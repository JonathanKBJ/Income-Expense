package handlers

import (
	"math"
	"testing"

	"expense-tracker/internal/models"
)

func TestComputeAccountDetail_UOB_TMRW(t *testing.T) {
	stmtDay := 17
	dueDay := 9
	acc := models.CreditAccount{
		ID:             "652752de-89bc-4970-ab5a-a080d5354ba6",
		Name:           "UOB TMRW",
		Type:           models.CreditAccountCreditCard,
		Bank:           "UOB",
		CreditLimit:    17000.00,
		CurrentBalance: 15593.00,
		StatementDay:   &stmtDay,
		PaymentDueDay:  &dueDay,
		InterestRate:   16.0,
		MinPaymentRate: 8.0,
		Status:         models.CreditStatusActive,
	}

	insts := []models.CreditInstallment{}

	minPayType := models.CreditPayMinimum
	txs := []models.CreditTransaction{
		{
			ID:          "payment-1",
			AccountID:   acc.ID,
			Type:        models.CreditTxPayment,
			Amount:      1323.56,
			Date:        "2026-10-01",
			PaymentType: &minPayType,
		},
		{
			ID:        "charge-1",
			AccountID: acc.ID,
			Type:      models.CreditTxCharge,
			Amount:    337.48,
			Date:      "2026-09-30",
		},
	}

	detail := computeAccountDetail(acc, insts, txs)

	if detail.AvailableCredit != 1407.00 {
		t.Errorf("expected AvailableCredit 1407.00, got %.2f", detail.AvailableCredit)
	}

	if !detail.IsPaidThisMonth {
		t.Errorf("expected IsPaidThisMonth true, got false")
	}

	if detail.TotalDueThisMonth != 0.00 {
		t.Errorf("expected TotalDueThisMonth 0.00, got %.2f", detail.TotalDueThisMonth)
	}

	if detail.EstimatedMinPayment != 1327.00 {
		t.Errorf("expected EstimatedMinPayment 1327.00, got %.2f", detail.EstimatedMinPayment)
	}

	if detail.PaidThisMonth != 1323.56 {
		t.Errorf("expected PaidThisMonth 1323.56, got %.2f", detail.PaidThisMonth)
	}

	// Next cycle interest: 15593 * 0.16 * 30 / 365 * 1.07 = ~219.41
	expectedInterest := math.Round(15593.00*0.16*(30.0/365.0)*1.07*100) / 100
	if detail.NextCycleEstimatedInterest != expectedInterest {
		t.Errorf("expected NextCycleEstimatedInterest %.2f, got %.2f", expectedInterest, detail.NextCycleEstimatedInterest)
	}

	// Next cycle statement: 15593 + 219.41 = 15812.41
	if detail.NextCycleEstimatedStatement != 15812.41 {
		t.Errorf("expected NextCycleEstimatedStatement 15812.41, got %.2f", detail.NextCycleEstimatedStatement)
	}

	// Next cycle min: ceil(15812.41 * 0.08) = 1265.00
	if detail.NextCycleEstimatedMin != 1265.00 {
		t.Errorf("expected NextCycleEstimatedMin 1265.00, got %.2f", detail.NextCycleEstimatedMin)
	}
}

func TestComputeAccountDetail_FirstChoice(t *testing.T) {
	stmtDay := 23
	dueDay := 13
	acc := models.CreditAccount{
		ID:             "38eee145-d3e3-4beb-906d-6946bb7a9a69",
		Name:           "First choice",
		Type:           models.CreditAccountPersonalLoan,
		Bank:           "BAY",
		CreditLimit:    23000.00,
		CurrentBalance: 28202.79,
		StatementDay:   &stmtDay,
		PaymentDueDay:  &dueDay,
		InterestRate:   25.0,
		MinPaymentRate: 3.706,
		Status:         models.CreditStatusActive,
	}

	// Chubb insurance recurring 100% full due
	insts := []models.CreditInstallment{
		{
			ID:            "inst-chubb",
			AccountID:     acc.ID,
			ItemName:      "KGIB_CHUBB SAMAGGI INSURANCE PCL_PA",
			TotalAmount:   45624.00,
			MonthlyAmount: 3802.00,
			TotalTerms:    12,
			PaidTerms:     0,
			InterestType:  "RECURRING",
			Status:        models.InstallmentStatusActive,
		},
	}

	txs := []models.CreditTransaction{}

	detail := computeAccountDetail(acc, insts, txs)

	// Total due this month must be 3802.00 + ceil((28202.79 - 3802.00) * 0.03706) = 3802.00 + 905.00 = 4707.00 (or ~4706.27)
	if detail.MonthlyInstallmentDue != 3802.00 {
		t.Errorf("expected MonthlyInstallmentDue 3802.00, got %.2f", detail.MonthlyInstallmentDue)
	}

	// RECURRING must NOT block unbilled installments!
	if detail.UnbilledInstallments != 0.0 {
		t.Errorf("expected UnbilledInstallments 0.0 for RECURRING, got %.2f", detail.UnbilledInstallments)
	}

	if detail.AvailableCredit != 0.0 {
		t.Errorf("expected AvailableCredit 0.0, got %.2f", detail.AvailableCredit)
	}

	// EstimatedMinPayment should be around 4,706 - 4,707
	if detail.EstimatedMinPayment < 4706.00 || detail.EstimatedMinPayment > 4707.00 {
		t.Errorf("expected EstimatedMinPayment ~4706.27, got %.2f", detail.EstimatedMinPayment)
	}

	if detail.TotalDueThisMonth < 4706.00 || detail.TotalDueThisMonth > 4707.00 {
		t.Errorf("expected TotalDueThisMonth ~4706.27, got %.2f", detail.TotalDueThisMonth)
	}
}

func TestComputeAccountDetail_Finnix_InterestOnly(t *testing.T) {
	stmtDay := 30
	dueDay := 6
	acc := models.CreditAccount{
		ID:             "finnix-test",
		Name:           "Finnix",
		Type:           models.CreditAccountPersonalLoan,
		Bank:           "SCB",
		CreditLimit:    30000.00,
		CurrentBalance: 25272.17,
		StatementDay:   &stmtDay,
		PaymentDueDay:  &dueDay,
		InterestRate:   33.0,
		MinPaymentRate: 0.0, // จ่ายเฉพาะดอกเบี้ย
		Status:         models.CreditStatusActive,
	}

	detail := computeAccountDetail(acc, []models.CreditInstallment{}, []models.CreditTransaction{})

	// Interest for 30 days = 25272.17 * 0.33 * 30 / 365 = 685.46
	expectedInterest := math.Round(25272.17*0.33*(30.0/365.0)*100) / 100
	if detail.EstimatedMinPayment != math.Ceil(expectedInterest) && detail.EstimatedMinPayment != expectedInterest {
		t.Errorf("expected EstimatedMinPayment around %.2f, got %.2f", expectedInterest, detail.EstimatedMinPayment)
	}

	// Next cycle min must be interest-only (685.46 or 686.00)
	if detail.NextCycleEstimatedMin != math.Ceil(expectedInterest) && detail.NextCycleEstimatedMin != expectedInterest {
		t.Errorf("expected NextCycleEstimatedMin around %.2f, got %.2f", expectedInterest, detail.NextCycleEstimatedMin)
	}
}
