package models

import "time"

// TelegramLinkCode represents a temporary pairing code between a web user/group and Telegram.
type TelegramLinkCode struct {
	Code      string    `json:"code"`
	UserID    string    `json:"userId"`
	GroupID   string    `json:"groupId"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// GenerateTelegramLinkCodeResponse is returned to the web frontend when a user generates a link code.
type GenerateTelegramLinkCodeResponse struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expiresAt"`
	ExpiresIn int    `json:"expiresIn"` // in seconds, e.g. 900 (15 min)
}

// VerifyTelegramLinkRequest is sent by Cloudflare Worker when a user submits a link code in a Telegram chat.
type VerifyTelegramLinkRequest struct {
	Code             string `json:"code"`
	TelegramUserID   string `json:"telegramUserId"`
	TelegramUsername string `json:"telegramUsername"`
	TelegramChatID   string `json:"telegramChatId"`
}

// VerifyTelegramLinkResponse confirms successful pairing.
type VerifyTelegramLinkResponse struct {
	UserID           string `json:"userId"`
	Username         string `json:"username"`
	GroupID          string `json:"groupId"`
	GroupName        string `json:"groupName"`
	TelegramUserID   string `json:"telegramUserId"`
	TelegramUsername string `json:"telegramUsername"`
	TelegramChatID   string `json:"telegramChatId"`
}

// TelegramPendingBillItem represents a pending transaction or bill for matching with a slip.
type TelegramPendingBillItem struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"`
	Status      string  `json:"status"`
	UserID      string  `json:"userId"`
	Username    string  `json:"username,omitempty"`
}

// TelegramPendingBillsResponse returns pending bills for matching.
type TelegramPendingBillsResponse struct {
	GroupID      string                    `json:"groupId"`
	GroupName    string                    `json:"groupName"`
	UserID       string                    `json:"userId"`
	PendingBills []TelegramPendingBillItem `json:"pendingBills"`
}

// TelegramPayBillRequest is sent by Cloudflare Worker to mark a pending bill as PAID with receipt image.
type TelegramPayBillRequest struct {
	TransactionID  string  `json:"transactionId"`
	PaidAmount     float64 `json:"paidAmount"`
	ReceiptImage   string  `json:"receiptImage"` // Base64
	TelegramUserID string  `json:"telegramUserId"`
}

// TelegramCreateTransactionRequest is sent by Cloudflare Worker to create a new transaction directly.
type TelegramCreateTransactionRequest struct {
	TelegramUserID string  `json:"telegramUserId"`
	TelegramChatID string  `json:"telegramChatId"`
	Type           string  `json:"type"` // INCOME or EXPENSE
	Category       string  `json:"category"`
	Description    string  `json:"description"`
	Amount         float64 `json:"amount"`
	Date           string  `json:"date"`
	ReceiptImage   string  `json:"receiptImage,omitempty"` // Base64
	Status         string  `json:"status,omitempty"`       // PENDING or PAID
}

// TelegramStatusResponse indicates whether the user and/or group is linked to Telegram.
type TelegramStatusResponse struct {
	IsUserLinked     bool    `json:"isUserLinked"`
	TelegramUsername *string `json:"telegramUsername,omitempty"`
	IsGroupLinked    bool    `json:"isGroupLinked"`
	TelegramChatID   *string `json:"telegramChatId,omitempty"`
}
