package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required for this operation")
		return
	}

	result, err := h.repo.GetAnnualSummary(r.Context(), year, groupID)
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

	// Determine wallet owner: default to current user, allow manual override
	// For batch, use the first request's userId or default to current user
	ownerUserID := createdByID
	if len(reqs) > 0 && reqs[0].UserID != nil && *reqs[0].UserID != "" && *reqs[0].UserID != createdByID {
		if _, err := h.groupRepo.GetMemberRole(r.Context(), groupID, *reqs[0].UserID); err != nil {
			writeError(w, http.StatusBadRequest, "owner is not a member of this group")
			return
		}
		ownerUserID = *reqs[0].UserID
	}

	err := h.repo.CreateBatch(r.Context(), reqs, ownerUserID, createdByID, groupID)
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
