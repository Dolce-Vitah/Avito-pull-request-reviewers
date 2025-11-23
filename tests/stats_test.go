package tests

import (
	"context"
	"database/sql"
	"testing"

	"pr-reviewer/internal/model"
)

func TestStatsEndpoint(t *testing.T) {
	s := SetupTestDB(t)
	ctx := context.Background()

	s.CreateTeam(ctx, &model.Team{
		TeamName: "Alpha",
		Members: []model.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
			{UserID: "u3", Username: "Charlie", IsActive: true},
		},
	})

	s.CreateTeam(ctx, &model.Team{
		TeamName: "Beta",
		Members: []model.TeamMember{
			{UserID: "u4", Username: "Dave", IsActive: true},
			{UserID: "u5", Username: "Eve", IsActive: false},
		},
	})

	dsn := "postgres://postgres:postgres@localhost:5432/pr_reviewer?sslmode=disable"
	setupDB, _ := sql.Open("pgx", dsn)
	defer setupDB.Close()

	setupDB.Exec("INSERT INTO pull_requests (id, name, author_id, status) VALUES ('pr-1', 'Code1', 'u1', 'OPEN')")
	setupDB.Exec("INSERT INTO reviewers (pull_request_id, user_id) VALUES ('pr-1', 'u2')")
	setupDB.Exec("INSERT INTO reviewers (pull_request_id, user_id) VALUES ('pr-1', 'u3')")

	setupDB.Exec("INSERT INTO pull_requests (id, name, author_id, status) VALUES ('pr-2', 'Code2', 'u4', 'MERGED')")
	setupDB.Exec("INSERT INTO reviewers (pull_request_id, user_id) VALUES ('pr-2', 'u2')")

	setupDB.Exec("INSERT INTO pull_requests (id, name, author_id, status) VALUES ('pr-3', 'Code3', 'u2', 'OPEN')")
	setupDB.Exec("INSERT INTO reviewers (pull_request_id, user_id) VALUES ('pr-3', 'u3')")

	stats, err := s.GetSystemStats(ctx)
	if err != nil {
		t.Fatalf("GetSystemStats failed: %v", err)
	}

	if stats.TotalTeams != 2 { t.Errorf("Expected 2 teams, got %d", stats.TotalTeams) }
	if stats.TotalUsers != 5 { t.Errorf("Expected 5 total users, got %d", stats.TotalUsers) }
	if stats.ActiveUsers != 4 { t.Errorf("Expected 4 active users, got %d", stats.ActiveUsers) }
	if stats.TotalPRs != 3 { t.Errorf("Expected 3 total PRs, got %d", stats.TotalPRs) }
	if stats.OpenPRs != 2 { t.Errorf("Expected 2 open PRs, got %d", stats.OpenPRs) }

	if stats.ReviewerCounts["Bob"] != 2 { t.Errorf("Expected Bob to have 2 reviews") }
	if stats.ReviewerCounts["Charlie"] != 2 { t.Errorf("Expected Charlie to have 2 reviews") }
}