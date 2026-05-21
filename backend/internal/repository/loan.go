package repository

import (
	"context"
	"database/sql"
	"expense-tracker/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type LoanRepository struct {
	db *sql.DB
}

func NewLoanRepository(db *sql.DB) *LoanRepository {
	return &LoanRepository{db: db}
}

// Create inserts a new loan record.
func (r *LoanRepository) Create(ctx context.Context, loan *models.Loan, userID, groupID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	loan.ID = uuid.New().String()
	loan.UserID = userID
	loan.GroupID = groupID
	loan.Status = models.LoanActive
	loan.CreatedAt = time.Now().UTC()
	loan.UpdatedAt = time.Now().UTC()

	query := `INSERT INTO loans (id, type, name, counterparty, principal, term_months, installment_amount, payment_day, interest_rate, start_date, end_date, status, notes, group_id, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		loan.ID, loan.Type, loan.Name, loan.Counterparty, loan.Principal,
		loan.TermMonths, loan.InstallmentAmount, loan.PaymentDay, loan.InterestRate,
		loan.StartDate, loan.EndDate, loan.Status, loan.Notes, groupID, userID, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create loan: %w", err)
	}
	return nil
}

// GetByID retrieves a single loan (must belong to the given group).
func (r *LoanRepository) GetByID(ctx context.Context, id, groupID string) (*models.Loan, error) {
	query := `SELECT id, type, name, counterparty, principal, term_months, installment_amount, payment_day, interest_rate, start_date, end_date, status, notes, group_id, user_id, created_at, updated_at
		FROM loans WHERE id = ? AND group_id = ?`
	row := r.db.QueryRowContext(ctx, query, id, groupID)

	var l models.Loan
	var cAt, uAt string
	err := row.Scan(&l.ID, &l.Type, &l.Name, &l.Counterparty, &l.Principal,
		&l.TermMonths, &l.InstallmentAmount, &l.PaymentDay, &l.InterestRate,
		&l.StartDate, &l.EndDate, &l.Status, &l.Notes, &l.GroupID, &l.UserID, &cAt, &uAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get loan: %w", err)
	}
	l.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
	l.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
	return &l, nil
}

// ListByGroup returns all loans in a group.
func (r *LoanRepository) ListByGroup(ctx context.Context, groupID string) ([]models.Loan, error) {
	query := `SELECT id, type, name, counterparty, principal, term_months, installment_amount, payment_day, interest_rate, start_date, end_date, status, notes, group_id, user_id, created_at, updated_at
		FROM loans WHERE group_id = ? ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to list loans: %w", err)
	}
	defer rows.Close()

	var loans []models.Loan
	for rows.Next() {
		var l models.Loan
		var cAt, uAt string
		if err := rows.Scan(&l.ID, &l.Type, &l.Name, &l.Counterparty, &l.Principal,
			&l.TermMonths, &l.InstallmentAmount, &l.PaymentDay, &l.InterestRate,
			&l.StartDate, &l.EndDate, &l.Status, &l.Notes, &l.GroupID, &l.UserID, &cAt, &uAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan loan: %w", err)
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
		l.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
		loans = append(loans, l)
	}
	if loans == nil {
		loans = []models.Loan{}
	}
	return loans, nil
}

// Update applies partial updates to a loan.
func (r *LoanRepository) Update(ctx context.Context, id, groupID string, req models.UpdateLoanRequest) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Get existing loan to merge
	existing, err := r.GetByID(ctx, id, groupID)
	if err != nil || existing == nil {
		return fmt.Errorf("loan not found")
	}

	query := `UPDATE loans SET updated_at = ?`
	args := []interface{}{now}

	if req.Name != nil {
		query += ", name = ?"
		args = append(args, *req.Name)
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

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update loan: %w", err)
	}
	return nil
}

// Delete removes a loan (CASCADE deletes entries via FK).
func (r *LoanRepository) Delete(ctx context.Context, id, groupID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM loans WHERE id = ? AND group_id = ?`, id, groupID)
	if err != nil {
		return fmt.Errorf("failed to delete loan: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("loan not found")
	}
	return nil
}

// --- Loan Entries ---

// AddEntry inserts a new entry for a loan.
func (r *LoanRepository) AddEntry(ctx context.Context, entry *models.LoanEntry) error {
	now := time.Now().UTC().Format(time.RFC3339)
	entry.ID = uuid.New().String()
	entry.CreatedAt = time.Now().UTC()
	entry.UpdatedAt = time.Now().UTC()

	query := `INSERT INTO loan_entries (id, loan_id, entry_type, amount, date, description, receipt_image, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.LoanID, entry.EntryType, entry.Amount,
		entry.Date, entry.Description, entry.ReceiptImage, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to add loan entry: %w", err)
	}
	return nil
}

// ListEntries returns all entries for a loan, ordered by date.
func (r *LoanRepository) ListEntries(ctx context.Context, loanID string) ([]models.LoanEntry, error) {
	query := `SELECT id, loan_id, entry_type, amount, date, description, receipt_image, created_at, updated_at
		FROM loan_entries WHERE loan_id = ? ORDER BY date ASC, created_at ASC`
	rows, err := r.db.QueryContext(ctx, query, loanID)
	if err != nil {
		return nil, fmt.Errorf("failed to list loan entries: %w", err)
	}
	defer rows.Close()

	var entries []models.LoanEntry
	for rows.Next() {
		var e models.LoanEntry
		var cAt, uAt string
		if err := rows.Scan(&e.ID, &e.LoanID, &e.EntryType, &e.Amount, &e.Date, &e.Description, &e.ReceiptImage, &cAt, &uAt); err != nil {
			return nil, fmt.Errorf("failed to scan loan entry: %w", err)
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
		e.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []models.LoanEntry{}
	}
	return entries, nil
}

// ListEntriesByLoanIDs fetches entries for multiple loans in a single query.
// Returns map[loanID][]LoanEntry for efficient batch lookups.
func (r *LoanRepository) ListEntriesByLoanIDs(ctx context.Context, loanIDs []string) (map[string][]models.LoanEntry, error) {
	if len(loanIDs) == 0 {
		return map[string][]models.LoanEntry{}, nil
	}

	query := `SELECT id, loan_id, entry_type, amount, date, description, receipt_image, created_at, updated_at
		FROM loan_entries WHERE loan_id IN (` + placeholders(len(loanIDs)) + `)
		ORDER BY date ASC, created_at ASC`

	args := make([]interface{}, len(loanIDs))
	for i, id := range loanIDs {
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list loan entries: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]models.LoanEntry)
	for rows.Next() {
		var e models.LoanEntry
		var cAt, uAt string
		if err := rows.Scan(&e.ID, &e.LoanID, &e.EntryType, &e.Amount, &e.Date, &e.Description, &e.ReceiptImage, &cAt, &uAt); err != nil {
			return nil, fmt.Errorf("failed to scan loan entry: %w", err)
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, cAt)
		e.UpdatedAt, _ = time.Parse(time.RFC3339, uAt)
		result[e.LoanID] = append(result[e.LoanID], e)
	}

	for _, id := range loanIDs {
		if _, ok := result[id]; !ok {
			result[id] = []models.LoanEntry{}
		}
	}

	return result, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	ph := "?"
	for i := 1; i < n; i++ {
		ph += ", ?"
	}
	return ph
}

// DeleteEntry removes a single entry.
func (r *LoanRepository) DeleteEntry(ctx context.Context, entryID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM loan_entries WHERE id = ?`, entryID)
	if err != nil {
		return fmt.Errorf("failed to delete loan entry: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entry not found")
	}
	return nil
}
