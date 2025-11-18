package handlers

import "net/http"

func (h *Handler) GetStatsAssignments(w http.ResponseWriter, r *http.Request) {
	stats, err := h.prSvc.GetAssignmentStats(r.Context())
	if err != nil {
		h.logError("fetch assignment stats failed", err)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIAssignmentStats(stats))
}
