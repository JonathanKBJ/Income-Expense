package repository

import (
	"context"
	"database/sql"
	"expense-tracker/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CreditRepository struct {
	db *sql.DB
}

func NewCreditRepository(db *sql.DB) *CreditRepository {
	return &CreditRepository{db: db}
}

// --- Accounts ---

// CreateAccount inserts a new credit account.
func (r *CreditRepository) CreateAccount(ctx context.Context, acc *models.CreditAccount, userID, groupID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	acc.ID = uuid.New().String()
	acc.UserID = userID
	acc.GroupID = groupID
	if acc.Status == "" {
		acc.Status = models.CreditStatusActive
	}
	if acc.MinPaymentRate <= 0 {
		acc.MinPaymentRate = 5.0
	}
	if acc.MinPaymentFloor < 0 {
		acc.MinPaymentFloor = 0
	}
	acc.CreatedAt = time.Now().UTC()
	acc.UpdatedAt = time.Now().UTC()

	query := `INSERT INTO credit_accounts (
		id, name, type, bank, credit_limit, current_balance, statement_day, payment_due_day,
		interest_rate, min_payment_rate, min_payment_floor, status, notes, group_id, user_id,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		acc.ID, acc.Name, acc.Type, acc.Bank, acc.CreditLimit, acc.CurrentBalance,
		acc.StatementDay, acc.PaymentDueDay, acc.InterestRate, acc.MinPaymentRate, acc.MinPaymentFloor,
		acc.Status, acc.Notes, groupID, userID, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create credit account: %w", err)
	}
	return nil
}

// GetAccountByID retrieves a single credit account belonging to groupID.
func (r *CreditRepository) GetAccountByID(ctx context.Context, id, groupID string) (*models.CreditAccount, error) {
	query := `SELECT id, name, type, bank, credit_limit, current_balance, statement_day, payment_due_day,
		interest_rate, min_payment_rate, min_payment_floor, status, notes, group_id, user_id,
		created_at, updated_at
		FROM credit_accounts WHERE id = ? AND group_id = ?`

	row := r.db.QueryRowContext(ctx, query, id, groupID)

	var a models.CreditAccount
	var cAt, uAt string
	err := row.Scan(
		&a.ID, &a.Name, &a.Type, &a.Bank, &a.CreditLimit, &a.CurrentBalance,
		&a.StatementDay, &a.PaymentDueDay, &a.InterestRate, &a.MinPaymentRate, &a.MinPaymentFloor,
		&a.Status, &a.Notes, &a.GroupID, &a.UserID, &cAt, &uAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get credit account: %w", err)
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
	return &a, nil
}

// ListAccountsByGroup retrieves all credit accounts in a group.
func (r *CreditRepository) ListAccountsByGroup(ctx context.Context, groupID string) ([]models.CreditAccount, error) {
	query := `SELECT id, name, type, bank, credit_limit, current_balance, statement_day, payment_due_day,
		interest_rate, min_payment_rate, min_payment_floor, status, notes, group_id, user_id,
		created_at, updated_at
		FROM credit_accounts WHERE group_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to list credit accounts: %w", err)
	}
	defer rows.Close()

	var accounts []models.CreditAccount
	for rows.Next() {
		var a models.CreditAccount
		var cAt, uAt string
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Type, &a.Bank, &a.CreditLimit, &a.CurrentBalance,
			&a.StatementDay, &a.PaymentDueDay, &a.InterestRate, &a.MinPaymentRate, &a.MinPaymentFloor,
			&a.Status, &a.Notes, &a.GroupID, &a.UserID, &cAt, &uAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan credit account: %w", err)
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
		accounts = append(accounts, a)
	}
	if accounts == nil {
		accounts = []models.CreditAccount{}
	}
	return accounts, nil
}

// UpdateAccount updates specified fields of a credit account.
func (r *CreditRepository) UpdateAccount(ctx context.Context, id, groupID string, req models.UpdateCreditAccountRequest) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := "UPDATE credit_accounts SET updated_at = ?"
	args := []interface{}{now}

	if req.Name != nil {
		query += ", name = ?"
		args = append(args, *req.Name)
	}
	if req.Type != nil {
		query += ", type = ?"
		args = append(args, *req.Type)
	}
	if req.Bank != nil {
		query += ", bank = ?"
		args = append(args, *req.Bank)
	}
	if req.CreditLimit != nil {
		query += ", credit_limit = ?"
		args = append(args, *req.CreditLimit)
	}
	if req.CurrentBalance != nil {
		query += ", current_balance = ?"
		args = append(args, *req.CurrentBalance)
	}
	if req.StatementDay != nil {
		query += ", statement_day = ?"
		args = append(args, *req.StatementDay)
	}
	if req.PaymentDueDay != nil {
		query += ", payment_due_day = ?"
		args = append(args, *req.PaymentDueDay)
	}
	if req.InterestRate != nil {
		query += ", interest_rate = ?"
		args = append(args, *req.InterestRate)
	}
	if req.MinPaymentRate != nil {
		query += ", min_payment_rate = ?"
		args = append(args, *req.MinPaymentRate)
	}
	if req.MinPaymentFloor != nil {
		query += ", min_payment_floor = ?"
		args = append(args, *req.MinPaymentFloor)
	}
	if req.Status != nil {
		query += ", status = ?"
		args = append(args, *req.Status)
	}
	if req.Notes != nil {
		query += ", notes = ?"
		args = append(args, *req.Notes)
	}

	query += " WHERE id = ? AND group_id = ?"
	args = append(args, id, groupID)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update credit account: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("credit account not found")
	}
	return nil
}

// DeleteAccount deletes a credit account and cascades to installments and transactions.
func (r *CreditRepository) DeleteAccount(ctx context.Context, id, groupID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM credit_accounts WHERE id = ? AND group_id = ?`, id, groupID)
	if err != nil {
		return fmt.Errorf("failed to delete credit account: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("credit account not found")
	}
	return nil
}

// --- Installments ---

// AddInstallment creates a new installment plan under an account.
func (r *CreditRepository) AddInstallment(ctx context.Context, inst *models.CreditInstallment, accountID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	inst.ID = uuid.New().String()
	inst.AccountID = accountID
	if inst.Status == "" {
		inst.Status = models.InstallmentStatusActive
	}
	if inst.InterestType == "" {
		inst.InterestType = "FLAT"
	}
	inst.CreatedAt = time.Now().UTC()
	inst.UpdatedAt = time.Now().UTC()

	query := `INSERT INTO credit_installments (
		id, account_id, item_name, total_amount, monthly_amount, total_terms, paid_terms,
		interest_rate, interest_type, remaining_balance, start_date, end_date, status, notes, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		inst.ID, accountID, inst.ItemName, inst.TotalAmount, inst.MonthlyAmount,
		inst.TotalTerms, inst.PaidTerms, inst.InterestRate, inst.InterestType, inst.RemainingBalance,
		inst.StartDate, inst.EndDate, inst.Status, inst.Notes,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to add installment: %w", err)
	}
	return nil
}
// ListInstallments returns all installment plans for an account.
func (r *CreditRepository) ListInstallments(ctx context.Context, accountID string) ([]models.CreditInstallment, error) {
	query := `SELECT id, account_id, item_name, total_amount, monthly_amount, total_terms, paid_terms,
		interest_rate, interest_type, remaining_balance, start_date, end_date, status, notes, created_at, updated_at
		FROM credit_installments WHERE account_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list installments: %w", err)
	}
	defer rows.Close()

	var list []models.CreditInstallment
	for rows.Next() {
		var item models.CreditInstallment
		var cAt, uAt string
		if err := rows.Scan(
			&item.ID, &item.AccountID, &item.ItemName, &item.TotalAmount, &item.MonthlyAmount,
			&item.TotalTerms, &item.PaidTerms, &item.InterestRate, &item.InterestType, &item.RemainingBalance,
			&item.StartDate, &item.EndDate, &item.Status, &item.Notes,
			&cAt, &uAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan installment: %w", err)
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
		list = append(list, item)
	}
	if list == nil {
		list = []models.CreditInstallment{}
	}
	return list, nil
}

// ListInstallmentsByAccountIDs returns installments mapped by accountID.
func (r *CreditRepository) ListInstallmentsByAccountIDs(ctx context.Context, accountIDs []string) (map[string][]models.CreditInstallment, error) {
	result := make(map[string][]models.CreditInstallment)
	for _, id := range accountIDs {
		result[id] = []models.CreditInstallment{}
	}
	if len(accountIDs) == 0 {
		return result, nil
	}

	query := `SELECT id, account_id, item_name, total_amount, monthly_amount, total_terms, paid_terms,
		interest_rate, interest_type, remaining_balance, start_date, end_date, status, notes, created_at, updated_at
		FROM credit_installments WHERE account_id IN (` + placeholders(len(accountIDs)) + `)
		ORDER BY created_at DESC`

	args := make([]interface{}, len(accountIDs))
	for i, id := range accountIDs {
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to batch list installments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.CreditInstallment
		var cAt, uAt string
		if err := rows.Scan(
			&item.ID, &item.AccountID, &item.ItemName, &item.TotalAmount, &item.MonthlyAmount,
			&item.TotalTerms, &item.PaidTerms, &item.InterestRate, &item.InterestType, &item.RemainingBalance,
			&item.StartDate, &item.EndDate, &item.Status, &item.Notes,
			&cAt, &uAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan installment: %w", err)
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
		result[item.AccountID] = append(result[item.AccountID], item)
	}
	return result, nil
}
// UpdateInstallment updates an installment plan.
func (r *CreditRepository) UpdateInstallment(ctx context.Context, id, accountID string, req models.UpdateCreditInstallmentRequest) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := "UPDATE credit_installments SET updated_at = ?"
	args := []interface{}{now}

	if req.ItemName != nil {
		query += ", item_name = ?"
		args = append(args, *req.ItemName)
	}
	if req.TotalAmount != nil {
		query += ", total_amount = ?"
		args = append(args, *req.TotalAmount)
	}
	if req.MonthlyAmount != nil {
		query += ", monthly_amount = ?"
		args = append(args, *req.MonthlyAmount)
	}
	if req.TotalTerms != nil {
		query += ", total_terms = ?"
		args = append(args, *req.TotalTerms)
	}
	if req.PaidTerms != nil {
		query += ", paid_terms = ?"
		args = append(args, *req.PaidTerms)
	}
	if req.InterestRate != nil {
		query += ", interest_rate = ?"
		args = append(args, *req.InterestRate)
	}
	if req.InterestType != nil {
		query += ", interest_type = ?"
		args = append(args, *req.InterestType)
	}
	if req.RemainingBalance != nil {
		query += ", remaining_balance = ?"
		args = append(args, *req.RemainingBalance)
	}
	if req.EndDate != nil {
		query += ", end_date = ?"
		args = append(args, *req.EndDate)
	}
	if req.Status != nil {
		query += ", status = ?"
		args = append(args, *req.Status)
	}
	if req.Notes != nil {
		query += ", notes = ?"
		args = append(args, *req.Notes)
	}

	query += " WHERE id = ? AND account_id = ?"
	args = append(args, id, accountID)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update installment: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("installment not found")
	}
	return nil
}

// DeleteInstallment deletes an installment plan.
func (r *CreditRepository) DeleteInstallment(ctx context.Context, id, accountID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM credit_installments WHERE id = ? AND account_id = ?`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete installment: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("installment not found")
	}
	return nil
}

// --- Transactions & Balance Handling ---

// AddTransaction inserts a charge or payment, updating account balance atomically.
func (r *CreditRepository) AddTransaction(ctx context.Context, tx *models.CreditTransaction, accountID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	tx.ID = uuid.New().String()
	tx.AccountID = accountID
	tx.CreatedAt = time.Now().UTC()
	tx.UpdatedAt = time.Now().UTC()

	// Begin DB transaction
	dbTx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin db tx: %w", err)
	}
	defer dbTx.Rollback()

	// 1. Insert transaction
	insertQuery := `INSERT INTO credit_transactions (
		id, account_id, type, amount, date, description, payment_type, installment_id, receipt_image,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = dbTx.ExecContext(ctx, insertQuery,
		tx.ID, accountID, tx.Type, tx.Amount, tx.Date, tx.Description,
		tx.PaymentType, tx.InstallmentID, tx.ReceiptImage, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert credit transaction: %w", err)
	}

	// 2. Adjust balance
	if tx.Type == models.CreditTxCharge {
		_, err = dbTx.ExecContext(ctx, `UPDATE credit_accounts SET current_balance = current_balance + ?, updated_at = ? WHERE id = ?`, tx.Amount, now, accountID)
	} else if tx.Type == models.CreditTxPayment {
		_, err = dbTx.ExecContext(ctx, `UPDATE credit_accounts SET current_balance = MAX(0, current_balance - ?), updated_at = ? WHERE id = ?`, tx.Amount, now, accountID)
	}
	if err != nil {
		return fmt.Errorf("failed to adjust credit balance: %w", err)
	}

	// 3. If payment is linked to installment, increment paid_terms
	if tx.Type == models.CreditTxPayment && tx.InstallmentID != nil && *tx.InstallmentID != "" {
		_, err = dbTx.ExecContext(ctx, `
			UPDATE credit_installments
			SET paid_terms = MIN(total_terms, paid_terms + 1),
			    status = CASE WHEN paid_terms + 1 >= total_terms THEN 'COMPLETED' ELSE status END,
			    updated_at = ?
			WHERE id = ? AND account_id = ?
		`, now, *tx.InstallmentID, accountID)
		if err != nil {
			return fmt.Errorf("failed to advance installment term: %w", err)
		}
	}

	return dbTx.Commit()
}

// ListTransactions returns all transactions for an account, newest first.
func (r *CreditRepository) ListTransactions(ctx context.Context, accountID string) ([]models.CreditTransaction, error) {
	query := `SELECT id, account_id, type, amount, date, description, payment_type, installment_id, receipt_image,
		created_at, updated_at
		FROM credit_transactions WHERE account_id = ? ORDER BY date DESC, created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list credit transactions: %w", err)
	}
	defer rows.Close()

	var list []models.CreditTransaction
	for rows.Next() {
		var item models.CreditTransaction
		var cAt, uAt string
		if err := rows.Scan(
			&item.ID, &item.AccountID, &item.Type, &item.Amount, &item.Date, &item.Description,
			&item.PaymentType, &item.InstallmentID, &item.ReceiptImage,
			&cAt, &uAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan credit transaction: %w", err)
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
		list = append(list, item)
	}
	if list == nil {
		list = []models.CreditTransaction{}
	}
	return list, nil
}

// ListTransactionsByAccountIDs returns transactions grouped by accountID.
func (r *CreditRepository) ListTransactionsByAccountIDs(ctx context.Context, accountIDs []string) (map[string][]models.CreditTransaction, error) {
	result := make(map[string][]models.CreditTransaction)
	for _, id := range accountIDs {
		result[id] = []models.CreditTransaction{}
	}
	if len(accountIDs) == 0 {
		return result, nil
	}

	query := `SELECT id, account_id, type, amount, date, description, payment_type, installment_id, receipt_image,
		created_at, updated_at
		FROM credit_transactions WHERE account_id IN (` + placeholders(len(accountIDs)) + `)
		ORDER BY date DESC, created_at DESC`

	args := make([]interface{}, len(accountIDs))
	for i, id := range accountIDs {
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to batch list credit transactions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.CreditTransaction
		var cAt, uAt string
		if err := rows.Scan(
			&item.ID, &item.AccountID, &item.Type, &item.Amount, &item.Date, &item.Description,
			&item.PaymentType, &item.InstallmentID, &item.ReceiptImage,
			&cAt, &uAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan credit transaction: %w", err)
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
		result[item.AccountID] = append(result[item.AccountID], item)
	}
	return result, nil
}

// DeleteTransaction deletes a transaction and reverses its balance impact.
func (r *CreditRepository) DeleteTransaction(ctx context.Context, id, accountID string) error {
	// First get the transaction to know type and amount
	var txType models.CreditTxType
	var amount float64
	var instID sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT type, amount, installment_id FROM credit_transactions WHERE id = ? AND account_id = ?`, id, accountID).Scan(&txType, &amount, &instID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("transaction not found")
		}
		return fmt.Errorf("failed to find transaction: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	dbTx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin db tx: %w", err)
	}
	defer dbTx.Rollback()

	// Reverse balance impact
	if txType == models.CreditTxCharge {
		_, err = dbTx.ExecContext(ctx, `UPDATE credit_accounts SET current_balance = MAX(0, current_balance - ?), updated_at = ? WHERE id = ?`, amount, now, accountID)
	} else if txType == models.CreditTxPayment {
		_, err = dbTx.ExecContext(ctx, `UPDATE credit_accounts SET current_balance = current_balance + ?, updated_at = ? WHERE id = ?`, amount, now, accountID)
	}
	if err != nil {
		return fmt.Errorf("failed to reverse balance impact: %w", err)
	}

	// Reverse installment term if was payment linked to installment
	if txType == models.CreditTxPayment && instID.Valid && instID.String != "" {
		_, _ = dbTx.ExecContext(ctx, `
			UPDATE credit_installments
			SET paid_terms = MAX(0, paid_terms - 1),
			    status = 'ACTIVE',
			    updated_at = ?
			WHERE id = ? AND account_id = ?
		`, now, instID.String, accountID)
	}

	// Delete row
	_, err = dbTx.ExecContext(ctx, `DELETE FROM credit_transactions WHERE id = ? AND account_id = ?`, id, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete credit transaction: %w", err)
	}

	return dbTx.Commit()
}
