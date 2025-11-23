package store

import (
	"context"
	"database/sql"
	"pr-reviewer/internal/model"
)

func (s *Store) SetUserActive(ctx context.Context, userID string, isActive bool) (*model.User, error) {
	query := `
		UPDATE users 
		SET is_active = $1 
		WHERE id = $2 
		RETURNING id, username, team_name, is_active
	`
	var u model.User
	err := s.db.QueryRowContext(ctx, query, isActive, userID).Scan(
		&u.ID, &u.Username, &u.TeamName, &u.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}