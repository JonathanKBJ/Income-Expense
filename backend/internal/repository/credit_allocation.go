package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"expense-tracker/internal/models"
	"github.com/google/uuid"
)

var (
	ErrCreditInvalidAllocation = errors.New("invalid credit allocation")
	ErrCreditConflict          = errors.New("credit balance conflict")
	ErrCreditNotFound          = errors.New("credit account or transaction not found")
)

// Keep every sum exactly representable in both integer cents and the existing
// REAL/wire representation. No arithmetic on split liabilities uses floats.
const maxCreditCents int64 = 900000000000000

func creditCents(value float64) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > float64(maxCreditCents)/100 {
		return 0, fmt.Errorf("%w: money must be finite, nonnegative and within range", ErrCreditInvalidAllocation)
	}
	rounded := math.Round(value * 100)
	if rounded/100 != value {
		return 0, fmt.Errorf("%w: money must have at most two decimal places", ErrCreditInvalidAllocation)
	}
	return int64(rounded), nil
}

func validateCreditConfig(basis *models.CreditLimitBasis, order *models.CreditAllocationOrder) error {
	if basis != nil && *basis != models.CreditLimitPrincipal && *basis != models.CreditLimitTotal {
		return fmt.Errorf("%w: unknown credit limit basis", ErrCreditInvalidAllocation)
	}
	if order != nil && *order != models.AllocationFeesInterestPrincipal && *order != models.AllocationInterestFeesPrincipal {
		return fmt.Errorf("%w: unknown allocation order", ErrCreditInvalidAllocation)
	}
	return nil
}

type creditBalanceState struct {
	mode                 models.CreditBalanceMode
	basis                models.CreditLimitBasis
	order                models.CreditAllocationOrder
	asOf                 sql.NullString
	total, interest, fee float64
	owner                string
}

// Acquire the SQLite write lock BEFORE reading either ledger history or the
// baseline. Every ledger writer and reconciler takes the same account guard;
// there is no read-before-lock window or stale balance overwrite.
func lockCreditAccount(ctx context.Context, tx *sql.Tx, id, groupID string, userID ...string) (*creditBalanceState, error) {
	query := `UPDATE credit_accounts SET updated_at = updated_at WHERE id = ?`
	args := []interface{}{id}
	if groupID != "" {
		query += " AND group_id = ?"
		args = append(args, groupID)
	}
	if len(userID) > 0 && userID[0] != "" {
		query += " AND user_id = ?"
		args = append(args, userID[0])
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("guard credit balance: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, ErrCreditNotFound
	}
	var state creditBalanceState
	err = tx.QueryRowContext(ctx, `SELECT balance_mode, current_balance, interest_balance, fee_balance,
		credit_limit_basis, allocation_order, balance_as_of_date, user_id FROM credit_accounts WHERE id = ?`, id).
		Scan(&state.mode, &state.total, &state.interest, &state.fee, &state.basis, &state.order, &state.asOf, &state.owner)
	if err != nil {
		return nil, err
	}
	if state.mode == "" {
		state.mode = models.CreditBalanceLegacy
	}
	if state.basis == "" {
		state.basis = models.CreditLimitPrincipal
	}
	if state.order == "" {
		state.order = models.AllocationFeesInterestPrincipal
	}
	if state.mode != models.CreditBalanceLegacy && state.mode != models.CreditBalancePrincipalSplit {
		return nil, fmt.Errorf("%w: unknown stored balance mode", ErrCreditConflict)
	}
	return &state, nil
}

func (s *creditBalanceState) buckets() (principal, interest, fee int64, err error) {
	total, err := creditCents(s.total)
	if err != nil {
		return 0, 0, 0, err
	}
	interest, err = creditCents(s.interest)
	if err != nil {
		return 0, 0, 0, err
	}
	fee, err = creditCents(s.fee)
	if err != nil {
		return 0, 0, 0, err
	}
	if interest > total || fee > total-interest {
		return 0, 0, 0, fmt.Errorf("%w: stored debt buckets exceed total", ErrCreditConflict)
	}
	return total - interest - fee, interest, fee, nil
}

func validateCreditLedgerDate(date string, state *creditBalanceState) error {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil || parsed.Format("2006-01-02") != date {
		return fmt.Errorf("%w: date must be a real YYYY-MM-DD date", ErrCreditInvalidAllocation)
	}
	if date > time.Now().In(installmentBillingLocation).Format("2006-01-02") {
		return fmt.Errorf("%w: transaction date cannot be in the future in Bangkok", ErrCreditInvalidAllocation)
	}
	if state.asOf.Valid && date <= state.asOf.String {
		return fmt.Errorf("%w: transactions on or before baseline %s are frozen", ErrCreditConflict, state.asOf.String)
	}
	return nil
}

func splitComponents(item *models.CreditTransaction) (principal, interest, fee int64, err error) {
	if item.PrincipalAmount == nil || item.InterestAmount == nil || item.FeeAmount == nil {
		return 0, 0, 0, fmt.Errorf("%w: all three components are required", ErrCreditInvalidAllocation)
	}
	principal, err = creditCents(*item.PrincipalAmount)
	if err != nil {
		return
	}
	interest, err = creditCents(*item.InterestAmount)
	if err != nil {
		return
	}
	fee, err = creditCents(*item.FeeAmount)
	if err != nil {
		return
	}
	amount, amountErr := creditCents(item.Amount)
	if amountErr != nil {
		err = amountErr
		return
	}
	if principal+interest+fee != amount || amount == 0 {
		err = fmt.Errorf("%w: components must sum exactly to positive amount", ErrCreditInvalidAllocation)
	}
	return
}

func setCreditComponents(item *models.CreditTransaction, principal, interest, fee int64, source models.CreditAllocationSource) {
	p, i, f := float64(principal)/100, float64(interest)/100, float64(fee)/100
	item.PrincipalAmount, item.InterestAmount, item.FeeAmount = &p, &i, &f
	item.AllocationSource = &source
}

func prepareSplitAllocation(item *models.CreditTransaction, state *creditBalanceState) error {
	if item.Type != models.CreditTxCharge && item.Type != models.CreditTxPayment {
		return fmt.Errorf("%w: type must be CHARGE or PAYMENT", ErrCreditInvalidAllocation)
	}
	if err := validateCreditLedgerDate(item.Date, state); err != nil {
		return err
	}
	if item.PaymentType != nil {
		switch *item.PaymentType {
		case models.CreditPayFull, models.CreditPayMinimum, models.CreditPayCustom, models.CreditPayInstallment:
		default:
			return fmt.Errorf("%w: unknown payment type", ErrCreditInvalidAllocation)
		}
	}
	if item.ReceiptImage != nil && strings.TrimSpace(*item.ReceiptImage) == "" {
		return fmt.Errorf("%w: receipt must not be empty when supplied", ErrCreditInvalidAllocation)
	}
	amount, err := creditCents(item.Amount)
	if err != nil {
		return err
	}
	if amount == 0 {
		return fmt.Errorf("%w: amount must be positive", ErrCreditInvalidAllocation)
	}
	provided := 0
	for _, component := range []*float64{item.PrincipalAmount, item.InterestAmount, item.FeeAmount} {
		if component != nil {
			provided++
		}
	}
	if provided == 3 {
		p, i, f, err := splitComponents(item)
		if err != nil {
			return err
		}
		setCreditComponents(item, p, i, f, models.AllocationActual)
		return nil
	}
	if provided != 0 || item.Type == models.CreditTxCharge {
		return fmt.Errorf("%w: split charges and explicit payments require all three components", ErrCreditInvalidAllocation)
	}
	p, i, f, err := state.buckets()
	if err != nil {
		return err
	}
	if amount > p+i+f {
		return fmt.Errorf("%w: payment exceeds posted debt", ErrCreditConflict)
	}
	if err := validateCreditConfig(&state.basis, &state.order); err != nil {
		return err
	}
	remaining := amount
	take := func(balance int64) int64 {
		allocated := min(balance, remaining)
		remaining -= allocated
		return allocated
	}
	var allocatedInterest, allocatedFee int64
	if state.order == models.AllocationInterestFeesPrincipal {
		allocatedInterest, allocatedFee = take(i), take(f)
	} else {
		allocatedFee, allocatedInterest = take(f), take(i)
	}
	setCreditComponents(item, take(p), allocatedInterest, allocatedFee, models.AllocationRule)
	return nil
}

func applySplitBalance(state *creditBalanceState, item *models.CreditTransaction, reverse bool) error {
	p, i, f, err := state.buckets()
	if err != nil {
		return err
	}
	ap, ai, af, err := splitComponents(item)
	if err != nil {
		return err
	}
	subtract := item.Type == models.CreditTxPayment
	if reverse {
		subtract = !subtract
	}
	if subtract {
		if ap > p || ai > i || af > f {
			return fmt.Errorf("%w: allocation exceeds a posted principal, interest or fee balance", ErrCreditConflict)
		}
		p, i, f = p-ap, i-ai, f-af
	} else {
		p, i, f = p+ap, i+ai, f+af
		if p+i+f > maxCreditCents {
			return fmt.Errorf("%w: balance exceeds supported money range", ErrCreditInvalidAllocation)
		}
	}
	state.total, state.interest, state.fee = float64(p+i+f)/100, float64(i)/100, float64(f)/100
	return nil
}

func persistSplitBalance(ctx context.Context, tx *sql.Tx, accountID string, state *creditBalanceState, now string) error {
	_, err := tx.ExecContext(ctx, `UPDATE credit_accounts SET current_balance = ?, interest_balance = ?, fee_balance = ?, updated_at = ? WHERE id = ?`,
		state.total, state.interest, state.fee, now, accountID)
	return err
}

func validateSplitInstallment(ctx context.Context, tx *sql.Tx, item *models.CreditTransaction, accountID string, reversing bool) error {
	if item.InstallmentID == nil || *item.InstallmentID == "" {
		return nil
	}
	if err := lockUnsettledInstallment(ctx, tx, *item.InstallmentID, accountID); err != nil {
		return err
	}
	var paid, total int
	var status models.InstallmentStatus
	err := tx.QueryRowContext(ctx, `SELECT paid_terms, total_terms, status FROM credit_installments WHERE id = ? AND account_id = ?`, *item.InstallmentID, accountID).Scan(&paid, &total, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInstallmentNotFound
	}
	if err != nil {
		return err
	}
	if !reversing && item.Type == models.CreditTxPayment && (paid >= total || status != models.InstallmentStatusActive) {
		return ErrInstallmentNotActive
	}
	return nil
}

func adjustCreditInstallment(ctx context.Context, tx *sql.Tx, item *models.CreditTransaction, accountID, now string, reverse bool) error {
	if item.Type != models.CreditTxPayment || item.InstallmentID == nil || *item.InstallmentID == "" {
		return nil
	}
	query := `UPDATE credit_installments SET paid_terms = MIN(total_terms, paid_terms + 1),
		status = CASE WHEN paid_terms + 1 >= total_terms THEN 'COMPLETED' ELSE status END, updated_at = ? WHERE id = ? AND account_id = ?`
	if reverse {
		query = `UPDATE credit_installments SET paid_terms = MAX(0, paid_terms - 1), status = 'ACTIVE', updated_at = ? WHERE id = ? AND account_id = ?`
	}
	_, err := tx.ExecContext(ctx, query, now, *item.InstallmentID, accountID)
	return err
}

func insertCreditTransaction(ctx context.Context, tx *sql.Tx, item *models.CreditTransaction, now string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO credit_transactions
		(id, account_id, type, amount, date, description, payment_type, installment_id, receipt_image,
		created_at, updated_at, principal_amount, interest_amount, fee_amount, allocation_source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.AccountID, item.Type, item.Amount, item.Date, item.Description, item.PaymentType, item.InstallmentID, item.ReceiptImage,
		now, now, item.PrincipalAmount, item.InterestAmount, item.FeeAmount, item.AllocationSource)
	return err
}

func (r *CreditRepository) addAllocatedTransaction(ctx context.Context, item *models.CreditTransaction, accountID string, legacyOnly bool) error {
	if item == nil {
		return fmt.Errorf("%w: transaction is required", ErrCreditInvalidAllocation)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	state, err := lockCreditAccount(ctx, tx, accountID, "")
	if err != nil {
		return err
	}
	if legacyOnly && state.mode != models.CreditBalanceLegacy {
		return fmt.Errorf("%w: legacy transaction rejected for split balance account", ErrCreditConflict)
	}
	item.ID, item.AccountID = uuid.New().String(), accountID
	item.CreatedAt, item.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	now := item.CreatedAt.Format(time.RFC3339)
	if state.mode == models.CreditBalancePrincipalSplit {
		if err := prepareSplitAllocation(item, state); err != nil {
			return err
		}
		if err := validateSplitInstallment(ctx, tx, item, accountID, false); err != nil {
			return err
		}
		if err := applySplitBalance(state, item, false); err != nil {
			return err
		}
		if err := persistSplitBalance(ctx, tx, accountID, state, now); err != nil {
			return err
		}
	} else {
		if item.PrincipalAmount != nil || item.InterestAmount != nil || item.FeeAmount != nil || item.AllocationSource != nil {
			return fmt.Errorf("%w: allocation fields require a reconciled split account", ErrCreditInvalidAllocation)
		}
		if item.InstallmentID != nil && *item.InstallmentID != "" {
			if err := lockUnsettledInstallment(ctx, tx, *item.InstallmentID, accountID); err != nil {
				return err
			}
		}
		// Preserve legacy duplicate handling, but never silently swallow genuine
		// split payments (even if they have identical amount/date/description).
		var duplicateID string
		_ = tx.QueryRowContext(ctx, `SELECT id FROM credit_transactions
			WHERE account_id = ? AND type = ? AND amount = ? AND date = ? AND description = ?
			AND created_at >= datetime('now', '-3 seconds') LIMIT 1`, accountID, item.Type, item.Amount, item.Date, item.Description).Scan(&duplicateID)
		if duplicateID != "" {
			return nil
		}
		if item.Type == models.CreditTxCharge {
			_, err = tx.ExecContext(ctx, `UPDATE credit_accounts SET current_balance = current_balance + ?, updated_at = ? WHERE id = ?`, item.Amount, now, accountID)
		} else if item.Type == models.CreditTxPayment {
			_, err = tx.ExecContext(ctx, `UPDATE credit_accounts SET current_balance = MAX(0, current_balance - ?), updated_at = ? WHERE id = ?`, item.Amount, now, accountID)
		}
		if err != nil {
			return err
		}
	}
	if err := insertCreditTransaction(ctx, tx, item, now); err != nil {
		return err
	}
	if err := adjustCreditInstallment(ctx, tx, item, accountID, now, false); err != nil {
		return err
	}
	return tx.Commit()
}

func loadCreditTransaction(ctx context.Context, tx *sql.Tx, accountID, id string) (*models.CreditTransaction, error) {
	var item models.CreditTransaction
	var created, updated string
	err := tx.QueryRowContext(ctx, `SELECT id, account_id, type, amount, date, description, payment_type, installment_id,
		receipt_image, principal_amount, interest_amount, fee_amount, allocation_source, created_at, updated_at
		FROM credit_transactions WHERE id = ? AND account_id = ?`, id, accountID).Scan(
		&item.ID, &item.AccountID, &item.Type, &item.Amount, &item.Date, &item.Description, &item.PaymentType, &item.InstallmentID,
		&item.ReceiptImage, &item.PrincipalAmount, &item.InterestAmount, &item.FeeAmount, &item.AllocationSource, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCreditNotFound
	}
	if err != nil {
		return nil, err
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339, created)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &item, nil
}

func (r *CreditRepository) deleteAllocatedTransaction(ctx context.Context, id, accountID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	state, err := lockCreditAccount(ctx, tx, accountID, "")
	if err != nil {
		return err
	}
	item, err := loadCreditTransaction(ctx, tx, accountID, id)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if state.mode == models.CreditBalancePrincipalSplit {
		if err := validateCreditLedgerDate(item.Date, state); err != nil {
			return err
		}
		if err := validateSplitInstallment(ctx, tx, item, accountID, true); err != nil {
			return err
		}
		if err := applySplitBalance(state, item, true); err != nil {
			return err
		}
		if err := persistSplitBalance(ctx, tx, accountID, state, now); err != nil {
			return err
		}
	} else {
		if item.InstallmentID != nil && *item.InstallmentID != "" {
			if err := lockUnsettledInstallment(ctx, tx, *item.InstallmentID, accountID); err != nil {
				return err
			}
		}
		if item.Type == models.CreditTxCharge {
			_, err = tx.ExecContext(ctx, `UPDATE credit_accounts SET current_balance = MAX(0, current_balance - ?), updated_at = ? WHERE id = ?`, item.Amount, now, accountID)
		} else if item.Type == models.CreditTxPayment {
			_, err = tx.ExecContext(ctx, `UPDATE credit_accounts SET current_balance = current_balance + ?, updated_at = ? WHERE id = ?`, item.Amount, now, accountID)
		}
		if err != nil {
			return err
		}
	}
	if err := adjustCreditInstallment(ctx, tx, item, accountID, now, true); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM credit_transactions WHERE id = ? AND account_id = ?`, id, accountID); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateTransaction replaces a post-baseline split row without ever exposing a
// deleted row or a reversed-only balance. RULE replacements allocate from the
// liabilities after reversing the original's persisted (not recomputed) split.
func (r *CreditRepository) UpdateTransaction(ctx context.Context, accountID string, txID string, replacement *models.CreditTransaction) error {
	if replacement == nil {
		return fmt.Errorf("%w: replacement is required", ErrCreditInvalidAllocation)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	state, err := lockCreditAccount(ctx, tx, accountID, "")
	if err != nil {
		return err
	}
	if state.mode != models.CreditBalancePrincipalSplit {
		return fmt.Errorf("%w: editing transactions requires a split account", ErrCreditInvalidAllocation)
	}
	original, err := loadCreditTransaction(ctx, tx, accountID, txID)
	if err != nil {
		return err
	}
	if err := validateCreditLedgerDate(original.Date, state); err != nil {
		return err
	}
	link := func(value *string) string {
		if value == nil {
			return ""
		}
		return *value
	}
	if link(original.InstallmentID) != link(replacement.InstallmentID) {
		return fmt.Errorf("%w: editing cannot change the installment link", ErrCreditInvalidAllocation)
	}
	origP, origI, origF, err := splitComponents(original)
	if err != nil {
		return err
	}
	curP, curI, curF, err := state.buckets()
	if err != nil {
		return err
	}

	allocState := *state
	if original.Type == models.CreditTxPayment {
		allocP := curP + origP
		allocI := curI + origI
		allocF := curF + origF
		allocState.total = float64(allocP+allocI+allocF) / 100
		allocState.interest = float64(allocI) / 100
		allocState.fee = float64(allocF) / 100
	}
	if err := prepareSplitAllocation(replacement, &allocState); err != nil {
		return err
	}
	newP, newI, newF, err := splitComponents(replacement)
	if err != nil {
		return err
	}

	var deltaP, deltaI, deltaF int64
	if original.Type == models.CreditTxCharge {
		deltaP -= origP
		deltaI -= origI
		deltaF -= origF
	} else {
		deltaP += origP
		deltaI += origI
		deltaF += origF
	}
	if replacement.Type == models.CreditTxCharge {
		deltaP += newP
		deltaI += newI
		deltaF += newF
	} else {
		deltaP -= newP
		deltaI -= newI
		deltaF -= newF
	}

	finalP := curP + deltaP
	finalI := curI + deltaI
	finalF := curF + deltaF
	if finalP < 0 || finalI < 0 || finalF < 0 {
		return fmt.Errorf("%w: allocation exceeds a posted principal, interest or fee balance", ErrCreditConflict)
	}
	finalTotal := finalP + finalI + finalF
	if finalTotal > maxCreditCents {
		return fmt.Errorf("%w: balance exceeds supported money range", ErrCreditInvalidAllocation)
	}

	if err := validateSplitInstallment(ctx, tx, original, accountID, true); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := adjustCreditInstallment(ctx, tx, original, accountID, now, true); err != nil {
		return err
	}
	if err := validateSplitInstallment(ctx, tx, replacement, accountID, false); err != nil {
		return err
	}
	if err := adjustCreditInstallment(ctx, tx, replacement, accountID, now, false); err != nil {
		return err
	}

	state.total = float64(finalTotal) / 100
	state.interest = float64(finalI) / 100
	state.fee = float64(finalF) / 100
	if err := persistSplitBalance(ctx, tx, accountID, state, now); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE credit_transactions SET type = ?, amount = ?, date = ?, description = ?, payment_type = ?,
		installment_id = ?, receipt_image = ?, principal_amount = ?, interest_amount = ?, fee_amount = ?, allocation_source = ?, updated_at = ?
		WHERE id = ? AND account_id = ?`, replacement.Type, replacement.Amount, replacement.Date, replacement.Description, replacement.PaymentType,
		replacement.InstallmentID, replacement.ReceiptImage, replacement.PrincipalAmount, replacement.InterestAmount, replacement.FeeAmount,
		replacement.AllocationSource, now, txID, accountID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	replacement.ID, replacement.AccountID, replacement.CreatedAt = txID, accountID, original.CreatedAt
	replacement.UpdatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

// ReconcileAccount adopts official balances. Historical allocations annotate
// existing receipts only; they never post a second charge/payment or cash row.
func (r *CreditRepository) ReconcileAccount(ctx context.Context, accountID string, groupID string, req models.ReconcileCreditAccountRequest, userID ...string) error {
	if groupID == "" {
		return ErrCreditNotFound
	}
	if err := validateCreditConfig(req.CreditLimitBasis, req.AllocationOrder); err != nil {
		return err
	}
	if err := validateCreditLedgerDate(req.AsOfDate, &creditBalanceState{}); err != nil {
		return err
	}
	if req.AsOfDate > time.Now().In(installmentBillingLocation).Format("2006-01-02") {
		return fmt.Errorf("%w: baseline date cannot be in the future in Bangkok", ErrCreditInvalidAllocation)
	}
	if strings.TrimSpace(req.Note) == "" {
		return fmt.Errorf("%w: baseline reference note is required", ErrCreditInvalidAllocation)
	}
	p, err := creditCents(req.PrincipalBalance)
	if err != nil {
		return err
	}
	i, err := creditCents(req.InterestBalance)
	if err != nil {
		return err
	}
	f, err := creditCents(req.FeeBalance)
	if err != nil {
		return err
	}
	if p+i+f > maxCreditCents {
		return fmt.Errorf("%w: baseline total exceeds supported range", ErrCreditInvalidAllocation)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	state, err := lockCreditAccount(ctx, tx, accountID, groupID, userID...)
	if err != nil {
		return err
	}
	if state.asOf.Valid && req.AsOfDate < state.asOf.String {
		return fmt.Errorf("%w: baseline cannot move backwards", ErrCreditConflict)
	}
	var latest sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT MAX(date) FROM credit_transactions WHERE account_id = ?`, accountID).Scan(&latest); err != nil {
		return err
	}
	if latest.Valid && req.AsOfDate < latest.String {
		return fmt.Errorf("%w: baseline must include the latest ledger date", ErrCreditConflict)
	}
	if req.CreditLimitBasis != nil {
		state.basis = *req.CreditLimitBasis
	}
	if req.AllocationOrder != nil {
		state.order = *req.AllocationOrder
	}
	seen := make(map[string]bool, len(req.HistoricalAllocations))
	now := time.Now().UTC().Format(time.RFC3339)
	for _, allocation := range req.HistoricalAllocations {
		if seen[allocation.TransactionID] {
			return fmt.Errorf("%w: duplicate historical allocation", ErrCreditInvalidAllocation)
		}
		seen[allocation.TransactionID] = true
		item, err := loadCreditTransaction(ctx, tx, accountID, allocation.TransactionID)
		if err != nil {
			return err
		}
		if item.Date > req.AsOfDate {
			return fmt.Errorf("%w: historical allocation falls after baseline", ErrCreditConflict)
		}
		item.PrincipalAmount, item.InterestAmount, item.FeeAmount = &allocation.PrincipalAmount, &allocation.InterestAmount, &allocation.FeeAmount
		if _, _, _, err := splitComponents(item); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE credit_transactions SET principal_amount = ?, interest_amount = ?, fee_amount = ?, allocation_source = 'ACTUAL', updated_at = ? WHERE id = ? AND account_id = ?`,
			allocation.PrincipalAmount, allocation.InterestAmount, allocation.FeeAmount, now, item.ID, accountID)
		if err != nil {
			return err
		}
	}
	history, err := json.Marshal(req.HistoricalAllocations)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO credit_balance_reconciliations
		(id, account_id, group_id, user_id, as_of_date, principal_balance, interest_balance, fee_balance,
		credit_limit_basis, allocation_order, note, prior_balance, prior_mode, historical_allocations, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, uuid.New().String(), accountID, groupID, state.owner, req.AsOfDate,
		req.PrincipalBalance, req.InterestBalance, req.FeeBalance, state.basis, state.order, strings.TrimSpace(req.Note), state.total, state.mode, string(history), now)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE credit_accounts SET balance_mode = 'PRINCIPAL_SPLIT', current_balance = ?, interest_balance = ?, fee_balance = ?,
		credit_limit_basis = ?, allocation_order = ?, balance_as_of_date = ?, balance_note = ?, updated_at = ? WHERE id = ? AND group_id = ?`,
		float64(p+i+f)/100, req.InterestBalance, req.FeeBalance, state.basis, state.order, req.AsOfDate, strings.TrimSpace(req.Note), now, accountID, groupID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func normalizeCreditAccount(account *models.CreditAccount) {
	if account.BalanceMode == "" {
		account.BalanceMode = models.CreditBalanceLegacy
	}
	if account.CreditLimitBasis == "" {
		account.CreditLimitBasis = models.CreditLimitPrincipal
	}
	if account.AllocationOrder == "" {
		account.AllocationOrder = models.AllocationFeesInterestPrincipal
	}
}
