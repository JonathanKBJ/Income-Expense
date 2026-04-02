import { useCallback, useEffect, useState } from "react";
import type {
  CreateTransactionRequest,
  Transaction,
  TransactionSummary,
  UpdateTransactionRequest,
} from "../types/transaction";
import * as api from "../api/transactions";

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
  const now = new Date();
  const [month, setMonth] = useState(now.getMonth() + 1); // 1-12
  const [year, setYear] = useState(now.getFullYear());
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [summary, setSummary] = useState<TransactionSummary>(defaultSummary);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getTransactions(month, year);
      setTransactions(data.transactions);
      setSummary(data.summary);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch");
    } finally {
      setLoading(false);
    }
  }, [month, year]);

  // Fetch when month/year changes
  useEffect(() => {
    refresh();
  }, [refresh]);

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
  };
}
