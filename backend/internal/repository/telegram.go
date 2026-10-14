package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"expense-tracker/internal/models"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TelegramRepository manages Telegram account pairings and bot operations.
type TelegramRepository struct {
	db *sql.DB
}

func NewTelegramRepository(db *sql.DB) *TelegramRepository {
	return &TelegramRepository{db: db}
}

// GenerateLinkCode creates a temporary 6-digit link code (valid for 15 minutes).
func (r *TelegramRepository) GenerateLinkCode(ctx context.Context, userID, groupID string) (*models.GenerateTelegramLinkCodeResponse, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(15 * time.Minute)
	nowStr := now.Format(time.RFC3339)
	expiresAtStr := expiresAt.Format(time.RFC3339)

	// Clean up expired codes or existing codes for this user
	_, _ = r.db.ExecContext(ctx, `DELETE FROM telegram_link_codes WHERE user_id = ? OR expires_at < ?`, userID, nowStr)

	// Generate a 6-digit random number prefixed with TG-
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate random code: %w", err)
	}
	code := fmt.Sprintf("TG-%06d", n.Int64()+100000)

	query := `INSERT INTO telegram_link_codes (code, user_id, group_id, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, query, code, userID, groupID, expiresAtStr, nowStr); err != nil {
		return nil, fmt.Errorf("failed to store link code: %w", err)
	}

	return &models.GenerateTelegramLinkCodeResponse{
		Code:      code,
		ExpiresAt: expiresAtStr,
		ExpiresIn: 900, // 15 minutes in seconds
	}, nil
}

// VerifyAndLink verifies a link code sent from a Telegram chat and links user & group.
func (r *TelegramRepository) VerifyAndLink(ctx context.Context, req models.VerifyTelegramLinkRequest) (*models.VerifyTelegramLinkResponse, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	code := strings.ToUpper(strings.TrimSpace(req.Code))
	nowStr := time.Now().UTC().Format(time.RFC3339)

	var userID, groupID, expiresAtStr string
	err = tx.QueryRowContext(ctx, `SELECT user_id, group_id, expires_at FROM telegram_link_codes WHERE code = ?`, code).Scan(&userID, &groupID, &expiresAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid or expired link code")
		}
		return nil, fmt.Errorf("failed to query link code: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil || time.Now().UTC().After(expiresAt) {
		_, _ = tx.ExecContext(ctx, `DELETE FROM telegram_link_codes WHERE code = ?`, code)
		return nil, errors.New("link code has expired")
	}

	// 1. Link user with Telegram User ID & Username
	_, err = tx.ExecContext(ctx, `UPDATE users SET telegram_user_id = ?, telegram_username = ?, updated_at = ? WHERE id = ?`,
		req.TelegramUserID, req.TelegramUsername, nowStr, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to link user: %w", err)
	}

	// 2. Link group with Telegram Chat ID (if provided)
	if req.TelegramChatID != "" {
		_, err = tx.ExecContext(ctx, `UPDATE groups SET telegram_chat_id = ? WHERE id = ?`, req.TelegramChatID, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to link group chat: %w", err)
		}
	}

	// 3. Delete used link code
	_, _ = tx.ExecContext(ctx, `DELETE FROM telegram_link_codes WHERE code = ?`, code)

	// 4. Fetch user and group names for display
	var username, groupName string
	_ = tx.QueryRowContext(ctx, `SELECT username FROM users WHERE id = ?`, userID).Scan(&username)
	_ = tx.QueryRowContext(ctx, `SELECT name FROM groups WHERE id = ?`, groupID).Scan(&groupName)

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.VerifyTelegramLinkResponse{
		UserID:           userID,
		Username:         username,
		GroupID:          groupID,
		GroupName:        groupName,
		TelegramUserID:   req.TelegramUserID,
		TelegramUsername: req.TelegramUsername,
		TelegramChatID:   req.TelegramChatID,
	}, nil
}

// GetStatus checks linking status for a given web user and group.
func (r *TelegramRepository) GetStatus(ctx context.Context, userID, groupID string) (*models.TelegramStatusResponse, error) {
	var telegramUserID, telegramUsername, telegramChatID sql.NullString

	_ = r.db.QueryRowContext(ctx, `SELECT telegram_user_id, telegram_username FROM users WHERE id = ?`, userID).Scan(&telegramUserID, &telegramUsername)
	_ = r.db.QueryRowContext(ctx, `SELECT telegram_chat_id FROM groups WHERE id = ?`, groupID).Scan(&telegramChatID)

	res := &models.TelegramStatusResponse{
		IsUserLinked:  telegramUserID.Valid && telegramUserID.String != "",
		IsGroupLinked: telegramChatID.Valid && telegramChatID.String != "",
	}
	if telegramUsername.Valid && telegramUsername.String != "" {
		res.TelegramUsername = &telegramUsername.String
	}
	if telegramChatID.Valid && telegramChatID.String != "" {
		res.TelegramChatID = &telegramChatID.String
	}
	return res, nil
}

// GetPendingBills finds pending transactions in the group linked to telegramChatID or telegramUserID.
func (r *TelegramRepository) GetPendingBills(ctx context.Context, telegramUserID, telegramChatID string) (*models.TelegramPendingBillsResponse, error) {
	var groupID, groupName string
	var userID, username string

	// Lookup user
	err := r.db.QueryRowContext(ctx, `SELECT id, username FROM users WHERE telegram_user_id = ?`, telegramUserID).Scan(&userID, &username)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}

	// Lookup group by telegram_chat_id first
	if telegramChatID != "" {
		err = r.db.QueryRowContext(ctx, `SELECT id, name FROM groups WHERE telegram_chat_id = ?`, telegramChatID).Scan(&groupID, &groupName)
	}

	// If group not found by chat_id but user exists, find user's primary group
	if groupID == "" && userID != "" {
		_ = r.db.QueryRowContext(ctx, `
			SELECT g.id, g.name FROM groups g
			JOIN group_members gm ON g.id = gm.group_id
			WHERE gm.user_id = ? LIMIT 1
		`, userID).Scan(&groupID, &groupName)
	}

	if groupID == "" {
		return nil, errors.New("group not found for this telegram chat or user; please link account first")
	}

	// Query pending transactions in this group
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.type, t.category, t.description, t.amount, t.date, t.status, t.user_id, COALESCE(u.username, '')
		FROM transactions t
		LEFT JOIN users u ON t.user_id = u.id
		WHERE t.group_id = ? AND t.status = 'PENDING'
		ORDER BY t.date DESC
		LIMIT 20
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending bills: %w", err)
	}
	defer rows.Close()

	var pendingBills []models.TelegramPendingBillItem
	for rows.Next() {
		var item models.TelegramPendingBillItem
		var statusStr sql.NullString
		if err := rows.Scan(&item.ID, &item.Type, &item.Category, &item.Description, &item.Amount, &item.Date, &statusStr, &item.UserID, &item.Username); err != nil {
			return nil, fmt.Errorf("failed to scan pending bill: %w", err)
		}
		if statusStr.Valid {
			item.Status = statusStr.String
		}
		pendingBills = append(pendingBills, item)
	}

	if pendingBills == nil {
		pendingBills = []models.TelegramPendingBillItem{}
	}

	return &models.TelegramPendingBillsResponse{
		GroupID:      groupID,
		GroupName:    groupName,
		UserID:       userID,
		PendingBills: pendingBills,
	}, nil
}

// PayPendingBill marks a pending bill as PAID and attaches the receipt image.
func (r *TelegramRepository) PayPendingBill(ctx context.Context, req models.TelegramPayBillRequest) (*models.Transaction, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	var existingAmount float64
	var groupID, userID string
	err := r.db.QueryRowContext(ctx, `SELECT amount, group_id, user_id FROM transactions WHERE id = ?`, req.TransactionID).Scan(&existingAmount, &groupID, &userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("transaction not found")
		}
		return nil, fmt.Errorf("failed to find transaction: %w", err)
	}

	paidAmount := req.PaidAmount
	if paidAmount <= 0 {
		paidAmount = existingAmount
	}

	var receiptImage *string
	if req.ReceiptImage != "" {
		receiptImage = &req.ReceiptImage
	}

	query := `UPDATE transactions SET status = 'PAID', paid_amount = ?, receipt_image = ?, updated_at = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, paidAmount, receiptImage, now, req.TransactionID); err != nil {
		return nil, fmt.Errorf("failed to update transaction to paid: %w", err)
	}

	// Fetch updated transaction
	var tx models.Transaction
	var status sql.NullString
	var paidAmt sql.NullFloat64
	var rcpt sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT id, type, category, description, amount, date, status, paid_amount, group_id, user_id, created_by, receipt_image, created_at, updated_at
		FROM transactions WHERE id = ?
	`, req.TransactionID).Scan(
		&tx.ID, &tx.Type, &tx.Category, &tx.Description, &tx.Amount, &tx.Date,
		&status, &paidAmt, &tx.GroupID, &tx.UserID, &tx.CreatedByID, &rcpt,
		&tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated transaction: %w", err)
	}
	if status.Valid {
		st := models.ExpenseStatus(status.String)
		tx.Status = &st
	}
	if paidAmt.Valid {
		tx.PaidAmount = &paidAmt.Float64
	}
	if rcpt.Valid {
		tx.ReceiptImage = &rcpt.String
	}

	return &tx, nil
}

// CreateTransactionFromTelegram creates a transaction directly from Telegram bot.
func (r *TelegramRepository) CreateTransactionFromTelegram(ctx context.Context, req models.TelegramCreateTransactionRequest) (*models.Transaction, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	var userID string
	var groupID string

	// Lookup user by telegram_user_id
	err := r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE telegram_user_id = ?`, req.TelegramUserID).Scan(&userID)
	if err != nil {
		return nil, errors.New("user not found; please link your Telegram account with the web app first")
	}

	// Lookup group by telegram_chat_id
	if req.TelegramChatID != "" {
		_ = r.db.QueryRowContext(ctx, `SELECT id FROM groups WHERE telegram_chat_id = ?`, req.TelegramChatID).Scan(&groupID)
	}

	// Fallback to user's primary group if not found
	if groupID == "" {
		_ = r.db.QueryRowContext(ctx, `SELECT group_id FROM group_members WHERE user_id = ? LIMIT 1`, userID).Scan(&groupID)
	}

	if groupID == "" {
		return nil, errors.New("group not found for this user")
	}

	id := uuid.New().String()
	txType := strings.ToUpper(strings.TrimSpace(req.Type))
	if txType != string(models.TypeIncome) && txType != string(models.TypeExpense) {
		txType = string(models.TypeExpense)
	}

	var status *string
	var paidAmount *float64
	var receiptImage *string

	if req.ReceiptImage != "" {
		receiptImage = &req.ReceiptImage
	}

	if txType == string(models.TypeExpense) {
		if receiptImage != nil || req.Status == string(models.StatusPaid) {
			s := string(models.StatusPaid)
			status = &s
			paidAmount = &req.Amount
		} else {
			s := string(models.StatusPending)
			status = &s
			zero := 0.0
			paidAmount = &zero
		}
	}

	date := req.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	query := `
		INSERT INTO transactions (id, type, category, description, amount, date, status, paid_amount, group_id, user_id, created_by, receipt_image, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.ExecContext(ctx, query, id, txType, req.Category, req.Description, req.Amount, date, status, paidAmount, groupID, userID, userID, receiptImage, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction from telegram: %w", err)
	}

	var tx models.Transaction
	tx.ID = id
	tx.Type = models.TransactionType(txType)
	tx.Category = req.Category
	tx.Description = req.Description
	tx.Amount = req.Amount
	tx.Date = date
	if status != nil {
		st := models.ExpenseStatus(*status)
		tx.Status = &st
	}
	tx.PaidAmount = paidAmount
	tx.GroupID = &groupID
	tx.UserID = &userID
	tx.CreatedByID = &userID
	tx.ReceiptImage = receiptImage
	tx.CreatedAt = now
	tx.UpdatedAt = now

	return &tx, nil
}

// GetGroupTelegramChatID gets the telegram_chat_id linked to the given group.
func (r *TelegramRepository) GetGroupTelegramChatID(ctx context.Context, groupID string) (string, error) {
	var chatID sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT telegram_chat_id FROM groups WHERE id = ?`, groupID).Scan(&chatID)
	if err != nil {
		return "", err
	}
	if !chatID.Valid {
		return "", nil
	}
	return chatID.String, nil
}

// IncrementUsage increments the daily usage counters for Gemini, Worker, or bot transactions.
func (r *TelegramRepository) IncrementUsage(ctx context.Context, req models.IncrementUsageRequest) error {
	today := time.Now().Format("2006-01-02")
	nowStr := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO daily_usage_stats (date, gemini_calls, worker_requests, bot_transactions, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			gemini_calls = gemini_calls + excluded.gemini_calls,
			worker_requests = worker_requests + excluded.worker_requests,
			bot_transactions = bot_transactions + excluded.bot_transactions,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query, today, req.GeminiCalls, req.WorkerRequests, req.BotTransactions, nowStr)
	if err != nil {
		return fmt.Errorf("failed to increment daily usage: %w", err)
	}
	return nil
}

// GetUsageStats calculates remaining quota across Gemini, Cloudflare, Turso, and Render.
func (r *TelegramRepository) GetUsageStats(ctx context.Context) (*models.TelegramUsageStatsResponse, error) {
	today := time.Now().Format("2006-01-02")

	var geminiCalls, workerRequests, botTransactions int
	err := r.db.QueryRowContext(ctx, `
		SELECT gemini_calls, worker_requests, bot_transactions
		FROM daily_usage_stats WHERE date = ?
	`, today).Scan(&geminiCalls, &workerRequests, &botTransactions)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query daily usage: %w", err)
	}

	var totalTransactions int
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&totalTransactions)

	geminiLimit := 1500
	geminiRemaining := geminiLimit - geminiCalls
	if geminiRemaining < 0 {
		geminiRemaining = 0
	}
	geminiPct := (float64(geminiCalls) / float64(geminiLimit)) * 100

	cfLimit := 100000
	cfRemaining := cfLimit - workerRequests
	if cfRemaining < 0 {
		cfRemaining = 0
	}
	cfPct := (float64(workerRequests) / float64(cfLimit)) * 100

	return &models.TelegramUsageStatsResponse{
		Date: today,
		Gemini: models.ServiceQuotaItem{
			Name:           "Google Gemini AI (Slip Parser)",
			UsedToday:      geminiCalls,
			DailyLimit:     geminiLimit,
			RemainingToday: geminiRemaining,
			Unit:           "requests/day",
			PercentUsed:    geminiPct,
			Note:           "15 RPM, 1,500 RPD (Free Tier)",
		},
		Cloudflare: models.ServiceQuotaItem{
			Name:           "Cloudflare Worker (Bot Gateway)",
			UsedToday:      workerRequests,
			DailyLimit:     cfLimit,
			RemainingToday: cfRemaining,
			Unit:           "requests/day",
			PercentUsed:    cfPct,
			Note:           "100,000 requests/day (Free Tier)",
		},
		Render: models.ServiceQuotaItem{
			Name:           "Render Backend (Go API)",
			UsedToday:      1,
			DailyLimit:     24,
			RemainingToday: 750,
			Unit:           "hours/month",
			PercentUsed:    100.0,
			Note:           "750 hours/month (Runs 24/7 if single service)",
		},
		Database: models.ServiceQuotaItem{
			Name:           "Turso Database (LibSQL)",
			UsedToday:      totalTransactions,
			DailyLimit:     1000000000,
			RemainingToday: 1000000000 - totalTransactions,
			Unit:           "total rows",
			PercentUsed:    0.01,
			Note:           "25M writes, 1B reads, 9GB storage (Free Tier)",
		},
		TotalTransactions: totalTransactions,
	}, nil
}
