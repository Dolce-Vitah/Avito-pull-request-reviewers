package api

import (
	"net/http"
)

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetSystemStats(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, stats)
}