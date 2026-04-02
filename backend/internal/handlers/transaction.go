package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"

	"github.com/go-chi/chi/v5"
)

// TransactionHandler handles HTTP requests for transaction operations.
type TransactionHandler struct {
	repo *repository.TransactionRepository
}

// NewTransactionHandler creates a new handler with the given repository.
func NewTransactionHandler(repo *repository.TransactionRepository) *TransactionHandler {
	return &TransactionHandler{repo: repo}
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

	result, err := h.repo.GetByMonthYear(r.Context(), month, year, groupID)
	if err != nil {
		log.Printf("ERROR: GetTransactions: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch transactions")
		return
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
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())

	transaction, err := h.repo.Create(r.Context(), req, userID, groupID)
	if err != nil {
		log.Printf("ERROR: CreateTransaction: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create transaction")
		return
	}

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
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())

	err := h.repo.CreateBatch(r.Context(), reqs, userID, groupID)
	if err != nil {
		log.Printf("ERROR: CreateTransactionsBatch: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create transactions in batch")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "batch created successfully"})
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
	groupID := middleware.GetGroupID(r.Context())

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
	groupID := middleware.GetGroupID(r.Context())

	if err := h.repo.Delete(r.Context(), id, groupID); err != nil {
		log.Printf("ERROR: DeleteTransaction: %v", err)
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "transaction deleted"})
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
