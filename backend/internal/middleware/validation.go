package middleware

import (
	"fmt"
	"regexp"
	"time"

	"expense-tracker/internal/models"
)

// dateRegex validates YYYY-MM-DD format.
var dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ValidateCreateRequest validates a CreateTransactionRequest.
// Enforces strict type-based rules:
//   - INCOME: status and paidAmount must be nil (rejected if present)
//   - EXPENSE: status defaults to PENDING if absent, paidAmount defaults to 0
//   - If status is PAID, paidAmount must be > 0
func ValidateCreateRequest(req *models.CreateTransactionRequest) error {
	// Validate required fields
	if req.Type != models.TypeIncome && req.Type != models.TypeExpense {
		return fmt.Errorf("type must be 'INCOME' or 'EXPENSE', got '%s'", req.Type)
	}

	if req.Category == "" {
		return fmt.Errorf("category is required")
	}

	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	if req.Date == "" {
		return fmt.Errorf("date is required")
	}

	if !dateRegex.MatchString(req.Date) {
		return fmt.Errorf("date must be in YYYY-MM-DD format")
	}

	// Validate the date is a real calendar date
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		return fmt.Errorf("date is not a valid calendar date: %s", req.Date)
	}

	// Type-specific validation
	switch req.Type {
	case models.TypeIncome:
		// INCOME must NOT have expense-only fields
		if req.Status != nil {
			return fmt.Errorf("income transactions must not have a status field")
		}
		if req.PaidAmount != nil {
			return fmt.Errorf("income transactions must not have a paidAmount field")
		}

	case models.TypeExpense:
		// Validate status if provided
		if req.Status != nil {
			if *req.Status != models.StatusPending && *req.Status != models.StatusPaid {
				return fmt.Errorf("status must be 'PENDING' or 'PAID', got '%s'", *req.Status)
			}
		}

		// Validate paidAmount if provided
		if req.PaidAmount != nil && *req.PaidAmount < 0 {
			return fmt.Errorf("paidAmount must be >= 0")
		}

		// Business rule: if status is PAID, paidAmount must be > 0
		if req.Status != nil && *req.Status == models.StatusPaid {
			if req.PaidAmount == nil || *req.PaidAmount <= 0 {
				return fmt.Errorf("when status is 'PAID', paidAmount must be greater than 0")
			}
		}
	}

	return nil
}

// ValidateUpdateRequest validates an UpdateTransactionRequest against
// the existing transaction to enforce type-based rules.
func ValidateUpdateRequest(req *models.UpdateTransactionRequest, existing *models.Transaction) error {
	// 1. Basic validation: Amount must be > 0 if provided
	if req.Amount != nil && *req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	// 2. Validate expense-specific fields
	if existing.Type == models.TypeIncome {
		// Cannot update INCOME transactions with expense-specific fields
		if req.Status != nil || req.PaidAmount != nil {
			return fmt.Errorf("cannot update status or paidAmount on an INCOME transaction")
		}
	}

	// 3. Must provide at least one field to update
	if req.Amount == nil && req.Status == nil && req.PaidAmount == nil {
		return fmt.Errorf("at least one of amount, status or paidAmount must be provided")
	}

	// Validate status value if provided
	if req.Status != nil {
		if *req.Status != models.StatusPending && *req.Status != models.StatusPaid {
			return fmt.Errorf("status must be 'PENDING' or 'PAID', got '%s'", *req.Status)
		}
	}

	// Validate paidAmount if provided
	if req.PaidAmount != nil && *req.PaidAmount < 0 {
		return fmt.Errorf("paidAmount must be >= 0")
	}

	// Business rule: when setting status to PAID, paidAmount must be > 0
	// Check both the new value and the existing value
	finalStatus := existing.Status
	if req.Status != nil {
		finalStatus = req.Status
	}

	finalPaidAmount := existing.PaidAmount
	if req.PaidAmount != nil {
		finalPaidAmount = req.PaidAmount
	}

	if finalStatus != nil && *finalStatus == models.StatusPaid {
		if finalPaidAmount == nil || *finalPaidAmount <= 0 {
			return fmt.Errorf("when status is 'PAID', paidAmount must be greater than 0")
		}
	}

	return nil
}
