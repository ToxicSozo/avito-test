package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ToxicSozo/avito-test/internal/model"
	"github.com/ToxicSozo/avito-test/internal/storage/postgres/repository"
)

const (
	statusOpen   = "OPEN"
	statusMerged = "MERGED"
)

type Service struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) CreateTeam(ctx context.Context, team model.Team) (*model.Team, error) {
	err := s.withTx(ctx, func(q repository.Querier) error {
		txRepo := repository.New(q)

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
	team, err := repository.New(s.pool).GetTeam(ctx, teamName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return team, nil
}

func (s *Service) SetUserActivity(ctx context.Context, userID string, isActive bool) (*model.User, error) {
	user, err := repository.New(s.pool).SetUserActivity(ctx, userID, isActive)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) CreatePullRequest(ctx context.Context, prID, name, authorID string) (*model.PullRequest, error) {
	err := s.withTx(ctx, func(q repository.Querier) error {
		txRepo := repository.New(q)

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

	return repository.New(s.pool).GetPullRequest(ctx, prID)
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (*model.PullRequest, error) {
	rows, err := repository.New(s.pool).MarkMerged(ctx, prID, statusMerged)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrNotFound
	}
	return repository.New(s.pool).GetPullRequest(ctx, prID)
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*model.PullRequest, string, error) {
	var replacedBy string
	err := s.withTx(ctx, func(q repository.Querier) error {
		txRepo := repository.New(q)

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

	pr, err := repository.New(s.pool).GetPullRequest(ctx, prID)
	if err != nil {
		return nil, "", err
	}
	return pr, replacedBy, nil
}

func (s *Service) ListReviewerAssignments(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	repo := repository.New(s.pool)
	if err := repo.EnsureUserExists(ctx, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return repo.ListAssignments(ctx, userID)
}

func (s *Service) GetAssignmentStats(ctx context.Context) (*model.AssignmentStats, error) {
	return repository.New(s.pool).AssignmentStats(ctx)
}

func (s *Service) withTx(ctx context.Context, fn func(repository.Querier) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx) // safe to ignore error if already committed
	}()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
