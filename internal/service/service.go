package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ToxicSozo/avito-test/internal/model"
)

const (
	statusOpen   = "OPEN"
	statusMerged = "MERGED"
)

// Service wraps all business logic around PR reviewer assignment.
type Service struct {
	db *pgxpool.Pool
}

// New creates a new service instance.
func New(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// CreateTeam stores a brand-new team with initial members.
func (s *Service) CreateTeam(ctx context.Context, team model.Team) (*model.Team, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `INSERT INTO teams (team_name) VALUES ($1)`, team.Name); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrTeamExists
		}
		return nil, err
	}

	for _, member := range team.Members {
		if member.TeamName == "" {
			member.TeamName = team.Name
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (user_id, username, team_name, is_active)
			VALUES ($1, $2, $3, $4)
		`, member.ID, member.Username, member.TeamName, member.IsActive); err != nil {
			if isUniqueViolation(err) {
				return nil, ErrUserExists
			}
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetTeam(ctx, team.Name)
}

// GetTeam returns team with all members by name.
func (s *Service) GetTeam(ctx context.Context, teamName string) (*model.Team, error) {
	var name string
	if err := s.db.QueryRow(ctx, `SELECT team_name FROM teams WHERE team_name = $1`, teamName).Scan(&name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_name = $1
		ORDER BY username
	`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.IsActive); err != nil {
			return nil, err
		}
		u.TeamName = name
		members = append(members, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.Team{
		Name:    name,
		Members: members,
	}, nil
}

// SetUserActivity toggles user activity flag.
func (s *Service) SetUserActivity(ctx context.Context, userID string, isActive bool) (*model.User, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE users
		SET is_active = $2
		WHERE user_id = $1
		RETURNING user_id, username, team_name, is_active
	`, userID, isActive)

	var user model.User
	if err := row.Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

// CreatePullRequest persists PR and assigns up to two reviewers from author's team.
func (s *Service) CreatePullRequest(ctx context.Context, prID, name, authorID string) (*model.PullRequest, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var teamName string
	if err := tx.QueryRow(ctx, `
		SELECT team_name
		FROM users
		WHERE user_id = $1
	`, authorID).Scan(&teamName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	reviewers, err := s.pickReviewers(ctx, tx, teamName, authorID, 2)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
		VALUES ($1, $2, $3, $4)
	`, prID, name, authorID, statusOpen); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPullRequestExists
		}
		return nil, err
	}

	for idx, reviewer := range reviewers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id, position)
			VALUES ($1, $2, $3)
		`, prID, reviewer, idx+1); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.getPullRequest(ctx, prID)
}

// MergePullRequest marks PR as merged (idempotent).
func (s *Service) MergePullRequest(ctx context.Context, prID string) (*model.PullRequest, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE pull_requests
		SET status = $2,
		    merged_at = COALESCE(merged_at, NOW())
		WHERE pull_request_id = $1
	`, prID, statusMerged)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return s.getPullRequest(ctx, prID)
}

// ReassignReviewer replaces reviewer with a random teammate.
func (s *Service) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*model.PullRequest, string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback(ctx)

	var status, authorID string
	if err := tx.QueryRow(ctx, `
		SELECT status, author_id
		FROM pull_requests
		WHERE pull_request_id = $1
		FOR UPDATE
	`, prID).Scan(&status, &authorID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}

	if status == statusMerged {
		return nil, "", ErrPullRequestMerged
	}

	var teamName string
	var position int
	if err := tx.QueryRow(ctx, `
		SELECT u.team_name, pr.position
		FROM pull_request_reviewers pr
		JOIN users u ON u.user_id = pr.reviewer_id
		WHERE pr.pull_request_id = $1 AND pr.reviewer_id = $2
		FOR UPDATE
	`, prID, oldReviewerID).Scan(&teamName, &position); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrReviewerMissing
		}
		return nil, "", err
	}

	newReviewer, err := s.pickReplacement(ctx, tx, teamName, oldReviewerID, prID, authorID)
	if err != nil {
		return nil, "", err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE pull_request_reviewers
		SET reviewer_id = $1
		WHERE pull_request_id = $2 AND position = $3
	`, newReviewer, prID, position); err != nil {
		return nil, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, "", err
	}

	pr, err := s.getPullRequest(ctx, prID)
	if err != nil {
		return nil, "", err
	}

	return pr, newReviewer, nil
}

// ListReviewerAssignments returns PRs assigned to user.
func (s *Service) ListReviewerAssignments(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	if err := s.ensureUserExists(ctx, userID); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_request_reviewers r
		JOIN pull_requests pr ON pr.pull_request_id = r.pull_request_id
		WHERE r.reviewer_id = $1
		ORDER BY pr.created_at DESC
	`, userID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return prs, nil
}

// AssignmentStats returns top reviewers ordered by assignment count.
func (s *Service) AssignmentStats(ctx context.Context, limit int) ([]model.AssignmentStat, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(ctx, `
		SELECT u.user_id, u.username, u.team_name, COALESCE(COUNT(r.pull_request_id), 0) AS assignments
		FROM users u
		LEFT JOIN pull_request_reviewers r ON r.reviewer_id = u.user_id
		GROUP BY u.user_id, u.username, u.team_name
		ORDER BY assignments DESC, u.username ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []model.AssignmentStat
	for rows.Next() {
		var stat model.AssignmentStat
		if err := rows.Scan(&stat.UserID, &stat.Username, &stat.TeamName, &stat.Assignments); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *Service) ensureUserExists(ctx context.Context, userID string) error {
	var id string
	if err := s.db.QueryRow(ctx, `SELECT user_id FROM users WHERE user_id = $1`, userID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) pickReviewers(ctx context.Context, q pgx.Tx, teamName, authorID string, limit int) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT user_id
		FROM users
		WHERE team_name = $1
		  AND is_active = TRUE
		  AND user_id <> $2
		ORDER BY random()
		LIMIT $3
	`, teamName, authorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reviewers, nil
}

func (s *Service) pickReplacement(ctx context.Context, q pgx.Tx, teamName, oldReviewerID, prID, authorID string) (string, error) {
	row := q.QueryRow(ctx, `
		SELECT user_id
		FROM users
		WHERE team_name = $1
		  AND is_active = TRUE
		  AND user_id <> $2
		  AND user_id <> $3
		  AND user_id NOT IN (
			SELECT reviewer_id FROM pull_request_reviewers WHERE pull_request_id = $4
		  )
		ORDER BY random()
		LIMIT 1
	`, teamName, oldReviewerID, authorID, prID)

	var userID string
	if err := row.Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNoCandidate
		}
		return "", err
	}
	return userID, nil
}

func (s *Service) getPullRequest(ctx context.Context, prID string) (*model.PullRequest, error) {
	row := s.db.QueryRow(ctx, `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`, prID)

	var pr model.PullRequest
	if err := row.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	reviewers, err := s.loadReviewers(ctx, prID)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers
	return &pr, nil
}

func (s *Service) loadReviewers(ctx context.Context, prID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT reviewer_id
		FROM pull_request_reviewers
		WHERE pull_request_id = $1
		ORDER BY position
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reviewers, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
