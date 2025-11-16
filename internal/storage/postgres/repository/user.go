package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ToxicSozo/avito-test/internal/model"
)

func (s *Store) SetUserActivity(ctx context.Context, userID string, active bool) (*model.User, error) {
	db := s.WithContext(ctx)

	var user User
	if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	user.IsActive = active
	if err := db.Save(&user).Error; err != nil {
		return nil, err
	}
	m := user.toModel()
	return &m, nil
}

func (s *Store) EnsureUserExists(ctx context.Context, userID string) error {
	err := s.WithContext(ctx).First(&User{}, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *Store) FindTeamName(ctx context.Context, userID string) (string, error) {
	var user User
	err := s.WithContext(ctx).Select("team_name").First(&user, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	return user.TeamName, nil
}

func (s *Store) PickReviewers(ctx context.Context, teamName, authorID string, limit int) ([]string, error) {
	rows, err := s.WithContext(ctx).
		Model(&User{}).
		Select("user_id").
		Where("team_name = ? AND is_active = ? AND user_id <> ?", teamName, true, authorID).
		Order("random()").
		Limit(limit).
		Rows()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

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
	rows, err := s.WithContext(ctx).
		Model(&User{}).
		Select("user_id").
		Where("team_name = ? AND is_active = ?", teamName, true).
		Where("user_id NOT IN ?", []string{oldReviewerID, authorID}).
		Where("user_id NOT IN (?)",
			s.WithContext(ctx).Model(&PullRequestReviewer{}).
				Select("reviewer_id").
				Where("pull_request_id = ?", prID),
		).
		Order("random()").
		Limit(1).
		Rows()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = rows.Close()
	}()

	if rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		return id, nil
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return "", ErrNotFound
}
