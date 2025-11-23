package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"pr-reviewer/internal/model"
)

func (s *Store) CreateTeam(ctx context.Context, team *model.Team) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "INSERT INTO teams (name) VALUES ($1)", team.TeamName)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { 
			return ErrTeamExists
		}
		return err
	}

	query := `
		INSERT INTO users (id, username, team_name, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			team_name = EXCLUDED.team_name,
			is_active = EXCLUDED.is_active
	`
	for _, m := range team.Members {
		_, err := tx.ExecContext(ctx, query, m.UserID, m.Username, team.TeamName, m.IsActive)
		if err != nil {
			return fmt.Errorf("failed to upsert user %s: %w", m.UserID, err)
		}
	}

	return tx.Commit()
}

func (s *Store) GetTeam(ctx context.Context, teamName string) (*model.Team, error) {
	var name string
	err := s.db.QueryRowContext(ctx, "SELECT name FROM teams WHERE name = $1", teamName).Scan(&name)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, "SELECT id, username, is_active FROM users WHERE team_name = $1", teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.TeamMember
	for rows.Next() {
		var m model.TeamMember
		if err := rows.Scan(&m.UserID, &m.Username, &m.IsActive); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return &model.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}

func (s *Store) BulkDeactivateAndReassign(ctx context.Context, userIDs []string) (map[string][]string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	queryDeactivate := `
		UPDATE users 
		SET is_active = false 
		WHERE id = ANY($1)
		RETURNING id, team_name
	`
	rows, err := tx.QueryContext(ctx, queryDeactivate, userIDs)
	if err != nil {
		return nil, err
	}
	rows.Close() 

	queryFindAssignments := `
		SELECT r.pull_request_id, r.user_id, p.author_id, u.team_name
		FROM reviewers r
		JOIN pull_requests p ON r.pull_request_id = p.id
		JOIN users u ON r.user_id = u.id
		WHERE r.user_id = ANY($1) AND p.status = 'OPEN'
	`
	rows, err = tx.QueryContext(ctx, queryFindAssignments, userIDs)
	if err != nil {
		return nil, err
	}

	type Assignment struct {
		PrID     string
		OldUser  string
		AuthorID string
		TeamName string
	}
	var tasks []Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.PrID, &a.OldUser, &a.AuthorID, &a.TeamName); err != nil {
			return nil, err
		}
		tasks = append(tasks, a)
	}
	rows.Close()

	reassignments := make(map[string][]string)

	stmtCandidate, err := tx.PrepareContext(ctx, `
		SELECT id FROM users 
		WHERE team_name = $1 
		  AND is_active = true 
		  AND id != $2 
		  AND id NOT IN (SELECT user_id FROM reviewers WHERE pull_request_id = $3)
		ORDER BY RANDOM() LIMIT 1
	`)
	if err != nil {
		return nil, err
	}
	defer stmtCandidate.Close()

	stmtSwap, err := tx.PrepareContext(ctx, `
		UPDATE reviewers SET user_id = $1 
		WHERE pull_request_id = $2 AND user_id = $3
	`)
	if err != nil {
		return nil, err
	}
	defer stmtSwap.Close()

	for _, task := range tasks {
		var newReviewerID string
		err := stmtCandidate.QueryRowContext(ctx, task.TeamName, task.AuthorID, task.PrID).Scan(&newReviewerID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue 
			}
			return nil, err
		}

		_, err = stmtSwap.ExecContext(ctx, newReviewerID, task.PrID, task.OldUser)
		if err != nil {
			return nil, err
		}
		reassignments[task.PrID] = append(reassignments[task.PrID], newReviewerID)
	}

	return reassignments, tx.Commit()
}