package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"expense-tracker/internal/database"
	"expense-tracker/internal/models"

	"github.com/google/uuid"
)

// TransactionRepository provides CRUD operations for transactions
// using the Repository pattern to decouple data access from business logic.
type TransactionRepository struct {
	db *database.DB
}

// NewTransactionRepository creates a new repository instance.
func NewTransactionRepository(db *database.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// GetByMonthYear retrieves all transactions for a given month/year and group.
func (r *TransactionRepository) GetByMonthYear(ctx context.Context, month, year int, groupID string) (*models.TransactionsResponse, error) {
	// Build date range: first day of month to last day of month
	startDate := fmt.Sprintf("%04d-%02d-01", year, month)
	// Use first day of next month as exclusive upper bound
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	endDate := fmt.Sprintf("%04d-%02d-01", nextYear, nextMonth)

	query := `
		SELECT id, type, category, description, amount, date,
		       status, paid_amount, group_id, user_id, created_at, updated_at
		FROM transactions
		WHERE date >= ? AND date < ? AND group_id = ?
		ORDER BY date DESC, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.Transaction
	summary := models.TransactionSummary{}

	for rows.Next() {
		var row models.TransactionRow
		err := rows.Scan(
			&row.ID,
			&row.Type,
			&row.Category,
			&row.Description,
			&row.Amount,
			&row.Date,
			&row.Status,
			&row.PaidAmount,
			&row.GroupID,
			&row.UserID,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction row: %w", err)
		}

		t := row.ToTransaction()
		transactions = append(transactions, t)

		// Aggregate summary metrics
		switch t.Type {
		case models.TypeIncome:
			summary.TotalIncome += t.Amount
		case models.TypeExpense:
			if t.Status != nil {
				switch *t.Status {
				case models.StatusPaid:
					if t.PaidAmount != nil {
						summary.TotalPaid += *t.PaidAmount
					}
				case models.StatusPending:
					summary.TotalPending += t.Amount
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows: %w", err)
	}

	// Return empty slice instead of nil for JSON serialization
	if transactions == nil {
		transactions = []models.Transaction{}
	}

	return &models.TransactionsResponse{
		Transactions: transactions,
		Summary:      summary,
	}, nil
}

// GetByID retrieves a single transaction by its ID and group ID.
func (r *TransactionRepository) GetByID(ctx context.Context, id, groupID string) (*models.Transaction, error) {
	query := `
		SELECT id, type, category, description, amount, date,
		       status, paid_amount, group_id, user_id, created_at, updated_at
		FROM transactions
		WHERE id = ? AND group_id = ?
	`

	var row models.TransactionRow
	err := r.db.QueryRowContext(ctx, query, id, groupID).Scan(
		&row.ID,
		&row.Type,
		&row.Category,
		&row.Description,
		&row.Amount,
		&row.Date,
		&row.Status,
		&row.PaidAmount,
		&row.GroupID,
		&row.UserID,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get transaction by id: %w", err)
	}

	t := row.ToTransaction()
	return &t, nil
}

// Create inserts a new transaction into the database for a specific user and group.
func (r *TransactionRepository) Create(ctx context.Context, req models.CreateTransactionRequest, userID, groupID string) (*models.Transaction, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	// Determine nullable field values based on transaction type
	var status *string
	var paidAmount *float64

	if req.Type == models.TypeExpense {
		if req.Status != nil {
			s := string(*req.Status)
			status = &s
		} else {
			// Default to PENDING for expenses
			s := string(models.StatusPending)
			status = &s
		}

		if req.PaidAmount != nil {
			paidAmount = req.PaidAmount
		} else {
			// Default to 0 for expenses
			zero := 0.0
			paidAmount = &zero
		}
	}
	// For INCOME: status and paidAmount remain nil

	query := `
		INSERT INTO transactions (id, type, category, description, amount, date, status, paid_amount, group_id, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		string(req.Type),
		req.Category,
		req.Description,
		req.Amount,
		req.Date,
		status,
		paidAmount,
		groupID,
		userID,
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert transaction: %w", err)
	}

	// Fetch and return the created record
	return r.GetByID(ctx, id, groupID)
}

// CreateBatch inserts multiple transactions in a single database transaction.
func (r *TransactionRepository) CreateBatch(ctx context.Context, requests []models.CreateTransactionRequest, userID, groupID string) error {
	if len(requests) == 0 {
		return nil
	}

	// Start a database transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	query := `
		INSERT INTO transactions (id, type, category, description, amount, date, status, paid_amount, group_id, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, req := range requests {
		id := uuid.New().String()
		var status *string
		var paidAmount *float64

		if req.Type == models.TypeExpense {
			if req.Status != nil {
				s := string(*req.Status)
				status = &s
			} else {
				s := string(models.StatusPending)
				status = &s
			}

			if req.PaidAmount != nil {
				paidAmount = req.PaidAmount
			} else {
				zero := 0.0
				paidAmount = &zero
			}
		}

		_, err := stmt.ExecContext(ctx,
			id,
			string(req.Type),
			req.Category,
			req.Description,
			req.Amount,
			req.Date,
			status,
			paidAmount,
			groupID,
			userID,
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert transaction in batch: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Update modifies an existing transaction belonging to a specific group.
func (r *TransactionRepository) Update(ctx context.Context, id, groupID string, req models.UpdateTransactionRequest) (*models.Transaction, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	// Build dynamic update query based on provided fields
	setClauses := []string{"updated_at = ?"}
	args := []interface{}{now}

	if req.Amount != nil {
		setClauses = append(setClauses, "amount = ?")
		args = append(args, *req.Amount)
	}

	if req.Status != nil {
		setClauses = append(setClauses, "status = ?")
		args = append(args, string(*req.Status))
	}

	if req.PaidAmount != nil {
		setClauses = append(setClauses, "paid_amount = ?")
		args = append(args, *req.PaidAmount)
	}

	args = append(args, id, groupID)

	query := fmt.Sprintf("UPDATE transactions SET %s WHERE id = ? AND group_id = ?",
		joinStrings(setClauses, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, nil // Not found
	}

	return r.GetByID(ctx, id, groupID)
}

// Delete removes a transaction by ID if it belongs to the specified group.
func (r *TransactionRepository) Delete(ctx context.Context, id, groupID string) error {
	query := `DELETE FROM transactions WHERE id = ? AND group_id = ?`

	result, err := r.db.ExecContext(ctx, query, id, groupID)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found")
	}

	return nil
}

// joinStrings joins a slice of strings with a separator.
// A simple helper to avoid importing strings package just for this.
func joinStrings(parts []string, sep string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += sep
		}
		result += part
	}
	return result
}
