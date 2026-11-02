package repository

import (
	"errors"
	"testing"
	"time"
)

func TestValidateEarlySettlementDate(t *testing.T) {
	// Bangkok has already rolled into 2 November while UTC is still 1 November.
	now := time.Date(2026, 11, 1, 18, 30, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		date  string
		valid bool
	}{
		{"today in Bangkok", "2026-11-02", true},
		{"earlier calendar day", "2026-10-31", true},
		{"leap day", "2024-02-29", true},
		{"future Bangkok day", "2026-11-03", false},
		{"non leap February", "2026-02-29", false},
		{"impossible day", "2026-04-31", false},
		{"missing date", "", false},
		{"not padded", "2026-1-02", false},
		{"timestamp instead of date", "2026-11-02T00:00:00Z", false},
		{"whitespace", " 2026-11-02", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEarlySettlementDate(tc.date, now)
			if tc.valid && err != nil {
				t.Fatalf("expected valid settlement date, got %v", err)
			}
			if !tc.valid && !errors.Is(err, ErrInvalidSettlementDate) {
				t.Fatalf("expected ErrInvalidSettlementDate, got %v", err)
			}
		})
	}
}
