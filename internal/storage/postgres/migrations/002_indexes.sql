CREATE INDEX IF NOT EXISTS idx_users_team ON users(team_name);

CREATE INDEX IF NOT EXISTS idx_pull_requests_author ON pull_requests(author_id);

CREATE INDEX IF NOT EXISTS idx_pr_reviewers_user ON pull_request_reviewers(reviewer_id);
