package service_test

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ToxicSozo/avito-test/internal/model"
	"github.com/ToxicSozo/avito-test/internal/service"
	pgstorage "github.com/ToxicSozo/avito-test/internal/storage/postgres"
)

func TestServiceIntegration_BusinessFlows(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skipf("docker not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := tcpostgres.Run(ctx,
		"postgres:16",
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.WithDatabase("reviewers"),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	defer func() {
		_ = container.Terminate(context.Background())
	}()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	var db *pgstorage.DB
	for i := 0; i < 10; i++ {
		db, err = pgstorage.New(ctx, dsn)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		t.Fatalf("init storage: %v", err)
	}
	defer db.Close()

	svc := service.New(db.Pool())

	team, err := svc.CreateTeam(ctx, model.Team{
		Name: "backend",
		Members: []model.User{
			{ID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
			{ID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
			{ID: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
			{ID: "u4", Username: "Dora", TeamName: "backend", IsActive: true},
		},
	})
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	if len(team.Members) != 4 {
		t.Fatalf("expected 4 team members, got %d", len(team.Members))
	}

	pr, err := svc.CreatePullRequest(ctx, "pr-1", "Search API", "u1")
	if err != nil {
		t.Fatalf("create pull request: %v", err)
	}
	if pr.AuthorID != "u1" {
		t.Fatalf("unexpected author id %s", pr.AuthorID)
	}
	if len(pr.AssignedReviewers) == 0 {
		t.Fatalf("expected reviewers to be assigned")
	}
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer == "u1" {
			t.Fatalf("author should not be assigned as reviewer")
		}
	}

	oldReviewer := pr.AssignedReviewers[0]
	prAfterReassign, replacedBy, err := svc.ReassignReviewer(ctx, pr.ID, oldReviewer)
	if err != nil {
		t.Fatalf("reassign reviewer: %v", err)
	}
	if replacedBy == oldReviewer {
		t.Fatalf("reassigned reviewer must differ from old reviewer")
	}
	if contains(prAfterReassign.AssignedReviewers, replacedBy) == false {
		t.Fatalf("new reviewer not present in PR assignment list")
	}

	merged, err := svc.MergePullRequest(ctx, pr.ID)
	if err != nil {
		t.Fatalf("merge pull request: %v", err)
	}
	if merged.Status != "MERGED" {
		t.Fatalf("expected MERGED status, got %s", merged.Status)
	}

	if _, _, err := svc.ReassignReviewer(ctx, pr.ID, replacedBy); !errors.Is(err, service.ErrPullRequestMerged) {
		t.Fatalf("expected ErrPullRequestMerged, got %v", err)
	}

	assignments, err := svc.ListReviewerAssignments(ctx, replacedBy)
	if err != nil {
		t.Fatalf("list reviewer assignments: %v", err)
	}
	if len(assignments) == 0 {
		t.Fatalf("expected assignments for %s", replacedBy)
	}

	stats, err := svc.AssignmentStats(ctx, 10)
	if err != nil {
		t.Fatalf("assignment stats: %v", err)
	}
	if len(stats) == 0 {
		t.Fatalf("expected stats entries")
	}
	if stats[0].Assignments == 0 {
		t.Fatalf("top reviewer should have assignments > 0")
	}

	_, err = svc.SetUserActivity(ctx, "u2", false)
	if err != nil {
		t.Fatalf("set user activity: %v", err)
	}

	secondPR, err := svc.CreatePullRequest(ctx, "pr-2", "Checkout flow", "u1")
	if err != nil {
		t.Fatalf("create second PR: %v", err)
	}
	if contains(secondPR.AssignedReviewers, "u2") {
		t.Fatal("inactive user should not be auto assigned")
	}
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
