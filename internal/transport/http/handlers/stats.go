package handlers

import (
	"net/http"

	"github.com/ToxicSozo/avito-test/api"
)

func (h *Handler) GetStatsAssignments(w http.ResponseWriter, r *http.Request, params api.GetStatsAssignmentsParams) {
	limit := defaultAssignmentLimit
	if params.Limit != nil {
		limit = *params.Limit
		if limit <= 0 {
			limit = defaultAssignmentLimit
		} else if limit > maxAssignmentLimit {
			h.logWarn("limit exceeded maximum, clamping", "raw", *params.Limit, "max", maxAssignmentLimit)
			limit = maxAssignmentLimit
		}
	}

	stats, err := h.prSvc.AssignmentStats(r.Context(), limit)
	if err != nil {
		h.logError("assignment stats failed", err, "limit", limit)
		writeJSONError(w, http.StatusInternalServerError, api.NOTFOUND, "failed to load stats")
		return
	}

	writeJSON(w, http.StatusOK, newAssignmentStatsResponse(stats))
}
