package models

type SyncDueMatchItem struct {
	TransactionID          string  `json:"transactionId"`
	TransactionDescription string  `json:"transactionDescription"`
	Category               string  `json:"category"`
	CurrentAmount          float64 `json:"currentAmount"`
	Date                   string  `json:"date"`
	Status                 string  `json:"status"`

	MatchedType            string  `json:"matchedType"` // "CREDIT_ACCOUNT" | "LOAN" | ""
	MatchedID              string  `json:"matchedId"`
	MatchedName            string  `json:"matchedName"`

	DueAmount              float64 `json:"dueAmount"`          // Minimum or installment due
	FullAmount             float64 `json:"fullAmount"`         // Full balance (for credit cards)
	SuggestedAmount        float64 `json:"suggestedAmount"`    // Default suggested amount
	SelectedAmountType     string  `json:"selectedAmountType"` // "DUE" | "FULL"
	Diff                   float64 `json:"diff"`               // SuggestedAmount - CurrentAmount
}

type SyncDueAvailableAccount struct {
	Type       string  `json:"type"` // "CREDIT_ACCOUNT" | "LOAN"
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	DueAmount  float64 `json:"dueAmount"`
	FullAmount float64 `json:"fullAmount"`
}

type SyncDuesPreviewResponse struct {
	Month             int                       `json:"month"`
	Year              int                       `json:"year"`
	Matches           []SyncDueMatchItem        `json:"matches"`
	AvailableAccounts []SyncDueAvailableAccount `json:"availableAccounts"`
}

type SyncDueUpdateItem struct {
	TransactionID   string  `json:"transactionId"`
	Amount          float64 `json:"amount"`
	CreditAccountID *string `json:"creditAccountId,omitempty"`
	LoanID          *string `json:"loanId,omitempty"`
}

type ApplySyncDuesRequest struct {
	Updates []SyncDueUpdateItem `json:"updates"`
}

type ApplySyncDuesResponse struct {
	UpdatedCount int `json:"updatedCount"`
}
