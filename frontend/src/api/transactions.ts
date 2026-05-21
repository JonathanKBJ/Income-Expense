import type {
  CreateTransactionRequest,
  Transaction,
  TransactionsResponse,
  UpdateTransactionRequest,
  AnnualSummaryResponse,
} from "../types/transaction";
import { apiFetch } from "./client";

/**
 * Fetch transactions for a specific month/year with summary metrics.
 * GET /api/transactions?month={M}&year={Y}
 */
export async function getTransactions(
  month: number,
  year: number
): Promise<TransactionsResponse> {
  return apiFetch(`/api/transactions?month=${month}&year=${year}`);
}

/**
 * Create a new transaction.
 * POST /api/transactions
 */
export async function createTransaction(
  req: CreateTransactionRequest
): Promise<Transaction> {
  return apiFetch("/api/transactions", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

/**
 * Create multiple transactions in a batch.
 * POST /api/transactions/batch
 */
export async function createTransactionsBatch(
  reqs: CreateTransactionRequest[]
): Promise<{ message: string }> {
  return apiFetch("/api/transactions/batch", {
    method: "POST",
    body: JSON.stringify(reqs),
  });
}

/**
 * Copy transactions to a different group.
 * POST /api/transactions/batch-to-group
 */
export async function createTransactionsBatchToGroup(
  reqs: CreateTransactionRequest[],
  targetGroupId: string
): Promise<{ message: string }> {
  return apiFetch("/api/transactions/batch-to-group", {
    method: "POST",
    body: JSON.stringify({ transactions: reqs, targetGroupId }),
  });
}

/**
 * Update an existing transaction's status and/or paidAmount.
 * PATCH /api/transactions/{id}
 */
export async function updateTransaction(
  id: string,
  req: UpdateTransactionRequest
): Promise<Transaction> {
  return apiFetch(`/api/transactions/${id}`, {
    method: "PATCH",
    body: JSON.stringify(req),
  });
}

/**
 * Delete a transaction by ID.
 * DELETE /api/transactions/{id}
 */
export async function deleteTransaction(id: string): Promise<void> {
  return apiFetch(`/api/transactions/${id}`, {
    method: "DELETE",
  });
}

/**
 * Delete multiple transactions in a batch.
 * DELETE /api/transactions/batch
 */
export async function deleteTransactionsBatch(ids: string[]): Promise<void> {
  return apiFetch("/api/transactions/batch", {
    method: "DELETE",
    body: JSON.stringify({ ids }),
  });
}

/**
 * Fetch annual summary statistics.
 * GET /api/transactions/annual?year={Y}
 */
export async function getAnnualSummary(year: number): Promise<AnnualSummaryResponse> {
  return apiFetch(`/api/transactions/annual?year=${year}`);
}
