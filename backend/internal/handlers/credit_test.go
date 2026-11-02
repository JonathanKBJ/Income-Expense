package handlers

import (
	"math"
	"testing"
	"time"

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

	detail := computeAccountDetail(acc, insts, txs, time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC))

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

func TestComputeAccountDetail_UOBCashPlus_BalancePercent(t *testing.T) {
	stmtDay := 17
	dueDay := 9
	acc := models.CreditAccount{
		ID:                 "0dab4204-7d3f-4050-a84c-af0bb74b7b58",
		Name:               "UOB CASH PLUS",
		Type:               models.CreditAccountCashCard,
		Bank:               "UOB",
		CreditLimit:        25500.00,
		CurrentBalance:     9962.37,
		StatementDay:       &stmtDay,
		PaymentDueDay:      &dueDay,
		InterestRate:       25.0,
		MinPaymentRate:     2.5,
		MinPaymentCalcMode: models.CalcModeBalancePercent,
		Status:             models.CreditStatusActive,
	}

	// Active installment with 486.41 monthly payment
	insts := []models.CreditInstallment{
		{
			ID:            "uob-inst-1",
			AccountID:     acc.ID,
			ItemName:      "Cash Plus Installment",
			TotalAmount:   15176.00,
			MonthlyAmount: 486.41,
			TotalTerms:    48,
			PaidTerms:     5,
			Status:        models.InstallmentStatusActive,
		},
	}
	payType := models.CreditPayMinimum
	txs := []models.CreditTransaction{
		{
			ID:          "uob-pay-1",
			AccountID:   acc.ID,
			Type:        models.CreditTxPayment,
			Amount:      742.00,
			Date:        "2026-09-30",
			PaymentType: &payType,
		},
	}

	detail := computeAccountDetail(acc, insts, txs, time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC))

	// Monthly installment due: 486.41
	if detail.MonthlyInstallmentDue != 486.41 {
		t.Errorf("expected MonthlyInstallmentDue 486.41, got %.2f", detail.MonthlyInstallmentDue)
	}

	// Next cycle estimated min MUST be 741.00 (NOT 941.00):
	// next cycle interest: 9962.37 * 0.25 * 30 / 365 = 204.71
	// revolving base next: 9962.37 + 204.71 = 10167.08
	// rev min next: ceil(10167.08 * 0.025) = 255.00
	// nextCycleMin = 486.41 + 255.00 = 741.41 -> ceil 742.00 (or 741.00)
	if detail.NextCycleEstimatedMin != 741.00 {
		t.Errorf("expected NextCycleEstimatedMin 741.00, got %.2f", detail.NextCycleEstimatedMin)
	}
}

func TestComputeAccountDetail_MoneyThunder_PrincipalPlusInterest(t *testing.T) {
	stmtDay := 9
	dueDay := 31
	principal := 79605.44
	interest := 1423.84

	acc := models.CreditAccount{
		ID:                 "78d75445-1cc5-4d2a-9818-5954737eca8c",
		Name:               "Money Thunder",
		Type:               models.CreditAccountPersonalLoan,
		Bank:               "SCB",
		CreditLimit:        82900.00,
		CurrentBalance:     principal,
		StatementDay:       &stmtDay,
		PaymentDueDay:      &dueDay,
		InterestRate:       33.0,
		MinPaymentRate:     2.0,
		MinPaymentCalcMode: models.CalcModePrincipalPlusInterest,
		Status:             models.CreditStatusActive,
	}

	// Interest charge billed on statement day
	txs := []models.CreditTransaction{
		{
			ID:          "mt-int-1",
			AccountID:   acc.ID,
			Type:        models.CreditTxCharge,
			Amount:      interest,
			Date:        "2026-10-09",
			Description: "ดอกเบี้ยรอบบิล 09/10/2569",
		},
	}

	detail := computeAccountDetail(acc, []models.CreditInstallment{}, txs, time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC))

	// Available credit must remain exactly 82900 - 79605.44 = 3294.56
	if math.Abs(detail.AvailableCredit-3294.56) > 0.01 {
		t.Errorf("expected AvailableCredit 3294.56, got %.2f", detail.AvailableCredit)
	}

	// Minimum payment due this cycle: (79605.44 * 0.02 = 1592.11) + 1423.84 = 3015.95
	expectedMin := 3015.95
	if math.Abs(detail.EstimatedMinPayment-expectedMin) > 0.01 {
		t.Errorf("expected EstimatedMinPayment %.2f, got %.2f", expectedMin, detail.EstimatedMinPayment)
	}
	if math.Abs(detail.TotalDueThisMonth-expectedMin) > 0.01 {
		t.Errorf("expected TotalDueThisMonth %.2f, got %.2f", expectedMin, detail.TotalDueThisMonth)
	}

	// Next cycle estimated min: (79605.44 * 0.02 = 1592.11) + (79605.44 * 0.33 * 30 / 365 = 2159.16) = 3751.27 -> ceil 3752.00
	if detail.NextCycleEstimatedMin != 3752.00 {
		t.Errorf("expected NextCycleEstimatedMin 3752.00, got %.2f", detail.NextCycleEstimatedMin)
	}
}

func TestComputeAccountDetail_OfficialStatementPayments(t *testing.T) {
	statementDay := 23
	statementDate, dueDate := "2026-10-23", "2026-11-12"
	statementBalance, minimum := 25648.94, 898.21
	asOf := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name      string
		payments  []models.CreditTransaction
		paid, due float64
	}{
		{name: "unpaid exact cents", due: 898.21},
		{name: "partial payment", payments: []models.CreditTransaction{{Type: models.CreditTxPayment, Amount: 300, Date: "2026-10-24"}}, paid: 300, due: 598.21},
		{name: "full payment", payments: []models.CreditTransaction{{Type: models.CreditTxPayment, Amount: 898.21, Date: "2026-10-26"}}, paid: 898.21},
		{name: "overpayment", payments: []models.CreditTransaction{{Type: models.CreditTxPayment, Amount: 1000, Date: "2026-10-26"}}, paid: 1000},
		{name: "multiple payments", payments: []models.CreditTransaction{
			{Type: models.CreditTxPayment, Amount: 300, Date: "2026-10-24"},
			{Type: models.CreditTxPayment, Amount: 598.21, Date: "2026-10-25"},
		}, paid: 898.21},
		{name: "no estimated twenty baht tolerance", payments: []models.CreditTransaction{{Type: models.CreditTxPayment, Amount: 880, Date: "2026-10-24"}}, paid: 880, due: 18.21},
		{name: "last cent remains due", payments: []models.CreditTransaction{{Type: models.CreditTxPayment, Amount: 898.20, Date: "2026-10-24"}}, paid: 898.20, due: 0.01},
		{name: "future payment ignored", payments: []models.CreditTransaction{{Type: models.CreditTxPayment, Amount: 898.21, Date: "2026-10-27"}}, due: 898.21},
		{name: "previous cycle payment ignored", payments: []models.CreditTransaction{{Type: models.CreditTxPayment, Amount: 3560.27, Date: "2026-09-30"}}, due: 898.21},
		{name: "cutoff day payment counts", payments: []models.CreditTransaction{{Type: models.CreditTxPayment, Amount: 300, Date: "2026-10-23"}}, paid: 300, due: 598.21},
	} {
		t.Run(tc.name, func(t *testing.T) {
			acc := models.CreditAccount{
				Type: models.CreditAccountCreditCard, CurrentBalance: statementBalance - tc.paid,
				StatementDay: &statementDay, StatementDate: &statementDate, StatementBalance: &statementBalance,
				StatementMinimumDue: &minimum, StatementDueDate: &dueDate, MinPaymentRate: 8,
			}
			detail := computeAccountDetail(acc, nil, tc.payments, asOf)
			if !detail.UsesStatementMinimum || detail.EstimatedMinPayment != minimum {
				t.Fatalf("official minimum lost: uses=%v minimum=%.2f", detail.UsesStatementMinimum, detail.EstimatedMinPayment)
			}
			if math.Abs(detail.PaidThisMonth-tc.paid) > 0.000001 || math.Abs(detail.TotalDueThisMonth-tc.due) > 0.000001 {
				t.Errorf("paid/due = %.2f/%.2f, want %.2f/%.2f", detail.PaidThisMonth, detail.TotalDueThisMonth, tc.paid, tc.due)
			}
			if detail.IsPaidThisMonth != (tc.due == 0) {
				t.Errorf("paid flag = %v for due %.2f", detail.IsPaidThisMonth, tc.due)
			}
		})
	}
}

func TestComputeAccountDetail_OfficialStatementIgnoresPaymentLabels(t *testing.T) {
	statementDay := 23
	statementDate, dueDate := "2026-10-23", "2026-11-12"
	balance, minimum := 25648.94, 898.21
	acc := models.CreditAccount{
		CurrentBalance: balance - 100, StatementDay: &statementDay, StatementDate: &statementDate,
		StatementBalance: &balance, StatementMinimumDue: &minimum, StatementDueDate: &dueDate,
	}
	for _, label := range []models.CreditPaymentType{models.CreditPayFull, models.CreditPayMinimum} {
		detail := computeAccountDetail(acc, nil, []models.CreditTransaction{
			{Type: models.CreditTxPayment, Amount: 100, Date: "2026-10-24", PaymentType: &label},
		}, time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC))
		if detail.IsPaidThisMonth || detail.TotalDueThisMonth != 798.21 {
			t.Errorf("%s underpayment incorrectly settled statement: paid=%v due=%.2f", label, detail.IsPaidThisMonth, detail.TotalDueThisMonth)
		}
	}
}

func TestComputeAccountDetail_OfficialStatementSurvivesNewChargesAndExpires(t *testing.T) {
	statementDay := 23
	statementDate, dueDate := "2026-10-23", "2026-11-12"
	balance, minimum := 25648.94, 898.21
	acc := models.CreditAccount{
		Type: models.CreditAccountCreditCard, CreditLimit: 50000, CurrentBalance: balance + 600,
		StatementDay: &statementDay, StatementDate: &statementDate, StatementBalance: &balance,
		StatementMinimumDue: &minimum, StatementDueDate: &dueDate, MinPaymentRate: 8,
	}
	txs := []models.CreditTransaction{
		{Type: models.CreditTxCharge, Amount: 500, Date: "2026-10-24", Description: "new purchase"},
		{Type: models.CreditTxCharge, Amount: 100, Date: "2026-10-25", Description: "interest"},
	}
	for _, date := range []time.Time{
		time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC),
	} {
		detail := computeAccountDetail(acc, nil, txs, date)
		if !detail.UsesStatementMinimum || detail.EstimatedMinPayment != minimum || detail.TotalDueThisMonth != minimum {
			t.Fatalf("snapshot minimum changed as of %s: %+v", date.Format("2006-01-02"), detail)
		}
		if detail.CurrentBalance != balance+600 || *detail.StatementBalance != balance {
			t.Fatal("live debt and original closing balance must remain separate")
		}
		if detail.NextCycleEstimatedMin == minimum {
			t.Fatal("next cycle must remain an independent estimate")
		}
	}
	nextCycle := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	detail := computeAccountDetail(acc, nil, txs, nextCycle)
	if detail.UsesStatementMinimum || detail.EstimatedMinPayment != 2100 {
		t.Fatalf("expired snapshot reused: uses=%v min=%.2f", detail.UsesStatementMinimum, detail.EstimatedMinPayment)
	}
	before := computeAccountDetail(acc, nil, nil, time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC))
	if before.UsesStatementMinimum {
		t.Fatal("future statement must not apply before cutoff")
	}
}

func TestComputeAccountDetail_OfficialStatementShortMonth(t *testing.T) {
	day := 31
	date, due := "2028-02-29", "2028-03-15"
	balance, minimum := 1000.0, 123.45
	acc := models.CreditAccount{
		StatementDay: &day, StatementDate: &date, StatementDueDate: &due,
		StatementBalance: &balance, StatementMinimumDue: &minimum, CurrentBalance: balance, MinPaymentRate: 5,
	}
	detail := computeAccountDetail(acc, nil, nil, time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC))
	if !detail.UsesStatementMinimum || detail.TotalDueThisMonth != minimum {
		t.Fatalf("last-day cutoff did not activate statement: %+v", detail)
	}
}

func TestComputeAccountDetail_NoSnapshotPreservesEstimatedTolerance(t *testing.T) {
	day := 23
	acc := models.CreditAccount{CurrentBalance: 9210, StatementDay: &day, MinPaymentRate: 8}
	detail := computeAccountDetail(acc, nil, []models.CreditTransaction{
		{Type: models.CreditTxPayment, Amount: 790, Date: "2026-10-24"},
	}, time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC))
	if detail.UsesStatementMinimum || detail.EstimatedMinPayment != 800 || detail.TotalDueThisMonth != 0 || !detail.IsPaidThisMonth {
		t.Fatalf("legacy estimated payment behavior changed: %+v", detail)
	}
}

func TestValidateStatementSnapshot(t *testing.T) {
	date, due := "2026-10-23", "2026-11-12"
	balance, minimum := 25648.94, 898.21
	if err := validateStatementSnapshot(&date, &balance, &minimum, &due, false); err != nil {
		t.Fatal(err)
	}
	if err := validateStatementSnapshot(nil, nil, nil, nil, false); err != nil {
		t.Fatal("omitting a snapshot must preserve it:", err)
	}
	if err := validateStatementSnapshot(nil, nil, nil, nil, true); err != nil {
		t.Fatal("explicit clear must be accepted:", err)
	}
	badDate, oldDue := "2026-02-30", "2026-10-22"
	negative, tooHigh, infinite, nan := -1.0, balance+1, math.Inf(1), math.NaN()
	for _, tc := range []struct {
		name             string
		date, due        *string
		balance, minimum *float64
		clear            bool
	}{
		{"partial", &date, nil, &balance, &minimum, false},
		{"impossible date", &badDate, &due, &balance, &minimum, false},
		{"due before statement", &date, &oldDue, &balance, &minimum, false},
		{"negative balance", &date, &due, &negative, &minimum, false},
		{"negative minimum", &date, &due, &balance, &negative, false},
		{"minimum above balance", &date, &due, &balance, &tooHigh, false},
		{"infinite balance", &date, &due, &infinite, &minimum, false},
		{"nan minimum", &date, &due, &balance, &nan, false},
		{"clear plus snapshot", &date, &due, &balance, &minimum, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateStatementSnapshot(tc.date, tc.balance, tc.minimum, tc.due, tc.clear); err == nil {
				t.Fatal("invalid snapshot accepted")
			}
		})
	}
}

func TestComputeAccountDetail_OfficialStatementBangkokBoundaries(t *testing.T) {
	day := 23
	date, dueDate := "2026-10-23", "2026-11-12"
	balance, minimum := 25648.94, 898.21
	account := models.CreditAccount{
		CurrentBalance: balance, StatementDay: &day, StatementDate: &date,
		StatementBalance: &balance, StatementMinimumDue: &minimum, StatementDueDate: &dueDate,
	}
	for _, tc := range []struct {
		name, instant string
		payment       bool
		active        bool
		remaining     float64
	}{
		{"before Thai cutoff", "2026-10-22T16:59:00Z", false, false, 0},
		{"Thai cutoff starts", "2026-10-22T17:00:00Z", false, true, 898.21},
		{"tomorrow in Thailand remains future", "2026-10-25T16:59:00Z", true, true, 898.21},
		{"Thai payment day begins before UTC", "2026-10-25T17:30:00Z", true, true, 0},
		{"last minute of Thai cycle", "2026-11-22T16:59:00Z", false, true, 898.21},
		{"next Thai cutoff expires snapshot", "2026-11-22T17:00:00Z", false, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asOf, err := time.Parse(time.RFC3339, tc.instant)
			if err != nil {
				t.Fatal(err)
			}
			var payments []models.CreditTransaction
			if tc.payment {
				payments = []models.CreditTransaction{{Type: models.CreditTxPayment, Amount: minimum, Date: "2026-10-26"}}
			}
			detail := computeAccountDetail(account, nil, payments, asOf)
			if detail.UsesStatementMinimum != tc.active {
				t.Fatalf("official statement active=%v, want %v", detail.UsesStatementMinimum, tc.active)
			}
			if tc.active && (detail.EstimatedMinPayment != minimum || detail.TotalDueThisMonth != tc.remaining) {
				t.Fatalf("original minimum/remaining = %.2f/%.2f, want %.2f/%.2f", detail.EstimatedMinPayment, detail.TotalDueThisMonth, minimum, tc.remaining)
			}
		})
	}
}

func TestDashboardPaymentSettlesStatementOnActualThaiPaymentDay(t *testing.T) {
	day := 23
	date, dueDate := "2026-10-23", "2026-11-12"
	balance, minimum := 25648.94, 898.21
	paidAt := time.Date(2026, 10, 25, 17, 30, 0, 0, time.UTC)
	for _, billDate := range []string{"2026-11-12", "2026-10-12"} {
		t.Run(billDate, func(t *testing.T) {
			bill := models.Transaction{Amount: minimum, Date: billDate, Description: "Uchoose"}
			payment := newDashboardCreditPayment("statement-account", &bill, paidAt)
			account := models.CreditAccount{
				CurrentBalance: balance - minimum, StatementDay: &day, StatementDate: &date,
				StatementBalance: &balance, StatementMinimumDue: &minimum, StatementDueDate: &dueDate,
			}
			detail := computeAccountDetail(account, nil, []models.CreditTransaction{payment}, paidAt)
			if !detail.UsesStatementMinimum || !detail.IsPaidThisMonth || detail.TotalDueThisMonth != 0 || detail.PaidThisMonth != minimum {
				t.Fatalf("paying dashboard bill dated %s did not settle today's official minimum: remaining=%.2f paid=%.2f", billDate, detail.TotalDueThisMonth, detail.PaidThisMonth)
			}
		})
	}
}
