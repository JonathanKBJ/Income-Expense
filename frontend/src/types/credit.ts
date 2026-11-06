export type CreditAccountType = "CREDIT_CARD" | "CASH_CARD" | "PERSONAL_LOAN";
export type CreditStatus = "ACTIVE" | "CLOSED";
export type InstallmentStatus = "ACTIVE" | "COMPLETED";
export type CreditTxType = "CHARGE" | "PAYMENT";
export type CreditPaymentType = "FULL" | "MINIMUM" | "INSTALLMENT" | "CUSTOM";
export type CreditBalanceMode = "LEGACY" | "PRINCIPAL_SPLIT";
export type CreditLimitBasis = "PRINCIPAL" | "TOTAL";
export type CreditAllocationOrder = "FEES_INTEREST_PRINCIPAL" | "INTEREST_FEES_PRINCIPAL";
export type CreditAllocationSource = "ACTUAL" | "RULE";

export type CreditMinPaymentCalcMode =
  | "BALANCE_PERCENT"
  | "PRINCIPAL_PERCENT_PLUS_INTEREST"
  | "INTEREST_ONLY";

export interface CreditAccount {
  id: string;
  name: string;
  type: CreditAccountType;
  bank: string;
  creditLimit: number;
  currentBalance: number;
  balanceMode?: CreditBalanceMode;
  interestBalance?: number;
  feeBalance?: number;
  creditLimitBasis?: CreditLimitBasis;
  allocationOrder?: CreditAllocationOrder;
  balanceAsOfDate?: string | null;
  balanceNote?: string;
  statementDay?: number | null;
  paymentDueDay?: number | null;
  statementDate?: string | null;
  statementBalance?: number | null;
  statementMinimumDue?: number | null;
  statementDueDate?: string | null;
  interestRate: number;
  minPaymentRate: number;
  minPaymentFloor: number;
  minPaymentCalcMode?: CreditMinPaymentCalcMode;
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
  earlySettlementDate?: string | null;
  earlySettlementNote?: string;
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
  principalAmount?: number | null;
  interestAmount?: number | null;
  feeAmount?: number | null;
  allocationSource?: CreditAllocationSource | null;
  createdAt: string;
  updatedAt: string;
}

export interface CreditAccountDetail extends CreditAccount {
  principalBalance?: number;
  availableCredit: number;
  creditUtilization: number;
  unbilledInstallments: number;
  monthlyInstallmentDue: number;
  estimatedMinPayment: number;
  usesStatementMinimum: boolean;
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
  statementDate?: string | null;
  statementBalance?: number | null;
  statementMinimumDue?: number | null;
  statementDueDate?: string | null;
  interestRate?: number;
  minPaymentRate?: number;
  minPaymentFloor?: number;
  minPaymentCalcMode?: CreditMinPaymentCalcMode;
  notes?: string;
}

export interface UpdateCreditAccountRequest {
  name?: string;
  type?: CreditAccountType;
  bank?: string;
  creditLimit?: number;
  currentBalance?: number;
  creditLimitBasis?: CreditLimitBasis;
  allocationOrder?: CreditAllocationOrder;
  statementDay?: number;
  paymentDueDay?: number;
  statementDate?: string | null;
  statementBalance?: number | null;
  statementMinimumDue?: number | null;
  statementDueDate?: string | null;
  clearStatement?: boolean;
  interestRate?: number;
  minPaymentRate?: number;
  minPaymentFloor?: number;
  minPaymentCalcMode?: CreditMinPaymentCalcMode;
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

export interface SettleInstallmentEarlyRequest {
  settlementDate: string;
  note?: string;
}

export interface CreateCreditTxRequest {
  type: CreditTxType;
  amount: number;
  date: string;
  description?: string;
  paymentType?: CreditPaymentType;
  installmentId?: string;
  receiptImage?: string;
  principalAmount?: number;
  interestAmount?: number;
  feeAmount?: number;
  allocationSource?: CreditAllocationSource;
}

export interface CreditHistoricalAllocation {
  transactionId: string;
  principalAmount: number;
  interestAmount: number;
  feeAmount: number;
}

export interface ReconcileCreditAccountRequest {
  asOfDate: string;
  principalBalance: number;
  interestBalance: number;
  feeBalance: number;
  note: string;
  creditLimitBasis?: CreditLimitBasis;
  allocationOrder?: CreditAllocationOrder;
  historicalAllocations?: CreditHistoricalAllocation[];
}
