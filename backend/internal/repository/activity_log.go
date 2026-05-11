package repository

import (
	"context"
	"database/sql"
	"fmt"

	"expense-tracker/internal/models"
)

// ActivityLogRepository handles the activity_log table (Phase 1: Group Awareness).
type ActivityLogRepository struct {
	db *sql.DB
}

// NewActivityLogRepository creates a new repository instance.
func NewActivityLogRepository(db *sql.DB) *ActivityLogRepository {
	return &ActivityLogRepository{db: db}
}

// CreateEntry inserts a new activity log entry.
func (r *ActivityLogRepository) CreateEntry(ctx context.Context, entry *models.ActivityLogEntry) error {
	query := `INSERT INTO activity_log (id, group_id, user_id, action, entity_type, entity_id, details, created_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID,
		entry.GroupID,
		entry.UserID,
		entry.Action,
		entry.EntityType,
		entry.EntityID,
		entry.Details,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert activity log: %w", err)
	}

	return nil
}

// GetRecentByGroup returns the most recent N activities for a group, with usernames.
func (r *ActivityLogRepository) GetRecentByGroup(ctx context.Context, groupID string, limit int) ([]models.ActivityLogEntry, error) {
	query := `
		SELECT a.id, a.group_id, a.user_id, u.username, a.action, a.entity_type, a.entity_id, a.details, a.created_at
		FROM activity_log a
		JOIN users u ON a.user_id = u.id
		WHERE a.group_id = ?
		ORDER BY a.created_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, groupID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query activity log: %w", err)
	}
	defer rows.Close()

	var entries []models.ActivityLogEntry
	for rows.Next() {
		var e models.ActivityLogEntry
		var entityID, details sql.NullString
		if err := rows.Scan(&e.ID, &e.GroupID, &e.UserID, &e.Username, &e.Action,
			&e.EntityType, &entityID, &details, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan activity log row: %w", err)
		}
		if entityID.Valid {
			e.EntityID = entityID.String
		}
		if details.Valid {
			e.Details = details.String
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activity log rows: %w", err)
	}

	if entries == nil {
		entries = []models.ActivityLogEntry{}
	}

	return entries, nil
}
