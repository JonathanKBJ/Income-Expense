package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/internal/models"
)

func TestTelegramHandler_BotSecretValidation(t *testing.T) {
	h := NewTelegramHandler(nil, "my-secret-token")

	// Missing header
	req := httptest.NewRequest(http.MethodGet, "/api/telegram/pending-bills", nil)
	if h.validateBotSecret(req) {
		t.Errorf("expected false for missing secret, got true")
	}

	// Correct X-Telegram-Bot-Secret header
	req.Header.Set("X-Telegram-Bot-Secret", "my-secret-token")
	if !h.validateBotSecret(req) {
		t.Errorf("expected true for valid X-Telegram-Bot-Secret, got false")
	}

	// Correct Bearer token
	req = httptest.NewRequest(http.MethodGet, "/api/telegram/pending-bills", nil)
	req.Header.Set("Authorization", "Bearer my-secret-token")
	if !h.validateBotSecret(req) {
		t.Errorf("expected true for valid Bearer token, got false")
	}

	// Empty secret configured -> permits in dev
	hNoSecret := NewTelegramHandler(nil, "")
	reqNoAuth := httptest.NewRequest(http.MethodGet, "/api/telegram/pending-bills", nil)
	if !hNoSecret.validateBotSecret(reqNoAuth) {
		t.Errorf("expected true when no secret configured, got false")
	}
}

func TestTelegramHandler_VerifyLink_Validation(t *testing.T) {
	h := NewTelegramHandler(nil, "secret123")

	// Unauthorized request
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/verify-link", nil)
	w := httptest.NewRecorder()
	h.VerifyLink(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}

	// Invalid payload (missing required fields)
	invalidPayload := models.VerifyTelegramLinkRequest{Code: ""}
	body, _ := json.Marshal(invalidPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/telegram/verify-link", bytes.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Secret", "secret123")
	w = httptest.NewRecorder()
	h.VerifyLink(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for empty code, got %d", w.Code)
	}
}

func TestTelegramHandler_GetPendingBills_Validation(t *testing.T) {
	h := NewTelegramHandler(nil, "secret123")

	// Missing query params
	req := httptest.NewRequest(http.MethodGet, "/api/telegram/pending-bills", nil)
	req.Header.Set("X-Telegram-Bot-Secret", "secret123")
	w := httptest.NewRecorder()
	h.GetPendingBills(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for missing params, got %d", w.Code)
	}
}

func TestTelegramHandler_PayPendingBill_Validation(t *testing.T) {
	h := NewTelegramHandler(nil, "secret123")

	// Missing transactionId
	emptyReq := models.TelegramPayBillRequest{TransactionID: ""}
	body, _ := json.Marshal(emptyReq)
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/pay-bill", bytes.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Secret", "secret123")
	w := httptest.NewRecorder()
	h.PayPendingBill(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for missing transactionId, got %d", w.Code)
	}
}

func TestTelegramHandler_CreateTransaction_Validation(t *testing.T) {
	h := NewTelegramHandler(nil, "secret123")

	// Missing telegramUserId
	invalidTx := models.TelegramCreateTransactionRequest{
		TelegramUserID: "",
		Amount:         100,
	}
	body, _ := json.Marshal(invalidTx)
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/transactions", bytes.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Secret", "secret123")
	w := httptest.NewRecorder()
	h.CreateTransaction(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for missing telegramUserId, got %d", w.Code)
	}

	// Invalid amount <= 0
	invalidTx = models.TelegramCreateTransactionRequest{
		TelegramUserID: "tg-1",
		Amount:         -50,
	}
	body, _ = json.Marshal(invalidTx)
	req = httptest.NewRequest(http.MethodPost, "/api/telegram/transactions", bytes.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Secret", "secret123")
	w = httptest.NewRecorder()
	h.CreateTransaction(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for non-positive amount, got %d", w.Code)
	}
}
