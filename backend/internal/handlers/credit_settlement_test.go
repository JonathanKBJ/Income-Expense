package handlers

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
)

func TestComputeAccountDetail_EarlySettlementReleasesOnlyClosedPlan(t *testing.T) {
	acc := models.CreditAccount{
		ID: "account", CreditLimit: 50000, CurrentBalance: 5000,
		MinPaymentRate: 5, Status: models.CreditStatusActive,
	}
	endDate := "2027-12-01"
	plan := models.CreditInstallment{
		ID: "settled-plan", AccountID: acc.ID, ItemName: "UOB installment",
		TotalAmount: 27048.72, MonthlyAmount: 486.64, TotalTerms: 48, PaidTerms: 33,
		RemainingBalance: 7299.57, InterestType: "FLAT", StartDate: "2023-12-01",
		EndDate: &endDate, Status: models.InstallmentStatusActive, Notes: "Original agreement",
	}
	other := models.CreditInstallment{
		ID: "unrelated-plan", AccountID: acc.ID, TotalAmount: 1200, MonthlyAmount: 200,
		TotalTerms: 6, PaidTerms: 1, RemainingBalance: 800, InterestType: "FLAT",
		Status: models.InstallmentStatusActive,
	}
	linkedID := plan.ID
	txs := []models.CreditTransaction{
		{ID: "historic-charge", AccountID: acc.ID, Type: models.CreditTxCharge, Amount: 486.64, Date: "2026-10-01", InstallmentID: &linkedID},
		{ID: "already-recorded-payment", AccountID: acc.ID, Type: models.CreditTxPayment, Amount: 19749.15, Date: "2026-10-02", InstallmentID: &linkedID},
	}
	history := append([]models.CreditTransaction(nil), txs...)
	ref := time.Date(2026, 11, 2, 12, 0, 0, 0, creditBillingLocation)
	before := computeAccountDetail(acc, []models.CreditInstallment{plan, other}, txs, ref)

	closed := plan
	settledDate := "2026-11-02"
	closed.Status = models.InstallmentStatusCompleted
	closed.RemainingBalance = 0
	closed.EarlySettlementDate = &settledDate
	closed.EarlySettlementNote = "Payment already recorded"
	after := computeAccountDetail(acc, []models.CreditInstallment{closed, other}, txs, ref)
	otherOnly := computeAccountDetail(acc, []models.CreditInstallment{other}, txs, ref)

	if math.Abs(after.AvailableCredit-before.AvailableCredit-7299.57) > 0.000001 {
		t.Fatalf("expected exactly 7299.57 of reserved credit released: before=%v after=%v", before.AvailableCredit, after.AvailableCredit)
	}
	if after.CurrentBalance != acc.CurrentBalance || after.UnbilledInstallments != 800 || after.MonthlyInstallmentDue != 200 || after.ActiveInstallmentCount != 1 {
		t.Fatalf("settlement must retain monetary balance and unrelated obligations: %+v", after)
	}
	if after.EstimatedMinPayment != otherOnly.EstimatedMinPayment || after.NextCycleEstimatedMin != otherOnly.NextCycleEstimatedMin || after.NextCycleEstimatedStatement != otherOnly.NextCycleEstimatedStatement {
		t.Fatalf("closed plan must not contribute to present/future estimates: after=%+v other=%+v", after, otherOnly)
	}
	if !reflect.DeepEqual(after.Transactions, history) || !reflect.DeepEqual(txs, history) || !reflect.DeepEqual(after.Installments[1], other) {
		t.Fatal("computation changed financial history or unrelated plan")
	}
	if after.Installments[0].PaidTerms != 33 || after.Installments[0].TotalTerms != 48 || after.Installments[0].MonthlyAmount != plan.MonthlyAmount || after.Installments[0].TotalAmount != plan.TotalAmount || after.Installments[0].StartDate != plan.StartDate || after.Installments[0].EndDate != plan.EndDate || after.Installments[0].Notes != plan.Notes {
		t.Fatal("closed plan must retain its original schedule, amounts and notes")
	}
}

func TestWriteInstallmentMutationError(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code int
	}{
		{repository.ErrInstallmentNotFound, http.StatusNotFound},
		{repository.ErrInvalidSettlementDate, http.StatusBadRequest},
		{repository.ErrInstallmentNotActive, http.StatusConflict},
		{repository.ErrInstallmentSettledEarly, http.StatusConflict},
	} {
		w := httptest.NewRecorder()
		if !writeInstallmentMutationError(w, fmt.Errorf("wrapped: %w", tc.err)) || w.Code != tc.code {
			t.Fatalf("error %v must map to %d; got %d", tc.err, tc.code, w.Code)
		}
	}
	if writeInstallmentMutationError(httptest.NewRecorder(), errors.New("database failure")) {
		t.Fatal("unexpected database failures must remain server errors")
	}
}
