package handlers

import (
	"math"
	"time"

	"expense-tracker/internal/models"
)

func splitCents(amount float64) float64 {
	return math.Round(amount*100) / 100
}

// computeSplitAccountDetail never posts estimates or interprets payment labels as
// money. The legacy branch intentionally remains separate and unchanged.
func computeSplitAccountDetail(acc models.CreditAccount, insts []models.CreditInstallment, txs []models.CreditTransaction, refDate ...time.Time) models.CreditAccountDetail {
	principal := splitCents(acc.CurrentBalance - acc.InterestBalance - acc.FeeBalance)
	detail := models.CreditAccountDetail{
		CreditAccount: acc, PrincipalBalance: principal, Installments: insts, Transactions: txs,
	}
	for _, inst := range insts {
		if inst.Status != models.InstallmentStatusActive || inst.PaidTerms >= inst.TotalTerms {
			continue
		}
		detail.MonthlyInstallmentDue += inst.MonthlyAmount
		detail.ActiveInstallmentCount++
		if inst.InterestType == "RECURRING" {
			continue
		}
		if inst.RemainingBalance > 0 {
			detail.UnbilledInstallments += inst.RemainingBalance
		} else if terms := inst.TotalTerms - inst.PaidTerms - 1; terms > 0 {
			detail.UnbilledInstallments += float64(terms) * inst.MonthlyAmount
		}
	}
	detail.MonthlyInstallmentDue = splitCents(detail.MonthlyInstallmentDue)
	detail.UnbilledInstallments = splitCents(detail.UnbilledInstallments)
	used := principal + detail.UnbilledInstallments
	if acc.CreditLimitBasis == models.CreditLimitTotal {
		used = acc.CurrentBalance + detail.UnbilledInstallments
	}
	if acc.CreditLimit > 0 {
		detail.AvailableCredit = math.Max(0, splitCents(acc.CreditLimit-used))
		detail.CreditUtilization = math.Min(100, used/acc.CreditLimit*100)
	}

	asOf := time.Now().In(creditBillingLocation)
	if len(refDate) > 0 && !refDate[0].IsZero() {
		asOf = refDate[0].In(creditBillingLocation)
	}
	asOfDate := asOf.Format("2006-01-02")
	cycleStart := getBillingCycleStartDate(acc.StatementDay, asOf)
	detail.UsesStatementMinimum = acc.StatementDay != nil && *acc.StatementDay >= 1 && *acc.StatementDay <= 31 &&
		acc.StatementDate != nil && acc.StatementBalance != nil && acc.StatementMinimumDue != nil && acc.StatementDueDate != nil &&
		validateStatementSnapshot(acc.StatementDate, acc.StatementBalance, acc.StatementMinimumDue, acc.StatementDueDate, false) == nil &&
		*acc.StatementDate == currentStatementCutoff(*acc.StatementDay, asOf)
	if detail.UsesStatementMinimum {
		cycleStart = *acc.StatementDate
	}

	// Reconstruct only known allocations, not inferred interest from descriptions
	// or historical unallocated rows. All dated payment money still counts toward
	// the cycle minimum, whether the allocation was ACTUAL or RULE.
	// Future-dated split entries are rejected at the repository boundary, ensuring
	// persisted balances and reconstructed statement debt reflect actual debt as of today.
	statementPrincipal, statementInterest, statementFee := principal, acc.InterestBalance, acc.FeeBalance
	for _, tx := range txs {
		if tx.Date < cycleStart || tx.Date > asOfDate {
			continue
		}
		if tx.Type == models.CreditTxPayment {
			detail.PaidThisMonth += tx.Amount
		}
		if tx.PrincipalAmount == nil || tx.InterestAmount == nil || tx.FeeAmount == nil {
			continue
		}
		if tx.Type == models.CreditTxPayment {
			statementPrincipal += *tx.PrincipalAmount
			statementInterest += *tx.InterestAmount
			statementFee += *tx.FeeAmount
		} else if tx.Type == models.CreditTxCharge && tx.Date > cycleStart {
			statementPrincipal -= *tx.PrincipalAmount
			statementInterest -= *tx.InterestAmount
			statementFee -= *tx.FeeAmount
		}
	}
	detail.PaidThisMonth = splitCents(detail.PaidThisMonth)
	rate, floor := math.Max(0, acc.MinPaymentRate), math.Max(0, acc.MinPaymentFloor)
	mode := acc.MinPaymentCalcMode
	if mode == "" {
		mode = models.CalcModeBalancePercent
		if rate == 0 {
			mode = models.CalcModeInterestOnly
		}
	}
	statementTotal := splitCents(statementPrincipal + statementInterest + statementFee)
	detail.EstimatedMinPayment = splitMinimum(mode, rate, floor, statementPrincipal, statementInterest, statementFee, detail.MonthlyInstallmentDue)
	detail.EstimatedMinPayment = math.Min(detail.EstimatedMinPayment, math.Max(0, statementTotal))
	if detail.UsesStatementMinimum {
		detail.EstimatedMinPayment = *acc.StatementMinimumDue
	}
	detail.TotalDueThisMonth = math.Max(0, splitCents(detail.EstimatedMinPayment-detail.PaidThisMonth))
	// Zero principal alone is not settlement: posted interest and fees are debt.
	detail.IsPaidThisMonth = detail.TotalDueThisMonth == 0 &&
		(detail.EstimatedMinPayment > 0 || acc.CurrentBalance == 0)
	if detail.UsesStatementMinimum {
		detail.IsPaidThisMonth = detail.TotalDueThisMonth == 0
	}

	if acc.CurrentBalance > 0 {
		// Forecast only. Interest never earns interest, and fees are not principal.
		if principal > 0 && acc.InterestRate > 0 {
			interest := principal * acc.InterestRate / 100 * 30 / 365
			if acc.Type == models.CreditAccountCreditCard {
				interest *= 1.07
			}
			detail.NextCycleEstimatedInterest = splitCents(interest)
		}
		detail.NextCycleEstimatedStatement = splitCents(acc.CurrentBalance + detail.NextCycleEstimatedInterest)
		detail.NextCycleEstimatedMin = splitMinimum(mode, rate, floor, principal,
			acc.InterestBalance+detail.NextCycleEstimatedInterest, acc.FeeBalance, detail.MonthlyInstallmentDue)
		detail.NextCycleEstimatedMin = math.Min(detail.NextCycleEstimatedMin, detail.NextCycleEstimatedStatement)
	}
	return detail
}

// Posted interest/fees are included exactly once. A schedule without an actual
// allocation is a minimum floor, not another posted liability or a guessed
// principal/interest breakdown. Callers cap the estimate by posted debt.
func splitMinimum(mode models.CreditMinPaymentCalcMode, rate, floor, principal, interest, fee, monthlyInstallment float64) float64 {
	var minimum float64
	switch mode {
	case models.CalcModePrincipalPlusInterest:
		minimum = principal*rate/100 + interest + fee
	case models.CalcModeInterestOnly:
		minimum = interest + fee
	default:
		minimum = (principal + interest + fee) * rate / 100
	}
	if principal+interest+fee > 0 && minimum < floor {
		minimum = floor
	}
	minimum = math.Max(minimum, monthlyInstallment)
	if mode == models.CalcModePrincipalPlusInterest {
		return splitCents(minimum)
	}
	return math.Ceil(minimum)
}
