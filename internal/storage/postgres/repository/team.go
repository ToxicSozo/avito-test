package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ToxicSozo/avito-test/internal/model"
)

func (s *Store) EnsureTeam(ctx context.Context, name string) error {
	return s.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&Team{Name: name}).Error
}

func (s *Store) UpsertMember(ctx context.Context, member model.User) error {
	db := s.WithContext(ctx)

	var existing User
	err := db.Where("user_id = ?", member.ID).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return db.Create(userFromModel(member)).Error
	case err != nil:
		return err
	default:
		if existing.TeamName != member.TeamName {
			return ErrUserTeamMismatch
		}
		existing.Username = member.Username
		existing.IsActive = member.IsActive
		return db.Save(&existing).Error
	}
}

func (s *Store) GetTeam(ctx context.Context, name string) (*model.Team, error) {
	var team Team
	err := s.WithContext(ctx).
		Preload("Members", func(db *gorm.DB) *gorm.DB {
			return db.Order("username ASC")
		}).
		First(&team, "team_name = ?", name).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return team.toModel(), nil
}
