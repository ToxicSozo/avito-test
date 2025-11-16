package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/ToxicSozo/avito-test/internal/model"
	"github.com/ToxicSozo/avito-test/internal/storage/postgres/repository"
)

const (
	statusOpen   = "OPEN"
	statusMerged = "MERGED"
)

type Service struct {
	repo *repository.Store
}

func New(db *gorm.DB) *Service {
	return &Service{
		repo: repository.New(db),
	}
}

func (s *Service) CreateTeam(ctx context.Context, team model.Team) (*model.Team, error) {
	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := repository.New(tx)

		if err := txRepo.EnsureTeam(ctx, team.Name); err != nil {
			return err
		}

		for _, member := range team.Members {
			if member.TeamName == "" {
				member.TeamName = team.Name
			}
			if err := txRepo.UpsertMember(ctx, member); err != nil {
				if errors.Is(err, repository.ErrUserTeamMismatch) {
					return ErrUserExists
				}
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetTeam(ctx, team.Name)
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*model.Team, error) {
	team, err := s.repo.GetTeam(ctx, teamName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return team, nil
}

func (s *Service) SetUserActivity(ctx context.Context, userID string, isActive bool) (*model.User, error) {
	user, err := s.repo.SetUserActivity(ctx, userID, isActive)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) CreatePullRequest(ctx context.Context, prID, name, authorID string) (*model.PullRequest, error) {
	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := repository.New(tx)

		teamName, err := txRepo.FindTeamName(ctx, authorID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}

		reviewers, err := txRepo.PickReviewers(ctx, teamName, authorID, 2)
		if err != nil {
			return err
		}

		if err := txRepo.CreatePullRequest(ctx, prID, name, authorID, statusOpen); err != nil {
			if isUniqueViolation(err) {
				return ErrPullRequestExists
			}
			return err
		}

		if err := txRepo.AddReviewers(ctx, prID, reviewers); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.repo.GetPullRequest(ctx, prID)
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (*model.PullRequest, error) {
	rows, err := s.repo.MarkMerged(ctx, prID, statusMerged)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrNotFound
	}
	return s.repo.GetPullRequest(ctx, prID)
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*model.PullRequest, string, error) {
	var replacedBy string
	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := repository.New(tx)

		pr, err := txRepo.LockPullRequest(ctx, prID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		if pr.Status == statusMerged {
			return ErrPullRequestMerged
		}

		rec, err := txRepo.LockReviewer(ctx, prID, oldReviewerID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrReviewerMissing
			}
			return err
		}

		teamName, err := txRepo.FindTeamName(ctx, oldReviewerID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrReviewerMissing
			}
			return err
		}

		candidate, err := txRepo.PickReplacement(ctx, teamName, oldReviewerID, prID, pr.AuthorID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNoCandidate
			}
			return err
		}

		if err := txRepo.UpdateReviewer(ctx, prID, rec.Position, candidate); err != nil {
			return err
		}

		replacedBy = candidate
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	pr, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, "", err
	}
	return pr, replacedBy, nil
}

func (s *Service) ListReviewerAssignments(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	if err := s.repo.EnsureUserExists(ctx, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.repo.ListAssignments(ctx, userID)
}

func (s *Service) AssignmentStats(ctx context.Context, limit int) ([]model.AssignmentStat, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.AssignmentStats(ctx, limit)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
