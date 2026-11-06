package repository

import (
	"context"
	"errors"
	"testing"

	"expense-tracker/internal/database"
	"expense-tracker/internal/models"
)

func TestDueSyncPreservesSplitAccountCashHistory(t *testing.T) {
	for _, tc := range []struct {
		name            string
		split           bool
		paid            bool
		initiallyMapped bool
		reject          bool
	}{
		{"paid split bill", true, true, true, true},
		{"paid bill newly mapped to split", true, true, false, true},
		{"pending split bill can accept estimate", true, false, true, false},
		{"legacy paid bill retains previous behavior", false, true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newAllocationTestDB(t)
			ctx := context.Background()
			credits := NewCreditRepository(db)
			account := models.CreditAccount{Name: tc.name, Type: models.CreditAccountPersonalLoan, CreditLimit: 1000, CurrentBalance: 500}
			if err := credits.CreateAccount(ctx, &account, "allocation-owner", "allocation-group"); err != nil {
				t.Fatal(err)
			}
			if tc.split {
				if err := credits.ReconcileAccount(ctx, account.ID, "allocation-group", models.ReconcileCreditAccountRequest{AsOfDate: "2020-01-01", PrincipalBalance: 500, Note: "test statement"}); err != nil {
					t.Fatal(err)
				}
			}
			repo := NewTransactionRepository(&database.DB{DB: db})
			status := models.StatusPending
			paidAmount := 0.0
			var receipt *string
			if tc.paid {
				status = models.StatusPaid
				paidAmount = 400
				value := "receipt"
				receipt = &value
			}
			req := models.CreateTransactionRequest{Type: models.TypeExpense, Category: "credit", Description: "bill", Amount: 400, Date: "2020-01-02", Status: &status, PaidAmount: &paidAmount, ReceiptImage: receipt}
			if tc.initiallyMapped {
				req.CreditAccountID = &account.ID
			}
			tx, err := repo.Create(ctx, req, "allocation-owner", "allocation-owner", "allocation-group")
			if err != nil {
				t.Fatal(err)
			}
			err = repo.BatchUpdateCreditLoanMapping(ctx, "allocation-group", []models.SyncDueUpdateItem{{TransactionID: tx.ID, Amount: 123, CreditAccountID: &account.ID}})
			if tc.reject && !errors.Is(err, ErrPaidSplitSyncConflict) {
				t.Fatalf("expected protected cash conflict, got %v", err)
			}
			if !tc.reject && err != nil {
				t.Fatal(err)
			}
			saved, err := repo.GetByID(ctx, tx.ID, "allocation-group")
			if err != nil {
				t.Fatal(err)
			}
			want := 123.0
			if tc.reject {
				want = 400
			}
			if saved.Amount != want || (tc.paid && (saved.PaidAmount == nil || *saved.PaidAmount != 400)) {
				t.Fatalf("cash amounts changed incorrectly: %#v", saved)
			}
		})
	}
}

func TestDueSyncProtectedRowRollsBackEntireBatch(t *testing.T) {
	db := newAllocationTestDB(t)
	ctx := context.Background()
	credits := NewCreditRepository(db)
	account := models.CreditAccount{Name: "split", Type: models.CreditAccountPersonalLoan, CreditLimit: 1000, CurrentBalance: 500}
	if err := credits.CreateAccount(ctx, &account, "allocation-owner", "allocation-group"); err != nil {
		t.Fatal(err)
	}
	if err := credits.ReconcileAccount(ctx, account.ID, "allocation-group", models.ReconcileCreditAccountRequest{AsOfDate: "2020-01-01", PrincipalBalance: 500, Note: "statement"}); err != nil {
		t.Fatal(err)
	}
	repo := NewTransactionRepository(&database.DB{DB: db})
	pending := models.StatusPending
	paid := models.StatusPaid
	zero := 0.0
	amount := 400.0
	receipt := "actual receipt"
	first, err := repo.Create(ctx, models.CreateTransactionRequest{Type: models.TypeExpense, Category: "credit", Amount: 100, Date: "2020-01-02", Status: &pending, PaidAmount: &zero, CreditAccountID: &account.ID}, "allocation-owner", "allocation-owner", "allocation-group")
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.Create(ctx, models.CreateTransactionRequest{Type: models.TypeExpense, Category: "credit", Amount: 400, Date: "2020-01-02", Status: &paid, PaidAmount: &amount, ReceiptImage: &receipt, CreditAccountID: &account.ID}, "allocation-owner", "allocation-owner", "allocation-group")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.BatchUpdateCreditLoanMapping(ctx, "allocation-group", []models.SyncDueUpdateItem{{TransactionID: first.ID, Amount: 200, CreditAccountID: &account.ID}, {TransactionID: second.ID, Amount: 10}})
	if !errors.Is(err, ErrPaidSplitSyncConflict) {
		t.Fatalf("expected protected split history conflict, got %v", err)
	}
	saved, err := repo.GetByID(ctx, first.ID, "allocation-group")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Amount != 100 {
		t.Fatalf("earlier batch row committed despite failure: %v", saved.Amount)
	}
}
