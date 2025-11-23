package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"pr-reviewer/internal/model"
	"pr-reviewer/internal/store"
)

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req model.Team
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.TeamName == "" || len(req.Members) == 0 {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name and members are required")
		return
	}

	err := h.store.CreateTeam(r.Context(), &req)
	if err != nil {
		if errors.Is(err, store.ErrTeamExists) {
			h.respondError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]any{"team": req})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name query param is required")
		return
	}

	team, err := h.store.GetTeam(r.Context(), teamName)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "NOT_FOUND", "team not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, team)
}

func (h *Handler) BulkDeactivate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}

	result, err := h.store.BulkDeactivateAndReassign(r.Context(), req.UserIDs)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]any{
		"deactivated_count": len(req.UserIDs),
		"reassignments":     result,
	})
}