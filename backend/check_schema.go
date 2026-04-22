package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	dsn := "file:income_expense.db"
	if len(os.Args) > 1 {
		dsn = os.Args[1]
	}

	db, err := sql.Open("libsql", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("PRAGMA table_info(transactions)")
	if err != nil {
		log.Fatalf("failed to query table info: %v", err)
	}
	defer rows.Close()

	fmt.Println("Columns in transactions table:")
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dflt_value sql.NullString
		err = rows.Scan(&cid, &name, &dtype, &notnull, &dflt_value, &pk)
		if err != nil {
			log.Fatalf("failed to scan row: %v", err)
		}
		fmt.Printf("- %s (%s)\n", name, dtype)
	}
}
