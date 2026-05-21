export type LoanType = "BORROW" | "LEND";
export type LoanEntryType = "WITHDRAWAL" | "DEPOSIT" | "INSTALLMENT";
export type LoanStatus = "ACTIVE" | "CLOSED";

export interface Loan {
  id: string;
  type: LoanType;
  name: string;
  counterparty: string;
  principal: number;
  termMonths?: number | null;
  installmentAmount?: number | null;
  paymentDay?: number | null;
  interestRate?: number | null;
  startDate: string;
  endDate?: string | null;
  status: LoanStatus;
  notes: string;
  groupId: string;
  userId: string;
  createdAt: string;
  updatedAt: string;
}

export interface LoanEntry {
  id: string;
  loanId: string;
  entryType: LoanEntryType;
  amount: number;
  date: string;
  description: string;
  receiptImage?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface LoanDetail extends Loan {
  entries: LoanEntry[];
  totalWithdrawn: number;
  totalDeposited: number;
  totalInstallments: number;
  outstanding: number;
  progressPercent: number;
}

export interface CreateLoanRequest {
  type: LoanType;
  name: string;
  counterparty: string;
  principal: number;
  termMonths?: number;
  installmentAmount?: number;
  paymentDay?: number;
  interestRate?: number;
  startDate: string;
  endDate?: string;
  notes?: string;
}

export interface CreateLoanEntryRequest {
  entryType: LoanEntryType;
  amount: number;
  date: string;
  description?: string;
  receiptImage?: string;
}
