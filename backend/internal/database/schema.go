package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('ADMIN', 'USER')),
    status        TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);
`

const createGroupsTable = `
CREATE TABLE IF NOT EXISTS groups (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL
);
`

const createGroupMembersTable = `
CREATE TABLE IF NOT EXISTS group_members (
    group_id TEXT NOT NULL,
    user_id  TEXT NOT NULL,
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`

// createTransactionsTable is the DDL for the transactions table.
const createTransactionsTable = `
CREATE TABLE IF NOT EXISTS transactions (
    id          TEXT PRIMARY KEY,
    type        TEXT NOT NULL CHECK (type IN ('INCOME', 'EXPENSE')),
    category    TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    amount      REAL NOT NULL CHECK (amount > 0),
    date        TEXT NOT NULL,
    status      TEXT CHECK (status IN ('PENDING', 'PAID') OR status IS NULL),
    paid_amount REAL CHECK (paid_amount >= 0 OR paid_amount IS NULL),
    group_id      TEXT,
    user_id       TEXT,
    receipt_image TEXT,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);
`

// createCategoriesTable is the DDL for the categories table.
const createCategoriesTable = `
CREATE TABLE IF NOT EXISTS categories (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('INCOME', 'EXPENSE')),
    group_id   TEXT,
    user_id    TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(name, type, group_id, user_id)
);
`

// createIndexes defines performance indexes for common query patterns.
const createIndexes = `
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_transactions_group ON transactions(group_id);
CREATE INDEX IF NOT EXISTS idx_categories_group ON categories(group_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_global_unique ON categories(name, type) WHERE group_id IS NULL AND user_id IS NULL;
`

// defaultCategories are seeded on first migration using INSERT OR IGNORE.
var defaultCategories = map[string][]string{
	"INCOME": {
		"Salary",
		"Freelance",
		"Investment",
		"Rental",
		"Gift",
		"Refund",
		"Other",
	},
	"EXPENSE": {
		"Food & Dining",
		"Transportation",
		"Housing",
		"Utilities",
		"Healthcare",
		"Entertainment",
		"Shopping",
		"Education",
		"Insurance",
		"Savings",
		"Other",
	},
}

// Migrate runs the database schema migration.
func (d *DB) Migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create new tables
	tables := []struct {
		name string
		ddl  string
	}{
		{"users", createUsersTable},
		{"groups", createGroupsTable},
		{"group_members", createGroupMembersTable},
		{"transactions", createTransactionsTable},
		{"categories", createCategoriesTable},
	}

	for _, t := range tables {
		if _, err := d.ExecContext(ctx, t.ddl); err != nil {
			return fmt.Errorf("failed to create %s table: %w", t.name, err)
		}
	}

	// Add columns to existing tables if they don't exist (handle evolution)
	// SQLite ALTER TABLE ADD COLUMN is limited but works for simple additions.
	alterations := []string{
		"ALTER TABLE transactions ADD COLUMN group_id TEXT",
		"ALTER TABLE transactions ADD COLUMN user_id TEXT",
		"ALTER TABLE transactions ADD COLUMN receipt_image TEXT",
		"ALTER TABLE categories ADD COLUMN group_id TEXT",
	}

	for _, sql := range alterations {
		// We ignore errors because the columns might already exist
		_, _ = d.ExecContext(ctx, sql)
	}

	// Migration for categories to update UNIQUE constraint and add user_id
	var hasUserID int
	_ = d.QueryRowContext(ctx, "SELECT count(*) FROM pragma_table_info('categories') WHERE name='user_id'").Scan(&hasUserID)
	if hasUserID == 0 {
		// Table exists but lacks user_id, or it's a fresh install where CREATE TABLE above already run
		// However, if CREATE TABLE IF NOT EXISTS already ran, hasUserID would be 1.
		// If it's an old table, we need to migrate.
		// Check if the table actually exists first (pragma_table_info returns nothing if table doesn't exist)
		var tableExists int
		_ = d.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='categories'").Scan(&tableExists)

		if tableExists > 0 {
			// Perform migration: rename, create new, copy, drop
			migrationSQL := []string{
				"ALTER TABLE categories RENAME TO categories_old",
				createCategoriesTable,
				"INSERT INTO categories (id, name, type, group_id, created_at, updated_at) SELECT id, name, type, group_id, created_at, updated_at FROM categories_old",
				"DROP TABLE categories_old",
			}
			for _, sql := range migrationSQL {
				if _, err := d.ExecContext(ctx, sql); err != nil {
					return fmt.Errorf("failed to migrate categories table: %w", err)
				}
			}
		}
	}

	// Create/Recreate UNIQUE constraint on categories for group_id
	// Note: SQLite doesn't support ALTER TABLE DROP CONSTRAINT or ADD CONSTRAINT.
	// We'll rely on the CREATE TABLE IF NOT EXISTS for new setups.
	// For existing ones, we might need a more complex migration if we want to enforce it.
	// For now, let's keep it simple.

	// Create indexes
	if _, err := d.ExecContext(ctx, createIndexes); err != nil {
		// If index creation fails due to existing duplicates, we handle that by cleaning up duplicates first
		// This can happen if the database already has many duplicates before this index was added.
	}

	// Clean up any duplicate global categories before proceeding
	cleanupSQL := `
		DELETE FROM categories 
		WHERE id NOT IN (
			SELECT MIN(id) 
			FROM categories 
			WHERE group_id IS NULL AND user_id IS NULL 
			GROUP BY name, type
		) AND group_id IS NULL AND user_id IS NULL;
	`
	_, _ = d.ExecContext(ctx, cleanupSQL)

	// Re-try create indexes (in case it failed above)
	if _, err := d.ExecContext(ctx, createIndexes); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Fix orphaned transactions and categories by assigning them to their user's primary group
	migrationSQL := []string{
		`UPDATE transactions 
		 SET group_id = (SELECT group_id FROM group_members WHERE user_id = transactions.user_id LIMIT 1) 
		 WHERE (group_id IS NULL OR group_id = '') AND user_id IS NOT NULL`,

		`UPDATE categories 
		 SET group_id = (SELECT group_id FROM group_members WHERE user_id = categories.user_id LIMIT 1) 
		 WHERE (group_id IS NULL OR group_id = '') AND user_id IS NOT NULL`,
	}
	for _, sql := range migrationSQL {
		_, _ = d.ExecContext(ctx, sql)
	}

	// Seed default categories for NULL group (global defaults or template)
	if err := d.seedCategories(ctx); err != nil {
		return fmt.Errorf("failed to seed default categories: %w", err)
	}

	return nil
}

// seedCategories inserts default categories if they don't already exist.
func (d *DB) seedCategories(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)

	for catType, names := range defaultCategories {
		for _, name := range names {
			id := uuid.New().String()
			_, err := d.ExecContext(ctx,
				`INSERT OR IGNORE INTO categories (id, name, type, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
				id, name, catType, now, now,
			)
			if err != nil {
				return fmt.Errorf("failed to seed category %q (%s): %w", name, catType, err)
			}
		}
	}

	return nil
}
