import { useCallback, useEffect, useMemo, useState } from "react";
import type {
  CompactTransaction,
  CreateTransactionRequest,
  MomChangeMap,
  Transaction,
  TransactionSummary,
  UpdateTransactionRequest,
} from "../types/transaction";
import * as api from "../api/transactions";

/**
 * Compute the previous month/year given a current month and year.
 */
function getPreviousMonth(month: number, year: number): { month: number; year: number } {
  if (month === 1) return { month: 12, year: year - 1 };
  return { month: month - 1, year };
}

/**
 * Build a composite key for matching transactions across months.
 * Matches by userId (wallet owner) + category + description + type.
 * This mirrors the clone/copy logic which preserves these fields.
 */
function makeMomKey(tx: CompactTransaction): string {
  const uid = tx.userId ?? "__nobody__";
  return `${uid}|${tx.type}|${tx.category}|${(tx.description || "").trim().toLowerCase()}`;
}

/**
 * Compute month-over-month change data by matching current transactions
 * against previous month transactions using the composite key.
 */
function computeMomChanges(
  currentTxs: Transaction[],
  prevTxs: CompactTransaction[]
): MomChangeMap {
  const map: MomChangeMap = {};

  // Build lookup from previous month
  const prevLookup: Record<string, CompactTransaction> = {};
  for (const pt of prevTxs) {
    const key = makeMomKey(pt);
    if (!prevLookup[key]) {
      prevLookup[key] = pt;
    }
  }

  for (const ct of currentTxs) {
    const key = makeMomKey(ct);
    const prev = prevLookup[key];
    if (prev) {
      const pct = prev.amount > 0
        ? Math.round(((ct.amount - prev.amount) / prev.amount) * 1000) / 10
        : ct.amount > 0 ? 100 : 0;
      map[ct.id] = {
        previousAmount: prev.amount,
        currentAmount: ct.amount,
        percentChange: pct,
        isNew: false,
        isUnchanged: ct.amount === prev.amount,
        prevTxId: prev.id,
      };
    }
  }

  return map;
}

/**
 * Compute summary from a transactions array (mirrors backend aggregation logic).
 */
function computeSummary(transactions: Transaction[]): TransactionSummary {
  const summary: TransactionSummary = { totalIncome: 0, totalPaid: 0, totalPending: 0 };
  for (const t of transactions) {
    if (t.type === "INCOME") {
      summary.totalIncome += t.amount;
    } else {
      if (t.status === "PAID") {
        summary.totalPaid += t.paidAmount ?? 0;
      } else if (t.status === "PENDING") {
        summary.totalPending += t.amount;
      }
    }
  }
  return summary;
}

interface UseTransactionsReturn {
  transactions: Transaction[];
  summary: TransactionSummary;
  loading: boolean;
  error: string | null;
  month: number;
  year: number;
  setMonth: (m: number) => void;
  setYear: (y: number) => void;
  refresh: () => Promise<void>;
  create: (req: CreateTransactionRequest) => Promise<void>;
  createBatch: (reqs: CreateTransactionRequest[]) => Promise<void>;
  update: (id: string, req: UpdateTransactionRequest) => Promise<void>;
  remove: (id: string) => Promise<void>;
  removeBatch: (ids: string[]) => Promise<void>;
  momChanges: MomChangeMap;
}

const defaultSummary: TransactionSummary = {
  totalIncome: 0,
  totalPaid: 0,
  totalPending: 0,
};

/**
 * Custom hook that manages transaction state, month/year navigation,
 * and all CRUD operations. Acts as the central state manager for the app.
 *
 * Performance optimizations:
 * - Previous month uses a lightweight "compact" endpoint (no receipt images, no JOINs).
 * - Create/Update/Delete use optimistic UI updates — local state is updated immediately
 *   after the API call succeeds, avoiding a full re-fetch. Falls back to full refresh on error.
 */
export function useTransactions(): UseTransactionsReturn {
  const [month, setMonth] = useState(() => {
    const saved = localStorage.getItem("filter_month");
    return saved ? parseInt(saved, 10) : new Date().getMonth() + 1;
  });
  const [year, setYear] = useState(() => {
    const saved = localStorage.getItem("filter_year");
    return saved ? parseInt(saved, 10) : new Date().getFullYear();
  });
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [summary, setSummary] = useState<TransactionSummary>(defaultSummary);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Previous month transactions for MoM comparison (compact format)
  const [prevMonthTransactions, setPrevMonthTransactions] = useState<CompactTransaction[]>([]);

  const momChanges = useMemo(
    () => computeMomChanges(transactions, prevMonthTransactions),
    [transactions, prevMonthTransactions]
  );

  /**
   * Full refresh — fetches current month (full) + previous month (compact) in parallel.
   * Called on month/year change or as fallback after a failed CRUD operation.
   */
  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const prev = getPreviousMonth(month, year);
      const [data, prevData] = await Promise.all([
        api.getTransactions(month, year),
        api.getTransactionsCompact(prev.month, prev.year),
      ]);
      setTransactions(data.transactions);
      setSummary(data.summary);
      setPrevMonthTransactions(prevData.transactions);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch");
    } finally {
      setLoading(false);
    }
  }, [month, year]);

  // Fetch when month/year changes
  useEffect(() => {
    refresh();
    localStorage.setItem("filter_month", month.toString());
    localStorage.setItem("filter_year", year.toString());
  }, [refresh, month, year]);

  /**
   * Create a transaction with optimistic UI update.
   * Inserts the returned transaction into local state without a full re-fetch.
   * Falls back to full refresh on error.
   */
  const create = useCallback(
    async (req: CreateTransactionRequest) => {
      try {
        const created = await api.createTransaction(req);
        setTransactions((prev) => {
          // Insert at the front to match server sort order (date DESC, created_at DESC)
          const next = [created, ...prev];
          setSummary(computeSummary(next));
          return next;
        });
      } catch (err) {
        // On error, fall back to full refresh to ensure consistency
        await refresh();
        throw err;
      }
    },
    [refresh]
  );

  /**
   * Create multiple transactions in batch.
   * Falls back to full refresh since the batch endpoint doesn't return individual transactions.
   */
  const createBatch = useCallback(
    async (reqs: CreateTransactionRequest[]) => {
      await api.createTransactionsBatch(reqs);
      await refresh();
    },
    [refresh]
  );

  /**
   * Update a transaction with optimistic UI update.
   * Replaces the updated transaction in local state without a full re-fetch.
   * Falls back to full refresh on error.
   */
  const update = useCallback(
    async (id: string, req: UpdateTransactionRequest) => {
      try {
        const updated = await api.updateTransaction(id, req);
        setTransactions((prev) => {
          const next = prev.map((t) => (t.id === id ? updated : t));
          setSummary(computeSummary(next));
          return next;
        });
      } catch (err) {
        await refresh();
        throw err;
      }
    },
    [refresh]
  );

  /**
   * Delete a transaction with optimistic UI update.
   * Removes the transaction from local state without a full re-fetch.
   * Falls back to full refresh on error.
   */
  const remove = useCallback(
    async (id: string) => {
      try {
        await api.deleteTransaction(id);
        setTransactions((prev) => {
          const next = prev.filter((t) => t.id !== id);
          setSummary(computeSummary(next));
          return next;
        });
      } catch (err) {
        await refresh();
        throw err;
      }
    },
    [refresh]
  );

  /**
   * Delete multiple transactions in batch with optimistic UI update.
   * Filters out the deleted IDs from local state; falls back to full refresh on error.
   */
  const removeBatch = useCallback(
    async (ids: string[]) => {
      try {
        await api.deleteTransactionsBatch(ids);
        const idSet = new Set(ids);
        setTransactions((prev) => {
          const next = prev.filter((t) => !idSet.has(t.id));
          setSummary(computeSummary(next));
          return next;
        });
      } catch (err) {
        await refresh();
        throw err;
      }
    },
    [refresh]
  );

  return {
    transactions,
    summary,
    loading,
    error,
    month,
    year,
    setMonth,
    setYear,
    refresh,
    create,
    createBatch,
    update,
    remove,
    removeBatch,
    momChanges,
  };
}