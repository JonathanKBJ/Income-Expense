package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"expense-tracker/internal/models"

	"github.com/google/uuid"
)

// GroupInviteRepository handles the group_invites table (Phase 2: Self-Service).
type GroupInviteRepository struct {
	db *sql.DB
}

// NewGroupInviteRepository creates a new repository instance.
func NewGroupInviteRepository(db *sql.DB) *GroupInviteRepository {
	return &GroupInviteRepository{db: db}
}

// CreateInvite generates a new invite code with 24-hour expiry.
func (r *GroupInviteRepository) CreateInvite(ctx context.Context, groupID, createdBy string) (*models.CreateInviteResponse, error) {
	id := uuid.New().String()
	code := uuid.New().String()
	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)

	query := `INSERT INTO group_invites (id, group_id, created_by, code, expires_at, created_at)
	           VALUES (?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		id,
		groupID,
		createdBy,
		code,
		expiresAt.Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create group invite: %w", err)
	}

	return &models.CreateInviteResponse{
		Code:      code,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

// ValidateInvite checks if an invite code is valid and not expired.
// Returns the group ID if valid, or an error if invalid/expired.
func (r *GroupInviteRepository) ValidateInvite(ctx context.Context, code string) (string, error) {
	// Clean up expired invites as part of validation
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = r.db.ExecContext(ctx, `DELETE FROM group_invites WHERE expires_at < ?`, now)

	query := `SELECT group_id FROM group_invites WHERE code = ?`
	var groupID string
	err := r.db.QueryRowContext(ctx, query, code).Scan(&groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("invalid or expired invite code")
		}
		return "", fmt.Errorf("failed to validate invite: %w", err)
	}

	return groupID, nil
}

// DeleteInvite removes an invite code (single-use).
func (r *GroupInviteRepository) DeleteInvite(ctx context.Context, code string) error {
	query := `DELETE FROM group_invites WHERE code = ?`
	_, err := r.db.ExecContext(ctx, query, code)
	if err != nil {
		return fmt.Errorf("failed to delete invite: %w", err)
	}
	return nil
}
