package store

import (
	"database/sql"
	"errors"
)

var (
	ErrTeamExists    = errors.New("team_name already exists")
	ErrNotFound      = errors.New("resource not found")
	ErrPRExists      = errors.New("pull_request_id already exists")
	ErrPRMerged      = errors.New("cannot reassign on merged PR")
	ErrNotAssigned   = errors.New("reviewer is not assigned to this PR")
	ErrNoCandidate   = errors.New("no active replacement candidate in team")
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}