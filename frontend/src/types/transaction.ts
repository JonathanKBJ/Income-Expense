// =============================================================================
// Transaction Type System
// Uses TypeScript discriminated unions to enforce that:
//   - INCOME transactions never have status or paidAmount
//   - EXPENSE transactions always have status and paidAmount
// =============================================================================

/** Transaction type discriminator */
export type TransactionType = "INCOME" | "EXPENSE";

/** Expense payment status */
export type ExpenseStatus = "PENDING" | "PAID";

// --- Domain Models (match Go structs / DB schema) ---

interface BaseTransaction {
  id: string;
  category: string;
  description: string;
  amount: number;
  date: string; // YYYY-MM-DD
  createdAt: string;
  updatedAt: string;
}

/** Income transaction — status & paidAmount are compile-time prohibited */
export interface IncomeTransaction extends BaseTransaction {
  type: "INCOME";
  status: null;
  paidAmount: null;
}

/** Expense transaction — status & paidAmount are required */
export interface ExpenseTransaction extends BaseTransaction {
  type: "EXPENSE";
  status: ExpenseStatus;
  paidAmount: number;
}

/** Discriminated union — narrows via `type` field */
export type Transaction = IncomeTransaction | ExpenseTransaction;

// --- API Request Types ---

export interface CreateIncomeRequest {
  type: "INCOME";
  category: string;
  description: string;
  amount: number;
  date: string;
}

export interface CreateExpenseRequest {
  type: "EXPENSE";
  category: string;
  description: string;
  amount: number;
  date: string;
  status: ExpenseStatus;
  paidAmount: number;
}

export type CreateTransactionRequest = CreateIncomeRequest | CreateExpenseRequest;

export interface UpdateTransactionRequest {
  amount?: number;
  status?: ExpenseStatus;
  paidAmount?: number;
}

// --- API Response Types ---

export interface TransactionSummary {
  totalIncome: number;
  totalPaid: number;
  totalPending: number;
}

export interface TransactionsResponse {
  transactions: Transaction[];
  summary: TransactionSummary;
}

export interface ErrorResponse {
  error: string;
  details?: string;
}

// --- Form State ---

export interface TransactionFormState {
  type: TransactionType;
  category: string;
  description: string;
  amount: string; // string for controlled input binding
  date: string;
  status: ExpenseStatus;
  paidAmount: string; // string for controlled input binding
}

// --- Utility type guards ---

export function isExpense(t: Transaction): t is ExpenseTransaction {
  return t.type === "EXPENSE";
}

export function isIncome(t: Transaction): t is IncomeTransaction {
  return t.type === "INCOME";
}

// --- Categories ---

export const INCOME_CATEGORIES = [
  "Salary",
  "Freelance",
  "Investment",
  "Rental",
  "Gift",
  "Refund",
  "Other",
] as const;

export const EXPENSE_CATEGORIES = [
  "Food & Dining",
  "Transportation",
  "Housing",
  "Utilities",
  "Healthcare",
  "Entertainment",
  "Shopping",
  "Education",
  "Insurance",
  "Savings",
  "Other",
] as const;
