package models

// Category represents an income or expense category that users can manage.
type Category struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      TransactionType `json:"type"` // INCOME or EXPENSE
	GroupID   *string         `json:"groupId"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
}

// CreateCategoryRequest is the payload for POST /api/categories.
type CreateCategoryRequest struct {
	Name string          `json:"name"`
	Type TransactionType `json:"type"`
}

// UpdateCategoryRequest is the payload for PATCH /api/categories/{id}.
type UpdateCategoryRequest struct {
	Name string `json:"name"`
}
