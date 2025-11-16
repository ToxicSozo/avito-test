package handlers

import (
	"net/http"

	"github.com/ToxicSozo/avito-test/api"
)

func (h *Handler) PostPullRequestCreate(w http.ResponseWriter, r *http.Request) {
	var payload api.PostPullRequestCreateJSONRequestBody
	if err := decodeJSONBody(w, r, &payload); err != nil {
		h.logError("failed to decode PR create payload", err, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "invalid JSON payload")
		return
	}

	pr, err := h.prSvc.CreatePullRequest(r.Context(), payload.PullRequestId, payload.PullRequestName, payload.AuthorId)
	if err != nil {
		h.logError("create pull request failed", err, "pr_id", payload.PullRequestId)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, pullRequestEnvelope{PullRequest: toAPIPullRequest(pr)})
}

func (h *Handler) PostPullRequestMerge(w http.ResponseWriter, r *http.Request) {
	var payload api.PostPullRequestMergeJSONRequestBody
	if err := decodeJSONBody(w, r, &payload); err != nil {
		h.logError("failed to decode merge payload", err, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "invalid JSON payload")
		return
	}

	pr, err := h.prSvc.MergePullRequest(r.Context(), payload.PullRequestId)
	if err != nil {
		h.logError("merge pull request failed", err, "pr_id", payload.PullRequestId)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, pullRequestEnvelope{PullRequest: toAPIPullRequest(pr)})
}

func (h *Handler) PostPullRequestReassign(w http.ResponseWriter, r *http.Request) {
	var payload api.PostPullRequestReassignJSONRequestBody
	if err := decodeJSONBody(w, r, &payload); err != nil {
		h.logError("failed to decode reassign payload", err, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "invalid JSON payload")
		return
	}

	pr, replacedBy, err := h.prSvc.ReassignReviewer(r.Context(), payload.PullRequestId, payload.OldUserId)
	if err != nil {
		h.logError("reassign reviewer failed", err, "pr_id", payload.PullRequestId, "old_user_id", payload.OldUserId)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reassignResponse{
		PullRequest: toAPIPullRequest(pr),
		ReplacedBy:  replacedBy,
	})
}
