CREATE TABLE IF NOT EXISTS teams (
    name VARCHAR(255) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL REFERENCES teams(name),
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS pull_requests (
    id VARCHAR(255) PRIMARY KEY,
    name TEXT NOT NULL,
    author_id VARCHAR(255) NOT NULL REFERENCES users(id),
    status VARCHAR(50) NOT NULL DEFAULT 'OPEN', -- OPEN, MERGED
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    merged_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS reviewers (
    pull_request_id VARCHAR(255) REFERENCES pull_requests(id),
    user_id VARCHAR(255) REFERENCES users(id),
    PRIMARY KEY (pull_request_id, user_id)
);

-- Index for quick lookup of reviewers by PR
CREATE INDEX idx_reviewers_pr_id ON reviewers(pull_request_id);
-- Index for looking up active users in a team for random assignment
CREATE INDEX idx_users_team_active ON users(team_name, is_active);