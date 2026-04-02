package repository

import (
	"context"
	"database/sql"
	"errors"
	"expense-tracker/internal/models"
	"fmt"
	"time"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts a new user into the database.
func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	now := time.Now().UTC().Format(time.RFC3339)
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	query := `
		INSERT INTO users (id, username, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.Role,
		user.Status,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByUsername finds a user by their username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT id, username, password_hash, role, status, created_at, updated_at FROM users WHERE username = ?`
	row := r.db.QueryRowContext(ctx, query, username)

	var user models.User
	var createdAtStr, updatedAtStr string
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.Status, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return &user, nil
}

// GetByID finds a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `SELECT id, username, password_hash, role, status, created_at, updated_at FROM users WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var user models.User
	var createdAtStr, updatedAtStr string
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.Status, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return &user, nil
}

// ListUsers returns all users in the system (for Admin).
func (r *UserRepository) ListUsers(ctx context.Context) ([]models.User, error) {
	query := `SELECT id, username, role, status, created_at, updated_at FROM users`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.Status, &createdAtStr, &updatedAtStr); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		user.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		users = append(users, user)
	}
	return users, nil
}

// UpdateStatus changes the status of a user.
func (r *UserRepository) UpdateStatus(ctx context.Context, id string, status models.UserStatus) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := `UPDATE users SET status = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, now, id)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}
	return nil
}
