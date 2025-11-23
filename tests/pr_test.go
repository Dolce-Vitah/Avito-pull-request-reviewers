package tests

import (
	"context"
	"testing"

	"pr-reviewer/internal/model"
	"pr-reviewer/internal/store"
)

func TestAutoAssignmentLogic(t *testing.T) {
	s := SetupTestDB(t)
	ctx := context.Background()

	s.CreateTeam(ctx, &model.Team{
		TeamName: "backend",
		Members: []model.TeamMember{
			{UserID: "author", Username: "Author", IsActive: true},
			{UserID: "r1", Username: "Rev1", IsActive: true},
			{UserID: "r2", Username: "Rev2", IsActive: true},
			{UserID: "inactive", Username: "Ghost", IsActive: false},
		},
	})

	pr := &model.PullRequest{
		ID:       "pr-1",
		Name:     "Fix stuff",
		AuthorID: "author",
	}

	err := s.CreatePullRequest(ctx, pr)
	if err != nil {
		t.Fatalf("CreatePullRequest failed: %v", err)
	}

	if len(pr.AssignedReviewers) != 2 {
		t.Errorf("Expected 2 reviewers, got %d", len(pr.AssignedReviewers))
	}

	for _, r := range pr.AssignedReviewers {
		if r == "author" {
			t.Error("Author was assigned as reviewer")
		}
		if r == "inactive" {
			t.Error("Inactive user was assigned as reviewer")
		}
	}
}

func TestReassignment(t *testing.T) {
	s := SetupTestDB(t)
	ctx := context.Background()

	s.CreateTeam(ctx, &model.Team{
		TeamName: "frontend",
		Members: []model.TeamMember{
			{UserID: "u1", Username: "Author", IsActive: true},
			{UserID: "u2", Username: "Reviewer", IsActive: true},
			{UserID: "u3", Username: "Reviewer2", IsActive: true},
			{UserID: "u4", Username: "Backup", IsActive: false},
		},
	})

	err := s.CreatePullRequest(ctx, &model.PullRequest{ID: "pr-1", Name: "UI Fix", AuthorID: "u1"})
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}

	_, _, err = s.ReassignReviewer(ctx, "pr-1", "u2")
	if err != store.ErrNoCandidate {
		t.Errorf("Expected ErrNoCandidate, got %v", err)
	}

	_, err = s.SetUserActive(ctx, "u4", true)
	if err != nil {
		t.Fatalf("Failed to activate user: %v", err)
	}

	_, newID, err := s.ReassignReviewer(ctx, "pr-1", "u2")
	if err != nil {
		t.Fatalf("Reassignment failed: %v", err)
	}
	if newID != "u4" {
		t.Errorf("Expected new reviewer to be u4, got %s", newID)
	}
}