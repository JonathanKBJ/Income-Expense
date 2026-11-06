package database

import (
	"context"
	"fmt"
)

// Check column existence explicitly: an unsuccessful migration must not look like
// an already-migrated database and leave allocation writers using a partial schema.
func (d *DB) migrateCreditAllocations(ctx context.Context) error {
	columns := []struct{ table, name, definition string }{
		{"credit_accounts", "balance_mode", "TEXT NOT NULL DEFAULT 'LEGACY'"},
		{"credit_accounts", "interest_balance", "REAL NOT NULL DEFAULT 0"},
		{"credit_accounts", "fee_balance", "REAL NOT NULL DEFAULT 0"},
		{"credit_accounts", "credit_limit_basis", "TEXT NOT NULL DEFAULT 'PRINCIPAL'"},
		{"credit_accounts", "allocation_order", "TEXT NOT NULL DEFAULT 'FEES_INTEREST_PRINCIPAL'"},
		{"credit_accounts", "balance_as_of_date", "TEXT"},
		{"credit_accounts", "balance_note", "TEXT NOT NULL DEFAULT ''"},
		{"credit_transactions", "principal_amount", "REAL"},
		{"credit_transactions", "interest_amount", "REAL"},
		{"credit_transactions", "fee_amount", "REAL"},
		{"credit_transactions", "allocation_source", "TEXT"},
	}
	for _, column := range columns {
		var exists int
		if err := d.QueryRowContext(ctx, "SELECT count(*) FROM pragma_table_info(?) WHERE name = ?", column.table, column.name).Scan(&exists); err != nil {
			return fmt.Errorf("inspect credit allocation column: %w", err)
		}
		if exists == 0 {
			if _, err := d.ExecContext(ctx, "ALTER TABLE "+column.table+" ADD COLUMN "+column.name+" "+column.definition); err != nil {
				return fmt.Errorf("migrate credit allocation column %s: %w", column.name, err)
			}
		}
	}
	return nil
}
