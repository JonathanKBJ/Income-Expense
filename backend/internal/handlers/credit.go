package handlers

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"

	"github.com/go-chi/chi/v5"
)

type CreditHandler struct {
	creditRepo *repository.CreditRepository
	groupRepo  *repository.GroupRepository
}

func NewCreditHandler(creditRepo *repository.CreditRepository, groupRepo *repository.GroupRepository) *CreditHandler {
	return &CreditHandler{
		creditRepo: creditRepo,
		groupRepo:  groupRepo,
	}
}
func (h *CreditHandler) getScopedUserID(ctx context.Context, groupID, userID string) string {
	settings, err := h.groupRepo.GetGroupSettings(ctx, groupID)
	if err == nil && !settings.ShareCredit {
		return userID
	}
	return ""
}

// ListAccounts handles GET /api/credit-accounts
func (h *CreditHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}
	scopedUser := h.getScopedUserID(r.Context(), groupID, userID)
	accounts, err := h.creditRepo.ListAccountsByGroup(r.Context(), groupID, scopedUser)
	if err != nil {
		log.Printf("ERROR: ListAccounts: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch credit accounts")
		return
	}

	accIDs := make([]string, len(accounts))
	for i, a := range accounts {
		accIDs[i] = a.ID
	}

	installmentsMap, err := h.creditRepo.ListInstallmentsByAccountIDs(r.Context(), accIDs)
	if err != nil {
		installmentsMap = make(map[string][]models.CreditInstallment)
	}

	txsMap, err := h.creditRepo.ListTransactionsByAccountIDs(r.Context(), accIDs)
	if err != nil {
		txsMap = make(map[string][]models.CreditTransaction)
	}

	details := make([]models.CreditAccountDetail, 0, len(accounts))
	var summary models.CreditDashboardSummary
	summary.TotalAccounts = len(accounts)

	for _, acc := range accounts {
		insts := installmentsMap[acc.ID]
		if insts == nil {
			insts = []models.CreditInstallment{}
		}
		txs := txsMap[acc.ID]
		if txs == nil {
			txs = []models.CreditTransaction{}
		}

		detail := computeAccountDetail(acc, insts, txs)
		details = append(details, detail)

		if acc.Status == models.CreditStatusActive {
			summary.ActiveAccounts++
			summary.TotalCreditLimit += acc.CreditLimit
			summary.TotalCurrentBalance += acc.CurrentBalance
			summary.TotalAvailableCredit += detail.AvailableCredit
			summary.TotalEstimatedDue += detail.TotalDueThisMonth
			summary.TotalNextCycleOutstanding += detail.NextCycleEstimatedStatement
			summary.TotalNextCycleEstimatedDue += detail.NextCycleEstimatedMin
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"accounts": details,
		"summary":  summary,
	})
}

// CreateAccount handles POST /api/credit-accounts
func (h *CreditHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "user and group identification required")
		return
	}
	// Creation is allowed; account is saved with creator's userID and groupID

	var req models.CreateCreditAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "account name is required")
		return
	}

	switch req.Type {
	case models.CreditAccountCreditCard, models.CreditAccountCashCard, models.CreditAccountPersonalLoan:
		// valid
	default:
		writeError(w, http.StatusBadRequest, "invalid credit account type")
		return
	}

	acc := models.CreditAccount{
		Name:            req.Name,
		Type:            req.Type,
		Bank:            strings.TrimSpace(req.Bank),
		CreditLimit:     req.CreditLimit,
		CurrentBalance:  req.CurrentBalance,
		StatementDay:    req.StatementDay,
		PaymentDueDay:   req.PaymentDueDay,
		InterestRate:    req.InterestRate,
		MinPaymentRate:  req.MinPaymentRate,
		MinPaymentFloor: req.MinPaymentFloor,
		Notes:           strings.TrimSpace(req.Notes),
	}

	if err := h.creditRepo.CreateAccount(r.Context(), &acc, userID, groupID); err != nil {
		log.Printf("ERROR: CreateAccount: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create credit account")
		return
	}

	detail := computeAccountDetail(acc, []models.CreditInstallment{}, []models.CreditTransaction{})
	writeJSON(w, http.StatusCreated, detail)
}

// GetAccountDetail handles GET /api/credit-accounts/{id}
func (h *CreditHandler) GetAccountDetail(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}
	scopedUser := h.getScopedUserID(r.Context(), groupID, userID)
	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID, scopedUser)
	if err != nil {
		log.Printf("ERROR: GetAccountDetail: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get credit account")
		return
	}
	if acc == nil {
		writeError(w, http.StatusNotFound, "credit account not found")
		return
	}

	insts, _ := h.creditRepo.ListInstallments(r.Context(), id)
	txs, _ := h.creditRepo.ListTransactions(r.Context(), id)

	detail := computeAccountDetail(*acc, insts, txs)
	writeJSON(w, http.StatusOK, detail)
}

// UpdateAccount handles PATCH /api/credit-accounts/{id}
func (h *CreditHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}
	scopedUser := h.getScopedUserID(r.Context(), groupID, userID)
	var req models.UpdateCreditAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.creditRepo.UpdateAccount(r.Context(), id, groupID, req, scopedUser); err != nil {
		log.Printf("ERROR: UpdateAccount: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update credit account")
		return
	}

	acc, _ := h.creditRepo.GetAccountByID(r.Context(), id, groupID, scopedUser)
	if acc == nil {
		writeError(w, http.StatusNotFound, "credit account not found")
		return
	}

	insts, _ := h.creditRepo.ListInstallments(r.Context(), id)
	txs, _ := h.creditRepo.ListTransactions(r.Context(), id)

	detail := computeAccountDetail(*acc, insts, txs)
	writeJSON(w, http.StatusOK, detail)
}

// DeleteAccount handles DELETE /api/credit-accounts/{id}
func (h *CreditHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}
	scopedUser := h.getScopedUserID(r.Context(), groupID, userID)
	if err := h.creditRepo.DeleteAccount(r.Context(), id, groupID, scopedUser); err != nil {
		log.Printf("ERROR: DeleteAccount: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete credit account")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "credit account deleted successfully"})
}

// --- Installments ---

// AddInstallment handles POST /api/credit-accounts/{id}/installments
func (h *CreditHandler) AddInstallment(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}
	scopedUser := h.getScopedUserID(r.Context(), groupID, userID)
	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID, scopedUser)
	if err != nil || acc == nil {
		writeError(w, http.StatusNotFound, "credit account not found")
		return
	}

	var req models.CreateCreditInstallmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.ItemName = strings.TrimSpace(req.ItemName)
	if req.ItemName == "" || req.TotalAmount <= 0 || req.MonthlyAmount <= 0 || req.TotalTerms <= 0 {
		writeError(w, http.StatusBadRequest, "item name, valid total/monthly amounts and terms are required")
		return
	}

	inst := models.CreditInstallment{
		AccountID:     id,
		ItemName:      req.ItemName,
		TotalAmount:   req.TotalAmount,
		MonthlyAmount: req.MonthlyAmount,
		TotalTerms:    req.TotalTerms,
		PaidTerms:     req.PaidTerms,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
		Notes:         strings.TrimSpace(req.Notes),
	}
	if req.InterestRate != nil {
		inst.InterestRate = *req.InterestRate
	}
	if req.InterestType != nil {
		inst.InterestType = *req.InterestType
	}
	if req.RemainingBalance != nil {
		inst.RemainingBalance = *req.RemainingBalance
	}
	if err := h.creditRepo.AddInstallment(r.Context(), &inst, id); err != nil {
		log.Printf("ERROR: AddInstallment: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to add installment")
		return
	}

	writeJSON(w, http.StatusCreated, inst)
}

// UpdateInstallment handles PATCH /api/credit-accounts/{id}/installments/{iid}
func (h *CreditHandler) UpdateInstallment(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	iid := chi.URLParam(r, "iid")
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}
	scopedUser := h.getScopedUserID(r.Context(), groupID, userID)
	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID, scopedUser)
	if err != nil || acc == nil {
		writeError(w, http.StatusNotFound, "credit account not found")
		return
	}

	var req models.UpdateCreditInstallmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.creditRepo.UpdateInstallment(r.Context(), iid, id, req); err != nil {
		log.Printf("ERROR: UpdateInstallment: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update installment")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "installment updated successfully"})
}

// DeleteInstallment handles DELETE /api/credit-accounts/{id}/installments/{iid}
func (h *CreditHandler) DeleteInstallment(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	iid := chi.URLParam(r, "iid")
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}
	scopedUser := h.getScopedUserID(r.Context(), groupID, userID)
	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID, scopedUser)
	if err != nil || acc == nil {
		writeError(w, http.StatusNotFound, "credit account not found")
		return
	}

	if err := h.creditRepo.DeleteInstallment(r.Context(), iid, id); err != nil {
		log.Printf("ERROR: DeleteInstallment: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete installment")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "installment deleted successfully"})
}

// --- Transactions ---

// AddTransaction handles POST /api/credit-accounts/{id}/transactions
func (h *CreditHandler) AddTransaction(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}
	scopedUser := h.getScopedUserID(r.Context(), groupID, userID)
	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID, scopedUser)
	if err != nil || acc == nil {
		writeError(w, http.StatusNotFound, "credit account not found")
		return
	}

	var req models.CreateCreditTxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type != models.CreditTxCharge && req.Type != models.CreditTxPayment {
		writeError(w, http.StatusBadRequest, "type must be CHARGE or PAYMENT")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}
	if req.Date == "" {
		writeError(w, http.StatusBadRequest, "date is required")
		return
	}

	tx := models.CreditTransaction{
		AccountID:     id,
		Type:          req.Type,
		Amount:        req.Amount,
		Date:          req.Date,
		Description:   strings.TrimSpace(req.Description),
		PaymentType:   req.PaymentType,
		InstallmentID: req.InstallmentID,
		ReceiptImage:  req.ReceiptImage,
	}

	if err := h.creditRepo.AddTransaction(r.Context(), &tx, id); err != nil {
		log.Printf("ERROR: AddTransaction: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to record transaction")
		return
	}

	// Fetch updated account detail
	updatedAcc, _ := h.creditRepo.GetAccountByID(r.Context(), id, groupID)
	insts, _ := h.creditRepo.ListInstallments(r.Context(), id)
	txs, _ := h.creditRepo.ListTransactions(r.Context(), id)

	detail := computeAccountDetail(*updatedAcc, insts, txs)
	writeJSON(w, http.StatusCreated, detail)
}

// DeleteTransaction handles DELETE /api/credit-accounts/{id}/transactions/{tid}
func (h *CreditHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")
	tid := chi.URLParam(r, "tid")
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}
	scopedUser := h.getScopedUserID(r.Context(), groupID, userID)
	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID, scopedUser)
	if err != nil || acc == nil {
		writeError(w, http.StatusNotFound, "credit account not found")
		return
	}

	if err := h.creditRepo.DeleteTransaction(r.Context(), tid, id); err != nil {
		log.Printf("ERROR: DeleteTransaction: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete transaction")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "transaction deleted successfully"})
}

// --- Computation Helper ---
func getBillingCycleStartDate(statementDay *int) string {
	today := time.Now().UTC()
	if statementDay == nil || *statementDay < 1 || *statementDay > 31 {
		return time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}

	stmtDay := *statementDay
	var year int
	var month time.Month

	if today.Day() >= stmtDay {
		year = today.Year()
		month = today.Month()
	} else {
		year = today.Year()
		month = today.Month() - 1
		if month < 1 {
			month = 12
			year--
		}
	}

	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	day := stmtDay
	if day > lastDay {
		day = lastDay
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}


func computeAccountDetail(acc models.CreditAccount, insts []models.CreditInstallment, txs []models.CreditTransaction) models.CreditAccountDetail {
	// Active installments monthly sum and unbilled future installments blocking credit limit
	var monthlyInstDue float64
	var unbilledInstallments float64
	activeInstCount := 0
	for _, inst := range insts {
		if inst.Status == models.InstallmentStatusActive && inst.PaidTerms < inst.TotalTerms {
			monthlyInstDue += inst.MonthlyAmount
			activeInstCount++

			// In banking practice, unbilled future installments block the credit line
			// But RECURRING charges (like insurance, subscriptions) are billed monthly and do not block future credit limits
			if inst.InterestType != "RECURRING" {
				if inst.RemainingBalance > 0 {
					unbilledInstallments += inst.RemainingBalance
				} else {
					remTerms := inst.TotalTerms - (inst.PaidTerms + 1)
					if remTerms > 0 {
						unbilledInstallments += float64(remTerms) * inst.MonthlyAmount
					}
				}
			}
		}
	}

	var availableCredit float64
	totalCreditUsed := acc.CurrentBalance + unbilledInstallments
	if acc.CreditLimit > 0 {
		availableCredit = acc.CreditLimit - totalCreditUsed
		if availableCredit < 0 {
			availableCredit = 0
		}
	}

	var utilization float64
	if acc.CreditLimit > 0 {
		utilization = (totalCreditUsed / acc.CreditLimit) * 100
		if utilization > 100 {
			utilization = 100
		}
	}

	// Cycle payments & charges
	cycleStart := getBillingCycleStartDate(acc.StatementDay)
	var paidThisCycle float64
	var chargesThisCycle float64
	for _, tx := range txs {
		if tx.Date >= cycleStart {
			if tx.Type == models.CreditTxPayment {
				paidThisCycle += tx.Amount
			} else if tx.Type == models.CreditTxCharge {
				chargesThisCycle += tx.Amount
			}
		}
	}

	// 1. Calculate original statement balance before cycle payments
	statementBal := acc.CurrentBalance + paidThisCycle - chargesThisCycle
	if statementBal < 0 {
		statementBal = 0
	}

	// 2. Minimum payment required for the current statement cycle
	rate := acc.MinPaymentRate
	if rate < 0 {
		rate = 0
	}
	floor := acc.MinPaymentFloor
	if floor < 0 {
		floor = 0
	}

	var minPayForCycle float64
	if statementBal > 0 {
		revolvingBalStmt := statementBal - monthlyInstDue
		if revolvingBalStmt < 0 {
			revolvingBalStmt = 0
		}
		var revMinStmt float64
		if rate > 0 {
			revMinStmt = revolvingBalStmt * (rate / 100.0)
		} else {
			// rate == 0: Interest-only payment (จ่ายเฉพาะดอกเบี้ย เช่น Finnix)
			if acc.InterestRate > 0 {
				rawInterest := revolvingBalStmt * (acc.InterestRate / 100.0) * (30.0 / 365.0)
				revMinStmt = math.Round(rawInterest * 100) / 100
			}
		}
		if floor > 0 && revMinStmt < floor && revolvingBalStmt > 0 {
			revMinStmt = floor
		}
		minPayForCycle = math.Ceil(monthlyInstDue + revMinStmt)
		if minPayForCycle > statementBal {
			minPayForCycle = statementBal
		}
	}

	// 3. Remaining due for the current cycle
	hasMinOrFullPayment := false
	for _, tx := range txs {
		if tx.Date >= cycleStart && tx.Type == models.CreditTxPayment {
			if tx.PaymentType != nil && (*tx.PaymentType == models.CreditPayMinimum || *tx.PaymentType == models.CreditPayFull) {
				hasMinOrFullPayment = true
				break
			}
		}
	}

	remainingDue := minPayForCycle - paidThisCycle
	if remainingDue <= 0 || hasMinOrFullPayment || (paidThisCycle > 0 && minPayForCycle-paidThisCycle <= 20.0) {
		remainingDue = 0
	}
	isPaidThisMonth := (minPayForCycle > 0 && (paidThisCycle >= minPayForCycle || hasMinOrFullPayment || remainingDue == 0)) || (acc.CurrentBalance == 0 && statementBal == 0)

	// 4. Estimated minimum payment for NEXT cycle (based on remaining current balance and interest accrued to next statement date)
	var nextCycleMin float64
	var nextCycleInterest float64
	var nextCycleEstimatedStmt float64

	if acc.CurrentBalance > 0 {
		revolvingBalCurr := acc.CurrentBalance
		unpaidInstThisCycle := monthlyInstDue - paidThisCycle
		if unpaidInstThisCycle > 0 {
			revolvingBalCurr = acc.CurrentBalance - unpaidInstThisCycle
			if revolvingBalCurr < 0 {
				revolvingBalCurr = 0
			}
		}
		// Calculate estimated interest accrued to next statement date (every statementDay)
		// Standard billing cycle is ~30 days. Daily interest = revolvingBal * (ir / 100) / 365
		if acc.InterestRate > 0 && revolvingBalCurr > 0 {
			rawInterest := revolvingBalCurr * (acc.InterestRate / 100.0) * (30.0 / 365.0)
			if acc.Type == models.CreditAccountCreditCard {
				// Thai banks add 7% VAT on credit card interest
				nextCycleInterest = math.Round(rawInterest * 1.07 * 100) / 100
			} else {
				// Personal loans / nano finance: interest is exempt from VAT
				nextCycleInterest = math.Round(rawInterest * 100) / 100
			}
		}

		nextCycleEstimatedStmt = math.Round((acc.CurrentBalance + nextCycleInterest) * 100) / 100

		// Next cycle minimum calculation includes revolving balance plus accrued interest
		revolvingBaseNext := revolvingBalCurr + nextCycleInterest
		var revMinCurr float64
		if rate > 0 {
			revMinCurr = revolvingBaseNext * (rate / 100.0)
		} else {
			// rate == 0: Interest-only payment (จ่ายเฉพาะดอกเบี้ย)
			revMinCurr = nextCycleInterest
		}
		if floor > 0 && revMinCurr < floor && revolvingBaseNext > 0 {
			revMinCurr = floor
		}
		nextCycleMin = math.Ceil(monthlyInstDue + revMinCurr)
		if nextCycleMin > nextCycleEstimatedStmt {
			nextCycleMin = nextCycleEstimatedStmt
		}
	}
	// Total due this month:
	totalDue := remainingDue

	return models.CreditAccountDetail{
		CreditAccount:          acc,
		AvailableCredit:        availableCredit,
		CreditUtilization:      utilization,
		UnbilledInstallments:   unbilledInstallments,
		MonthlyInstallmentDue:  monthlyInstDue,
		TotalDueThisMonth:      totalDue,
		EstimatedMinPayment:         minPayForCycle,
		PaidThisMonth:               paidThisCycle,
		IsPaidThisMonth:             isPaidThisMonth,
		NextCycleEstimatedMin:       nextCycleMin,
		NextCycleEstimatedInterest:  nextCycleInterest,
		NextCycleEstimatedStatement: nextCycleEstimatedStmt,
		ActiveInstallmentCount:      activeInstCount,
		Installments:           insts,
		Transactions:           txs,
	}
}
