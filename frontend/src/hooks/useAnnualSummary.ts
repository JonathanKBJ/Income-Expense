import { useCallback, useEffect, useState } from "react";
import type { AnnualSummaryResponse } from "../types/transaction";
import * as api from "../api/transactions";

interface UseAnnualSummaryReturn {
  summary: AnnualSummaryResponse | null;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

export function useAnnualSummary(year: number): UseAnnualSummaryReturn {
  const [summary, setSummary] = useState<AnnualSummaryResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getAnnualSummary(year);
      setSummary(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch annual summary");
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return {
    summary,
    loading,
    error,
    refresh,
  };
}
