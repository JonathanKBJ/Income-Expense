import type { 
  Category, 
  CreateCategoryRequest, 
  UpdateCategoryRequest 
} from "../types/category";
import { apiFetch } from "./client";

/**
 * Fetch categories, optionally filtered by type (INCOME or EXPENSE).
 * GET /api/categories?type={INCOME|EXPENSE}
 */
export async function getCategories(type?: string): Promise<Category[]> {
  const url = type ? `/api/categories?type=${type}` : "/api/categories";
  return apiFetch(url);
}

/**
 * Create a new category.
 * POST /api/categories
 */
export async function createCategory(req: CreateCategoryRequest): Promise<Category> {
  return apiFetch("/api/categories", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

/**
 * Update an existing category name.
 * PATCH /api/categories/{id}
 */
export async function updateCategory(id: string, req: UpdateCategoryRequest): Promise<Category> {
  return apiFetch(`/api/categories/${id}`, {
    method: "PATCH",
    body: JSON.stringify(req),
  });
}

/**
 * Delete a category by ID.
 * DELETE /api/categories/{id}
 */
export async function deleteCategory(id: string): Promise<void> {
  return apiFetch(`/api/categories/${id}`, {
    method: "DELETE",
  });
}
