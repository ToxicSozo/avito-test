package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/ToxicSozo/avito-test/api"
	"github.com/ToxicSozo/avito-test/internal/domain/service"
	"github.com/ToxicSozo/avito-test/internal/model"
)

const (
	maxRequestBodyBytes    int64 = 1 << 20
	defaultAssignmentLimit       = 50
	maxAssignmentLimit           = 500
)

type Handler struct {
	teamSvc TeamService
	userSvc UserService
	prSvc   PullRequestService

	log *logrus.Entry
}

func NewHandler(
	teamSvc TeamService,
	userSvc UserService,
	prSvc PullRequestService,
	logger *logrus.Entry,
) *Handler {
	return &Handler{
		teamSvc: teamSvc,
		userSvc: userSvc,
		prSvc:   prSvc,
		log:     logger,
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

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrTeamExists):
		writeJSONError(w, http.StatusBadRequest, api.TEAMEXISTS, "team already exists")
	case errors.Is(err, service.ErrUserExists):
		writeJSONError(w, http.StatusBadRequest, api.USEREXISTS, "user already exists")
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

func (h *Handler) logError(msg string, err error, keyvals ...any) {
	if h.log == nil {
		return
	}
	entry := h.log
	if err != nil {
		entry = entry.WithError(err)
	}
	if len(keyvals) > 0 {
		entry = entry.WithFields(makeFields(keyvals...))
	}
	entry.Error(msg)
}

func (h *Handler) logWarn(msg string, keyvals ...any) {
	if h.log == nil {
		return
	}
	entry := h.log
	if len(keyvals) > 0 {
		entry = entry.WithFields(makeFields(keyvals...))
	}
	entry.Warn(msg)
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

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer func() {
		_ = r.Body.Close()
	}()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func makeFields(values ...any) logrus.Fields {
	fields := logrus.Fields{}
	for i := 0; i < len(values)-1; i += 2 {
		key, ok := values[i].(string)
		if !ok {
			continue
		}
		fields[key] = values[i+1]
	}
	return fields
}
