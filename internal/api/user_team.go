package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"pr-reviewer/internal/store"
)

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.UserID == "" {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}

	updatedUser, err := h.store.SetUserActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]any{"user": updatedUser})
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id query param is required")
		return
	}

	prs, err := h.store.GetReviewsForUser(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]any{
		"user_id":       userID,
		"pull_requests": prs,
	})
}