package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ToxicSozo/avito-test/internal/model"
)

func (s *Store) EnsureTeam(ctx context.Context, name string) error {
	_, err := s.q.Exec(ctx, `
		INSERT INTO teams (team_name)
		VALUES ($1)
		ON CONFLICT (team_name) DO NOTHING
	`, name)
	return err
}

func (s *Store) UpsertMember(ctx context.Context, member model.User) error {
	var existing model.User
	err := s.q.QueryRow(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id = $1
	`, member.ID).Scan(&existing.ID, &existing.Username, &existing.TeamName, &existing.IsActive)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if member.TeamName == "" {
			return ErrUserTeamMismatch
		}
		_, err = s.q.Exec(ctx, `
			INSERT INTO users (user_id, username, team_name, is_active)
			VALUES ($1, $2, $3, $4)
		`, member.ID, member.Username, member.TeamName, member.IsActive)
		return err
	case err != nil:
		return err
	default:
		if existing.TeamName != member.TeamName {
			return ErrUserTeamMismatch
		}
		_, err = s.q.Exec(ctx, `
			UPDATE users
			SET username = $2,
			    is_active = $3
			WHERE user_id = $1
		`, member.ID, member.Username, member.IsActive)
		return err
	}
}

func (s *Store) GetTeam(ctx context.Context, name string) (*model.Team, error) {
	var teamName string
	if err := s.q.QueryRow(ctx, `
		SELECT team_name
		FROM teams
		WHERE team_name = $1
	`, name).Scan(&teamName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	rows, err := s.q.Query(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name = $1
		ORDER BY username ASC
	`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.User
	for rows.Next() {
		var m model.User
		if err := rows.Scan(&m.ID, &m.Username, &m.TeamName, &m.IsActive); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.Team{
		Name:    teamName,
		Members: members,
	}, nil
}
