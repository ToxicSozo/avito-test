package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ToxicSozo/avito-test/internal/model"
)

func (s *Store) CreatePullRequest(ctx context.Context, prID, name, authorID, status string) error {
	pr := &PullRequest{
		ID:       prID,
		Name:     name,
		AuthorID: authorID,
		Status:   status,
	}
	return s.WithContext(ctx).Create(pr).Error
}

func (s *Store) AddReviewers(ctx context.Context, prID string, reviewers []string) error {
	if len(reviewers) == 0 {
		return nil
	}
	records := make([]PullRequestReviewer, 0, len(reviewers))
	for idx, reviewer := range reviewers {
		records = append(records, PullRequestReviewer{
			PullRequestID: prID,
			ReviewerID:    reviewer,
			Position:      idx + 1,
		})
	}
	return s.WithContext(ctx).Create(&records).Error
}

func (s *Store) GetPullRequest(ctx context.Context, prID string) (*model.PullRequest, error) {
	var pr PullRequest
	err := s.WithContext(ctx).
		Preload("Reviewers", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		First(&pr, "pull_request_id = ?", prID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return pr.toModel(), nil
}

func (s *Store) MarkMerged(ctx context.Context, prID, status string) (int64, error) {
	res := s.WithContext(ctx).Model(&PullRequest{}).
		Where("pull_request_id = ?", prID).
		Updates(map[string]any{
			"status":    status,
			"merged_at": gorm.Expr("COALESCE(merged_at, NOW())"),
		})
	return res.RowsAffected, res.Error
}

func (s *Store) LockPullRequest(ctx context.Context, prID string) (*PullRequest, error) {
	var pr PullRequest
	err := s.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&pr, "pull_request_id = ?", prID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &pr, nil
}

func (s *Store) LockReviewer(ctx context.Context, prID, reviewerID string) (*PullRequestReviewer, error) {
	var rec PullRequestReviewer
	err := s.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&rec, "pull_request_id = ? AND reviewer_id = ?", prID, reviewerID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}

func (s *Store) UpdateReviewer(ctx context.Context, prID string, position int, newReviewer string) error {
	return s.WithContext(ctx).Model(&PullRequestReviewer{}).
		Where("pull_request_id = ? AND position = ?", prID, position).
		Update("reviewer_id", newReviewer).Error
}

func (s *Store) ListAssignments(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	rows, err := s.WithContext(ctx).
		Table("pull_request_reviewers AS r").
		Select("pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status").
		Joins("JOIN pull_requests pr ON pr.pull_request_id = r.pull_request_id").
		Where("r.reviewer_id = ?", userID).
		Order("pr.created_at DESC").
		Rows()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

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

func (s *Store) AssignmentStats(ctx context.Context, limit int) ([]model.AssignmentStat, error) {
	rows, err := s.WithContext(ctx).
		Table("users AS u").
		Select("u.user_id, u.username, u.team_name, COALESCE(COUNT(r.pull_request_id), 0) AS assignments").
		Joins("LEFT JOIN pull_request_reviewers r ON r.reviewer_id = u.user_id").
		Group("u.user_id, u.username, u.team_name").
		Order("assignments DESC, u.username ASC").
		Limit(limit).
		Rows()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var stats []model.AssignmentStat
	for rows.Next() {
		var stat model.AssignmentStat
		if err := rows.Scan(&stat.UserID, &stat.Username, &stat.TeamName, &stat.Assignments); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}
