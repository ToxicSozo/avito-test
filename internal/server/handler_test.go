package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ToxicSozo/avito-test/internal/model"
)

type stubService struct {
	stats []model.AssignmentStat
	err   error
	limit int
}

func (s *stubService) CreateTeam(context.Context, model.Team) (*model.Team, error) {
	return nil, nil
}
func (s *stubService) GetTeam(context.Context, string) (*model.Team, error) {
	return nil, nil
}
func (s *stubService) SetUserActivity(context.Context, string, bool) (*model.User, error) {
	return nil, nil
}
func (s *stubService) CreatePullRequest(context.Context, string, string, string) (*model.PullRequest, error) {
	return nil, nil
}
func (s *stubService) MergePullRequest(context.Context, string) (*model.PullRequest, error) {
	return nil, nil
}
func (s *stubService) ReassignReviewer(context.Context, string, string) (*model.PullRequest, string, error) {
	return nil, "", nil
}
func (s *stubService) ListReviewerAssignments(context.Context, string) ([]model.PullRequestShort, error) {
	return nil, nil
}
func (s *stubService) AssignmentStats(ctx context.Context, limit int) ([]model.AssignmentStat, error) {
	s.limit = limit
	return s.stats, s.err
}

func TestGetAssignmentStats_Success(t *testing.T) {
	stub := &stubService{
		stats: []model.AssignmentStat{
			{UserID: "u1", Username: "Alice", TeamName: "payments", Assignments: 3},
			{UserID: "u2", Username: "Bob", TeamName: "infra", Assignments: 1},
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandler(stub, stub, stub, logger, "admin", "user")

	req := httptest.NewRequest(http.MethodGet, "/stats/assignments?limit=10", nil)
	req.Header.Set("Authorization", "Bearer admin")
	rr := httptest.NewRecorder()

	h.GetAssignmentStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if stub.limit != 10 {
		t.Fatalf("expected limit 10, got %d", stub.limit)
	}

	var body struct {
		Stats []map[string]interface{} `json:"stats"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Stats) != 2 {
		t.Fatalf("expected 2 stats entries, got %d", len(body.Stats))
	}
	if body.Stats[0]["user_id"] != "u1" {
		t.Fatalf("expected first user_id u1, got %v", body.Stats[0]["user_id"])
	}
}

func TestGetAssignmentStats_Unauthorized(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandler(&stubService{}, &stubService{}, &stubService{}, logger, "admin", "user")

	req := httptest.NewRequest(http.MethodGet, "/stats/assignments", nil)
	rr := httptest.NewRecorder()

	h.GetAssignmentStats(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}
