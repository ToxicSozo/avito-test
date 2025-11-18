package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ToxicSozo/avito-test/internal/model"
)

func (s *Store) SetUserActivity(ctx context.Context, userID string, active bool) (*model.User, error) {
	var user model.User
	err := s.q.QueryRow(ctx, `
		UPDATE users
		SET is_active = $2
		WHERE user_id = $1
		RETURNING user_id, username, team_name, is_active
	`, userID, active).Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *Store) EnsureUserExists(ctx context.Context, userID string) error {
	var id string
	err := s.q.QueryRow(ctx, `
		SELECT user_id
		FROM users
		WHERE user_id = $1
	`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *Store) FindTeamName(ctx context.Context, userID string) (string, error) {
	var teamName string
	err := s.q.QueryRow(ctx, `
		SELECT team_name
		FROM users
		WHERE user_id = $1
	`, userID).Scan(&teamName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return teamName, nil
}

func (s *Store) PickReviewers(ctx context.Context, teamName, authorID string, limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}

	rows, err := s.q.Query(ctx, `
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
	return reviewers, rows.Err()
}

func (s *Store) PickReplacement(ctx context.Context, teamName, oldReviewerID, prID, authorID string) (string, error) {
	var candidate string
	err := s.q.QueryRow(ctx, `
		SELECT user_id
		FROM users
		WHERE team_name = $1
		  AND is_active = TRUE
		  AND user_id <> $2
		  AND user_id <> $4
		  AND user_id NOT IN (
			  SELECT reviewer_id FROM pull_request_reviewers WHERE pull_request_id = $3
		  )
		ORDER BY random()
		LIMIT 1
	`, teamName, oldReviewerID, prID, authorID).Scan(&candidate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return candidate, nil
}
