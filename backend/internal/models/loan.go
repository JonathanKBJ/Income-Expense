package models

import "time"

// LoanType represents whether we borrowed or lent money.
type LoanType string

const (
	LoanBorrow LoanType = "BORROW"
	LoanLend   LoanType = "LEND"
)

// LoanEntryType classifies a movement within a loan.
type LoanEntryType string

const (
	EntryWithdrawal  LoanEntryType = "WITHDRAWAL"
	EntryDeposit     LoanEntryType = "DEPOSIT"
	EntryInstallment LoanEntryType = "INSTALLMENT"
)

// LoanStatus tracks whether the loan is still active.
type LoanStatus string

const (
	LoanActive LoanStatus = "ACTIVE"
	LoanClosed LoanStatus = "CLOSED"
)

// Loan represents a borrowing or lending record.
type Loan struct {
	ID                string    `json:"id"`
	Type              LoanType  `json:"type"`
	Name              string    `json:"name"`
	Counterparty      string    `json:"counterparty"`
	Principal         float64   `json:"principal"`
	TermMonths        *int      `json:"termMonths,omitempty"`
	InstallmentAmount *float64  `json:"installmentAmount,omitempty"`
	PaymentDay        *int      `json:"paymentDay,omitempty"`
	InterestRate      *float64  `json:"interestRate,omitempty"`
	StartDate         string    `json:"startDate"`
	EndDate           *string   `json:"endDate,omitempty"`
	Status            LoanStatus `json:"status"`
	Notes             string    `json:"notes"`
	GroupID           string    `json:"groupId"`
	UserID            string    `json:"userId"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// LoanEntry represents a single movement (withdrawal/deposit/installment) within a loan.
type LoanEntry struct {
	ID           string        `json:"id"`
	LoanID       string        `json:"loanId"`
	EntryType    LoanEntryType `json:"entryType"`
	Amount       float64       `json:"amount"`
	Date         string        `json:"date"`
	Description  string        `json:"description"`
	ReceiptImage *string       `json:"receiptImage,omitempty"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

// LoanDetail wraps a loan with its entries and computed summary fields.
type LoanDetail struct {
	Loan
	Entries           []LoanEntry `json:"entries"`
	TotalWithdrawn    float64     `json:"totalWithdrawn"`
	TotalDeposited    float64     `json:"totalDeposited"`
	TotalInstallments float64     `json:"totalInstallments"`
	Outstanding       float64     `json:"outstanding"`
	ProgressPercent   float64     `json:"progressPercent"`
}

// --- Request DTOs ---

type CreateLoanRequest struct {
	Type              LoanType `json:"type"`
	Name              string   `json:"name"`
	Counterparty      string   `json:"counterparty"`
	Principal         float64  `json:"principal"`
	TermMonths        *int     `json:"termMonths,omitempty"`
	InstallmentAmount *float64 `json:"installmentAmount,omitempty"`
	PaymentDay        *int     `json:"paymentDay,omitempty"`
	InterestRate      *float64 `json:"interestRate,omitempty"`
	StartDate         string   `json:"startDate"`
	EndDate           *string  `json:"endDate,omitempty"`
	Notes             string   `json:"notes"`
}

type UpdateLoanRequest struct {
	Name              *string    `json:"name,omitempty"`
	Counterparty      *string    `json:"counterparty,omitempty"`
	TermMonths        *int       `json:"termMonths,omitempty"`
	InstallmentAmount *float64   `json:"installmentAmount,omitempty"`
	PaymentDay        *int       `json:"paymentDay,omitempty"`
	InterestRate      *float64   `json:"interestRate,omitempty"`
	EndDate           *string    `json:"endDate,omitempty"`
	Status            *LoanStatus `json:"status,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
}

type CreateLoanEntryRequest struct {
	EntryType    LoanEntryType `json:"entryType"`
	Amount       float64       `json:"amount"`
	Date         string        `json:"date"`
	Description  string        `json:"description"`
	ReceiptImage *string       `json:"receiptImage,omitempty"`
}
