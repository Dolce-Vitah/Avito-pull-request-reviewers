package store

import (
	"context"
)

type Stats struct {
	TotalTeams      int            `json:"total_teams"`
	TotalUsers      int            `json:"total_users"`
	ActiveUsers     int            `json:"active_users"`
	TotalPRs        int            `json:"total_prs"`
	OpenPRs         int            `json:"open_prs"`
	BusiestReviewer string         `json:"busiest_reviewer"`
	ReviewerCounts  map[string]int `json:"reviewer_counts"`
}

func (s *Store) GetSystemStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{
		ReviewerCounts: make(map[string]int),
	}

	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM teams").Scan(&stats.TotalTeams)
	if err != nil { return nil, err }

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active) FROM users").
		Scan(&stats.TotalUsers, &stats.ActiveUsers)
	if err != nil { return nil, err }

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'OPEN') FROM pull_requests").
		Scan(&stats.TotalPRs, &stats.OpenPRs)
	if err != nil { return nil, err }

	rows, err := s.db.QueryContext(ctx, `
		SELECT u.username, COUNT(r.pull_request_id) as cnt
		FROM reviewers r
		JOIN users u ON r.user_id = u.id
		GROUP BY u.username
		ORDER BY cnt DESC
		LIMIT 5
	`)
	if err != nil { return nil, err }
	defer rows.Close()

	first := true
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil { return nil, err }
		
		stats.ReviewerCounts[name] = count
		if first {
			stats.BusiestReviewer = name
			first = false
		}
	}

	return stats, nil
}