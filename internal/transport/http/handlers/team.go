package handlers

import (
	"net/http"

	"github.com/ToxicSozo/avito-test/api"
)

func (h *Handler) PostTeamAdd(w http.ResponseWriter, r *http.Request) {
	var payload api.Team
	if err := decodeJSONBody(w, r, &payload); err != nil {
		h.logError("failed to decode team payload", err, "remote_addr", r.RemoteAddr)
		writeJSONError(w, http.StatusBadRequest, api.NOTFOUND, "invalid JSON payload")
		return
	}

	team, err := h.teamSvc.CreateTeam(r.Context(), teamPayloadToModel(payload))
	if err != nil {
		h.logError("create team failed", err, "team_name", payload.TeamName)
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, teamEnvelope{Team: toAPITeam(team)})
}

func (h *Handler) GetTeamGet(w http.ResponseWriter, r *http.Request, params api.GetTeamGetParams) {
	team, err := h.teamSvc.GetTeam(r.Context(), params.TeamName)
	if err != nil {
		h.logError("get team failed", err, "team_name", params.TeamName)
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITeam(team))
}
