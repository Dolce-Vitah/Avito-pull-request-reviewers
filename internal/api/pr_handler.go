package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"pr-reviewer/internal/model"
	"pr-reviewer/internal/store"
)

func (h *Handler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req model.PullRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.ID == "" || req.Name == "" || req.AuthorID == "" {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id, pull_request_name, and author_id are required")
		return
	}

	err := h.store.CreatePullRequest(r.Context(), &req)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "NOT_FOUND", "author not found")
			return
		}
		if errors.Is(err, store.ErrPRExists) {
			h.respondError(w, http.StatusConflict, "PR_EXISTS", "PR id already exists")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]any{"pr": req})
}

func (h *Handler) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.PullRequestID == "" {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id is required")
		return
	}

	pr, err := h.store.MergePullRequest(r.Context(), req.PullRequestID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "NOT_FOUND", "PR not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]any{"pr": pr})
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.PullRequestID == "" || req.OldUserID == "" {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id and old_user_id are required")
		return
	}

	pr, replacedBy, err := h.store.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			h.respondError(w, http.StatusNotFound, "NOT_FOUND", "PR or user not found")
		case errors.Is(err, store.ErrPRMerged):
			h.respondError(w, http.StatusConflict, "PR_MERGED", "cannot reassign on merged PR")
		case errors.Is(err, store.ErrNotAssigned):
			h.respondError(w, http.StatusConflict, "NOT_ASSIGNED", "reviewer is not assigned to this PR")
		case errors.Is(err, store.ErrNoCandidate):
			h.respondError(w, http.StatusConflict, "NO_CANDIDATE", "no active replacement candidate in team")
		default:
			h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]any{
		"pr":          pr,
		"replaced_by": replacedBy,
	})
}