package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expense-tracker/internal/database"
	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"
)

func TestUpdateTransaction_UnchangedDeletedCreditAccountDoesNotBlockCashEdits(t *testing.T) {
	db, err := database.New("file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}

	groupID := "test-group"
	userID := "test-user"
	now := "2026-11-06T00:00:00Z"
	_, err = db.Exec(`INSERT INTO groups (id, name, created_at) VALUES (?, 'Test Group', ?)`, groupID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users (id, username, password_hash, role, status, created_at, updated_at) VALUES (?, 'user1', 'hash', 'USER', 'ACTIVE', ?, ?)`, userID, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO group_members (group_id, user_id) VALUES (?, ?)`, groupID, userID)
	if err != nil {
		t.Fatal(err)
	}

	creditRepo := repository.NewCreditRepository(db.DB)
	groupRepo := repository.NewGroupRepository(db.DB)
	txRepo := repository.NewTransactionRepository(db)
	handler := NewTransactionHandler(txRepo, groupRepo, nil).WithCreditAndLoans(creditRepo, nil)

	// Create legacy credit account
	acc := models.CreditAccount{
		Name:        "Old Card",
		Type:        models.CreditAccountCreditCard,
		CreditLimit: 5000,
	}
	if err := creditRepo.CreateAccount(context.Background(), &acc, userID, groupID); err != nil {
		t.Fatal(err)
	}

	// Create cash expense linked to it
	status := models.StatusPending
	tx, err := txRepo.Create(context.Background(), models.CreateTransactionRequest{
		Type:            models.TypeExpense,
		Category:        "Dining",
		Description:     "Dinner",
		Amount:          200,
		Date:            "2026-11-06",
		Status:          &status,
		CreditAccountID: &acc.ID,
	}, userID, userID, groupID)
	if err != nil {
		t.Fatal(err)
	}

	// Delete the credit account
	if err := creditRepo.DeleteAccount(context.Background(), acc.ID, groupID); err != nil {
		t.Fatal(err)
	}

	// Update cash expense description without changing creditAccountId
	newDesc := "Dinner with friends"
	patchBody := `{"description": "Dinner with friends"}`
	req := httptest.NewRequest("PATCH", "/api/transactions/"+tx.ID, strings.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.GroupIDKey, groupID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", tx.ID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.UpdateTransaction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for cash edit with unchanged deleted credit mapping, got %d: %s", w.Code, w.Body.String())
	}

	updated, err := txRepo.GetByID(context.Background(), tx.ID, groupID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Description != newDesc {
		t.Fatalf("expected description %q, got %q", newDesc, updated.Description)
	}
}
