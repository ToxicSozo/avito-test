package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ToxicSozo/avito-test/internal/model"
)

const (
	statusOpenID   int16 = 1
	statusMergedID int16 = 2
)

var statusNameToID = map[string]int16{
	"OPEN":   statusOpenID,
	"MERGED": statusMergedID,
}

func statusIDToName(id int16) (string, error) {
	for name, mapped := range statusNameToID {
		if mapped == id {
			return name, nil
		}
	}
	return "", fmt.Errorf("unknown status id: %d", id)
}

func (s *Store) CreatePullRequest(ctx context.Context, prID, name, authorID, status string) error {
	statusID, ok := statusNameToID[status]
	if !ok {
		return fmt.Errorf("unknown status: %s", status)
	}
	_, err := s.q.Exec(ctx, `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status_id)
		VALUES ($1, $2, $3, $4)
	`, prID, name, authorID, statusID)
	return err
}

func (s *Store) AddReviewers(ctx context.Context, prID string, reviewers []string) error {
	if len(reviewers) == 0 {
		return nil
	}
	for idx, reviewer := range reviewers {
		if _, err := s.q.Exec(ctx, `
			INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id, position)
			VALUES ($1, $2, $3)
		`, prID, reviewer, idx+1); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetPullRequest(ctx context.Context, prID string) (*model.PullRequest, error) {
	var pr model.PullRequest
	var createdAt time.Time
	var mergedAt *time.Time
	var statusID int16
	err := s.q.QueryRow(ctx, `
		SELECT pr.pull_request_id,
		       pr.pull_request_name,
		       pr.author_id,
		       pr.status_id,
		       pr.created_at,
		       pr.merged_at
		FROM pull_requests pr
		WHERE pr.pull_request_id = $1
	`, prID).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &statusID, &createdAt, &mergedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	statusName, err := statusIDToName(statusID)
	if err != nil {
		return nil, err
	}
	pr.Status = statusName
	pr.CreatedAt = createdAt
	pr.MergedAt = mergedAt

	rows, err := s.q.Query(ctx, `
		SELECT reviewer_id
		FROM pull_request_reviewers
		WHERE pull_request_id = $1
		ORDER BY position ASC
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var reviewer string
		if err := rows.Scan(&reviewer); err != nil {
			return nil, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &pr, nil
}

func (s *Store) MarkMerged(ctx context.Context, prID, status string) (int64, error) {
	statusID, ok := statusNameToID[status]
	if !ok {
		return 0, fmt.Errorf("unknown status: %s", status)
	}
	tag, err := s.q.Exec(ctx, `
		UPDATE pull_requests
		SET status_id = $2,
		    merged_at = COALESCE(merged_at, NOW())
		WHERE pull_request_id = $1
	`, prID, statusID)
	return tag.RowsAffected(), err
}

type lockedPullRequest struct {
	ID       string
	AuthorID string
	Status   string
}

func (s *Store) LockPullRequest(ctx context.Context, prID string) (*lockedPullRequest, error) {
	var pr lockedPullRequest
	var statusID int16
	err := s.q.QueryRow(ctx, `
		SELECT pull_request_id, author_id, status_id
		FROM pull_requests
		WHERE pull_request_id = $1
		FOR UPDATE
	`, prID).Scan(&pr.ID, &pr.AuthorID, &statusID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	statusName, err := statusIDToName(statusID)
	if err != nil {
		return nil, err
	}
	pr.Status = statusName
	return &pr, nil
}

type reviewerRecord struct {
	Position int
}

func (s *Store) LockReviewer(ctx context.Context, prID, reviewerID string) (*reviewerRecord, error) {
	var rec reviewerRecord
	err := s.q.QueryRow(ctx, `
		SELECT position
		FROM pull_request_reviewers
		WHERE pull_request_id = $1
		  AND reviewer_id = $2
		FOR UPDATE
	`, prID, reviewerID).Scan(&rec.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}

func (s *Store) UpdateReviewer(ctx context.Context, prID string, position int, newReviewer string) error {
	tag, err := s.q.Exec(ctx, `
		UPDATE pull_request_reviewers
		SET reviewer_id = $3
		WHERE pull_request_id = $1
		  AND position = $2
	`, prID, position, newReviewer)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListAssignments(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	rows, err := s.q.Query(ctx, `
		SELECT pr.pull_request_id,
		       pr.pull_request_name,
		       pr.author_id,
		       st.status_name
		FROM pull_request_reviewers r
		JOIN pull_requests pr ON pr.pull_request_id = r.pull_request_id
		JOIN pull_request_statuses st ON st.status_id = pr.status_id
		WHERE r.reviewer_id = $1
		ORDER BY pr.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.PullRequestShort
	for rows.Next() {
		var pr model.PullRequestShort
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		result = append(result, pr)
	}
	return result, rows.Err()
}
