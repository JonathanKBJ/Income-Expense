package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// TransactionHandler handles HTTP requests for transaction operations.
type TransactionHandler struct {
	repo         *repository.TransactionRepository
	groupRepo    *repository.GroupRepository
	activityRepo *repository.ActivityLogRepository
	telegramRepo *repository.TelegramRepository
	creditRepo   *repository.CreditRepository
	loanRepo     *repository.LoanRepository
	botToken     string
}

// NewTransactionHandler creates a new handler with the given repositories.
func NewTransactionHandler(
	repo *repository.TransactionRepository,
	groupRepo *repository.GroupRepository,
	activityRepo *repository.ActivityLogRepository,
) *TransactionHandler {
	return &TransactionHandler{
		repo:         repo,
		groupRepo:    groupRepo,
		activityRepo: activityRepo,
	}
}

// WithTelegram attaches telegram repository and bot token for group notifications.
func (h *TransactionHandler) WithTelegram(telegramRepo *repository.TelegramRepository, botToken string) *TransactionHandler {
	h.telegramRepo = telegramRepo
	h.botToken = botToken
	return h
}

// WithCreditAndLoans attaches credit and loan repositories for two-way sync and payments.
func (h *TransactionHandler) WithCreditAndLoans(creditRepo *repository.CreditRepository, loanRepo *repository.LoanRepository) *TransactionHandler {
	h.creditRepo = creditRepo
	h.loanRepo = loanRepo
	return h
}

// GetTransactions handles GET /api/transactions?month={M}&year={Y}
// Returns filtered transactions and aggregated summary metrics.
func (h *TransactionHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters with defaults to current month/year
	now := time.Now()
	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	month := int(now.Month())
	year := now.Year()

	if monthStr != "" {
		m, err := strconv.Atoi(monthStr)
		if err != nil || m < 1 || m > 12 {
			writeError(w, http.StatusBadRequest, "month must be between 1 and 12")
			return
		}
		month = m
	}

	if yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err != nil || y < 2000 || y > 2100 {
			writeError(w, http.StatusBadRequest, "year must be between 2000 and 2100")
			return
		}
		year = y
	}

	// Injected by AuthMiddleware
	groupID := middleware.GetGroupID(r.Context())
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required for this operation")
		return
	}

	result, err := h.repo.GetByMonthYear(r.Context(), month, year, groupID)
	if err != nil {
		log.Printf("ERROR: GetTransactions: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch transactions")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetTransactionsCompact handles GET /api/transactions/compact?month={M}&year={Y}
// Returns a lightweight response (no receipt images, no JOINs) for MoM comparison.
func (h *TransactionHandler) GetTransactionsCompact(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	if monthStr != "" {
		m, err := strconv.Atoi(monthStr)
		if err != nil || m < 1 || m > 12 {
			writeError(w, http.StatusBadRequest, "month must be between 1 and 12")
			return
		}
		month = m
	}

	if yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err != nil || y < 2000 || y > 2100 {
			writeError(w, http.StatusBadRequest, "year must be between 2000 and 2100")
			return
		}
		year = y
	}

	groupID := middleware.GetGroupID(r.Context())
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required for this operation")
		return
	}

	result, err := h.repo.GetByMonthYearCompact(r.Context(), month, year, groupID)
	if err != nil {
		log.Printf("ERROR: GetTransactionsCompact: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch transactions")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetAnnualSummary handles GET /api/transactions/annual?year={Y}
func (h *TransactionHandler) GetAnnualSummary(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	yearStr := r.URL.Query().Get("year")
	year := now.Year()

	if yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err != nil || y < 2000 || y > 2100 {
			writeError(w, http.StatusBadRequest, "year must be between 2000 and 2100")
			return
		}
		year = y
	}

	// Injected by AuthMiddleware
	groupID := middleware.GetGroupID(r.Context())
	userID := middleware.GetUserID(r.Context())
	if groupID == "" || userID == "" {
		writeError(w, http.StatusForbidden, "group identification required for this operation")
		return
	}

	var filterUserID string
	settings, sErr := h.groupRepo.GetGroupSettings(r.Context(), groupID)
	if sErr == nil && !settings.ShareAnnual {
		filterUserID = userID
	}

	result, err := h.repo.GetAnnualSummary(r.Context(), year, groupID, filterUserID)
	if err != nil {
		log.Printf("ERROR: GetAnnualSummary: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch annual summary")
		return
	}

	// Return empty slices for frontend map safety if no data
	if result.MonthlyData == nil {
		result.MonthlyData = []models.MonthlySummary{}
	}
	if result.CategoryData == nil {
		result.CategoryData = []models.CategorySummary{}
	}

	writeJSON(w, http.StatusOK, result)
}

// CreateTransaction handles POST /api/transactions
// Creates a new income or expense transaction.
func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	// Validate the request (enforces Income vs Expense rules)
	if err := middleware.ValidateCreateRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Injected by AuthMiddleware
	createdByID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())

	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required for this operation")
		return
	}

	// Determine wallet owner: default to current user, allow manual override
	ownerUserID := createdByID
	if req.UserID != nil && *req.UserID != "" && *req.UserID != createdByID {
		// Validate that the requested owner is a member of this group
		if _, err := h.groupRepo.GetMemberRole(r.Context(), groupID, *req.UserID); err != nil {
			writeError(w, http.StatusBadRequest, "owner is not a member of this group")
			return
		}
		ownerUserID = *req.UserID
	}

	transaction, err := h.repo.Create(r.Context(), req, ownerUserID, createdByID, groupID)
	if err != nil {
		log.Printf("ERROR: CreateTransaction: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create transaction")
		return
	}

	// Log activity for multi-member groups
	h.logActivity(r.Context(), groupID, createdByID, "CREATE_TRANSACTION", "transaction", transaction.ID,
		fmt.Sprintf(`{"amount":%.2f,"type":"%s","category":"%s"}`, transaction.Amount, transaction.Type, transaction.Category))

	writeJSON(w, http.StatusCreated, transaction)
	// Notify linked Telegram group in the background
	h.notifyTelegramGroup(groupID, transaction)

	// Two-way payment reflection to credit cards / loans if created directly as PAID
	if transaction.Status != nil && *transaction.Status == models.StatusPaid {
		if h.creditRepo != nil && transaction.CreditAccountID != nil && *transaction.CreditAccountID != "" {
			go h.recordCreditCardPaymentFromDashboard(context.Background(), *transaction.CreditAccountID, transaction)
		}
		if h.loanRepo != nil && transaction.LoanID != nil && *transaction.LoanID != "" {
			go h.recordLoanPaymentFromDashboard(context.Background(), *transaction.LoanID, transaction)
		}
	}

}

// CreateTransactionsBatch handles POST /api/transactions/batch
func (h *TransactionHandler) CreateTransactionsBatch(w http.ResponseWriter, r *http.Request) {
	var reqs []models.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	// Validate each request
	for _, req := range reqs {
		if err := middleware.ValidateCreateRequest(&req); err != nil {
			writeError(w, http.StatusBadRequest, "validation failed: "+err.Error())
			return
		}
	}

	// Injected by AuthMiddleware
	createdByID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())

	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required for this operation")
		return
	}

	// Validate any explicit wallet owner per request. CreateBatch will preserve
	// each request's userId, falling back to the recorder when it is omitted.
	validatedOwners := map[string]bool{createdByID: true}
	for _, req := range reqs {
		if req.UserID == nil || *req.UserID == "" || validatedOwners[*req.UserID] {
			continue
		}
		if _, err := h.groupRepo.GetMemberRole(r.Context(), groupID, *req.UserID); err != nil {
			writeError(w, http.StatusBadRequest, "owner is not a member of this group")
			return
		}
		validatedOwners[*req.UserID] = true
	}

	err := h.repo.CreateBatch(r.Context(), reqs, createdByID, createdByID, groupID)
	if err != nil {
		log.Printf("ERROR: CreateTransactionsBatch: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create transactions in batch")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "batch created successfully"})
}

// CreateTransactionsBatchToGroup handles POST /api/transactions/batch-to-group
// Allows copying transactions to a different group (member must belong to both groups).
func (h *TransactionHandler) CreateTransactionsBatchToGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Transactions  []models.CreateTransactionRequest `json:"transactions"`
		TargetGroupID string                            `json:"targetGroupId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if req.TargetGroupID == "" {
		writeError(w, http.StatusBadRequest, "targetGroupId is required")
		return
	}

	if len(req.Transactions) == 0 {
		writeError(w, http.StatusBadRequest, "no transactions provided")
		return
	}

	userID := middleware.GetUserID(r.Context())

	// Verify user is member of target group
	if _, err := h.groupRepo.GetMemberRole(r.Context(), req.TargetGroupID, userID); err != nil {
		writeError(w, http.StatusForbidden, "you are not a member of the target group")
		return
	}

	// Validate each request
	for _, t := range req.Transactions {
		if err := middleware.ValidateCreateRequest(&t); err != nil {
			writeError(w, http.StatusBadRequest, "validation failed: "+err.Error())
			return
		}
	}

	validatedOwners := map[string]bool{userID: true}
	for _, t := range req.Transactions {
		if t.UserID == nil || *t.UserID == "" || validatedOwners[*t.UserID] {
			continue
		}
		if _, err := h.groupRepo.GetMemberRole(r.Context(), req.TargetGroupID, *t.UserID); err != nil {
			writeError(w, http.StatusBadRequest, "owner is not a member of the target group")
			return
		}
		validatedOwners[*t.UserID] = true
	}

	err := h.repo.CreateBatch(r.Context(), req.Transactions, userID, userID, req.TargetGroupID)
	if err != nil {
		log.Printf("ERROR: CreateTransactionsBatchToGroup: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create transactions in target group")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "batch copied to target group successfully"})
}

// UpdateTransaction handles PATCH /api/transactions/{id}
// Updates status and/or paidAmount for expense transactions only.
func (h *TransactionHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "transaction id is required")
		return
	}

	// Injected by AuthMiddleware
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required for this operation")
		return
	}

	// Fetch existing transaction
	existing, err := h.repo.GetByID(r.Context(), id, groupID)
	if err != nil {
		log.Printf("ERROR: UpdateTransaction (fetch): %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch transaction")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}

	var req models.UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	// Validate the update against the existing transaction
	if err := middleware.ValidateUpdateRequest(&req, existing); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// If changing wallet owner, validate new owner is a group member
	if req.UserID != nil && *req.UserID != "" {
		if _, err := h.groupRepo.GetMemberRole(r.Context(), groupID, *req.UserID); err != nil {
			writeError(w, http.StatusBadRequest, "new owner is not a member of this group")
			return
		}
	}

	updated, err := h.repo.Update(r.Context(), id, groupID, req)
	if err != nil {
		log.Printf("ERROR: UpdateTransaction (update): %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update transaction")
		return
	}
	if updated == nil {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}

	// Two-way payment reflection to credit cards / loans when marked as PAID
	if (existing.Status == nil || *existing.Status != models.StatusPaid) &&
		(updated.Status != nil && *updated.Status == models.StatusPaid) {
		if h.creditRepo != nil && updated.CreditAccountID != nil && *updated.CreditAccountID != "" {
			go h.recordCreditCardPaymentFromDashboard(context.Background(), *updated.CreditAccountID, updated)
		}
		if h.loanRepo != nil && updated.LoanID != nil && *updated.LoanID != "" {
			go h.recordLoanPaymentFromDashboard(context.Background(), *updated.LoanID, updated)
		}
	}

	// Log activity for multi-member groups
	h.logActivity(r.Context(), groupID, userID, "UPDATE_TRANSACTION", "transaction", id,
		fmt.Sprintf(`{"amount":%.2f,"type":"%s","category":"%s"}`, updated.Amount, updated.Type, updated.Category))

	writeJSON(w, http.StatusOK, updated)
}

// DeleteTransaction handles DELETE /api/transactions/{id}
func (h *TransactionHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "transaction id is required")
		return
	}

	// Injected by AuthMiddleware
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required for this operation")
		return
	}

	if err := h.repo.Delete(r.Context(), id, groupID); err != nil {
		log.Printf("ERROR: DeleteTransaction: %v", err)
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}

	// Log activity for multi-member groups
	h.logActivity(r.Context(), groupID, userID, "DELETE_TRANSACTION", "transaction", id, "")

	writeJSON(w, http.StatusOK, map[string]string{"message": "transaction deleted"})
}

// DeleteTransactionsBatch handles DELETE /api/transactions/batch
func (h *TransactionHandler) DeleteTransactionsBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "no transaction ids provided")
		return
	}

	// Injected by AuthMiddleware
	groupID := middleware.GetGroupID(r.Context())
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required for this operation")
		return
	}

	if err := h.repo.DeleteBatch(r.Context(), req.IDs, groupID); err != nil {
		log.Printf("ERROR: DeleteTransactionsBatch: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete transactions in batch")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "transactions deleted successfully"})
}

// logActivity creates an activity log entry only for multi-member groups.
func (h *TransactionHandler) logActivity(ctx context.Context, groupID, userID, action, entityType, entityID, details string) {
	isMulti, err := h.groupRepo.GroupHasMultipleMembers(ctx, groupID)
	if err != nil || !isMulti {
		return
	}

	entry := &models.ActivityLogEntry{
		ID:         uuid.New().String(),
		GroupID:    groupID,
		UserID:     userID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    details,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := h.activityRepo.CreateEntry(ctx, entry); err != nil {
		log.Printf("WARN: failed to log activity: %v", err)
	}
}

// notifyTelegramGroup broadcasts a transaction notification to the group's linked Telegram chat.
func (h *TransactionHandler) notifyTelegramGroup(groupID string, tx *models.Transaction) {
	if h.telegramRepo == nil || h.botToken == "" || tx == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		chatID, err := h.telegramRepo.GetGroupTelegramChatID(ctx, groupID)
		if err != nil || chatID == "" {
			return
		}

		typeEmoji := "💸"
		typeName := "รายจ่าย"
		if tx.Type == models.TypeIncome {
			typeEmoji = "💰"
			typeName = "รายรับ"
		}

		userDisplay := "สมาชิกในกลุ่ม"
		if tx.CreatedByUsername != nil && *tx.CreatedByUsername != "" {
			userDisplay = "@" + *tx.CreatedByUsername
		} else if tx.OwnerUsername != nil && *tx.OwnerUsername != "" {
			userDisplay = "@" + *tx.OwnerUsername
		}

		text := fmt.Sprintf(
			"📢 *บันทึก%sใหม่ผ่านเว็บ*\n%s *%s*: ฿%.2f\n🏷️ หมวดหมู่: %s\n📅 วันที่: %s\n👤 โดย: %s",
			typeName, typeEmoji, tx.Description, tx.Amount, tx.Category, tx.Date, userDisplay,
		)

		payload := map[string]string{
			"chat_id":    chatID,
			"text":       text,
			"parse_mode": "Markdown",
		}
		jsonBody, err := json.Marshal(payload)
		if err != nil {
			return
		}

		url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", h.botToken)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("WARN: failed to send Telegram notification: %v", err)
			return
		}
		defer resp.Body.Close()
	}()
}

// GetWalletSummary handles GET /api/transactions/wallet-summary?month=M&year=Y
// Returns per-member wallet balances grouped by user_id (wallet owner).
func (h *TransactionHandler) GetWalletSummary(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required for this operation")
		return
	}

	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")
	if monthStr == "" || yearStr == "" {
		writeError(w, http.StatusBadRequest, "month and year query parameters are required")
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		writeError(w, http.StatusBadRequest, "month must be between 1 and 12")
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 || year > 2100 {
		writeError(w, http.StatusBadRequest, "year must be between 2000 and 2100")
		return
	}

	result, err := h.repo.GetWalletSummary(r.Context(), month, year, groupID)
	if err != nil {
		log.Printf("ERROR: GetWalletSummary: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch wallet summary")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *TransactionHandler) recordCreditCardPaymentFromDashboard(ctx context.Context, accountID string, tx *models.Transaction) {
	paidAmount := tx.Amount
	if tx.PaidAmount != nil && *tx.PaidAmount > 0 {
		paidAmount = *tx.PaidAmount
	}

	creditTx := models.CreditTransaction{
		ID:           uuid.New().String(),
		AccountID:    accountID,
		Type:         models.CreditTxPayment,
		Amount:       paidAmount,
		Date:         tx.Date,
		Description:  fmt.Sprintf("ชำระผ่าน Dashboard: %s", tx.Description),
		ReceiptImage: tx.ReceiptImage,
	}

	// Link to active installment if exists
	insts, err := h.creditRepo.ListInstallments(ctx, accountID)
	if err == nil {
		for _, inst := range insts {
			if inst.Status == models.InstallmentStatusActive && inst.PaidTerms < inst.TotalTerms {
				creditTx.InstallmentID = &inst.ID
				break
			}
		}
	}

	payType := models.CreditPayCustom
	creditTx.PaymentType = &payType

	if err := h.creditRepo.AddTransaction(ctx, &creditTx, accountID); err != nil {
		log.Printf("WARN: recordCreditCardPaymentFromDashboard failed: %v", err)
	}
}

func (h *TransactionHandler) recordLoanPaymentFromDashboard(ctx context.Context, loanID string, tx *models.Transaction) {
	paidAmount := tx.Amount
	if tx.PaidAmount != nil && *tx.PaidAmount > 0 {
		paidAmount = *tx.PaidAmount
	}

	entry := models.LoanEntry{
		ID:           uuid.New().String(),
		LoanID:       loanID,
		EntryType:    models.EntryInstallment,
		Amount:       paidAmount,
		Date:         tx.Date,
		Description:  fmt.Sprintf("ชำระผ่าน Dashboard: %s", tx.Description),
		ReceiptImage: tx.ReceiptImage,
	}

	if err := h.loanRepo.AddEntry(ctx, &entry); err != nil {
		log.Printf("WARN: recordLoanPaymentFromDashboard failed: %v", err)
	}
}

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

func matchAccountForDescription(desc string, accounts []models.SyncDueAvailableAccount) *models.SyncDueAvailableAccount {
	normDesc := normalizeName(desc)
	if normDesc == "" {
		return nil
	}

	// 1. Direct exact or substring match
	for i := range accounts {
		normName := normalizeName(accounts[i].Name)
		if normName == "" {
			continue
		}
		if normDesc == normName || strings.Contains(normDesc, normName) || strings.Contains(normName, normDesc) {
			return &accounts[i]
		}
	}

	// 2. Specific alias patterns
	for i := range accounts {
		normName := normalizeName(accounts[i].Name)
		// UOB Cash Plus
		if (strings.Contains(normDesc, "cashplus") || strings.Contains(normDesc, "cashplusuob")) && strings.Contains(normName, "cashplus") {
			return &accounts[i]
		}
		// UOB TMRW / UOB Credit
		if (strings.Contains(normDesc, "uob") || strings.Contains(normDesc, "tmrw")) && (strings.Contains(normName, "tmrw") || strings.Contains(normName, "uob")) && !strings.Contains(normDesc, "cashplus") && !strings.Contains(normName, "cashplus") {
			return &accounts[i]
		}
		// SCB Up Cash
		if (strings.Contains(normDesc, "upcash") || strings.Contains(normDesc, "scbupcash")) && strings.Contains(normName, "upcash") {
			return &accounts[i]
		}
		// First choice
		if (strings.Contains(normDesc, "firstchoice") || strings.Contains(normDesc, "first")) && strings.Contains(normName, "firstchoice") {
			return &accounts[i]
		}
		// Money Thunder
		if (strings.Contains(normDesc, "moneythunder") || strings.Contains(normDesc, "thunder")) && strings.Contains(normName, "moneythunder") {
			return &accounts[i]
		}
		// Finnix
		if strings.Contains(normDesc, "finnix") && strings.Contains(normName, "finnix") {
			return &accounts[i]
		}
		// Car loan
		if (strings.Contains(normDesc, "ผ่อนรถ") || strings.Contains(normDesc, "cimb")) && (strings.Contains(normName, "ผ่อนรถ") || strings.Contains(normName, "cimb")) {
			return &accounts[i]
		}
		// Krungthai
		if (strings.Contains(normDesc, "krungthai") || strings.Contains(normDesc, "ktb")) && (strings.Contains(normName, "krungthai") || strings.Contains(normName, "ktb")) {
			return &accounts[i]
		}
	}

	return nil
}

// GetSyncDuesPreview handles GET /api/transactions/sync-credit-dues?month=M&year=Y
func (h *TransactionHandler) GetSyncDuesPreview(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	month := int(now.Month())
	year := now.Year()
	if mStr := r.URL.Query().Get("month"); mStr != "" {
		if m, err := strconv.Atoi(mStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}
	if yStr := r.URL.Query().Get("year"); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil && y >= 2000 && y <= 2100 {
			year = y
		}
	}
	groupID := middleware.GetGroupID(r.Context())
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}

	// 1. Fetch transactions for month/year
	txRes, err := h.repo.GetByMonthYear(r.Context(), month, year, groupID)
	if err != nil {
		log.Printf("ERROR: GetSyncDuesPreview (txs): %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch transactions")
		return
	}

	// 2. Fetch credit accounts
	availAccounts := make([]models.SyncDueAvailableAccount, 0)
	if h.creditRepo != nil {
		accs, err := h.creditRepo.ListAccountsByGroup(r.Context(), groupID)
		if err == nil {
			for _, acc := range accs {
				insts, _ := h.creditRepo.ListInstallments(r.Context(), acc.ID)
				txs, _ := h.creditRepo.ListTransactions(r.Context(), acc.ID)
				detail := computeAccountDetail(acc, insts, txs)

				due := detail.TotalDueThisMonth
				if due <= 0 {
					due = detail.EstimatedMinPayment
				}
				if due <= 0 && detail.CurrentBalance > 0 {
					due = detail.CurrentBalance
				}

				availAccounts = append(availAccounts, models.SyncDueAvailableAccount{
					Type:       "CREDIT_ACCOUNT",
					ID:         acc.ID,
					Name:       acc.Name,
					DueAmount:  due,
					FullAmount: detail.CurrentBalance,
				})
			}
		}
	}

	// 3. Fetch active borrow loans
	if h.loanRepo != nil {
		loans, err := h.loanRepo.ListByGroup(r.Context(), groupID)
		if err == nil {
			for _, l := range loans {
				if l.Type == models.LoanBorrow && l.Status == models.LoanActive {
					instAmt := 0.0
					if l.InstallmentAmount != nil {
						instAmt = *l.InstallmentAmount
					}
					availAccounts = append(availAccounts, models.SyncDueAvailableAccount{
						Type:       "LOAN",
						ID:         l.ID,
						Name:       l.Name,
						DueAmount:  instAmt,
						FullAmount: l.Principal,
					})
				}
			}
		}
	}

	// 4. Match transactions
	matches := make([]models.SyncDueMatchItem, 0)

	for _, tx := range txRes.Transactions {
		if tx.Type != models.TypeExpense {
			continue
		}

		var matchedAcc *models.SyncDueAvailableAccount

		// Priority 1: Already mapped by ID
		if tx.CreditAccountID != nil && *tx.CreditAccountID != "" {
			for i := range availAccounts {
				if availAccounts[i].Type == "CREDIT_ACCOUNT" && availAccounts[i].ID == *tx.CreditAccountID {
					matchedAcc = &availAccounts[i]
					break
				}
			}
		} else if tx.LoanID != nil && *tx.LoanID != "" {
			for i := range availAccounts {
				if availAccounts[i].Type == "LOAN" && availAccounts[i].ID == *tx.LoanID {
					matchedAcc = &availAccounts[i]
					break
				}
			}
		}

		// Priority 2: Fuzzy / keyword match by description
		if matchedAcc == nil {
			matchedAcc = matchAccountForDescription(tx.Description, availAccounts)
		}

		// Only include if matched, or if category suggests credit card / loan
		isCreditCategory := strings.Contains(strings.ToLower(tx.Category), "บัตร") ||
			strings.Contains(strings.ToLower(tx.Category), "credit") ||
			strings.Contains(strings.ToLower(tx.Category), "หนี้") ||
			strings.Contains(strings.ToLower(tx.Category), "เงินกู้") ||
			strings.Contains(strings.ToLower(tx.Category), "loan")

		if matchedAcc != nil || isCreditCategory {
			statusStr := "PENDING"
			if tx.Status != nil {
				statusStr = string(*tx.Status)
			}

			item := models.SyncDueMatchItem{
				TransactionID:          tx.ID,
				TransactionDescription: tx.Description,
				Category:               tx.Category,
				CurrentAmount:          tx.Amount,
				Date:                   tx.Date,
				Status:                 statusStr,
			}

			if matchedAcc != nil {
				item.MatchedType = matchedAcc.Type
				item.MatchedID = matchedAcc.ID
				item.MatchedName = matchedAcc.Name
				item.DueAmount = matchedAcc.DueAmount
				item.FullAmount = matchedAcc.FullAmount
				item.SuggestedAmount = matchedAcc.DueAmount
				item.SelectedAmountType = "DUE"
				item.Diff = math.Round((item.SuggestedAmount-item.CurrentAmount)*100) / 100
			}

			matches = append(matches, item)
		}
	}

	writeJSON(w, http.StatusOK, models.SyncDuesPreviewResponse{
		Month:             month,
		Year:              year,
		Matches:           matches,
		AvailableAccounts: availAccounts,
	})
}

// ApplySyncDues handles POST /api/transactions/apply-sync-dues
func (h *TransactionHandler) ApplySyncDues(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}

	var req models.ApplySyncDuesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.repo.BatchUpdateCreditLoanMapping(r.Context(), groupID, req.Updates); err != nil {
		log.Printf("ERROR: ApplySyncDues: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to apply sync updates: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.ApplySyncDuesResponse{
		UpdatedCount: len(req.Updates),
	})
}

// --- Response helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("ERROR: failed to encode JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{Error: message})
}
