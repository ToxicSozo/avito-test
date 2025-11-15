package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ToxicSozo/avito-test/internal/api"
	"github.com/ToxicSozo/avito-test/internal/model"
	"github.com/ToxicSozo/avito-test/internal/service"
)

const (
	authHeader         = "Authorization"
	bearerPrefix       = "Bearer "
	errorCodeUserExist = api.ErrorResponseErrorCode("USER_EXISTS")
	unauthCode         = api.ErrorResponseErrorCode("UNAUTHORIZED")
)

// Handler implements api.ServerInterface.
type Handler struct {
	teamSvc TeamService
	userSvc UserService
	prSvc   PullRequestService

	log *slog.Logger

	adminToken string
	userToken  string
}

// NewHandler wires dependencies.
func NewHandler(teamSvc TeamService, userSvc UserService, prSvc PullRequestService, logger *slog.Logger, adminToken, userToken string) *Handler {
	return &Handler{
		teamSvc:    teamSvc,
		userSvc:    userSvc,
		prSvc:      prSvc,
		log:        logger,
		adminToken: adminToken,
		userToken:  userToken,
	}
}

type TeamService interface {
	CreateTeam(ctx context.Context, team model.Team) (*model.Team, error)
	GetTeam(ctx context.Context, teamName string) (*model.Team, error)
}

type UserService interface {
	SetUserActivity(ctx context.Context, userID string, isActive bool) (*model.User, error)
}

type PullRequestService interface {
	CreatePullRequest(ctx context.Context, prID, name, authorID string) (*model.PullRequest, error)
	MergePullRequest(ctx context.Context, prID string) (*model.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*model.PullRequest, string, error)
	ListReviewerAssignments(ctx context.Context, userID string) ([]model.PullRequestShort, error)
	AssignmentStats(ctx context.Context, limit int) ([]model.AssignmentStat, error)
}

var (
	_ TeamService        = (*service.Service)(nil)
	_ UserService        = (*service.Service)(nil)
	_ PullRequestService = (*service.Service)(nil)
)

// PostTeamAdd handles /team/add.
func (h *Handler) PostTeamAdd(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	var payload api.Team
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logError("failed to decode team payload", err, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "invalid JSON payload")
		return
	}
	if payload.TeamName == "" {
		h.logWarn("team_name is missing", "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "team_name is required")
		return
	}

	members := make([]model.User, 0, len(payload.Members))
	for _, m := range payload.Members {
		if m.UserId == "" || m.Username == "" {
			h.logWarn("team member is missing fields", "team_name", payload.TeamName)
			writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "all members must have user_id and username")
			return
		}
		members = append(members, model.User{
			ID:       m.UserId,
			Username: m.Username,
			TeamName: payload.TeamName,
			IsActive: m.IsActive,
		})
	}

	team, err := h.teamSvc.CreateTeam(r.Context(), model.Team{
		Name:    payload.TeamName,
		Members: members,
	})
	if err != nil {
		h.logError("create team failed", err, "team_name", payload.TeamName)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]api.Team{
		"team": toAPITeam(team),
	})
}

// GetTeamGet handles /team/get.
func (h *Handler) GetTeamGet(w http.ResponseWriter, r *http.Request, params api.GetTeamGetParams) {
	if !h.requireAdmin(w, r) {
		return
	}

	team, err := h.teamSvc.GetTeam(r.Context(), params.TeamName)
	if err != nil {
		h.logError("get team failed", err, "team_name", params.TeamName)
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITeam(team))
}

// PostUsersSetIsActive handles /users/setIsActive.
func (h *Handler) PostUsersSetIsActive(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	var payload struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logError("failed to decode setIsActive payload", err, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "invalid JSON payload")
		return
	}
	if payload.UserID == "" {
		h.logWarn("setIsActive missing user_id", "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "user_id is required")
		return
	}

	user, err := h.userSvc.SetUserActivity(r.Context(), payload.UserID, payload.IsActive)
	if err != nil {
		h.logError("set user activity failed", err, "user_id", payload.UserID)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]api.User{
		"user": toAPIUser(user),
	})
}

// PostPullRequestCreate handles PR creation.
func (h *Handler) PostPullRequestCreate(w http.ResponseWriter, r *http.Request) {
	if !h.requireUser(w, r) {
		return
	}

	var payload api.PostPullRequestCreateJSONBody
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logError("failed to decode PR create payload", err, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "invalid JSON payload")
		return
	}
	if payload.PullRequestId == "" || payload.PullRequestName == "" || payload.AuthorId == "" {
		h.logWarn("PR create missing fields", "author_id", payload.AuthorId)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "pull_request_id, pull_request_name and author_id are required")
		return
	}

	pr, err := h.prSvc.CreatePullRequest(r.Context(), payload.PullRequestId, payload.PullRequestName, payload.AuthorId)
	if err != nil {
		h.logError("create pull request failed", err, "pr_id", payload.PullRequestId)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]api.PullRequest{
		"pr": toAPIPullRequest(pr),
	})
}

// PostPullRequestMerge handles PR merge.
func (h *Handler) PostPullRequestMerge(w http.ResponseWriter, r *http.Request) {
	if !h.requireUser(w, r) {
		return
	}

	var payload api.PostPullRequestMergeJSONBody
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logError("failed to decode merge payload", err, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "invalid JSON payload")
		return
	}
	if payload.PullRequestId == "" {
		h.logWarn("merge missing pull_request_id", "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "pull_request_id is required")
		return
	}

	pr, err := h.prSvc.MergePullRequest(r.Context(), payload.PullRequestId)
	if err != nil {
		h.logError("merge pull request failed", err, "pr_id", payload.PullRequestId)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]api.PullRequest{
		"pr": toAPIPullRequest(pr),
	})
}

// PostPullRequestReassign handles reviewer reassign.
func (h *Handler) PostPullRequestReassign(w http.ResponseWriter, r *http.Request) {
	if !h.requireUser(w, r) {
		return
	}

	var payload api.PostPullRequestReassignJSONBody
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logError("failed to decode reassign payload", err, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "invalid JSON payload")
		return
	}
	if payload.PullRequestId == "" || payload.OldUserId == "" {
		h.logWarn("reassign missing fields", "pr_id", payload.PullRequestId)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "pull_request_id and old_user_id are required")
		return
	}

	pr, replacedBy, err := h.prSvc.ReassignReviewer(r.Context(), payload.PullRequestId, payload.OldUserId)
	if err != nil {
		h.logError("reassign reviewer failed", err, "pr_id", payload.PullRequestId, "old_user_id", payload.OldUserId)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pr":          toAPIPullRequest(pr),
		"replaced_by": replacedBy,
	})
}

// GetUsersGetReview returns PR assigned to reviewer.
func (h *Handler) GetUsersGetReview(w http.ResponseWriter, r *http.Request, params api.GetUsersGetReviewParams) {
	if !h.requireUser(w, r) {
		return
	}

	prs, err := h.prSvc.ListReviewerAssignments(r.Context(), params.UserId)
	if err != nil {
		h.logError("list reviewer assignments failed", err, "user_id", params.UserId)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       params.UserId,
		"pull_requests": toAPIPullRequestsShort(prs),
	})
}

// GetAssignmentStats returns reviewer assignment stats (custom endpoint).
func (h *Handler) GetAssignmentStats(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		} else {
			h.logWarn("invalid limit parameter", "raw", raw)
		}
	}

	stats, err := h.prSvc.AssignmentStats(r.Context(), limit)
	if err != nil {
		h.logError("assignment stats failed", err, "limit", limit)
		writeJSONError(w, http.StatusInternalServerError, api.NOTFOUND, "failed to load stats")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stats": toAPIStats(stats),
	})
}

func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	token := extractToken(r.Header.Get(authHeader))
	if token == "" || token != h.adminToken {
		h.logWarn("admin token rejected", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusUnauthorized, unauthCode, "admin token is required")
		return false
	}
	return true
}

func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) bool {
	token := extractToken(r.Header.Get(authHeader))
	if token == h.userToken || token == h.adminToken {
		return true
	}
	h.logWarn("user token rejected", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
	writeJSONError(w, http.StatusUnauthorized, unauthCode, "user token is required")
	return false
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrTeamExists):
		writeJSONError(w, http.StatusBadRequest, api.TEAMEXISTS, "team already exists")
	case errors.Is(err, service.ErrUserExists):
		writeJSONError(w, http.StatusBadRequest, errorCodeUserExist, "user already exists")
	case errors.Is(err, service.ErrPullRequestExists):
		writeJSONError(w, http.StatusConflict, api.PREXISTS, "pull request already exists")
	case errors.Is(err, service.ErrPullRequestMerged):
		writeJSONError(w, http.StatusConflict, api.PRMERGED, "pull request already merged")
	case errors.Is(err, service.ErrReviewerMissing):
		writeJSONError(w, http.StatusConflict, api.NOTASSIGNED, "reviewer is not assigned to this pull request")
	case errors.Is(err, service.ErrNoCandidate):
		writeJSONError(w, http.StatusConflict, api.NOCANDIDATE, "no active replacement candidate found")
	case errors.Is(err, service.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, api.NOTFOUND, "resource not found")
	default:
		writeJSONError(w, http.StatusInternalServerError, api.NOTFOUND, "internal server error")
	}
}

func extractToken(header string) string {
	if header == "" {
		return ""
	}
	if !strings.HasPrefix(header, bearerPrefix) {
		return header
	}
	return strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
}

func toAPITeam(team *model.Team) api.Team {
	members := make([]api.TeamMember, 0, len(team.Members))
	for _, m := range team.Members {
		members = append(members, api.TeamMember{
			UserId:   m.ID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	return api.Team{
		TeamName: team.Name,
		Members:  members,
	}
}

func toAPIUser(user *model.User) api.User {
	return api.User{
		UserId:   user.ID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}
}

func toAPIPullRequest(pr *model.PullRequest) api.PullRequest {
	response := api.PullRequest{
		PullRequestId:   pr.ID,
		PullRequestName: pr.Name,
		AuthorId:        pr.AuthorID,
		Status:          api.PullRequestStatus(pr.Status),
		AssignedReviewers: func() []string {
			if pr.AssignedReviewers == nil {
				return []string{}
			}
			return pr.AssignedReviewers
		}(),
		CreatedAt: &pr.CreatedAt,
	}
	if pr.MergedAt != nil {
		response.MergedAt = pr.MergedAt
	}
	return response
}

func toAPIPullRequestsShort(prs []model.PullRequestShort) []api.PullRequestShort {
	result := make([]api.PullRequestShort, 0, len(prs))
	for _, pr := range prs {
		result = append(result, api.PullRequestShort{
			PullRequestId:   pr.ID,
			PullRequestName: pr.Name,
			AuthorId:        pr.AuthorID,
			Status:          api.PullRequestShortStatus(pr.Status),
		})
	}
	return result
}

func toAPIStats(stats []model.AssignmentStat) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(stats))
	for _, s := range stats {
		result = append(result, map[string]interface{}{
			"user_id":     s.UserID,
			"username":    s.Username,
			"team_name":   s.TeamName,
			"assignments": s.Assignments,
		})
	}
	return result
}

func (h *Handler) logError(msg string, err error, keyvals ...any) {
	if h.log == nil {
		return
	}
	fields := append([]any{"error", err}, keyvals...)
	h.log.Error(msg, fields...)
}

func (h *Handler) logWarn(msg string, keyvals ...any) {
	if h.log == nil {
		return
	}
	h.log.Warn(msg, keyvals...)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, code api.ErrorResponseErrorCode, message string) {
	resp := api.ErrorResponse{}
	resp.Error.Code = code
	resp.Error.Message = message
	writeJSON(w, status, resp)
}
