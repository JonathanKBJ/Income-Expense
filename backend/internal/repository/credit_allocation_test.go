package repository

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"expense-tracker/internal/database"
	"expense-tracker/internal/models"
	_ "modernc.org/sqlite"
)

// Every regression uses a throwaway file and the real migrations/repository.
// Nothing reads application configuration or opens a production connection.
func newAllocationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "allocation.db")) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(4)
	wrapped := &database.DB{DB: db}
	if err := wrapped.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := wrapped.Migrate(); err != nil {
		t.Fatalf("idempotent migration: %v", err)
	}
	for _, statement := range []string{
		`INSERT INTO users (id, username, password_hash, role, status, created_at, updated_at) VALUES ('allocation-owner', 'allocation-owner', 'offline', 'USER', 'ACTIVE', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO users (id, username, password_hash, role, status, created_at, updated_at) VALUES ('allocation-other', 'allocation-other', 'offline', 'USER', 'ACTIVE', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO groups (id, name, created_at) VALUES ('allocation-group', 'Offline allocation', '2026-01-01T00:00:00Z')`,
		`INSERT INTO group_members (group_id, user_id, role) VALUES ('allocation-group', 'allocation-owner', 'OWNER'), ('allocation-group', 'allocation-other', 'EDITOR')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func newAllocationAccount(t *testing.T, repo *CreditRepository, balance float64) *models.CreditAccount {
	t.Helper()
	day := 15
	account := &models.CreditAccount{Name: "Offline card", Type: models.CreditAccountCashCard, CreditLimit: 82900, CurrentBalance: balance, StatementDay: &day}
	if err := repo.CreateAccount(context.Background(), account, "allocation-owner", "allocation-group"); err != nil {
		t.Fatal(err)
	}
	return account
}

func reconcileAllocation(t *testing.T, repo *CreditRepository, id string, principal, interest, fee float64) {
	t.Helper()
	if err := repo.ReconcileAccount(context.Background(), id, "allocation-group", models.ReconcileCreditAccountRequest{
		AsOfDate: "2026-01-01", PrincipalBalance: principal, InterestBalance: interest, FeeBalance: fee, Note: "Offline official statement",
	}, "allocation-owner"); err != nil {
		t.Fatal(err)
	}
}

func assertAllocationBalances(t *testing.T, repo *CreditRepository, id string, principal, interest, fee float64) {
	t.Helper()
	account, err := repo.GetAccountByID(context.Background(), id, "allocation-group", "allocation-owner")
	if err != nil || account == nil {
		t.Fatalf("load account: %v", err)
	}
	total, _ := creditCents(account.CurrentBalance)
	wantP, _ := creditCents(principal)
	wantI, _ := creditCents(interest)
	wantF, _ := creditCents(fee)
	gotI, _ := creditCents(account.InterestBalance)
	gotF, _ := creditCents(account.FeeBalance)
	if total-gotI-gotF != wantP || gotI != wantI || gotF != wantF {
		t.Fatalf("got principal/interest/fee %.2f/%.2f/%.2f, want %.2f/%.2f/%.2f", float64(total-gotI-gotF)/100, account.InterestBalance, account.FeeBalance, principal, interest, fee)
	}
}

func allocationPtr(value float64) *float64 { return &value }

func explicitAllocation(kind models.CreditTxType, amount, principal, interest, fee float64) *models.CreditTransaction {
	return &models.CreditTransaction{Type: kind, Date: "2026-01-02", Amount: amount,
		PrincipalAmount: allocationPtr(principal), InterestAmount: allocationPtr(interest), FeeAmount: allocationPtr(fee)}
}

func TestCreditReceiptBaselineAdoptionDoesNotPostCashOrTouchLegacy(t *testing.T) {
	ctx := context.Background()
	db := newAllocationTestDB(t)
	repo := NewCreditRepository(db)
	account := newAllocationAccount(t, repo, 82612.66)
	other := newAllocationAccount(t, repo, 1800)
	receipt := "receipt-offline"
	historical := &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 4000, Date: "2026-10-31", Description: "official receipt", ReceiptImage: &receipt}
	if err := repo.AddTransaction(ctx, historical, account.ID); err != nil {
		t.Fatal(err)
	}
	req := models.ReconcileCreditAccountRequest{AsOfDate: "2026-10-31", PrincipalBalance: 78013.33, InterestBalance: 599.33, FeeBalance: 0,
		Note: "Official 31 October receipt", HistoricalAllocations: []models.CreditHistoricalAllocation{{TransactionID: historical.ID, PrincipalAmount: 1592.11, InterestAmount: 2407.89, FeeAmount: 0}}}
	if err := repo.ReconcileAccount(ctx, account.ID, "allocation-group", req, "allocation-owner"); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 78013.33, 599.33, 0)
	updated, err := repo.GetAccountByID(ctx, account.ID, "allocation-group")
	if err != nil {
		t.Fatal(err)
	}
	if updated.BalanceMode != models.CreditBalancePrincipalSplit || updated.CreditLimitBasis != models.CreditLimitPrincipal || updated.BalanceAsOfDate == nil || *updated.BalanceAsOfDate != req.AsOfDate {
		t.Fatalf("baseline fields missing: %+v", updated)
	}
	limit, _ := creditCents(updated.CreditLimit)
	principal, _ := creditCents(78013.33)
	if limit-principal != 488667 {
		t.Fatal("receipt availability must be 4886.67")
	}
	rows, err := repo.ListTransactions(ctx, account.ID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("receipt duplicated: %d, %v", len(rows), err)
	}
	if rows[0].ID != historical.ID || rows[0].PrincipalAmount == nil || *rows[0].PrincipalAmount != 1592.11 || *rows[0].InterestAmount != 2407.89 || *rows[0].AllocationSource != models.AllocationActual || rows[0].ReceiptImage == nil || *rows[0].ReceiptImage != receipt {
		t.Fatalf("receipt allocation not adopted: %+v", rows[0])
	}
	var cashRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&cashRows); err != nil || cashRows != 0 {
		t.Fatalf("unexpected cash posting: %d %v", cashRows, err)
	}
	assertAllocationBalances(t, repo, other.ID, 1800, 0, 0)
	legacy, _ := repo.GetAccountByID(ctx, other.ID, "allocation-group")
	if legacy.BalanceMode != models.CreditBalanceLegacy {
		t.Fatal("other account opted in")
	}
	// The existing legacy clamped-overpayment and reversal behavior stays intact.
	legacyPayment := &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 2000, Date: "2026-11-01", Description: "legacy overpayment"}
	if err := repo.AddTransaction(ctx, legacyPayment, other.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, other.ID, 0, 0, 0)
	if err := repo.DeleteTransaction(ctx, legacyPayment.ID, other.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, other.ID, 2000, 0, 0)
	if err := repo.ReconcileAccount(ctx, account.ID, "allocation-group", req); err != nil {
		t.Fatal(err)
	}
	var auditCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM credit_balance_reconciliations WHERE account_id = ?`, account.ID).Scan(&auditCount); err != nil || auditCount != 2 {
		t.Fatalf("immutable snapshots: %d %v", auditCount, err)
	}
	if _, err := db.Exec(`UPDATE credit_balance_reconciliations SET note = 'rewrite' WHERE account_id = ?`, account.ID); err == nil {
		t.Fatal("audit update must be prohibited")
	}
	if _, err := db.Exec(`DELETE FROM credit_balance_reconciliations WHERE account_id = ?`, account.ID); err == nil {
		t.Fatal("audit deletion must be prohibited")
	}
}

func TestCreditSplitChargeActualRuleAndConfigReversalRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewCreditRepository(newAllocationTestDB(t))
	account := newAllocationAccount(t, repo, 0)
	reconcileAllocation(t, repo, account.ID, 1000, 100, 10)
	charge := explicitAllocation(models.CreditTxCharge, 125, 100, 20, 5)
	if err := repo.AddTransaction(ctx, charge, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 1100, 120, 15)
	payment := explicitAllocation(models.CreditTxPayment, 55, 30, 20, 5)
	if err := repo.AddTransaction(ctx, payment, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 1070, 100, 10)
	if err := repo.DeleteTransaction(ctx, payment.ID, account.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteTransaction(ctx, charge.ID, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 1000, 100, 10)
	rule := &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 25, Date: "2026-01-02"}
	if err := repo.AddTransaction(ctx, rule, account.ID); err != nil {
		t.Fatal(err)
	}
	if rule.AllocationSource == nil || *rule.AllocationSource != models.AllocationRule || *rule.PrincipalAmount != 0 || *rule.InterestAmount != 15 || *rule.FeeAmount != 10 {
		t.Fatalf("wrong default waterfall: %+v", rule)
	}
	order := models.AllocationInterestFeesPrincipal
	basis := models.CreditLimitTotal
	if err := repo.UpdateAccount(ctx, account.ID, "allocation-group", models.UpdateCreditAccountRequest{AllocationOrder: &order, CreditLimitBasis: &basis}); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteTransaction(ctx, rule.ID, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 1000, 100, 10)
	nextRule := &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 25, Date: "2026-01-02"}
	if err := repo.AddTransaction(ctx, nextRule, account.ID); err != nil {
		t.Fatal(err)
	}
	if *nextRule.InterestAmount != 25 || *nextRule.FeeAmount != 0 {
		t.Fatal("new configuration did not affect future allocation")
	}
	// Identical split inputs are real ledger entries, not silently deduplicated.
	for range 2 {
		if err := repo.AddTransaction(ctx, explicitAllocation(models.CreditTxPayment, 1, 1, 0, 0), account.ID); err != nil {
			t.Fatal(err)
		}
	}
	assertAllocationBalances(t, repo, account.ID, 998, 75, 10)
	batch, err := repo.ListTransactionsByAccountIDs(ctx, []string{account.ID})
	if err != nil || len(batch[account.ID]) != 3 {
		t.Fatalf("split duplicate swallowed or batch scan failed: %+v %v", batch, err)
	}
	for _, row := range batch[account.ID] {
		if row.PrincipalAmount == nil || row.AllocationSource == nil {
			t.Fatal("batch lost stored allocation")
		}
	}
	accounts, err := repo.ListAccountsByGroup(ctx, "allocation-group")
	if err != nil || len(accounts) != 1 || accounts[0].AllocationOrder != order || accounts[0].InterestBalance != 75 {
		t.Fatalf("account list lost buckets: %+v %v", accounts, err)
	}
	cronAccounts, err := repo.FindAccountsByStatementDay(ctx, 15)
	if err != nil || len(cronAccounts) != 1 || cronAccounts[0].BalanceMode != models.CreditBalancePrincipalSplit {
		t.Fatalf("cron lost mode: %+v %v", cronAccounts, err)
	}
}

func TestCreditSplitRejectsInvalidAndExcessAllocationsAtomically(t *testing.T) {
	ctx := context.Background()
	db := newAllocationTestDB(t)
	repo := NewCreditRepository(db)
	account := newAllocationAccount(t, repo, 0)
	reconcileAllocation(t, repo, account.ID, 100, 20, 5)
	cases := []struct {
		name string
		item *models.CreditTransaction
		want error
	}{
		{"negative", explicitAllocation(models.CreditTxPayment, 1, -1, 2, 0), ErrCreditInvalidAllocation},
		{"nonfinite", explicitAllocation(models.CreditTxPayment, math.Inf(1), 1, 0, 0), ErrCreditInvalidAllocation},
		{"nan", explicitAllocation(models.CreditTxCharge, 1, math.NaN(), 0, 0), ErrCreditInvalidAllocation},
		{"fractional cent", explicitAllocation(models.CreditTxPayment, 1.001, 1.001, 0, 0), ErrCreditInvalidAllocation},
		{"sum mismatch", explicitAllocation(models.CreditTxPayment, 2, 1, 0, 0), ErrCreditInvalidAllocation},
		{"principal overdraw", explicitAllocation(models.CreditTxPayment, 101, 101, 0, 0), ErrCreditConflict},
		{"interest overdraw", explicitAllocation(models.CreditTxPayment, 21, 0, 21, 0), ErrCreditConflict},
		{"fee overdraw", explicitAllocation(models.CreditTxPayment, 6, 0, 0, 6), ErrCreditConflict},
		{"partial components", &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 1, Date: "2026-01-02", PrincipalAmount: allocationPtr(1)}, ErrCreditInvalidAllocation},
		{"charge requires allocation", &models.CreditTransaction{Type: models.CreditTxCharge, Amount: 1, Date: "2026-01-02"}, ErrCreditInvalidAllocation},
		{"rule overpayment", &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 126, Date: "2026-01-02"}, ErrCreditConflict},
		{"frozen day", &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 1, Date: "2026-01-01"}, ErrCreditConflict},
		{"invalid date", &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 1, Date: "2026-02-30"}, ErrCreditInvalidAllocation},
		{"future date", &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 1, Date: "2099-01-01"}, ErrCreditInvalidAllocation},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := repo.AddTransaction(ctx, test.item, account.ID); !errors.Is(err, test.want) {
				t.Fatalf("want %v, got %v", test.want, err)
			}
			assertAllocationBalances(t, repo, account.ID, 100, 20, 5)
			rows, err := repo.ListTransactions(ctx, account.ID)
			if err != nil || len(rows) != 0 {
				t.Fatalf("rejected allocation posted row: %d %v", len(rows), err)
			}
		})
	}
	newBalance := 99.0
	legacyMode := models.CreditBalanceLegacy
	for _, request := range []models.UpdateCreditAccountRequest{{CurrentBalance: &newBalance}, {InterestBalance: &newBalance}, {BalanceMode: &legacyMode}} {
		if err := repo.UpdateAccount(ctx, account.ID, "allocation-group", request); !errors.Is(err, ErrCreditConflict) {
			t.Fatalf("ordinary update bypassed reconciliation: %v", err)
		}
	}
	assertAllocationBalances(t, repo, account.ID, 100, 20, 5)
}

func TestCreditSplitUpdateRollbackExactReversalAndBaselineFreeze(t *testing.T) {
	ctx := context.Background()
	repo := NewCreditRepository(newAllocationTestDB(t))
	account := newAllocationAccount(t, repo, 0)
	reconcileAllocation(t, repo, account.ID, 100, 20, 5)
	original := explicitAllocation(models.CreditTxPayment, 15, 10, 3, 2)
	if err := repo.AddTransaction(ctx, original, account.ID); err != nil {
		t.Fatal(err)
	}
	replacement := explicitAllocation(models.CreditTxPayment, 25, 20, 4, 1)
	if err := repo.UpdateTransaction(ctx, account.ID, original.ID, replacement); err != nil {
		t.Fatal(err)
	}
	if replacement.ID != original.ID || !replacement.CreatedAt.Equal(original.CreatedAt.Truncate(time.Second)) {
		t.Fatal("editing changed immutable transaction identity")
	}
	assertAllocationBalances(t, repo, account.ID, 80, 16, 4)
	invalid := explicitAllocation(models.CreditTxPayment, 200, 200, 0, 0)
	if err := repo.UpdateTransaction(ctx, account.ID, original.ID, invalid); !errors.Is(err, ErrCreditConflict) {
		t.Fatalf("expected rejected edit: %v", err)
	}
	assertAllocationBalances(t, repo, account.ID, 80, 16, 4)
	rows, err := repo.ListTransactions(ctx, account.ID)
	if err != nil || len(rows) != 1 || rows[0].Amount != 25 {
		t.Fatalf("failed edit changed row: %+v %v", rows, err)
	}
	if err := repo.DeleteTransaction(ctx, original.ID, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 100, 20, 5)
	charge := explicitAllocation(models.CreditTxCharge, 10, 10, 0, 0)
	if err := repo.AddTransaction(ctx, charge, account.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReconcileAccount(ctx, account.ID, "allocation-group", models.ReconcileCreditAccountRequest{AsOfDate: "2026-01-02", PrincipalBalance: 110, InterestBalance: 20, FeeBalance: 5, Note: "New official baseline"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteTransaction(ctx, charge.ID, account.ID); !errors.Is(err, ErrCreditConflict) {
		t.Fatalf("baseline allowed deletion: %v", err)
	}
	if err := repo.UpdateTransaction(ctx, account.ID, charge.ID, explicitAllocation(models.CreditTxCharge, 5, 5, 0, 0)); !errors.Is(err, ErrCreditConflict) {
		t.Fatalf("baseline allowed edit: %v", err)
	}
	assertAllocationBalances(t, repo, account.ID, 110, 20, 5)
}

func TestCreditReconcileScopeHistoryAndPartialFailureRollback(t *testing.T) {
	ctx := context.Background()
	db := newAllocationTestDB(t)
	repo := NewCreditRepository(db)
	account := newAllocationAccount(t, repo, 100)
	other := newAllocationAccount(t, repo, 200)
	history := &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 10, Date: "2026-01-03", Description: "history"}
	foreign := &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 20, Date: "2026-01-02", Description: "other"}
	if err := repo.AddTransaction(ctx, history, account.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddTransaction(ctx, foreign, other.ID); err != nil {
		t.Fatal(err)
	}
	req := models.ReconcileCreditAccountRequest{AsOfDate: "2026-01-03", PrincipalBalance: 90, Note: "Official receipt", HistoricalAllocations: []models.CreditHistoricalAllocation{{TransactionID: history.ID, PrincipalAmount: 10}}}
	if err := repo.ReconcileAccount(ctx, account.ID, "wrong-group", req); !errors.Is(err, ErrCreditNotFound) {
		t.Fatalf("cross-group reconcile: %v", err)
	}
	if err := repo.ReconcileAccount(ctx, account.ID, "allocation-group", req, "allocation-other"); !errors.Is(err, ErrCreditNotFound) {
		t.Fatalf("cross-owner reconcile: %v", err)
	}
	old := req
	old.AsOfDate = "2026-01-02"
	if err := repo.ReconcileAccount(ctx, account.ID, "allocation-group", old); !errors.Is(err, ErrCreditConflict) {
		t.Fatalf("baseline dropped subsequent history: %v", err)
	}
	req.HistoricalAllocations = append(req.HistoricalAllocations, models.CreditHistoricalAllocation{TransactionID: foreign.ID, PrincipalAmount: 20})
	if err := repo.ReconcileAccount(ctx, account.ID, "allocation-group", req); !errors.Is(err, ErrCreditNotFound) {
		t.Fatalf("cross-account history reconcile: %v", err)
	}
	rows, err := repo.ListTransactions(ctx, account.ID)
	if err != nil || len(rows) != 1 || rows[0].PrincipalAmount != nil || rows[0].AllocationSource != nil {
		t.Fatalf("failed reconcile partially annotated receipt: %+v %v", rows, err)
	}
	var audits int
	if err := db.QueryRow(`SELECT COUNT(*) FROM credit_balance_reconciliations`).Scan(&audits); err != nil || audits != 0 {
		t.Fatalf("failed reconcile wrote audit: %d %v", audits, err)
	}
	assertAllocationBalances(t, repo, account.ID, 90, 0, 0)
	assertAllocationBalances(t, repo, other.ID, 180, 0, 0)
}

func TestCreditSplitConcurrentPaymentsNeverOverdraw(t *testing.T) {
	ctx := context.Background()
	repo := NewCreditRepository(newAllocationTestDB(t))
	account := newAllocationAccount(t, repo, 0)
	reconcileAllocation(t, repo, account.ID, 100, 0, 0)
	start := make(chan struct{})
	errorsCh := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			errorsCh <- repo.AddTransaction(ctx, &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 100, Date: "2026-01-02"}, account.ID)
		}()
	}
	close(start)
	workers.Wait()
	close(errorsCh)
	succeeded, conflicted := 0, 0
	for err := range errorsCh {
		if err == nil {
			succeeded++
		} else if errors.Is(err, ErrCreditConflict) {
			conflicted++
		} else {
			t.Fatalf("unexpected concurrent writer error: %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("success/conflict = %d/%d", succeeded, conflicted)
	}
	assertAllocationBalances(t, repo, account.ID, 0, 0, 0)
	rows, err := repo.ListTransactions(ctx, account.ID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("concurrent overpayment posted: %d %v", len(rows), err)
	}
}

func TestCreditSplitLinkedInstallmentEditAndDeleteRestoreTerms(t *testing.T) {
	ctx := context.Background()
	repo := NewCreditRepository(newAllocationTestDB(t))
	account := newAllocationAccount(t, repo, 0)
	reconcileAllocation(t, repo, account.ID, 100, 20, 5)
	installment := &models.CreditInstallment{ItemName: "Actual purchase", TotalAmount: 100, MonthlyAmount: 50, TotalTerms: 2, StartDate: "2026-01-01"}
	if err := repo.AddInstallment(ctx, installment, account.ID); err != nil {
		t.Fatal(err)
	}
	first := explicitAllocation(models.CreditTxPayment, 10, 10, 0, 0)
	first.InstallmentID = &installment.ID
	if err := repo.AddTransaction(ctx, first, account.ID); err != nil {
		t.Fatal(err)
	}
	second := explicitAllocation(models.CreditTxPayment, 10, 10, 0, 0)
	second.InstallmentID = &installment.ID
	if err := repo.AddTransaction(ctx, second, account.ID); err != nil {
		t.Fatal(err)
	}
	plans, err := repo.ListInstallments(ctx, account.ID)
	if err != nil || len(plans) != 1 || plans[0].PaidTerms != 2 || plans[0].Status != models.InstallmentStatusCompleted {
		t.Fatalf("linked payment did not complete plan: %+v %v", plans, err)
	}
	replacement := explicitAllocation(models.CreditTxPayment, 12, 12, 0, 0)
	replacement.InstallmentID = &installment.ID
	if err := repo.UpdateTransaction(ctx, account.ID, second.ID, replacement); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 78, 20, 5)
	plans, err = repo.ListInstallments(ctx, account.ID)
	if err != nil || plans[0].PaidTerms != 2 || plans[0].Status != models.InstallmentStatusCompleted {
		t.Fatalf("edit double-advanced plan: %+v %v", plans, err)
	}
	unlinked := explicitAllocation(models.CreditTxPayment, 1, 1, 0, 0)
	if err := repo.UpdateTransaction(ctx, account.ID, second.ID, unlinked); !errors.Is(err, ErrCreditInvalidAllocation) {
		t.Fatalf("edit changed link: %v", err)
	}
	if err := repo.DeleteTransaction(ctx, second.ID, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 90, 20, 5)
	plans, err = repo.ListInstallments(ctx, account.ID)
	if err != nil || plans[0].PaidTerms != 1 || plans[0].Status != models.InstallmentStatusActive {
		t.Fatalf("delete did not restore plan: %+v %v", plans, err)
	}
	if err := repo.DeleteTransaction(ctx, first.ID, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 100, 20, 5)
}

func TestCreditSplitDeletingConsumedChargeFailsAtomically(t *testing.T) {
	ctx := context.Background()
	repo := NewCreditRepository(newAllocationTestDB(t))
	account := newAllocationAccount(t, repo, 0)
	reconcileAllocation(t, repo, account.ID, 0, 0, 0)
	charge := explicitAllocation(models.CreditTxCharge, 20, 10, 5, 5)
	if err := repo.AddTransaction(ctx, charge, account.ID); err != nil {
		t.Fatal(err)
	}
	payment := &models.CreditTransaction{Type: models.CreditTxPayment, Amount: 6, Date: "2026-01-03"}
	if err := repo.AddTransaction(ctx, payment, account.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteTransaction(ctx, charge.ID, account.ID); !errors.Is(err, ErrCreditConflict) {
		t.Fatalf("charge reversal overdraw not rejected: %v", err)
	}
	assertAllocationBalances(t, repo, account.ID, 10, 4, 0)
	rows, err := repo.ListTransactions(ctx, account.ID)
	if err != nil || len(rows) != 2 {
		t.Fatalf("failed reversal deleted row: %+v %v", rows, err)
	}
	if err := repo.DeleteTransaction(ctx, payment.ID, account.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteTransaction(ctx, charge.ID, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 0, 0, 0)
}

func TestCreditSplitConsumedChargeEditSucceedsAndNegativeDeltaRejected(t *testing.T) {
	ctx := context.Background()
	repo := NewCreditRepository(newAllocationTestDB(t))
	account := newAllocationAccount(t, repo, 0)
	reconcileAllocation(t, repo, account.ID, 0, 0, 0)

	// Post a 100 principal charge.
	charge := explicitAllocation(models.CreditTxCharge, 100, 100, 0, 0)
	charge.Description = "Original Charge"
	if err := repo.AddTransaction(ctx, charge, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 100, 0, 0)

	// Fully consume the charge with a 100 principal payment.
	payment := explicitAllocation(models.CreditTxPayment, 100, 100, 0, 0)
	if err := repo.AddTransaction(ctx, payment, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 0, 0, 0)

	// 1. Identical edit succeeds on fully consumed charge.
	identical := explicitAllocation(models.CreditTxCharge, 100, 100, 0, 0)
	identical.Description = "Identical Edit"
	if err := repo.UpdateTransaction(ctx, account.ID, charge.ID, identical); err != nil {
		t.Fatalf("identical consumed charge edit failed: %v", err)
	}
	assertAllocationBalances(t, repo, account.ID, 0, 0, 0)

	// 2. Relabelled edit (description and receipt) succeeds on fully consumed charge.
	receipt := "receipt-image-url"
	relabelled := explicitAllocation(models.CreditTxCharge, 100, 100, 0, 0)
	relabelled.Description = "Relabelled Grocery"
	relabelled.ReceiptImage = &receipt
	if err := repo.UpdateTransaction(ctx, account.ID, charge.ID, relabelled); err != nil {
		t.Fatalf("relabelled consumed charge edit failed: %v", err)
	}
	assertAllocationBalances(t, repo, account.ID, 0, 0, 0)
	rows, err := repo.ListTransactions(ctx, account.ID)
	if err != nil || len(rows) != 2 {
		t.Fatalf("unexpected rows: %+v %v", rows, err)
	}
	var updatedCharge *models.CreditTransaction
	for i := range rows {
		if rows[i].ID == charge.ID {
			updatedCharge = &rows[i]
		}
	}
	if updatedCharge == nil || updatedCharge.Description != "Relabelled Grocery" || updatedCharge.ReceiptImage == nil || *updatedCharge.ReceiptImage != receipt {
		t.Fatalf("relabelled charge was not saved correctly: %+v", updatedCharge)
	}

	// 3. Increased consumed charge edit succeeds with valid final buckets.
	increased := explicitAllocation(models.CreditTxCharge, 150, 150, 0, 0)
	increased.Description = "Increased Charge"
	if err := repo.UpdateTransaction(ctx, account.ID, charge.ID, increased); err != nil {
		t.Fatalf("increased consumed charge edit failed: %v", err)
	}
	assertAllocationBalances(t, repo, account.ID, 50, 0, 0)

	// 4. Invalid final negative delta rejected atomically.
	// Since current principal is 50 (from 150 charge - 100 payment), reducing the charge to 40
	// would result in net delta -110, final principal -60 < 0.
	invalidReduced := explicitAllocation(models.CreditTxCharge, 40, 40, 0, 0)
	invalidReduced.Description = "Invalid Reduction"
	if err := repo.UpdateTransaction(ctx, account.ID, charge.ID, invalidReduced); !errors.Is(err, ErrCreditConflict) {
		t.Fatalf("expected ErrCreditConflict for invalid negative delta, got %v", err)
	}
	// Balances and charge amount must remain unchanged.
	assertAllocationBalances(t, repo, account.ID, 50, 0, 0)
	rows, err = repo.ListTransactions(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if rows[i].ID == charge.ID && rows[i].Amount != 150 {
			t.Fatalf("failed edit modified transaction amount: %+v", rows[i])
		}
	}

	// Future-dated replacement edit is rejected atomically.
	futureReplacement := explicitAllocation(models.CreditTxCharge, 150, 150, 0, 0)
	futureReplacement.Date = "2099-01-01"
	if err := repo.UpdateTransaction(ctx, account.ID, charge.ID, futureReplacement); !errors.Is(err, ErrCreditInvalidAllocation) {
		t.Fatalf("expected ErrCreditInvalidAllocation for future date edit, got %v", err)
	}
	assertAllocationBalances(t, repo, account.ID, 50, 0, 0)

	// 5. Deletion must still reject removing a consumed charge.
	// Fully pay the remaining 50 principal first.
	payRemaining := explicitAllocation(models.CreditTxPayment, 50, 50, 0, 0)
	if err := repo.AddTransaction(ctx, payRemaining, account.ID); err != nil {
		t.Fatal(err)
	}
	assertAllocationBalances(t, repo, account.ID, 0, 0, 0)
	// Now deleting the 150 charge should fail because it is fully consumed.
	if err := repo.DeleteTransaction(ctx, charge.ID, account.ID); !errors.Is(err, ErrCreditConflict) {
		t.Fatalf("expected ErrCreditConflict when deleting consumed charge, got %v", err)
	}
	assertAllocationBalances(t, repo, account.ID, 0, 0, 0)
}

func TestAddLegacyTransactionRefusesSplitModeAndPreservesLegacyBehavior(t *testing.T) {
	ctx := context.Background()
	repo := NewCreditRepository(newAllocationTestDB(t))

	// Create an account in LEGACY mode with balance and an active installment.
	account := newAllocationAccount(t, repo, 100)
	installment := &models.CreditInstallment{ItemName: "Legacy Purchase", TotalAmount: 100, MonthlyAmount: 50, TotalTerms: 2, StartDate: "2026-01-01"}
	if err := repo.AddInstallment(ctx, installment, account.ID); err != nil {
		t.Fatal(err)
	}

	// Sequential race scenario:
	// Caller captures intent to post legacy payment to account.
	staleLegacyPayment := &models.CreditTransaction{
		Type:          models.CreditTxPayment,
		Amount:        50,
		Date:          "2026-01-02",
		Description:   "Dashboard cash payment",
		InstallmentID: &installment.ID,
	}

	// Meanwhile, the account is converted to PRINCIPAL_SPLIT via ReconcileAccount.
	reconcileReq := models.ReconcileCreditAccountRequest{
		AsOfDate:         "2026-01-02",
		PrincipalBalance: 100,
		InterestBalance:  0,
		FeeBalance:       0,
		Note:             "Official statement adoption",
	}
	if err := repo.ReconcileAccount(ctx, account.ID, "allocation-group", reconcileReq, "allocation-owner"); err != nil {
		t.Fatal(err)
	}

	// Now execute AddLegacyTransaction with the stale intent.
	// Must refuse split mode under account lock and return ErrCreditConflict.
	err := repo.AddLegacyTransaction(ctx, staleLegacyPayment, account.ID)
	if !errors.Is(err, ErrCreditConflict) {
		t.Fatalf("expected ErrCreditConflict for AddLegacyTransaction on split account, got %v", err)
	}

	// Verify no row was inserted, no installment term incremented, balances untouched.
	rows, err := repo.ListTransactions(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no transactions written, got %d", len(rows))
	}

	plans, err := repo.ListInstallments(ctx, account.ID)
	if err != nil || len(plans) != 1 || plans[0].PaidTerms != 0 || plans[0].Status != models.InstallmentStatusActive {
		t.Fatalf("installment was modified on rejected legacy transaction: %+v %v", plans, err)
	}

	assertAllocationBalances(t, repo, account.ID, 100, 0, 0)

	// Verify normal AddTransaction with RULE payment still works on split account.
	rulePayment := &models.CreditTransaction{
		Type:        models.CreditTxPayment,
		Amount:      30,
		Date:        "2026-01-03",
		Description: "Normal split payment",
	}
	if err := repo.AddTransaction(ctx, rulePayment, account.ID); err != nil {
		t.Fatalf("normal AddTransaction failed on split account: %v", err)
	}
	assertAllocationBalances(t, repo, account.ID, 70, 0, 0)

	// Now verify unconverted LEGACY account behavior remains completely unchanged.
	legacyAcc := newAllocationAccount(t, repo, 200)
	legacyInst := &models.CreditInstallment{ItemName: "Legacy Item", TotalAmount: 200, MonthlyAmount: 50, TotalTerms: 4, StartDate: "2026-01-01"}
	if err := repo.AddInstallment(ctx, legacyInst, legacyAcc.ID); err != nil {
		t.Fatal(err)
	}

	validLegacyPayment := &models.CreditTransaction{
		Type:          models.CreditTxPayment,
		Amount:        50,
		Date:          "2026-01-02",
		Description:   "Valid legacy payment",
		InstallmentID: &legacyInst.ID,
	}
	if err := repo.AddLegacyTransaction(ctx, validLegacyPayment, legacyAcc.ID); err != nil {
		t.Fatalf("AddLegacyTransaction failed on unconverted legacy account: %v", err)
	}

	legacyRows, err := repo.ListTransactions(ctx, legacyAcc.ID)
	if err != nil || len(legacyRows) != 1 {
		t.Fatalf("expected 1 legacy transaction, got %d %v", len(legacyRows), err)
	}

	legacyPlans, err := repo.ListInstallments(ctx, legacyAcc.ID)
	if err != nil || len(legacyPlans) != 1 || legacyPlans[0].PaidTerms != 1 {
		t.Fatalf("legacy installment not advanced: %+v %v", legacyPlans, err)
	}

	// Verify legacy balance reduced from 200 to 150
	fetchedAcc, err := repo.GetAccountByID(ctx, legacyAcc.ID, "allocation-group", "allocation-owner")
	if err != nil || fetchedAcc.CurrentBalance != 150 {
		t.Fatalf("expected legacy current_balance 150, got %v %v", fetchedAcc, err)
	}
}
