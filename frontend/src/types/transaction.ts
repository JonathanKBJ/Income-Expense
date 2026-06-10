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
  userId?: string;          // wallet owner
  createdById?: string;     // who recorded this
  receiptImage?: string;
  createdByUsername?: string; // recorder username (multi-member groups)
  ownerUsername?: string;     // wallet owner username (multi-member groups)
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
  userId?: string;       // optional: override wallet owner
  receiptImage?: string;
}

export interface CreateExpenseRequest {
  type: "EXPENSE";
  category: string;
  description: string;
  amount: number;
  date: string;
  status: ExpenseStatus;
  paidAmount: number;
  userId?: string;       // optional: override wallet owner
  receiptImage?: string;
}

export type CreateTransactionRequest = CreateIncomeRequest | CreateExpenseRequest;

export interface UpdateTransactionRequest {
  amount?: number;
  status?: ExpenseStatus;
  paidAmount?: number;
  userId?: string;       // optional: change wallet owner
  receiptImage?: string;
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

/** Lightweight transaction used for month-over-month comparison (no receipt image, status, or timestamps) */
export interface CompactTransaction {
  id: string;
  type: TransactionType;
  category: string;
  description: string;
  amount: number;
  userId?: string;
}

export interface CompactTransactionsResponse {
  transactions: CompactTransaction[];
}

export interface MonthlySummary {
  month: number;
  income: number;
  expense: number;
}

export interface CategorySummary {
  category: string;
  type: TransactionType;
  amount: number;
}

export interface AnnualSummaryResponse {
  year: number;
  totalIncome: number;
  totalExpense: number;
  netBalance: number;
  monthlyData: MonthlySummary[];
  categoryData: CategorySummary[];
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
  userId?: string;    // optional: wallet owner override
  receiptImage?: string;
}

// --- Month-over-Month Change ---

export interface MomChange {
  previousAmount: number;
  currentAmount: number;
  percentChange: number; // positive = increased, negative = decreased
  isNew: boolean;        // no matching transaction in previous month
  isUnchanged: boolean;  // exact same amount
  prevTxId: string;      // previous month's transaction ID (empty if new)
}

export type MomChangeMap = Record<string, MomChange>; // keyed by current tx ID

// --- Utility type guards ---

export function isExpense(t: Transaction): t is ExpenseTransaction {
  return t.type === "EXPENSE";
}

export function isIncome(t: Transaction): t is IncomeTransaction {
  return t.type === "INCOME";
}

// --- Wallet Summary ---

export interface WalletMemberSummary {
  userId: string;
  username: string;
  totalIncome: number;
  totalExpense: number;
  totalPaid: number;
  totalPending: number;
  netBalance: number;
}

export interface WalletSummaryResponse {
  month: number;
  year: number;
  members: WalletMemberSummary[];
  groupTotal: WalletMemberSummary;
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
