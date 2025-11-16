package handlers

import (
	"net/http"

	"github.com/ToxicSozo/avito-test/api"
)

func (h *Handler) PostUsersSetIsActive(w http.ResponseWriter, r *http.Request) {
	var payload api.PostUsersSetIsActiveJSONRequestBody
	if err := decodeJSONBody(w, r, &payload); err != nil {
		h.logError("failed to decode setIsActive payload", err, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "invalid JSON payload")
		return
	}

	user, err := h.userSvc.SetUserActivity(r.Context(), payload.UserId, payload.IsActive)
	if err != nil {
		h.logError("set user activity failed", err, "user_id", payload.UserId)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, userEnvelope{User: toAPIUser(user)})
}

func (h *Handler) GetUsersGetReview(w http.ResponseWriter, r *http.Request, params api.GetUsersGetReviewParams) {
	prs, err := h.prSvc.ListReviewerAssignments(r.Context(), params.UserId)
	if err != nil {
		h.logError("list reviewer assignments failed", err, "user_id", params.UserId)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newReviewerAssignmentsResponse(params.UserId, prs))
}
