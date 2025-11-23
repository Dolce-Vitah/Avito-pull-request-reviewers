package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"pr-reviewer/internal/model"
)

func (s *Store) CreatePullRequest(ctx context.Context, pr *model.PullRequest) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var teamName string
	err = tx.QueryRowContext(ctx, "SELECT team_name FROM users WHERE id = $1", pr.AuthorID).Scan(&teamName)
	if err == sql.ErrNoRows {
		return ErrNotFound 
	}
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, 
		"INSERT INTO pull_requests (id, name, author_id, status) VALUES ($1, $2, $3, 'OPEN')",
		pr.ID, pr.Name, pr.AuthorID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrPRExists
		}
		return err
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM users 
		WHERE team_name = $1 AND is_active = true AND id != $2
		ORDER BY RANDOM() 
		LIMIT 2
	`, teamName, pr.AuthorID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var rid string
		if err := rows.Scan(&rid); err != nil {
			return err
		}
		reviewers = append(reviewers, rid)
	}
	rows.Close()

	for _, rid := range reviewers {
		_, err := tx.ExecContext(ctx, "INSERT INTO reviewers (pull_request_id, user_id) VALUES ($1, $2)", pr.ID, rid)
		if err != nil {
			return err
		}
	}

	pr.Status = "OPEN"
	pr.AssignedReviewers = reviewers
	return tx.Commit()
}

func (s *Store) MergePullRequest(ctx context.Context, prID string) (*model.PullRequest, error) {
	query := `
		UPDATE pull_requests 
		SET status = 'MERGED', merged_at = COALESCE(merged_at, CURRENT_TIMESTAMP)
		WHERE id = $1
		RETURNING id, name, author_id, status, merged_at
	`
	var pr model.PullRequest
	err := s.db.QueryRowContext(ctx, query, prID).Scan(
		&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.MergedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	reviewers, err := s.getReviewers(ctx, prID)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (s *Store) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*model.PullRequest, string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()

	var status string
	var authorID string
	err = tx.QueryRowContext(ctx, "SELECT status, author_id FROM pull_requests WHERE id = $1", prID).Scan(&status, &authorID)
	if err == sql.ErrNoRows {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}

	if status == "MERGED" {
		return nil, "", ErrPRMerged
	}

	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM reviewers WHERE pull_request_id = $1 AND user_id = $2)", prID, oldUserID).Scan(&exists)
	if err != nil {
		return nil, "", err
	}
	if !exists {
		return nil, "", ErrNotAssigned
	}

	var teamName string
	err = tx.QueryRowContext(ctx, "SELECT team_name FROM users WHERE id = $1", oldUserID).Scan(&teamName)
	if err == sql.ErrNoRows {
		return nil, "", ErrNotFound
	}

	candidateQuery := `
		SELECT u.id 
		FROM users u
		WHERE u.team_name = $1 
		  AND u.is_active = true 
		  AND u.id != $2
		  AND u.id != $3
		  AND u.id NOT IN (SELECT user_id FROM reviewers WHERE pull_request_id = $4)
		ORDER BY RANDOM()
		LIMIT 1
	`
	var newUserID string
	err = tx.QueryRowContext(ctx, candidateQuery, teamName, oldUserID, authorID, prID).Scan(&newUserID)
	if err == sql.ErrNoRows {
		return nil, "", ErrNoCandidate
	}
	if err != nil {
		return nil, "", err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM reviewers WHERE pull_request_id = $1 AND user_id = $2", prID, oldUserID)
	if err != nil {
		return nil, "", err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO reviewers (pull_request_id, user_id) VALUES ($1, $2)", prID, newUserID)
	if err != nil {
		return nil, "", err
	}

	if err := tx.Commit(); err != nil {
		return nil, "", err
	}

	pr, err := s.getPullRequest(ctx, prID)
	if err != nil {
		return nil, "", err
	}

	return pr, newUserID, nil
}

func (s *Store) GetReviewsForUser(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	query := `
		SELECT p.id, p.name, p.author_id, p.status
		FROM pull_requests p
		JOIN reviewers r ON p.id = r.pull_request_id
		WHERE r.user_id = $1
		ORDER BY p.created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []model.PullRequestShort
	for rows.Next() {
		var pr model.PullRequestShort
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}
	if prs == nil {
		prs = []model.PullRequestShort{}
	}
	return prs, nil
}

func (s *Store) getPullRequest(ctx context.Context, prID string) (*model.PullRequest, error) {
	var pr model.PullRequest
	query := "SELECT id, name, author_id, status FROM pull_requests WHERE id = $1"
	err := s.db.QueryRowContext(ctx, query, prID).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status)
	if err != nil {
		return nil, err
	}
	
	reviewers, err := s.getReviewers(ctx, prID)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers
	return &pr, nil
}

func (s *Store) getReviewers(ctx context.Context, prID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT user_id FROM reviewers WHERE pull_request_id = $1", prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revs []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		revs = append(revs, r)
	}
	if revs == nil {
		revs = []string{}
	}
	return revs, nil
}