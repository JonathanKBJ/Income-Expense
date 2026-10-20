package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"expense-tracker/internal/config"
	"expense-tracker/internal/repository"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func TestCreditHandler_CronPostInstallments_DryRun(t *testing.T) {
	_, _ = config.Load() // in case env already set
	// try loading from backend/.env if running from package dir
	data, _ := os.ReadFile("../../.env")
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) == 2 && !strings.HasPrefix(parts[0], "#") {
			os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
	cfg, err := config.Load()
	if err != nil || cfg.TursoDatabaseURL == "" {
		t.Skip("Skipping test: Turso database not configured")
	}

	db, err := sql.Open("libsql", cfg.DSN())
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	creditRepo := repository.NewCreditRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	telegramRepo := repository.NewTelegramRepository(db)

	handler := NewCreditHandler(creditRepo, groupRepo).WithTelegram(telegramRepo, cfg.TelegramBotToken, cfg.TelegramBotSecret)

	// Test DryRun with targetDay = 17 (or 18)
	req := httptest.NewRequest(http.MethodGet, "/api/telegram/cron/post-installments?day=17&dry_run=true", nil)
	req.Header.Set("X-Telegram-Bot-Secret", cfg.TelegramBotSecret)
	w := httptest.NewRecorder()

	handler.CronPostInstallments(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("CronPostInstallments failed with status %d: %s", w.Code, w.Body.String())
	}

	var resp CronPostInstallmentsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	t.Logf("Dry run result: Checked %d accounts, found %d installment items", resp.Accounts, len(resp.Items))
	for _, item := range resp.Items {
		t.Logf(" - Account: %s (%s), Item: %s, Term: %d/%d, Amount: %.2f, Status: %s",
			item.AccountName, item.AccountID, item.ItemName, item.Term, item.TotalTerms, item.Amount, item.Status)
	}
}
