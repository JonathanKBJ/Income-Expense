package models

import "database/sql"

// TransactionType represents the kind of financial transaction.
type TransactionType string

const (
	TypeIncome  TransactionType = "INCOME"
	TypeExpense TransactionType = "EXPENSE"
)

// ExpenseStatus represents the payment status of an expense.
type ExpenseStatus string

const (
	StatusPending ExpenseStatus = "PENDING"
	StatusPaid    ExpenseStatus = "PAID"
)

// Transaction is the core domain model representing a financial transaction.
// For INCOME transactions: Status and PaidAmount are always nil/null.
// For EXPENSE transactions: Status is required (PENDING/PAID), PaidAmount >= 0.
type Transaction struct {
	ID          string          `json:"id"`
	Type        TransactionType `json:"type"`
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Amount      float64         `json:"amount"`
	Date        string          `json:"date"`       // YYYY-MM-DD
	Status      *ExpenseStatus  `json:"status"`      // nil for INCOME
	PaidAmount  *float64        `json:"paidAmount"`  // nil for INCOME
	GroupID     *string         `json:"groupId"`     // Owner group
	UserID      *string         `json:"userId"`      // Creator user
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}

// TransactionRow is used for scanning database rows where nullable fields
// are represented as sql.Null* types.
type TransactionRow struct {
	ID          string
	Type        string
	Category    string
	Description string
	Amount      float64
	Date        string
	Status      sql.NullString
	PaidAmount  sql.NullFloat64
	GroupID     sql.NullString
	UserID      sql.NullString
	CreatedAt   string
	UpdatedAt   string
}

// ToTransaction converts a database row representation to the domain model.
func (r *TransactionRow) ToTransaction() Transaction {
	t := Transaction{
		ID:          r.ID,
		Type:        TransactionType(r.Type),
		Category:    r.Category,
		Description: r.Description,
		Amount:      r.Amount,
		Date:        r.Date,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	if r.Status.Valid {
		status := ExpenseStatus(r.Status.String)
		t.Status = &status
	}

	if r.PaidAmount.Valid {
		t.PaidAmount = &r.PaidAmount.Float64
	}

	if r.GroupID.Valid {
		t.GroupID = &r.GroupID.String
	}

	if r.UserID.Valid {
		t.UserID = &r.UserID.String
	}

	return t
}

// --- Request / Response DTOs ---

// CreateTransactionRequest is the payload for POST /api/transactions.
type CreateTransactionRequest struct {
	Type        TransactionType `json:"type"`
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Amount      float64         `json:"amount"`
	Date        string          `json:"date"`
	Status      *ExpenseStatus  `json:"status,omitempty"`
	PaidAmount  *float64        `json:"paidAmount,omitempty"`
}

// UpdateTransactionRequest is the payload for PATCH /api/transactions/{id}.
type UpdateTransactionRequest struct {
	Amount     *float64       `json:"amount,omitempty"`
	Status     *ExpenseStatus `json:"status,omitempty"`
	PaidAmount *float64       `json:"paidAmount,omitempty"`
}

// TransactionSummary holds aggregated dashboard metrics for a given month.
type TransactionSummary struct {
	TotalIncome  float64 `json:"totalIncome"`
	TotalPaid    float64 `json:"totalPaid"`
	TotalPending float64 `json:"totalPending"`
}

// TransactionsResponse is the envelope for GET /api/transactions.
type TransactionsResponse struct {
	Transactions []Transaction      `json:"transactions"`
	Summary      TransactionSummary `json:"summary"`
}

// ErrorResponse is a standard error payload returned by the API.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
