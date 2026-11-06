package models

import "time"

// CreditAccountType represents the credit facility type.
type CreditAccountType string

const (
	CreditAccountCreditCard   CreditAccountType = "CREDIT_CARD"
	CreditAccountCashCard     CreditAccountType = "CASH_CARD"
	CreditAccountPersonalLoan CreditAccountType = "PERSONAL_LOAN"
)

// CreditStatus represents whether the credit account is active or closed.
type CreditStatus string

const (
	CreditStatusActive CreditStatus = "ACTIVE"
	CreditStatusClosed CreditStatus = "CLOSED"
)

// InstallmentStatus tracks the status of a fixed installment plan.
type InstallmentStatus string

const (
	InstallmentStatusActive    InstallmentStatus = "ACTIVE"
	InstallmentStatusCompleted InstallmentStatus = "COMPLETED"
)

// CreditTxType differentiates charges/withdrawals from payments.
type CreditTxType string

const (
	CreditTxCharge  CreditTxType = "CHARGE"
	CreditTxPayment CreditTxType = "PAYMENT"
)

// CreditPaymentType specifies how a payment was settled.
type CreditPaymentType string

const (
	CreditPayFull        CreditPaymentType = "FULL"
	CreditPayMinimum     CreditPaymentType = "MINIMUM"
	CreditPayInstallment CreditPaymentType = "INSTALLMENT"
	CreditPayCustom      CreditPaymentType = "CUSTOM"
)

// CreditMinPaymentCalcMode defines how the monthly minimum payment and interest are calculated.
type CreditMinPaymentCalcMode string

const (
	CalcModeBalancePercent        CreditMinPaymentCalcMode = "BALANCE_PERCENT"
	CalcModePrincipalPlusInterest CreditMinPaymentCalcMode = "PRINCIPAL_PERCENT_PLUS_INTEREST"
	CalcModeInterestOnly          CreditMinPaymentCalcMode = "INTEREST_ONLY"
)

type CreditBalanceMode string
type CreditLimitBasis string
type CreditAllocationOrder string
type CreditAllocationSource string

const (
	CreditBalanceLegacy             CreditBalanceMode      = "LEGACY"
	CreditBalancePrincipalSplit     CreditBalanceMode      = "PRINCIPAL_SPLIT"
	CreditLimitPrincipal            CreditLimitBasis       = "PRINCIPAL"
	CreditLimitTotal                CreditLimitBasis       = "TOTAL"
	AllocationFeesInterestPrincipal CreditAllocationOrder  = "FEES_INTEREST_PRINCIPAL"
	AllocationInterestFeesPrincipal CreditAllocationOrder  = "INTEREST_FEES_PRINCIPAL"
	AllocationActual                CreditAllocationSource = "ACTUAL"
	AllocationRule                  CreditAllocationSource = "RULE"
)

// CreditAccount represents a credit card or personal loan account.
type CreditAccount struct {
	ID                  string                   `json:"id"`
	Name                string                   `json:"name"`
	Type                CreditAccountType        `json:"type"`
	Bank                string                   `json:"bank"`
	CreditLimit         float64                  `json:"creditLimit"`
	CurrentBalance      float64                  `json:"currentBalance"`
	BalanceMode         CreditBalanceMode        `json:"balanceMode"`
	InterestBalance     float64                  `json:"interestBalance"`
	FeeBalance          float64                  `json:"feeBalance"`
	CreditLimitBasis    CreditLimitBasis         `json:"creditLimitBasis"`
	AllocationOrder     CreditAllocationOrder    `json:"allocationOrder"`
	BalanceAsOfDate     *string                  `json:"balanceAsOfDate"`
	BalanceNote         string                   `json:"balanceNote"`
	StatementDay        *int                     `json:"statementDay,omitempty"`
	PaymentDueDay       *int                     `json:"paymentDueDay,omitempty"`
	StatementDate       *string                  `json:"statementDate"`
	StatementBalance    *float64                 `json:"statementBalance"`
	StatementMinimumDue *float64                 `json:"statementMinimumDue"`
	StatementDueDate    *string                  `json:"statementDueDate"`
	InterestRate        float64                  `json:"interestRate"`
	MinPaymentRate      float64                  `json:"minPaymentRate"`
	MinPaymentFloor     float64                  `json:"minPaymentFloor"`
	MinPaymentCalcMode  CreditMinPaymentCalcMode `json:"minPaymentCalcMode"`
	Status              CreditStatus             `json:"status"`
	Notes               string                   `json:"notes"`
	GroupID             string                   `json:"groupId"`
	UserID              string                   `json:"userId"`
	CreatedAt           time.Time                `json:"createdAt"`
	UpdatedAt           time.Time                `json:"updatedAt"`
}

// CreditInstallment represents a fixed term installment plan under a credit account.
type CreditInstallment struct {
	ID                  string            `json:"id"`
	AccountID           string            `json:"accountId"`
	ItemName            string            `json:"itemName"`
	TotalAmount         float64           `json:"totalAmount"`
	MonthlyAmount       float64           `json:"monthlyAmount"`
	TotalTerms          int               `json:"totalTerms"`
	PaidTerms           int               `json:"paidTerms"`
	InterestRate        float64           `json:"interestRate"`
	InterestType        string            `json:"interestType"`
	RemainingBalance    float64           `json:"remainingBalance"`
	StartDate           string            `json:"startDate"`
	EndDate             *string           `json:"endDate,omitempty"`
	Status              InstallmentStatus `json:"status"`
	Notes               string            `json:"notes"`
	EarlySettlementDate *string           `json:"earlySettlementDate"`
	EarlySettlementNote string            `json:"earlySettlementNote"`
	CreatedAt           time.Time         `json:"createdAt"`
	UpdatedAt           time.Time         `json:"updatedAt"`
}

// CreditTransaction records an expenditure or payment.
type CreditTransaction struct {
	ID               string                  `json:"id"`
	AccountID        string                  `json:"accountId"`
	Type             CreditTxType            `json:"type"`
	Amount           float64                 `json:"amount"`
	Date             string                  `json:"date"`
	Description      string                  `json:"description"`
	PaymentType      *CreditPaymentType      `json:"paymentType,omitempty"`
	InstallmentID    *string                 `json:"installmentId,omitempty"`
	ReceiptImage     *string                 `json:"receiptImage,omitempty"`
	PrincipalAmount  *float64                `json:"principalAmount,omitempty"`
	InterestAmount   *float64                `json:"interestAmount,omitempty"`
	FeeAmount        *float64                `json:"feeAmount,omitempty"`
	AllocationSource *CreditAllocationSource `json:"allocationSource,omitempty"`
	CreatedAt        time.Time               `json:"createdAt"`
	UpdatedAt        time.Time               `json:"updatedAt"`
}

// CreditAccountDetail wraps a credit account with installments, transactions, and computed metrics.
type CreditAccountDetail struct {
	CreditAccount
	PrincipalBalance            float64             `json:"principalBalance"`
	AvailableCredit             float64             `json:"availableCredit"`
	CreditUtilization           float64             `json:"creditUtilization"`
	UnbilledInstallments        float64             `json:"unbilledInstallments"`
	MonthlyInstallmentDue       float64             `json:"monthlyInstallmentDue"`
	TotalDueThisMonth           float64             `json:"totalDueThisMonth"`
	EstimatedMinPayment         float64             `json:"estimatedMinPayment"`
	UsesStatementMinimum        bool                `json:"usesStatementMinimum"`
	PaidThisMonth               float64             `json:"paidThisMonth"`
	IsPaidThisMonth             bool                `json:"isPaidThisMonth"`
	NextCycleEstimatedMin       float64             `json:"nextCycleEstimatedMin"`
	NextCycleEstimatedInterest  float64             `json:"nextCycleEstimatedInterest"`
	NextCycleEstimatedStatement float64             `json:"nextCycleEstimatedStatement"`
	ActiveInstallmentCount      int                 `json:"activeInstallmentCount"`
	Installments                []CreditInstallment `json:"installments"`
	Transactions                []CreditTransaction `json:"transactions"`
}

// CreditDashboardSummary represents the top-level aggregates for the credit cards & loans dashboard.
type CreditDashboardSummary struct {
	TotalCreditLimit           float64 `json:"totalCreditLimit"`
	TotalCurrentBalance        float64 `json:"totalCurrentBalance"`
	TotalAvailableCredit       float64 `json:"totalAvailableCredit"`
	TotalEstimatedDue          float64 `json:"totalEstimatedDue"`
	TotalNextCycleOutstanding  float64 `json:"totalNextCycleOutstanding"`
	TotalNextCycleEstimatedDue float64 `json:"totalNextCycleEstimatedDue"`
	TotalAccounts              int     `json:"totalAccounts"`
	ActiveAccounts             int     `json:"activeAccounts"`
}

// --- Request DTOs ---

type CreateCreditAccountRequest struct {
	Name                string                    `json:"name"`
	Type                CreditAccountType         `json:"type"`
	Bank                string                    `json:"bank"`
	CreditLimit         float64                   `json:"creditLimit"`
	CurrentBalance      float64                   `json:"currentBalance"`
	StatementDay        *int                      `json:"statementDay,omitempty"`
	PaymentDueDay       *int                      `json:"paymentDueDay,omitempty"`
	StatementDate       *string                   `json:"statementDate,omitempty"`
	StatementBalance    *float64                  `json:"statementBalance,omitempty"`
	StatementMinimumDue *float64                  `json:"statementMinimumDue,omitempty"`
	StatementDueDate    *string                   `json:"statementDueDate,omitempty"`
	InterestRate        float64                   `json:"interestRate"`
	MinPaymentRate      float64                   `json:"minPaymentRate"`
	MinPaymentFloor     float64                   `json:"minPaymentFloor"`
	MinPaymentCalcMode  *CreditMinPaymentCalcMode `json:"minPaymentCalcMode,omitempty"`
	Notes               string                    `json:"notes"`
}

type UpdateCreditAccountRequest struct {
	Name                *string                   `json:"name,omitempty"`
	Type                *CreditAccountType        `json:"type,omitempty"`
	Bank                *string                   `json:"bank,omitempty"`
	CreditLimit         *float64                  `json:"creditLimit,omitempty"`
	CurrentBalance      *float64                  `json:"currentBalance,omitempty"`
	BalanceMode         *CreditBalanceMode        `json:"balanceMode,omitempty"`
	InterestBalance     *float64                  `json:"interestBalance,omitempty"`
	FeeBalance          *float64                  `json:"feeBalance,omitempty"`
	CreditLimitBasis    *CreditLimitBasis         `json:"creditLimitBasis,omitempty"`
	AllocationOrder     *CreditAllocationOrder    `json:"allocationOrder,omitempty"`
	BalanceAsOfDate     *string                   `json:"balanceAsOfDate,omitempty"`
	BalanceNote         *string                   `json:"balanceNote,omitempty"`
	StatementDay        *int                      `json:"statementDay,omitempty"`
	PaymentDueDay       *int                      `json:"paymentDueDay,omitempty"`
	StatementDate       *string                   `json:"statementDate,omitempty"`
	StatementBalance    *float64                  `json:"statementBalance,omitempty"`
	StatementMinimumDue *float64                  `json:"statementMinimumDue,omitempty"`
	StatementDueDate    *string                   `json:"statementDueDate,omitempty"`
	ClearStatement      bool                      `json:"clearStatement,omitempty"`
	InterestRate        *float64                  `json:"interestRate,omitempty"`
	MinPaymentRate      *float64                  `json:"minPaymentRate,omitempty"`
	MinPaymentFloor     *float64                  `json:"minPaymentFloor,omitempty"`
	MinPaymentCalcMode  *CreditMinPaymentCalcMode `json:"minPaymentCalcMode,omitempty"`
	Status              *CreditStatus             `json:"status,omitempty"`
	Notes               *string                   `json:"notes,omitempty"`
}

type CreateCreditInstallmentRequest struct {
	ItemName         string   `json:"itemName"`
	TotalAmount      float64  `json:"totalAmount"`
	MonthlyAmount    float64  `json:"monthlyAmount"`
	TotalTerms       int      `json:"totalTerms"`
	PaidTerms        int      `json:"paidTerms"`
	InterestRate     *float64 `json:"interestRate,omitempty"`
	InterestType     *string  `json:"interestType,omitempty"`
	RemainingBalance *float64 `json:"remainingBalance,omitempty"`
	StartDate        string   `json:"startDate"`
	EndDate          *string  `json:"endDate,omitempty"`
	Notes            string   `json:"notes"`
}

type UpdateCreditInstallmentRequest struct {
	ItemName         *string            `json:"itemName,omitempty"`
	TotalAmount      *float64           `json:"totalAmount,omitempty"`
	MonthlyAmount    *float64           `json:"monthlyAmount,omitempty"`
	TotalTerms       *int               `json:"totalTerms,omitempty"`
	PaidTerms        *int               `json:"paidTerms,omitempty"`
	InterestRate     *float64           `json:"interestRate,omitempty"`
	InterestType     *string            `json:"interestType,omitempty"`
	RemainingBalance *float64           `json:"remainingBalance,omitempty"`
	EndDate          *string            `json:"endDate,omitempty"`
	Status           *InstallmentStatus `json:"status,omitempty"`
	Notes            *string            `json:"notes,omitempty"`
}

type SettleCreditInstallmentRequest struct {
	SettlementDate string `json:"settlementDate"`
	Note           string `json:"note,omitempty"`
}

type CreateCreditTxRequest struct {
	Type             CreditTxType            `json:"type"`
	Amount           float64                 `json:"amount"`
	Date             string                  `json:"date"`
	Description      string                  `json:"description"`
	PaymentType      *CreditPaymentType      `json:"paymentType,omitempty"`
	InstallmentID    *string                 `json:"installmentId,omitempty"`
	ReceiptImage     *string                 `json:"receiptImage,omitempty"`
	PrincipalAmount  *float64                `json:"principalAmount,omitempty"`
	InterestAmount   *float64                `json:"interestAmount,omitempty"`
	FeeAmount        *float64                `json:"feeAmount,omitempty"`
	AllocationSource *CreditAllocationSource `json:"allocationSource,omitempty"`
}

type CreditHistoricalAllocation struct {
	TransactionID   string  `json:"transactionId"`
	PrincipalAmount float64 `json:"principalAmount"`
	InterestAmount  float64 `json:"interestAmount"`
	FeeAmount       float64 `json:"feeAmount"`
}

type ReconcileCreditAccountRequest struct {
	AsOfDate              string                       `json:"asOfDate"`
	PrincipalBalance      float64                      `json:"principalBalance"`
	InterestBalance       float64                      `json:"interestBalance"`
	FeeBalance            float64                      `json:"feeBalance"`
	Note                  string                       `json:"note"`
	CreditLimitBasis      *CreditLimitBasis            `json:"creditLimitBasis,omitempty"`
	AllocationOrder       *CreditAllocationOrder       `json:"allocationOrder,omitempty"`
	HistoricalAllocations []CreditHistoricalAllocation `json:"historicalAllocations,omitempty"`
}
