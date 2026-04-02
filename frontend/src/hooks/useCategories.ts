import { useCallback, useEffect, useState } from "react";
import type {
  Category,
  CreateCategoryRequest,
  UpdateCategoryRequest,
} from "../types/category";
import * as api from "../api/categories";

interface UseCategoriesReturn {
  categories: Category[];
  incomeCategories: Category[];
  expenseCategories: Category[];
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  create: (req: CreateCategoryRequest) => Promise<void>;
  update: (id: string, req: UpdateCategoryRequest) => Promise<void>;
  remove: (id: string) => Promise<string | null>;
}

/**
 * Custom hook that manages category state and all CRUD operations.
 */
export function useCategories(): UseCategoriesReturn {
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getCategories();
      setCategories(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch categories");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const incomeCategories = categories.filter((c) => c.type === "INCOME");
  const expenseCategories = categories.filter((c) => c.type === "EXPENSE");

  const create = useCallback(
    async (req: CreateCategoryRequest) => {
      await api.createCategory(req);
      await refresh();
    },
    [refresh]
  );

  const update = useCallback(
    async (id: string, req: UpdateCategoryRequest) => {
      await api.updateCategory(id, req);
      await refresh();
    },
    [refresh]
  );

  const remove = useCallback(
    async (id: string): Promise<string | null> => {
      try {
        await api.deleteCategory(id);
        await refresh();
        return null;
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Failed to delete";
        return msg;
      }
    },
    [refresh]
  );

  return {
    categories,
    incomeCategories,
    expenseCategories,
    loading,
    error,
    refresh,
    create,
    update,
    remove,
  };
}
