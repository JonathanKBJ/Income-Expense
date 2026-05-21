package repository

import (
	"context"
	"database/sql"
	"expense-tracker/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
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

// AddMember adds a user to a specific group with the given role.
// A user can now belong to multiple groups simultaneously.
func (r *GroupRepository) AddMember(ctx context.Context, groupID, userID string, role models.GroupRole) error {
	query := `INSERT INTO group_members (group_id, user_id, role) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, groupID, userID, string(role))
	if err != nil {
		return fmt.Errorf("failed to add group member: %w", err)
	}
	return nil
}

// CreateUserGroup creates a new group and adds the creator as OWNER.
func (r *GroupRepository) CreateUserGroup(ctx context.Context, userID, name string) (*models.UserGroup, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	groupID := uuid.New().String()

	group := &models.UserGroup{
		ID:   groupID,
		Name: name,
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO groups (id, name, created_at) VALUES (?, ?, ?)`, groupID, name, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO group_members (group_id, user_id, role) VALUES (?, ?, ?)`, groupID, userID, string(models.RoleOwner))
	if err != nil {
		return nil, fmt.Errorf("failed to add owner to group: %w", err)
	}

	group.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return group, tx.Commit()
}

// ListUserGroups returns all groups the user belongs to with member counts and roles.
func (r *GroupRepository) ListUserGroups(ctx context.Context, userID string) ([]models.GroupInfo, error) {
	query := `
		SELECT g.id, g.name, gm.role,
			(SELECT COUNT(*) FROM group_members WHERE group_id = g.id) as member_count
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = ?
		ORDER BY g.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user groups: %w", err)
	}
	defer rows.Close()

	var groups []models.GroupInfo
	for rows.Next() {
		var g models.GroupInfo
		var role string
		if err := rows.Scan(&g.ID, &g.Name, &role, &g.MemberCount); err != nil {
			return nil, fmt.Errorf("failed to scan user group: %w", err)
		}
		g.MyRole = models.GroupRole(role)
		g.Members = []models.GroupMember{} // empty, detailed members not needed for list
		groups = append(groups, g)
	}

	if groups == nil {
		groups = []models.GroupInfo{}
	}
	return groups, nil
}

// GetUserGroupID returns the first group ID for a specific user (backward compat).
func (r *GroupRepository) GetUserGroupID(ctx context.Context, userID string) (string, error) {
	query := `SELECT gm.group_id FROM group_members gm
		JOIN groups g ON g.id = gm.group_id
		WHERE gm.user_id = ?
		ORDER BY g.created_at ASC LIMIT 1`
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

// DeleteGroup deletes a group and all associated data (CASCADE via FK).
// Only the group owner should call this.
func (r *GroupRepository) DeleteGroup(ctx context.Context, groupID, userID string) error {
	// Verify user is OWNER of the group
	role, err := r.GetMemberRole(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to verify ownership: %w", err)
	}
	if role != models.RoleOwner {
		return fmt.Errorf("only the group owner can delete the group")
	}

	_, err = r.db.ExecContext(ctx, `DELETE FROM groups WHERE id = ?`, groupID)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	return nil
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

// DeleteMember removes a single membership without restoring to personal group.
// Used when a user leaves a group but keeps other groups.
func (r *GroupRepository) DeleteMember(ctx context.Context, groupID, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}
	return nil
}

// RemoveMember removes a specific user from a group and returns them to their personal group.
func (r *GroupRepository) RemoveMember(ctx context.Context, groupID, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Get the user's username
	var username string
	err = tx.QueryRowContext(ctx, `SELECT username FROM users WHERE id = ?`, userID).Scan(&username)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// 2. Remove from current group
	_, err = tx.ExecContext(ctx, `DELETE FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove group member: %w", err)
	}

	// 3. Find personal group (name pattern: 'username's Group')
	personalGroupName := fmt.Sprintf("%s's Group", username)
	var personalGroupID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM groups WHERE name = ? LIMIT 1`, personalGroupName).Scan(&personalGroupID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No personal group found, just commit the removal
			return tx.Commit()
		}
		return fmt.Errorf("failed to find personal group: %w", err)
	}

	// 4. Restore membership to personal group
	// Note: We use INSERT OR IGNORE just in case they are already in it somehow (though internal logic should prevent it)
	_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO group_members (group_id, user_id, role) VALUES (?, ?, ?)`, personalGroupID, userID, string(models.RoleOwner))
	if err != nil {
		return fmt.Errorf("failed to restore to personal group: %w", err)
	}

	return tx.Commit()
}

// GetGroupInfo returns full group details including members and the requesting user's role.
func (r *GroupRepository) GetGroupInfo(ctx context.Context, groupID, userID string) (*models.GroupInfo, error) {
	// Get group name and member count
	var info models.GroupInfo
	err := r.db.QueryRowContext(ctx, `
		SELECT g.id, g.name, (SELECT COUNT(*) FROM group_members WHERE group_id = g.id)
		FROM groups g WHERE g.id = ?
	`, groupID).Scan(&info.ID, &info.Name, &info.MemberCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("group not found")
		}
		return nil, fmt.Errorf("failed to get group info: %w", err)
	}

	// Get member list with roles
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.username, gm.role, gm.group_id
		FROM users u
		JOIN group_members gm ON u.id = gm.user_id
		WHERE gm.group_id = ?
		ORDER BY gm.ROWID ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m models.GroupMember
		var gID string
		if err := rows.Scan(&m.UserID, &m.Username, &m.Role, &gID); err != nil {
			return nil, fmt.Errorf("failed to scan group member: %w", err)
		}
		// joinedAt is not stored on group_members yet; set from group creation as approximation
		if m.UserID == userID {
			info.MyRole = m.Role
		}
		info.Members = append(info.Members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating group members: %w", err)
	}

	if info.Members == nil {
		info.Members = []models.GroupMember{}
	}

	return &info, nil
}

// UpdateGroupName renames a group (OWNER only — enforced at handler level).
func (r *GroupRepository) UpdateGroupName(ctx context.Context, groupID, newName string) error {
	query := `UPDATE groups SET name = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, newName, groupID)
	if err != nil {
		return fmt.Errorf("failed to update group name: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("group not found")
	}
	return nil
}

// GetMemberRole returns the role of a user in a group.
func (r *GroupRepository) GetMemberRole(ctx context.Context, groupID, userID string) (models.GroupRole, error) {
	query := `SELECT role FROM group_members WHERE group_id = ? AND user_id = ?`
	var role string
	err := r.db.QueryRowContext(ctx, query, groupID, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("member not found in group")
		}
		return "", fmt.Errorf("failed to get member role: %w", err)
	}
	return models.GroupRole(role), nil
}

// UpdateMemberRole changes a member's role (OWNER only — enforced at handler level).
func (r *GroupRepository) UpdateMemberRole(ctx context.Context, groupID, userID string, newRole models.GroupRole) error {
	query := `UPDATE group_members SET role = ? WHERE group_id = ? AND user_id = ?`
	result, err := r.db.ExecContext(ctx, query, string(newRole), groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("member not found in group")
	}
	return nil
}

// GroupHasMultipleMembers returns true if the group has more than 1 member.
func (r *GroupRepository) GroupHasMultipleMembers(ctx context.Context, groupID string) (bool, error) {
	query := `SELECT COUNT(*) FROM group_members WHERE group_id = ?`
	var count int
	err := r.db.QueryRowContext(ctx, query, groupID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to count group members: %w", err)
	}
	return count > 1, nil
}
