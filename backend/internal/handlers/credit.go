package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var creditBillingLocation = time.FixedZone("Asia/Bangkok", 7*60*60)

type CreditHandler struct {
	creditRepo   *repository.CreditRepository
	groupRepo    *repository.GroupRepository
	telegramRepo *repository.TelegramRepository
	botToken     string
	botSecret    string
}

func NewCreditHandler(creditRepo *repository.CreditRepository, groupRepo *repository.GroupRepository) *CreditHandler {
	return &CreditHandler{
		creditRepo: creditRepo,
		groupRepo:  groupRepo,
	}
}

func (h *CreditHandler) WithTelegram(telegramRepo *repository.TelegramRepository, botToken, botSecret string) *CreditHandler {
	h.telegramRepo = telegramRepo
	h.botToken = botToken
	h.botSecret = botSecret
	return h
}

func (h *CreditHandler) validateBotSecret(r *http.Request) bool {
	if h.botSecret == "" {
		return true
	}
	secret := r.Header.Get("X-Telegram-Bot-Secret")
	if secret == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			secret = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	return secret == h.botSecret
}

func (h *CreditHandler) sendInstallmentTelegramAlert(groupID, accName string, inst models.CreditInstallment, termNum int, newBalance float64, dueDay *int) bool {
	if h.telegramRepo == nil || h.botToken == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chatID, err := h.telegramRepo.GetGroupTelegramChatID(ctx, groupID)
	if err != nil || chatID == "" {
		return false
	}

	dueStr := "-"
	if dueDay != nil && *dueDay >= 1 && *dueDay <= 31 {
		dueStr = fmt.Sprintf("วันที่ %d", *dueDay)
	}

	text := fmt.Sprintf(
		"🔔 *แจ้งเตือนรอบบิลอัตโนมัติ (Statement Due)*\n\n"+
			"💳 บัญชี: *%s*\n"+
			"📋 รายการ: *%s (งวดที่ %d/%d)*\n"+
			"💸 ยอดค่างวดรอบนี้: *฿%.2f*\n"+
			"📊 ยอดหนี้รวมคงค้างใหม่: *฿%.2f*\n"+
			"📅 วันครบกำหนดชำระ: *%s*",
		accName, inst.ItemName, termNum, inst.TotalTerms,
		inst.MonthlyAmount, newBalance, dueStr,
	)

	payload := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return false
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", h.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: sendInstallmentTelegramAlert: %v", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

type CronInstallmentItemResult struct {
	AccountID     string  `json:"accountId"`
	AccountName   string  `json:"accountName"`
	InstallmentID string  `json:"installmentId"`
	ItemName      string  `json:"itemName"`
	Term          int     `json:"term"`
	TotalTerms    int     `json:"totalTerms"`
	Amount        float64 `json:"amount"`
	NewBalance    float64 `json:"newBalance"`
	Status        string  `json:"status"`
	TelegramSent  bool    `json:"telegramSent"`
}

type CronPostInstallmentsResponse struct {
	ProcessedAt string                      `json:"processedAt"`
	TargetDay   int                         `json:"targetDay"`
	DryRun      bool                        `json:"dryRun"`
	Accounts    int                         `json:"accountsChecked"`
	Items       []CronInstallmentItemResult `json:"items"`
}

// CronPostInstallments handles POST and GET /api/telegram/cron/post-installments and /api/credit/cron/post-installments
func (h *CreditHandler) CronPostInstallments(w http.ResponseWriter, r *http.Request) {
	if !h.validateBotSecret(r) {
		userID := middleware.GetUserID(r.Context())
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized: valid bot secret or auth token required")
			return
		}
	}

	// Determine Bangkok time and target statement day
	bkkLoc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		bkkLoc = time.FixedZone("Asia/Bangkok", 7*3600)
	}
	nowBkk := time.Now().In(bkkLoc)
	targetDay := nowBkk.Day()

	// Check overrides from query param or JSON body
	dryRun := false
	if r.URL.Query().Get("dry_run") == "true" {
		dryRun = true
	}
	if dayStr := r.URL.Query().Get("day"); dayStr != "" {
		if d, err := strconv.Atoi(dayStr); err == nil && d >= 1 && d <= 31 {
			targetDay = d
		}
	}

	if r.Method == http.MethodPost && r.Body != nil {
		var req struct {
			Day    *int  `json:"day"`
			DryRun *bool `json:"dryRun"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.Day != nil && *req.Day >= 1 && *req.Day <= 31 {
				targetDay = *req.Day
			}
			if req.DryRun != nil {
				dryRun = *req.DryRun
			}
		}
	}

	accounts, err := h.creditRepo.FindAccountsByStatementDay(r.Context(), targetDay)
	if err != nil {
		log.Printf("ERROR: FindAccountsByStatementDay: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to query accounts for statement day")
		return
	}

	resp := CronPostInstallmentsResponse{
		ProcessedAt: nowBkk.Format(time.RFC3339),
		TargetDay:   targetDay,
		DryRun:      dryRun,
		Accounts:    len(accounts),
		Items:       []CronInstallmentItemResult{},
	}

	sinceDate := nowBkk.AddDate(0, 0, -32).Format("2006-01-02")
	todayStr := nowBkk.Format("2006-01-02")

	for _, acc := range accounts {
		insts, err := h.creditRepo.ListInstallments(r.Context(), acc.ID)
		if err != nil {
			log.Printf("WARN: ListInstallments failed for acc %s: %v", acc.ID, err)
			continue
		}

		currentAccBalance := acc.CurrentBalance

		for _, inst := range insts {
			if inst.Status != models.InstallmentStatusActive {
				continue
			}
			if inst.PaidTerms >= inst.TotalTerms {
				continue
			}

			termNum := inst.PaidTerms + 1
			alreadyCharged, err := h.creditRepo.IsInstallmentChargedInCycle(r.Context(), acc.ID, inst.ID, termNum, inst.TotalTerms, sinceDate)
			if err != nil {
				log.Printf("WARN: IsInstallmentChargedInCycle failed for acc %s, inst %s: %v", acc.ID, inst.ID, err)
				continue
			}

			if alreadyCharged {
				resp.Items = append(resp.Items, CronInstallmentItemResult{
					AccountID:     acc.ID,
					AccountName:   acc.Name,
					InstallmentID: inst.ID,
					ItemName:      inst.ItemName,
					Term:          termNum,
					TotalTerms:    inst.TotalTerms,
					Amount:        inst.MonthlyAmount,
					NewBalance:    currentAccBalance,
					Status:        "SKIPPED_ALREADY_CHARGED",
					TelegramSent:  false,
				})
				continue
			}

			// Post CHARGE transaction
			newBal := currentAccBalance + inst.MonthlyAmount
			telegramSent := false

			if !dryRun {
				tx := models.CreditTransaction{
					ID:            uuid.New().String(),
					AccountID:     acc.ID,
					Type:          models.CreditTxCharge,
					Amount:        inst.MonthlyAmount,
					Date:          todayStr,
					Description:   fmt.Sprintf("ค่างวด %s (งวดที่ %d/%d)", inst.ItemName, termNum, inst.TotalTerms),
					InstallmentID: &inst.ID,
				}
				if err := h.creditRepo.AddTransaction(r.Context(), &tx, acc.ID); err != nil {
					log.Printf("ERROR: AddTransaction for cron installment failed: %v", err)
					continue
				}
				currentAccBalance = newBal
				telegramSent = h.sendInstallmentTelegramAlert(acc.GroupID, acc.Name, inst, termNum, newBal, acc.PaymentDueDay)
			}

			resp.Items = append(resp.Items, CronInstallmentItemResult{
				AccountID:     acc.ID,
				AccountName:   acc.Name,
				InstallmentID: inst.ID,
				ItemName:      inst.ItemName,
				Term:          termNum,
				TotalTerms:    inst.TotalTerms,
				Amount:        inst.MonthlyAmount,
				NewBalance:    newBal,
				Status:        "POSTED",
				TelegramSent:  telegramSent,
			})
		}
	}

	writeJSON(w, http.StatusOK, resp)
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

	if err := validateStatementSnapshot(req.StatementDate, req.StatementBalance, req.StatementMinimumDue, req.StatementDueDate, false); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	calcMode := models.CalcModeBalancePercent
	if req.MinPaymentCalcMode != nil && *req.MinPaymentCalcMode != "" {
		calcMode = *req.MinPaymentCalcMode
	}

	acc := models.CreditAccount{
		Name:                req.Name,
		Type:                req.Type,
		Bank:                strings.TrimSpace(req.Bank),
		CreditLimit:         req.CreditLimit,
		CurrentBalance:      req.CurrentBalance,
		StatementDay:        req.StatementDay,
		PaymentDueDay:       req.PaymentDueDay,
		StatementDate:       req.StatementDate,
		StatementBalance:    req.StatementBalance,
		StatementMinimumDue: req.StatementMinimumDue,
		StatementDueDate:    req.StatementDueDate,
		InterestRate:        req.InterestRate,
		MinPaymentRate:      req.MinPaymentRate,
		MinPaymentFloor:     req.MinPaymentFloor,
		MinPaymentCalcMode:  calcMode,
		Notes:               strings.TrimSpace(req.Notes),
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
	if err := validateStatementSnapshot(req.StatementDate, req.StatementBalance, req.StatementMinimumDue, req.StatementDueDate, req.ClearStatement); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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
func getBillingCycleStartDate(statementDay *int, refDate ...time.Time) string {
	today := time.Now().UTC()
	if len(refDate) > 0 && !refDate[0].IsZero() {
		today = refDate[0]
	}
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

// validateStatementSnapshot accepts an omitted snapshot or a complete official
// statement. Clearing and replacing in the same request is ambiguous.
func validateStatementSnapshot(date *string, balance, minimum *float64, due *string, clear bool) error {
	count := 0
	for _, present := range []bool{date != nil, balance != nil, minimum != nil, due != nil} {
		if present {
			count++
		}
	}
	if clear && count > 0 {
		return fmt.Errorf("clearStatement cannot be combined with statement fields")
	}
	if count == 0 {
		return nil
	}
	if count != 4 {
		return fmt.Errorf("all four statement fields are required together")
	}
	statementDate, err := time.Parse("2006-01-02", *date)
	if err != nil || statementDate.Format("2006-01-02") != *date {
		return fmt.Errorf("statementDate must be a valid YYYY-MM-DD date")
	}
	dueDate, err := time.Parse("2006-01-02", *due)
	if err != nil || dueDate.Format("2006-01-02") != *due || dueDate.Before(statementDate) {
		return fmt.Errorf("statementDueDate must be a valid YYYY-MM-DD date on or after statementDate")
	}
	if math.IsNaN(*balance) || math.IsInf(*balance, 0) || *balance < 0 ||
		math.IsNaN(*minimum) || math.IsInf(*minimum, 0) || *minimum < 0 || *minimum > *balance {
		return fmt.Errorf("statement amounts must be finite and nonnegative, with minimum no greater than balance")
	}
	return nil
}

// currentStatementCutoff clamps short months before deciding which cycle is
// current (a day-31 account closes on February's last day).
func currentStatementCutoff(day int, asOf time.Time) string {
	cutoff := func(year int, month time.Month) time.Time {
		last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
		return time.Date(year, month, min(day, last), 0, 0, 0, 0, time.UTC)
	}
	current := cutoff(asOf.Year(), asOf.Month())
	if asOf.Format("2006-01-02") < current.Format("2006-01-02") {
		current = cutoff(asOf.Year(), asOf.Month()-1)
	}
	return current.Format("2006-01-02")
}

func computeAccountDetail(acc models.CreditAccount, insts []models.CreditInstallment, txs []models.CreditTransaction, refDate ...time.Time) models.CreditAccountDetail {
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
	cycleStart := getBillingCycleStartDate(acc.StatementDay, refDate...)
	asOf := time.Now().In(creditBillingLocation)
	if len(refDate) > 0 && !refDate[0].IsZero() {
		asOf = refDate[0].In(creditBillingLocation)
	}
	usesStatementMinimum := acc.StatementDay != nil && *acc.StatementDay >= 1 && *acc.StatementDay <= 31 &&
		acc.StatementDate != nil && acc.StatementBalance != nil && acc.StatementMinimumDue != nil && acc.StatementDueDate != nil &&
		validateStatementSnapshot(acc.StatementDate, acc.StatementBalance, acc.StatementMinimumDue, acc.StatementDueDate, false) == nil &&
		*acc.StatementDate == currentStatementCutoff(*acc.StatementDay, asOf)
	if usesStatementMinimum {
		cycleStart = *acc.StatementDate
	}
	asOfDate := asOf.Format("2006-01-02")
	var paidThisCycle float64
	var chargesThisCycle float64
	var interestChargesThisCycle float64
	for _, tx := range txs {
		if tx.Date >= cycleStart && (!usesStatementMinimum || tx.Date <= asOfDate) {
			if tx.Type == models.CreditTxPayment {
				paidThisCycle += tx.Amount
			} else if tx.Type == models.CreditTxCharge {
				descLower := strings.ToLower(tx.Description)
				isInterest := strings.Contains(descLower, "ดอกเบี้ย") || strings.Contains(descLower, "interest")
				isInstallment := (tx.InstallmentID != nil && *tx.InstallmentID != "") ||
					strings.Contains(descLower, "ค่างวด") || strings.Contains(descLower, "installment")

				if isInterest {
					interestChargesThisCycle += tx.Amount
				} else if isInstallment {
					// Installment charges are part of the statement, not post-cutoff new swipes
					continue
				} else if tx.Date > cycleStart {
					// Only new revolving purchases strictly after statement cutoff reduce statement balance
					chargesThisCycle += tx.Amount
				}
			}
		}
	}

	// 1. Calculate original statement balance before cycle payments
	statementBal := acc.CurrentBalance + paidThisCycle - chargesThisCycle
	if statementBal < 0 {
		statementBal = 0
	}
	if usesStatementMinimum {
		statementBal = *acc.StatementBalance
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

	calcMode := acc.MinPaymentCalcMode
	if calcMode == "" {
		if rate == 0 {
			calcMode = models.CalcModeInterestOnly
		} else {
			calcMode = models.CalcModeBalancePercent
		}
	}

	var minPayForCycle float64
	if statementBal > 0 && !usesStatementMinimum {
		revolvingBalStmt := statementBal - monthlyInstDue
		if revolvingBalStmt < 0 {
			revolvingBalStmt = 0
		}
		var revMinStmt float64
		switch calcMode {
		case models.CalcModePrincipalPlusInterest:
			// Principal % + 100% of cycle interest (e.g. Money Thunder)
			cycleInterest := interestChargesThisCycle
			if cycleInterest == 0 && acc.InterestRate > 0 {
				rawInterest := revolvingBalStmt * (acc.InterestRate / 100.0) * (30.0 / 365.0)
				cycleInterest = math.Round(rawInterest*100) / 100
			}
			revMinStmt = math.Round(((revolvingBalStmt*(rate/100.0))+cycleInterest)*100) / 100
			if floor > 0 && revMinStmt < floor && revolvingBalStmt > 0 {
				revMinStmt = floor
			}
			minPayForCycle = math.Round((monthlyInstDue+revMinStmt)*100) / 100

		case models.CalcModeInterestOnly:
			// Interest only 100% (e.g. Finnix)
			if acc.InterestRate > 0 {
				rawInterest := revolvingBalStmt * (acc.InterestRate / 100.0) * (30.0 / 365.0)
				revMinStmt = math.Round(rawInterest*100) / 100
			}
			if floor > 0 && revMinStmt < floor && revolvingBalStmt > 0 {
				revMinStmt = floor
			}
			minPayForCycle = math.Ceil(monthlyInstDue + revMinStmt)

		default:
			// Balance % (Standard credit cards, UOB Cash Plus, First Choice)
			if rate > 0 {
				revMinStmt = revolvingBalStmt * (rate / 100.0)
			} else if acc.InterestRate > 0 {
				rawInterest := revolvingBalStmt * (acc.InterestRate / 100.0) * (30.0 / 365.0)
				revMinStmt = math.Round(rawInterest*100) / 100
			}
			if floor > 0 && revMinStmt < floor && revolvingBalStmt > 0 {
				revMinStmt = floor
			}
			minPayForCycle = math.Ceil(monthlyInstDue + revMinStmt)
		}

		statementTotal := statementBal
		if calcMode == models.CalcModePrincipalPlusInterest && interestChargesThisCycle > 0 {
			statementTotal += interestChargesThisCycle
		}
		if minPayForCycle > statementTotal {
			minPayForCycle = statementTotal
		}
	}
	if usesStatementMinimum {
		minPayForCycle = *acc.StatementMinimumDue
	}

	// 3. Remaining due for the current cycle
	hasMinOrFullPayment := false
	for _, tx := range txs {
		if !usesStatementMinimum && tx.Date >= cycleStart && tx.Type == models.CreditTxPayment {
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
	if usesStatementMinimum {
		// An official minimum is settled by money, not payment labels or the
		// legacy estimated-payment tolerance. Keep the bank's original cents.
		remainingDue = math.Max(0, math.Round((minPayForCycle-paidThisCycle)*100)/100)
		isPaidThisMonth = remainingDue == 0
	}

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
				nextCycleInterest = math.Round(rawInterest*1.07*100) / 100
			} else {
				// Personal loans / nano finance: interest is exempt from VAT
				nextCycleInterest = math.Round(rawInterest*100) / 100
			}
		}

		nextCycleEstimatedStmt = math.Round((acc.CurrentBalance+nextCycleInterest)*100) / 100

		// Next cycle minimum calculation includes revolving balance plus accrued interest
		revolvingBaseNext := revolvingBalCurr + nextCycleInterest
		var revMinCurr float64
		switch calcMode {
		case models.CalcModePrincipalPlusInterest:
			// Principal % + 100% of next cycle interest
			revMinCurr = (revolvingBalCurr * (rate / 100.0)) + nextCycleInterest
		case models.CalcModeInterestOnly:
			// Interest only (Finnix)
			revMinCurr = nextCycleInterest
		default:
			// Balance % (Standard credit card, UOB Cash Plus, First Choice)
			if rate > 0 {
				revMinCurr = revolvingBaseNext * (rate / 100.0)
			} else {
				revMinCurr = nextCycleInterest
			}
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
		CreditAccount:               acc,
		AvailableCredit:             availableCredit,
		CreditUtilization:           utilization,
		UnbilledInstallments:        unbilledInstallments,
		MonthlyInstallmentDue:       monthlyInstDue,
		TotalDueThisMonth:           totalDue,
		EstimatedMinPayment:         minPayForCycle,
		UsesStatementMinimum:        usesStatementMinimum,
		PaidThisMonth:               paidThisCycle,
		IsPaidThisMonth:             isPaidThisMonth,
		NextCycleEstimatedMin:       nextCycleMin,
		NextCycleEstimatedInterest:  nextCycleInterest,
		NextCycleEstimatedStatement: nextCycleEstimatedStmt,
		ActiveInstallmentCount:      activeInstCount,
		Installments:                insts,
		Transactions:                txs,
	}
}

// ComputeAccountDetailPublic exposes computeAccountDetail for external verification.
func ComputeAccountDetailPublic(acc models.CreditAccount, insts []models.CreditInstallment, txs []models.CreditTransaction, refDate ...time.Time) models.CreditAccountDetail {
	return computeAccountDetail(acc, insts, txs, refDate...)
}
