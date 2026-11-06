package handlers

import (
	"context"
	"fmt"

	"expense-tracker/internal/models"
)

const creditPostingManualRequired = "MANUAL_REQUIRED"

// Dashboard expenses track cash. Split-account debt is posted separately with
// an explicit allocation; the legacy asynchronous bridge has no reversal link.
func (h *TransactionHandler) dashboardCreditPostingStatus(ctx context.Context, accountID *string, groupID, userID string, newlySupplied bool) (string, error) {
	if accountID == nil || *accountID == "" || h.creditRepo == nil {
		return "", nil
	}
	scopedUser := ""
	if h.groupRepo != nil {
		settings, err := h.groupRepo.GetGroupSettings(ctx, groupID)
		if err != nil {
			return "", err
		}
		if !settings.ShareCredit {
			scopedUser = userID
		}
	}
	account, err := h.creditRepo.GetAccountByID(ctx, *accountID, groupID, scopedUser)
	if err != nil {
		if !newlySupplied {
			// An unchanged stale mapping on an existing cash entry must not block
			// unrelated cash updates if the credit account was deleted.
			return "", nil
		}
		return "", err
	}
	if account == nil {
		if !newlySupplied {
			return "", nil
		}
		return "", fmt.Errorf("linked credit account is unavailable")
	}
	if account.BalanceMode == models.CreditBalanceMode("PRINCIPAL_SPLIT") {
		return creditPostingManualRequired, nil
	}
	return "", nil
}
