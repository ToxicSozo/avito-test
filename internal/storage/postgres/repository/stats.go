package repository

import (
	"context"

	"github.com/ToxicSozo/avito-test/internal/model"
)

func (s *Store) AssignmentStats(ctx context.Context) (*model.AssignmentStats, error) {
	stats := model.AssignmentStats{}

	userRows, err := s.q.Query(ctx, `
		SELECT reviewer_id, COUNT(*) AS assignments
		FROM pull_request_reviewers
		GROUP BY reviewer_id
		ORDER BY assignments DESC, reviewer_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer userRows.Close()

	for userRows.Next() {
		var rec model.AssignmentCountByUser
		if err := userRows.Scan(&rec.UserID, &rec.Assignments); err != nil {
			return nil, err
		}
		stats.ByUser = append(stats.ByUser, rec)
	}
	if err := userRows.Err(); err != nil {
		return nil, err
	}

	prRows, err := s.q.Query(ctx, `
		SELECT pull_request_id, COUNT(*) AS reviewers
		FROM pull_request_reviewers
		GROUP BY pull_request_id
		ORDER BY reviewers DESC, pull_request_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer prRows.Close()

	for prRows.Next() {
		var rec model.AssignmentCountByPullRequest
		if err := prRows.Scan(&rec.PullRequestID, &rec.Reviewers); err != nil {
			return nil, err
		}
		stats.ByPullRequest = append(stats.ByPullRequest, rec)
	}
	if err := prRows.Err(); err != nil {
		return nil, err
	}

	return &stats, nil
}
