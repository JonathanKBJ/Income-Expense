package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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

// ListAccounts handles GET /api/credit-accounts
func (h *CreditHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}

	accounts, err := h.creditRepo.ListAccountsByGroup(r.Context(), groupID)
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
	id := chi.URLParam(r, "id")

	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID)
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
	id := chi.URLParam(r, "id")

	var req models.UpdateCreditAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.creditRepo.UpdateAccount(r.Context(), id, groupID, req); err != nil {
		log.Printf("ERROR: UpdateAccount: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update credit account")
		return
	}

	acc, _ := h.creditRepo.GetAccountByID(r.Context(), id, groupID)
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
	id := chi.URLParam(r, "id")

	if err := h.creditRepo.DeleteAccount(r.Context(), id, groupID); err != nil {
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
	id := chi.URLParam(r, "id")

	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID)
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
		Notes:         strings.TrimSpace(req.Notes),
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
	id := chi.URLParam(r, "id")
	iid := chi.URLParam(r, "iid")

	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID)
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
	id := chi.URLParam(r, "id")
	iid := chi.URLParam(r, "iid")

	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID)
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
	id := chi.URLParam(r, "id")

	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID)
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
	id := chi.URLParam(r, "id")
	tid := chi.URLParam(r, "tid")

	acc, err := h.creditRepo.GetAccountByID(r.Context(), id, groupID)
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

func computeAccountDetail(acc models.CreditAccount, insts []models.CreditInstallment, txs []models.CreditTransaction) models.CreditAccountDetail {
	var availableCredit float64
	if acc.CreditLimit > 0 {
		availableCredit = acc.CreditLimit - acc.CurrentBalance
		if availableCredit < 0 {
			availableCredit = 0
		}
	}

	var utilization float64
	if acc.CreditLimit > 0 {
		utilization = (acc.CurrentBalance / acc.CreditLimit) * 100
	}

	// Estimated minimum payment
	var minPay float64
	if acc.CurrentBalance > 0 {
		rate := acc.MinPaymentRate
		if rate <= 0 {
			rate = 5.0
		}
		floor := acc.MinPaymentFloor
		if floor <= 0 {
			floor = 500.0
		}

		minPay = acc.CurrentBalance * (rate / 100.0)
		if minPay < floor {
			minPay = floor
		}
		if minPay > acc.CurrentBalance {
			minPay = acc.CurrentBalance
		}
	}

	// Active installments monthly sum
	var monthlyInstDue float64
	activeInstCount := 0
	for _, inst := range insts {
		if inst.Status == models.InstallmentStatusActive && inst.PaidTerms < inst.TotalTerms {
			monthlyInstDue += inst.MonthlyAmount
			activeInstCount++
		}
	}

	// Total due this month:
	var totalDue float64
	if acc.Type == models.CreditAccountPersonalLoan {
		if monthlyInstDue > 0 {
			totalDue = monthlyInstDue
		} else {
			totalDue = minPay
		}
	} else {
		// Credit / Cash Card: Estimated Minimum + active installments (if distinct)
		// Or simply the minimum payment if balance includes installments
		totalDue = minPay + monthlyInstDue
	}

	return models.CreditAccountDetail{
		CreditAccount:          acc,
		AvailableCredit:        availableCredit,
		CreditUtilization:      utilization,
		EstimatedMinPayment:    minPay,
		MonthlyInstallmentDue:  monthlyInstDue,
		TotalDueThisMonth:      totalDue,
		ActiveInstallmentCount: activeInstCount,
		Installments:           insts,
		Transactions:           txs,
	}
}
