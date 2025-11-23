package tests

import (
	"context"
	"database/sql"
	"testing"

	"pr-reviewer/internal/model"
)

func TestCreateTeamAndUsers(t *testing.T) {
	s := SetupTestDB(t)
	ctx := context.Background()

	team := &model.Team{
		TeamName: "backend",
		Members: []model.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}

	err := s.CreateTeam(ctx, team)
	if err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}

	fetched, err := s.GetTeam(ctx, "backend")
	if err != nil {
		t.Fatalf("GetTeam failed: %v", err)
	}

	if len(fetched.Members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(fetched.Members))
	}
}

func TestBulkDeactivation(t *testing.T) {
	s := SetupTestDB(t)
	ctx := context.Background()

	s.CreateTeam(ctx, &model.Team{
		TeamName: "devops",
		Members: []model.TeamMember{
			{UserID: "u1", Username: "Author", IsActive: true},
			{UserID: "u2", Username: "Leaver1", IsActive: true},
			{UserID: "u3", Username: "Leaver2", IsActive: true},
			{UserID: "u4", Username: "Stay1", IsActive: true},
			{UserID: "u5", Username: "Stay2", IsActive: true},
		},
	})

	dsn := "postgres://postgres:postgres@localhost:5432/pr_reviewer?sslmode=disable"
	setupDB, _ := sql.Open("pgx", dsn)
	defer setupDB.Close()

	_, err := setupDB.Exec("INSERT INTO pull_requests (id, name, author_id, status) VALUES ('pr-1', 'Fix1', 'u1', 'OPEN')")
	if err != nil { t.Fatal(err) }
	setupDB.Exec("INSERT INTO reviewers (pull_request_id, user_id) VALUES ('pr-1', 'u2')")
	setupDB.Exec("INSERT INTO reviewers (pull_request_id, user_id) VALUES ('pr-1', 'u3')")

	_, err = setupDB.Exec("INSERT INTO pull_requests (id, name, author_id, status) VALUES ('pr-2', 'Fix2', 'u1', 'OPEN')")
	if err != nil { t.Fatal(err) }
	setupDB.Exec("INSERT INTO reviewers (pull_request_id, user_id) VALUES ('pr-2', 'u2')")
	setupDB.Exec("INSERT INTO reviewers (pull_request_id, user_id) VALUES ('pr-2', 'u4')")

	toDeactivate := []string{"u2", "u3"}
	result, err := s.BulkDeactivateAndReassign(ctx, toDeactivate)
	if err != nil {
		t.Fatalf("BulkDeactivate failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 PRs in result map, got %d", len(result))
	}
	if len(result["pr-1"]) != 2 {
		t.Errorf("Expected 2 replacements for pr-1, got %d", len(result["pr-1"]))
	}
	if len(result["pr-2"]) != 1 {
		t.Errorf("Expected 1 replacement for pr-2, got %d", len(result["pr-2"]))
	}

	var count int
	setupDB.QueryRow("SELECT count(*) FROM users WHERE is_active = false AND id IN ('u2', 'u3')").Scan(&count)
	if count != 2 {
		t.Error("Users u2 and u3 should be inactive")
	}

	rows, _ := setupDB.Query("SELECT user_id FROM reviewers WHERE pull_request_id = 'pr-1'")
	var revs []string
	for rows.Next() {
		var r string
		rows.Scan(&r)
		revs = append(revs, r)
	}
	rows.Close()
	
	hasU4 := false
	hasU5 := false
	for _, r := range revs {
		if r == "u4" { hasU4 = true }
		if r == "u5" { hasU5 = true }
		if r == "u2" || r == "u3" { t.Error("Leavers still assigned to PR-1") }
	}
	if !hasU4 || !hasU5 {
		t.Errorf("PR-1 should be assigned to u4 and u5. Got: %v", revs)
	}
}