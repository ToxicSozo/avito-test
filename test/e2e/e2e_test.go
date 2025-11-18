package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ToxicSozo/avito-test/api"
	"github.com/ToxicSozo/avito-test/internal/app"
	"github.com/ToxicSozo/avito-test/internal/config"
	"github.com/ToxicSozo/avito-test/pkg/logger"
)

func TestEndToEndScenario(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pg := startPostgres(t, ctx)
	defer func() {
		require.NoError(t, pg.Terminate(context.Background()))
	}()

	port := freePort(t)

	cfg := &config.Config{
		Port:            strconv.Itoa(port),
		DatabaseURL:     pg.dsn,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}

	log := logger.New()
	log.SetOutput(io.Discard)

	appCtx, appCancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run(appCtx, cfg, log)
	}()

	waitForReady(t, port)

	client := &http.Client{Timeout: 5 * time.Second}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	teamPayload := api.Team{
		TeamName: "backend",
		Members: []api.TeamMember{
			{UserId: "u1", Username: "Alice", IsActive: true},
			{UserId: "u2", Username: "Bob", IsActive: true},
			{UserId: "u3", Username: "Charlie", IsActive: true},
			{UserId: "u4", Username: "Diana", IsActive: true},
		},
	}

	var teamResp struct {
		Team api.Team `json:"team"`
	}
	doRequest(t, client, baseURL+"/team/add", http.MethodPost, teamPayload, http.StatusCreated, &teamResp)
	require.Equal(t, teamPayload.TeamName, teamResp.Team.TeamName)
	require.Len(t, teamResp.Team.Members, len(teamPayload.Members))

	prPayload := api.PostPullRequestCreateJSONRequestBody{
		PullRequestId:   "pr-1001",
		PullRequestName: "Add search",
		AuthorId:        "u1",
	}

	var prResp struct {
		PR api.PullRequest `json:"pr"`
	}
	doRequest(t, client, baseURL+"/pullRequest/create", http.MethodPost, prPayload, http.StatusCreated, &prResp)
	require.Equal(t, prPayload.PullRequestId, prResp.PR.PullRequestId)
	require.Equal(t, api.PullRequestStatus("OPEN"), prResp.PR.Status)
	require.Len(t, prResp.PR.AssignedReviewers, 2)
	require.NotContains(t, prResp.PR.AssignedReviewers, prPayload.AuthorId)

	oldReviewer := prResp.PR.AssignedReviewers[0]
	reassignPayload := api.PostPullRequestReassignJSONRequestBody{
		PullRequestId: prPayload.PullRequestId,
		OldUserId:     oldReviewer,
	}

	var reassignResp struct {
		PR         api.PullRequest `json:"pr"`
		ReplacedBy string          `json:"replaced_by"`
	}
	doRequest(t, client, baseURL+"/pullRequest/reassign", http.MethodPost, reassignPayload, http.StatusOK, &reassignResp)
	require.NotEqual(t, oldReviewer, reassignResp.ReplacedBy)
	require.Contains(t, reassignResp.PR.AssignedReviewers, reassignResp.ReplacedBy)

	mergePayload := api.PostPullRequestMergeJSONRequestBody{
		PullRequestId: prPayload.PullRequestId,
	}
	var mergeResp struct {
		PR api.PullRequest `json:"pr"`
	}
	doRequest(t, client, baseURL+"/pullRequest/merge", http.MethodPost, mergePayload, http.StatusOK, &mergeResp)
	require.Equal(t, api.PullRequestStatus("MERGED"), mergeResp.PR.Status)
	require.NotNil(t, mergeResp.PR.MergedAt)

	var mergeConflict api.ErrorResponse
	doRequest(t, client, baseURL+"/pullRequest/reassign", http.MethodPost, reassignPayload, http.StatusConflict, &mergeConflict)
	require.Equal(t, api.PRMERGED, mergeConflict.Error.Code)

	var assignmentsResp struct {
		UserID       string                 `json:"user_id"`
		PullRequests []api.PullRequestShort `json:"pull_requests"`
	}
	doRequest(t, client, fmt.Sprintf("%s/users/getReview?user_id=%s", baseURL, reassignResp.ReplacedBy), http.MethodGet, nil, http.StatusOK, &assignmentsResp)
	require.Equal(t, reassignResp.ReplacedBy, assignmentsResp.UserID)
	require.Len(t, assignmentsResp.PullRequests, 1)
	require.Equal(t, prPayload.PullRequestId, assignmentsResp.PullRequests[0].PullRequestId)

	appCancel()
	require.NoError(t, <-errCh)
}

func doRequest(t *testing.T, client *http.Client, url, method string, body any, expectedStatus int, out any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode, string(data))

	if out != nil {
		require.NoError(t, json.Unmarshal(data, out))
	}
}

type postgresContainer struct {
	testcontainers.Container
	dsn string
}

func startPostgres(t *testing.T, ctx context.Context) *postgresContainer {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "secret",
			"POSTGRES_DB":       "reviewer",
			"POSTGRES_USER":     "postgres",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://postgres:secret@%s:%s/reviewer?sslmode=disable", host, mappedPort.Port())

	return &postgresContainer{
		Container: container,
		dsn:       dsn,
	}
}

func waitForReady(t *testing.T, port int) {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/team/get?team_name=__ping__", port)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server did not become ready on port %d", port)
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = l.Close()
	}()
	return l.Addr().(*net.TCPAddr).Port
}
