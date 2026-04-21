package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"expense-tracker/internal/database"
	"expense-tracker/internal/models"

	"github.com/google/uuid"
)

// CategoryRepository provides CRUD operations for categories.
type CategoryRepository struct {
	db *database.DB
}

// NewCategoryRepository creates a new repository instance.
func NewCategoryRepository(db *database.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// GetAll retrieves all categories for a group or user, optionally filtered by type.
func (r *CategoryRepository) GetAll(ctx context.Context, userID, groupID, catType string) ([]models.Category, error) {
	var query string
	var args []interface{}

	// Query categories that belong to the user's group OR the user themselves
	// If groupID is provided, we prioritize group categories.
	// Filter by type if provided.
	baseQuery := `SELECT id, name, type, group_id, user_id, created_at, updated_at FROM categories WHERE `
	var filters []string

	if catType != "" {
		filters = append(filters, "type = ?")
		args = append(args, catType)
	}

	if groupID != "" {
		filters = append(filters, "group_id = ?")
		args = append(args, groupID)
	} else {
		filters = append(filters, "user_id = ?")
		args = append(args, userID)
	}

	query = baseQuery + strings.Join(filters, " AND ") + " ORDER BY type ASC, name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.GroupID, &cat.UserID, &cat.CreatedAt, &cat.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}
		categories = append(categories, cat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}

	if categories == nil {
		categories = []models.Category{}
	}

	return categories, nil
}

// GetByID retrieves a single category by its ID and either group ID or user ID.
func (r *CategoryRepository) GetByID(ctx context.Context, id, userID, groupID string) (*models.Category, error) {
	query := `SELECT id, name, type, group_id, user_id, created_at, updated_at FROM categories WHERE id = ? AND (group_id = ? OR user_id = ?)`

	var cat models.Category
	err := r.db.QueryRowContext(ctx, query, id, groupID, userID).Scan(
		&cat.ID, &cat.Name, &cat.Type, &cat.GroupID, &cat.UserID, &cat.CreatedAt, &cat.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get category by id: %w", err)
	}

	return &cat, nil
}

// Create inserts a new category for a specific group or user.
func (r *CategoryRepository) Create(ctx context.Context, req models.CreateCategoryRequest, userID, groupID string) (*models.Category, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	query := `INSERT INTO categories (id, name, type, user_id, group_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`

	var gid, uid interface{}
	if groupID != "" {
		gid = groupID
	}
	if userID != "" {
		uid = userID
	}

	_, err := r.db.ExecContext(ctx, query, id, req.Name, string(req.Type), uid, gid, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert category: %w", err)
	}

	return r.GetByID(ctx, id, userID, groupID)
}

// Update modifies a category's name.
func (r *CategoryRepository) Update(ctx context.Context, id, userID, groupID string, req models.UpdateCategoryRequest) (*models.Category, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	query := `UPDATE categories SET name = ?, updated_at = ? WHERE id = ? AND (group_id = ? OR user_id = ?)`

	result, err := r.db.ExecContext(ctx, query, req.Name, now, id, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, nil
	}

	return r.GetByID(ctx, id, userID, groupID)
}

// Delete removes a category by ID if it belongs to the specified group or user.
func (r *CategoryRepository) Delete(ctx context.Context, id, userID, groupID string) error {
	query := `DELETE FROM categories WHERE id = ? AND (group_id = ? OR user_id = ?)`

	result, err := r.db.ExecContext(ctx, query, id, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("category not found")
	}

	return nil
}

// IsCategoryInUse checks if a category name+type combination is referenced by any transactions within the group or by the user.
func (r *CategoryRepository) IsCategoryInUse(ctx context.Context, name, catType, userID, groupID string) (bool, error) {
	query := `SELECT COUNT(*) FROM transactions WHERE category = ? AND type = ? AND (group_id = ? OR user_id = ?)`

	var count int
	err := r.db.QueryRowContext(ctx, query, name, catType, groupID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check category usage: %w", err)
	}

	return count > 0, nil
}
