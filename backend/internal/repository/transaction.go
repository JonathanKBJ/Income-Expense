package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
// Includes createdByUsername only when the group has more than 1 member.
func (r *TransactionRepository) GetByMonthYear(ctx context.Context, month, year int, groupID string) (*models.TransactionsResponse, error) {
	cycle := fmt.Sprintf("%04d-%02d", year, month)

	query := `
		SELECT t.id, t.type, t.category, t.description, t.amount, t.date,
		       t.status, t.paid_amount, t.group_id, t.user_id, t.created_by, t.receipt_image,
		       t.billing_cycle,
		       rec_u.username as created_by_username,
		       own_u.username as owner_username,
		       t.created_at, t.updated_at
		FROM transactions t
		LEFT JOIN users rec_u ON t.created_by = rec_u.id
		LEFT JOIN users own_u ON t.user_id = own_u.id
		WHERE COALESCE(t.billing_cycle, strftime('%Y-%m', t.date)) = ? AND t.group_id = ?
		ORDER BY t.date DESC, t.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, cycle, groupID)
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
			&row.CreatedByID,
			&row.ReceiptImage,
			&row.BillingCycle,
			&row.CreatedByUsername,
			&row.OwnerUsername,
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

// GetByMonthYearCompact retrieves a minimal set of fields for MoM comparison.
// Only returns id, type, category, description, amount, and user_id — no receipt_image, status, or JOINs.
func (r *TransactionRepository) GetByMonthYearCompact(ctx context.Context, month, year int, groupID string) (*models.CompactTransactionsResponse, error) {
	cycle := fmt.Sprintf("%04d-%02d", year, month)

	query := `
		SELECT t.id, t.type, t.category, t.description, t.amount, t.user_id
		FROM transactions t
		WHERE COALESCE(t.billing_cycle, strftime('%Y-%m', t.date)) = ? AND t.group_id = ?
	`

	rows, err := r.db.QueryContext(ctx, query, cycle, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query compact transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]models.CompactTransaction, 0)
	for rows.Next() {
		var t models.CompactTransaction
		var userID sql.NullString
		if err := rows.Scan(&t.ID, &t.Type, &t.Category, &t.Description, &t.Amount, &userID); err != nil {
			return nil, fmt.Errorf("failed to scan compact transaction: %w", err)
		}
		if userID.Valid {
			val := userID.String
			t.UserID = &val
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating compact rows: %w", err)
	}

	return &models.CompactTransactionsResponse{
		Transactions: transactions,
	}, nil
}

// GetByID retrieves a single transaction by its ID and group ID.
func (r *TransactionRepository) GetByID(ctx context.Context, id, groupID string) (*models.Transaction, error) {
	query := `
		SELECT t.id, t.type, t.category, t.description, t.amount, t.date,
		       t.status, t.paid_amount, t.group_id, t.user_id, t.created_by, t.receipt_image,
		       t.billing_cycle,
		       rec_u.username as created_by_username,
		       own_u.username as owner_username,
		       t.created_at, t.updated_at
		FROM transactions t
		LEFT JOIN users rec_u ON t.created_by = rec_u.id
		LEFT JOIN users own_u ON t.user_id = own_u.id
		WHERE t.id = ? AND t.group_id = ?
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
		&row.CreatedByID,
		&row.ReceiptImage,
		&row.BillingCycle,
		&row.CreatedByUsername,
		&row.OwnerUsername,
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

// Create inserts a new transaction into the database for a specific owner, recorder, and group.
// ownerUserID = wallet owner (user_id), createdByID = who recorded this (created_by / JWT user).
func (r *TransactionRepository) Create(ctx context.Context, req models.CreateTransactionRequest, ownerUserID, createdByID, groupID string) (*models.Transaction, error) {
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

		// Logic: If receipt is attached, set to PAID automatically
		if req.ReceiptImage != nil && *req.ReceiptImage != "" {
			s := string(models.StatusPaid)
			status = &s
			paidAmount = &req.Amount
		}
	}
	// For INCOME: status and paidAmount remain nil

	billingCycle := ""
	if req.BillingCycle != nil && *req.BillingCycle != "" {
		billingCycle = *req.BillingCycle
	} else {
		var cutoffDay int
		_ = r.db.QueryRowContext(ctx, `SELECT cycle_cutoff_day FROM group_settings WHERE group_id = ?`, groupID).Scan(&cutoffDay)
		if cutoffDay < 1 || cutoffDay > 31 {
			cutoffDay = 1
		}

		parsedDate, parseErr := time.Parse("2006-01-02", req.Date)
		if parseErr == nil && cutoffDay > 1 && parsedDate.Day() >= cutoffDay {
			nextMonth := parsedDate.AddDate(0, 1, 0)
			billingCycle = nextMonth.Format("2006-01")
		} else if parseErr == nil {
			billingCycle = parsedDate.Format("2006-01")
		} else if len(req.Date) >= 7 {
			billingCycle = req.Date[:7]
		} else {
			billingCycle = time.Now().Format("2006-01")
		}
	}

	query := `
		INSERT INTO transactions (id, type, category, description, amount, date, status, paid_amount, group_id, user_id, created_by, receipt_image, billing_cycle, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		ownerUserID,
		createdByID,
		req.ReceiptImage,
		billingCycle,
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
// defaultOwnerUserID = fallback wallet owner (user_id), createdByID = who recorded this (created_by / JWT user).
// Each request can override the wallet owner with req.UserID.
func (r *TransactionRepository) CreateBatch(ctx context.Context, requests []models.CreateTransactionRequest, defaultOwnerUserID, createdByID, groupID string) error {
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
		INSERT INTO transactions (id, type, category, description, amount, date, status, paid_amount, group_id, user_id, created_by, receipt_image, billing_cycle, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		ownerUserID := defaultOwnerUserID

		if req.UserID != nil && *req.UserID != "" {
			ownerUserID = *req.UserID
		}

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

		billingCycle := ""
		if req.BillingCycle != nil && *req.BillingCycle != "" {
			billingCycle = *req.BillingCycle
		} else if len(req.Date) >= 7 {
			billingCycle = req.Date[:7]
		} else {
			billingCycle = time.Now().Format("2006-01")
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
			ownerUserID,
			createdByID,
			req.ReceiptImage,
			billingCycle,
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

	if req.ReceiptImage != nil {
		setClauses = append(setClauses, "receipt_image = ?")
		args = append(args, *req.ReceiptImage)
	}

	if req.UserID != nil {
		setClauses = append(setClauses, "user_id = ?")
		args = append(args, *req.UserID)
	}
	if req.Date != nil && *req.Date != "" {
		setClauses = append(setClauses, "date = ?")
		args = append(args, *req.Date)
	}

	if req.Category != nil && *req.Category != "" {
		setClauses = append(setClauses, "category = ?")
		args = append(args, *req.Category)
	}

	if req.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *req.Description)
	}

	if req.BillingCycle != nil && *req.BillingCycle != "" {
		setClauses = append(setClauses, "billing_cycle = ?")
		args = append(args, *req.BillingCycle)
	}

	// Fetch existing transaction to validate status change rules
	existing, err := r.GetByID(ctx, id, groupID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	// Logic: If status is being changed to PAID for an expense, require a receipt
	if req.Status != nil && *req.Status == models.StatusPaid && existing.Type == models.TypeExpense {
		hasNewReceipt := req.ReceiptImage != nil && *req.ReceiptImage != ""
		hasExistingReceipt := existing.ReceiptImage != nil && *existing.ReceiptImage != ""

		if !hasNewReceipt && !hasExistingReceipt {
			return nil, fmt.Errorf("receipt image is required to set expense status to PAID")
		}

		// If no paid amount provided, default to full amount when marking as PAID
		if req.PaidAmount == nil && (existing.PaidAmount == nil || *existing.PaidAmount == 0) {
			setClauses = append(setClauses, "paid_amount = ?")
			args = append(args, existing.Amount)
		}
	}

	// If a receipt is being added to an expense, automatically set status to PAID
	// Guard: only append status if it wasn't already appended above (req.Status == nil)
	// to prevent duplicate SET clauses in the SQL query.
	if existing.Type == models.TypeExpense && req.ReceiptImage != nil && *req.ReceiptImage != "" && req.Status == nil {
		setClauses = append(setClauses, "status = ?")
		args = append(args, string(models.StatusPaid))

		if req.PaidAmount == nil && (existing.PaidAmount == nil || *existing.PaidAmount == 0) {
			setClauses = append(setClauses, "paid_amount = ?")
			args = append(args, existing.Amount)
		}
	}

	args = append(args, id, groupID)

	query := fmt.Sprintf("UPDATE transactions SET %s WHERE id = ? AND group_id = ?",
		strings.Join(setClauses, ", "))

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

// DeleteBatch removes multiple transactions by their IDs if they belong to the specified group.
func (r *TransactionRepository) DeleteBatch(ctx context.Context, ids []string, groupID string) error {
	if len(ids) == 0 {
		return nil
	}

	// Build query with IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+1)
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	args[len(ids)] = groupID

	query := fmt.Sprintf("DELETE FROM transactions WHERE id IN (%s) AND group_id = ?",
		strings.Join(placeholders, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete transactions in batch: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no transactions found to delete")
	}

	return nil
}

// GetAnnualSummary retrieves and aggregates all transactions for a given year and group, optionally scoped to a user.
func (r *TransactionRepository) GetAnnualSummary(ctx context.Context, year int, groupID string, filterUserID ...string) (*models.AnnualSummaryResponse, error) {
	startCycle := fmt.Sprintf("%04d-01", year)
	endCycle := fmt.Sprintf("%04d-12", year)

	query := `
		SELECT type, category, amount, COALESCE(billing_cycle, strftime('%Y-%m', date)) as cycle, status, paid_amount
		FROM transactions
		WHERE COALESCE(billing_cycle, strftime('%Y-%m', date)) >= ? AND COALESCE(billing_cycle, strftime('%Y-%m', date)) <= ? AND group_id = ?
	`
	args := []interface{}{startCycle, endCycle, groupID}
	if len(filterUserID) > 0 && filterUserID[0] != "" {
		query += " AND (user_id = ? OR created_by = ?)"
		args = append(args, filterUserID[0], filterUserID[0])
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query annual transactions: %w", err)
	}
	defer rows.Close()

	summary := &models.AnnualSummaryResponse{
		Year: year,
	}

	monthlyMap := make(map[int]*models.MonthlySummary)
	for i := 1; i <= 12; i++ {
		monthlyMap[i] = &models.MonthlySummary{Month: i}
	}

	categoryMap := make(map[string]*models.CategorySummary)

	for rows.Next() {
		var txType string
		var category string
		var amount float64
		var cycleStr string
		var status sql.NullString
		var paidAmount sql.NullFloat64

		if err := rows.Scan(&txType, &category, &amount, &cycleStr, &status, &paidAmount); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// parse month from YYYY-MM
		var tYear, tMonth int
		fmt.Sscanf(cycleStr, "%04d-%02d", &tYear, &tMonth)
		if tMonth < 1 || tMonth > 12 {
			tMonth = 1
		}
		tType := models.TransactionType(txType)

		// calculate active amount for expenses (pending + paid)
		if tType == models.TypeExpense {
			// For expenses, we just use the full amount for trend and category summaries.
			// Alternatively if we only count paidAmount:
			// activeAmount = 0
			// if paidAmount.Valid { activeAmount += paidAmount.Float64 }
			// But usually budget tracks the total expense amount.
			// So we use amount directly.
			summary.TotalExpense += amount
			monthlyMap[tMonth].Expense += amount
		} else {
			summary.TotalIncome += amount
			monthlyMap[tMonth].Income += amount
		}

		// Category aggregation
		catKey := fmt.Sprintf("%s|%s", tType, category)
		if _, exists := categoryMap[catKey]; !exists {
			categoryMap[catKey] = &models.CategorySummary{
				Category: category,
				Type:     tType,
				Amount:   0,
			}
		}
		categoryMap[catKey].Amount += amount
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows: %w", err)
	}

	summary.NetBalance = summary.TotalIncome - summary.TotalExpense

	// Convert maps to slices
	for i := 1; i <= 12; i++ {
		summary.MonthlyData = append(summary.MonthlyData, *monthlyMap[i])
	}

	for _, v := range categoryMap {
		summary.CategoryData = append(summary.CategoryData, *v)
	}

	return summary, nil
}

// GetWalletSummary returns per-member wallet breakdown for a group/month, grouped by user_id (wallet owner).
// Uses LEFT JOIN so members with no transactions still appear (with zeroes).
func (r *TransactionRepository) GetWalletSummary(ctx context.Context, month, year int, groupID string) (*models.WalletSummaryResponse, error) {
	datePrefix := fmt.Sprintf("%04d-%02d", year, month)

	query := `
		SELECT
			gm.user_id,
			u.username,
			COALESCE(SUM(CASE WHEN t.type = 'INCOME' THEN t.amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE
				WHEN t.type = 'EXPENSE' AND t.status = 'PAID' THEN COALESCE(t.paid_amount, t.amount)
				WHEN t.type = 'EXPENSE' AND t.status = 'PENDING' THEN t.amount
				WHEN t.type = 'EXPENSE' THEN t.amount
				ELSE 0
			END), 0) AS total_expense,
			COALESCE(SUM(CASE WHEN t.type = 'EXPENSE' AND t.status = 'PAID' THEN COALESCE(t.paid_amount, t.amount) ELSE 0 END), 0) AS total_paid,
			COALESCE(SUM(CASE WHEN t.type = 'EXPENSE' AND t.status = 'PENDING' THEN t.amount ELSE 0 END), 0) AS total_pending
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		LEFT JOIN transactions t ON t.user_id = gm.user_id
			AND t.group_id = ?
			AND COALESCE(t.billing_cycle, strftime('%Y-%m', t.date)) = ?
		WHERE gm.group_id = ?
		GROUP BY gm.user_id, u.username
		ORDER BY u.username
	`

	rows, err := r.db.QueryContext(ctx, query, groupID, datePrefix, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query wallet summary: %w", err)
	}
	defer rows.Close()

	var members []models.WalletMemberSummary
	var groupTotal models.WalletMemberSummary

	for rows.Next() {
		var m models.WalletMemberSummary
		if err := rows.Scan(&m.UserID, &m.Username, &m.TotalIncome, &m.TotalExpense, &m.TotalPaid, &m.TotalPending); err != nil {
			return nil, fmt.Errorf("failed to scan wallet member: %w", err)
		}
		m.NetBalance = m.TotalIncome - m.TotalExpense
		members = append(members, m)

		groupTotal.TotalIncome += m.TotalIncome
		groupTotal.TotalExpense += m.TotalExpense
		groupTotal.TotalPaid += m.TotalPaid
		groupTotal.TotalPending += m.TotalPending
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating wallet rows: %w", err)
	}

	if members == nil {
		members = []models.WalletMemberSummary{}
	}

	groupTotal.NetBalance = groupTotal.TotalIncome - groupTotal.TotalExpense
	groupTotal.Username = "All Members"
	groupTotal.UserID = groupID

	return &models.WalletSummaryResponse{
		Month:      month,
		Year:       year,
		Members:    members,
		GroupTotal: groupTotal,
	}, nil
}
