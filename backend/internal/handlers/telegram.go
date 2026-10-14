package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
)

// TelegramHandler handles Telegram bot and pairing HTTP requests.
type TelegramHandler struct {
	telegramRepo *repository.TelegramRepository
	botSecret    string
}

func NewTelegramHandler(telegramRepo *repository.TelegramRepository, botSecret string) *TelegramHandler {
	return &TelegramHandler{
		telegramRepo: telegramRepo,
		botSecret:    botSecret,
	}
}

// validateBotSecret checks if the incoming request has the valid bot secret header.
func (h *TelegramHandler) validateBotSecret(r *http.Request) bool {
	if h.botSecret == "" {
		// If secret is not configured, permit in development/test
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

// GetStatus handles GET /api/me/telegram/status (JWT protected).
func (h *TelegramHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())
	if userID == "" || groupID == "" {
		writeError(w, http.StatusBadRequest, "user or group context missing")
		return
	}

	status, err := h.telegramRepo.GetStatus(r.Context(), userID, groupID)
	if err != nil {
		log.Printf("ERROR: GetTelegramStatus: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get telegram status")
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// GenerateLinkCode handles POST /api/me/telegram/link-code (JWT protected).
func (h *TelegramHandler) GenerateLinkCode(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())
	if userID == "" || groupID == "" {
		writeError(w, http.StatusBadRequest, "user or group context missing")
		return
	}

	res, err := h.telegramRepo.GenerateLinkCode(r.Context(), userID, groupID)
	if err != nil {
		log.Printf("ERROR: GenerateTelegramLinkCode: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to generate link code")
		return
	}

	writeJSON(w, http.StatusCreated, res)
}

// VerifyLink handles POST /api/telegram/verify-link (Called by Bot/Worker).
func (h *TelegramHandler) VerifyLink(w http.ResponseWriter, r *http.Request) {
	if !h.validateBotSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid or missing bot secret")
		return
	}

	var req models.VerifyTelegramLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if req.Code == "" || req.TelegramUserID == "" {
		writeError(w, http.StatusBadRequest, "code and telegramUserId are required")
		return
	}

	res, err := h.telegramRepo.VerifyAndLink(r.Context(), req)
	if err != nil {
		log.Printf("WARN: VerifyTelegramLink: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// GetPendingBills handles GET /api/telegram/pending-bills (Called by Bot/Worker).
func (h *TelegramHandler) GetPendingBills(w http.ResponseWriter, r *http.Request) {
	if !h.validateBotSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid or missing bot secret")
		return
	}

	telegramUserID := r.URL.Query().Get("telegram_user_id")
	telegramChatID := r.URL.Query().Get("telegram_chat_id")

	if telegramUserID == "" && telegramChatID == "" {
		writeError(w, http.StatusBadRequest, "telegram_user_id or telegram_chat_id is required")
		return
	}

	res, err := h.telegramRepo.GetPendingBills(r.Context(), telegramUserID, telegramChatID)
	if err != nil {
		log.Printf("WARN: GetTelegramPendingBills: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// PayPendingBill handles POST /api/telegram/pay-bill (Called by Bot/Worker).
func (h *TelegramHandler) PayPendingBill(w http.ResponseWriter, r *http.Request) {
	if !h.validateBotSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid or missing bot secret")
		return
	}

	var req models.TelegramPayBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if req.TransactionID == "" {
		writeError(w, http.StatusBadRequest, "transactionId is required")
		return
	}

	tx, err := h.telegramRepo.PayPendingBill(r.Context(), req)
	if err != nil {
		log.Printf("ERROR: PayPendingBill: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to pay bill: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tx)
}

// CreateTransaction handles POST /api/telegram/transactions (Called by Bot/Worker).
func (h *TelegramHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	if !h.validateBotSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid or missing bot secret")
		return
	}

	var req models.TelegramCreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if req.TelegramUserID == "" {
		writeError(w, http.StatusBadRequest, "telegramUserId is required")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}

	tx, err := h.telegramRepo.CreateTransactionFromTelegram(r.Context(), req)
	if err != nil {
		log.Printf("ERROR: CreateTransactionFromTelegram: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create transaction: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, tx)
}

// GetUsage handles GET /api/telegram/usage and GET /api/me/telegram/usage.
func (h *TelegramHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Telegram-Bot-Secret") != "" || h.botSecret != "" {
		if !h.validateBotSecret(r) && middleware.GetUserID(r.Context()) == "" {
			writeError(w, http.StatusUnauthorized, "invalid or missing bot secret")
			return
		}
	}

	stats, err := h.telegramRepo.GetUsageStats(r.Context())
	if err != nil {
		log.Printf("ERROR: GetTelegramUsage: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get usage stats: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// IncrementUsage handles POST /api/telegram/usage/increment (Called by Worker).
func (h *TelegramHandler) IncrementUsage(w http.ResponseWriter, r *http.Request) {
	if !h.validateBotSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid or missing bot secret")
		return
	}

	var req models.IncrementUsageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if err := h.telegramRepo.IncrementUsage(r.Context(), req); err != nil {
		log.Printf("ERROR: IncrementUsage: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to increment usage: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "usage incremented"})
}
