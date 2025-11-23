package tests

import (
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"pr-reviewer/internal/store"
)

func SetupTestDB(t *testing.T) *store.Store {
	dsn := "postgres://postgres:postgres@localhost:5432/pr_reviewer?sslmode=disable"
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}

	tables := []string{"reviewers", "pull_requests", "users", "teams"}
	for _, table := range tables {
		_, err := db.Exec("TRUNCATE TABLE " + table + " CASCADE")
		if err != nil {
			t.Fatalf("Failed to truncate table %s: %v", table, err)
		}
	}

	return store.New(db)
}