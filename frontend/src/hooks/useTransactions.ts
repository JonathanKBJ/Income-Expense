import { useCallback, useEffect, useMemo, useState } from "react";
import type {
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
function makeMomKey(tx: Transaction): string {
  const uid = tx.userId ?? "__nobody__";
  return `${uid}|${tx.type}|${tx.category}|${(tx.description || "").trim().toLowerCase()}`;
}

/**
 * Compute month-over-month change data by matching current transactions
 * against previous month transactions using the composite key.
 */
function computeMomChanges(
  currentTxs: Transaction[],
  prevTxs: Transaction[]
): MomChangeMap {
  const map: MomChangeMap = {};

  // Build lookup from previous month
  const prevLookup: Record<string, Transaction> = {};
  for (const pt of prevTxs) {
    const key = makeMomKey(pt);
    // If duplicate key in prev month, keep the first one (highest amount or first seen)
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

  // Previous month transactions for MoM comparison
  const [prevMonthTransactions, setPrevMonthTransactions] = useState<Transaction[]>([]);

  const momChanges = useMemo(
    () => computeMomChanges(transactions, prevMonthTransactions),
    [transactions, prevMonthTransactions]
  );

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const prev = getPreviousMonth(month, year);
      const [data, prevData] = await Promise.all([
        api.getTransactions(month, year),
        api.getTransactions(prev.month, prev.year),
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

  const create = useCallback(
    async (req: CreateTransactionRequest) => {
      await api.createTransaction(req);
      await refresh();
    },
    [refresh]
  );

  const createBatch = useCallback(
    async (reqs: CreateTransactionRequest[]) => {
      await api.createTransactionsBatch(reqs);
      await refresh();
    },
    [refresh]
  );

  const update = useCallback(
    async (id: string, req: UpdateTransactionRequest) => {
      await api.updateTransaction(id, req);
      await refresh();
    },
    [refresh]
  );

  const remove = useCallback(
    async (id: string) => {
      await api.deleteTransaction(id);
      await refresh();
    },
    [refresh]
  );

  const removeBatch = useCallback(
    async (ids: string[]) => {
      await api.deleteTransactionsBatch(ids);
      await refresh();
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