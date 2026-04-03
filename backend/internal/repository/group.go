package repository

import (
	"context"
	"database/sql"
	"expense-tracker/internal/models"
	"fmt"
	"time"
)

type GroupRepository struct {
	db *sql.DB
}

func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// CreateGroup adds a new group to the database.
func (r *GroupRepository) CreateGroup(ctx context.Context, group *models.UserGroup) error {
	now := time.Now().UTC().Format(time.RFC3339)
	group.CreatedAt = time.Now().UTC()

	query := `INSERT INTO groups (id, name, created_at) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, group.ID, group.Name, now)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}
	return nil
}

// AddMember adds a user to a specific group.
func (r *GroupRepository) AddMember(ctx context.Context, groupID, userID string) error {
	// 1 User can only belong to 1 Group (as per requirement)
	// We check and remove they from any existing group first for consistency
	// but the UI/Logic should ideally prevent this too.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing groups for this user
	_, err = tx.ExecContext(ctx, `DELETE FROM group_members WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to clear old membership: %w", err)
	}

	// Add new group membership
	query := `INSERT INTO group_members (group_id, user_id) VALUES (?, ?)`
	_, err = tx.ExecContext(ctx, query, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to add group member: %w", err)
	}

	return tx.Commit()
}

// GetUserGroupID returns the group ID for a specific user.
func (r *GroupRepository) GetUserGroupID(ctx context.Context, userID string) (string, error) {
	query := `SELECT group_id FROM group_members WHERE user_id = ? LIMIT 1`
	var groupID string
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No group found
		}
		return "", fmt.Errorf("failed to get user group: %w", err)
	}
	return groupID, nil
}

// ListGroups returns all groups (for Admin).
func (r *GroupRepository) ListGroups(ctx context.Context) ([]models.UserGroup, error) {
	query := `SELECT id, name, created_at FROM groups`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer rows.Close()

	var groups []models.UserGroup
	for rows.Next() {
		var g models.UserGroup
		var createdAtStr string
		if err := rows.Scan(&g.ID, &g.Name, &createdAtStr); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		groups = append(groups, g)
	}
	return groups, nil
}

// GetGroupMembers returns all users belonging to a specific group.
func (r *GroupRepository) GetGroupMembers(ctx context.Context, groupID string) ([]models.User, error) {
	query := `
		SELECT u.id, u.username, u.role, u.status 
		FROM users u
		JOIN group_members gm ON u.id = gm.user_id
		WHERE gm.group_id = ?
	`
	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Status); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

// RemoveMember removes a specific user from a group.
func (r *GroupRepository) RemoveMember(ctx context.Context, groupID, userID string) error {
	query := `DELETE FROM group_members WHERE group_id = ? AND user_id = ?`
	_, err := r.db.ExecContext(ctx, query, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove group member: %w", err)
	}
	return nil
}
