export type CreditAccountType = "CREDIT_CARD" | "CASH_CARD" | "PERSONAL_LOAN";
export type CreditStatus = "ACTIVE" | "CLOSED";
export type InstallmentStatus = "ACTIVE" | "COMPLETED";
export type CreditTxType = "CHARGE" | "PAYMENT";
export type CreditPaymentType = "FULL" | "MINIMUM" | "INSTALLMENT" | "CUSTOM";

export interface CreditAccount {
  id: string;
  name: string;
  type: CreditAccountType;
  bank: string;
  creditLimit: number;
  currentBalance: number;
  statementDay?: number | null;
  paymentDueDay?: number | null;
  interestRate: number;
  minPaymentRate: number;
  minPaymentFloor: number;
  status: CreditStatus;
  notes: string;
  groupId: string;
  userId: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreditInstallment {
  id: string;
  accountId: string;
  itemName: string;
  totalAmount: number;
  monthlyAmount: number;
  totalTerms: number;
  paidTerms: number;
  interestRate?: number;
  interestType?: "FLAT" | "EFFECTIVE" | "RECURRING";
  remainingBalance?: number;
  startDate: string;
  endDate?: string | null;
  status: InstallmentStatus;
  notes: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreditTransaction {
  id: string;
  accountId: string;
  type: CreditTxType;
  amount: number;
  date: string;
  description: string;
  paymentType?: CreditPaymentType | null;
  installmentId?: string | null;
  receiptImage?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface CreditAccountDetail extends CreditAccount {
  availableCredit: number;
  creditUtilization: number;
  unbilledInstallments: number;
  monthlyInstallmentDue: number;
  estimatedMinPayment: number;
  totalDueThisMonth: number;
  paidThisMonth: number;
  isPaidThisMonth: boolean;
  nextCycleEstimatedMin: number;
  nextCycleEstimatedInterest?: number;
  nextCycleEstimatedStatement?: number;
  activeInstallmentCount: number;
  installments: CreditInstallment[];
  transactions: CreditTransaction[];
}

export interface CreditDashboardSummary {
  totalCreditLimit: number;
  totalCurrentBalance: number;
  totalAvailableCredit: number;
  totalEstimatedDue: number;
  totalNextCycleOutstanding?: number;
  totalNextCycleEstimatedDue?: number;
  totalAccounts: number;
  activeAccounts: number;
}

export interface CreditAccountsResponse {
  accounts: CreditAccountDetail[];
  summary: CreditDashboardSummary;
}

export interface CreateCreditAccountRequest {
  name: string;
  type: CreditAccountType;
  bank?: string;
  creditLimit: number;
  currentBalance?: number;
  statementDay?: number;
  paymentDueDay?: number;
  interestRate?: number;
  minPaymentRate?: number;
  minPaymentFloor?: number;
  notes?: string;
}

export interface UpdateCreditAccountRequest {
  name?: string;
  type?: CreditAccountType;
  bank?: string;
  creditLimit?: number;
  currentBalance?: number;
  statementDay?: number;
  paymentDueDay?: number;
  interestRate?: number;
  minPaymentRate?: number;
  minPaymentFloor?: number;
  status?: CreditStatus;
  notes?: string;
}

export interface CreateCreditInstallmentRequest {
  itemName: string;
  totalAmount: number;
  monthlyAmount: number;
  totalTerms: number;
  paidTerms?: number;
  interestRate?: number;
  interestType?: "FLAT" | "EFFECTIVE" | "RECURRING";
  remainingBalance?: number;
  startDate: string;
  endDate?: string;
  notes?: string;
}

export interface UpdateCreditInstallmentRequest {
  itemName?: string;
  totalAmount?: number;
  monthlyAmount?: number;
  totalTerms?: number;
  paidTerms?: number;
  interestRate?: number;
  interestType?: "FLAT" | "EFFECTIVE" | "RECURRING";
  remainingBalance?: number;
  endDate?: string;
  status?: InstallmentStatus;
  notes?: string;
}

export interface CreateCreditTxRequest {
  type: CreditTxType;
  amount: number;
  date: string;
  description?: string;
  paymentType?: CreditPaymentType;
  installmentId?: string;
  receiptImage?: string;
}
