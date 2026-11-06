package database

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCreditAllocationMigrationPreservesLegacyBalancesAndUnknownHistory(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "legacy.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// This is the pre-allocation account/ledger shape, not the new CREATE TABLE
	// statements. It exercises ALTER migration and verifies no guessed backfill.
	for _, statement := range []string{
		`CREATE TABLE credit_accounts (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL, bank TEXT NOT NULL DEFAULT '',
			credit_limit REAL NOT NULL DEFAULT 0, current_balance REAL NOT NULL DEFAULT 0,
			statement_day INTEGER, payment_due_day INTEGER, interest_rate REAL DEFAULT 0,
			min_payment_rate REAL DEFAULT 5, min_payment_floor REAL DEFAULT 500,
			min_payment_calc_mode TEXT NOT NULL DEFAULT 'BALANCE_PERCENT', status TEXT NOT NULL DEFAULT 'ACTIVE',
			notes TEXT DEFAULT '', group_id TEXT NOT NULL, user_id TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE credit_transactions (
			id TEXT PRIMARY KEY, account_id TEXT NOT NULL, type TEXT NOT NULL, amount REAL NOT NULL,
			date TEXT NOT NULL, description TEXT DEFAULT '', payment_type TEXT, installment_id TEXT, receipt_image TEXT,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`INSERT INTO credit_accounts (id, name, type, credit_limit, current_balance, group_id, user_id, created_at, updated_at)
			VALUES ('legacy-a', 'Untouched A', 'CASH_CARD', 82900, 78013.33, 'legacy-g', 'legacy-u', '2026-01-01', '2026-01-01'),
			('legacy-b', 'Untouched B', 'CREDIT_CARD', 25000, 1500.51, 'legacy-g', 'legacy-u', '2026-01-01', '2026-01-01')`,
		`INSERT INTO credit_transactions (id, account_id, type, amount, date, created_at, updated_at)
			VALUES ('legacy-payment', 'legacy-a', 'PAYMENT', 4000, '2026-01-01', '2026-01-01', '2026-01-01')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	wrapped := &DB{DB: db}
	for range 2 {
		if err := wrapped.Migrate(); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := db.Query(`SELECT id, current_balance, balance_mode, interest_balance, fee_balance, credit_limit_basis, allocation_order, balance_as_of_date, balance_note FROM credit_accounts ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, mode, basis, order, note string
		var total, interest, fee float64
		var asOf sql.NullString
		if err := rows.Scan(&id, &total, &mode, &interest, &fee, &basis, &order, &asOf, &note); err != nil {
			t.Fatal(err)
		}
		want := map[string]float64{"legacy-a": 78013.33, "legacy-b": 1500.51}[id]
		if total != want || mode != "LEGACY" || interest != 0 || fee != 0 || basis != "PRINCIPAL" || order != "FEES_INTEREST_PRINCIPAL" || asOf.Valid || note != "" {
			t.Fatalf("migration changed legacy account %s: %.2f %s %.2f %.2f %s %s %v %s", id, total, mode, interest, fee, basis, order, asOf, note)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("migration lost accounts: %d", count)
	}
	var principal, interest, fee sql.NullFloat64
	var source sql.NullString
	if err := db.QueryRow(`SELECT principal_amount, interest_amount, fee_amount, allocation_source FROM credit_transactions WHERE id = 'legacy-payment'`).Scan(&principal, &interest, &fee, &source); err != nil {
		t.Fatal(err)
	}
	if principal.Valid || interest.Valid || fee.Valid || source.Valid {
		t.Fatal("migration invented historical allocations")
	}
	var audits int
	if err := db.QueryRow(`SELECT count(*) FROM credit_balance_reconciliations`).Scan(&audits); err != nil || audits != 0 {
		t.Fatalf("migration invented baselines: %d %v", audits, err)
	}
}
