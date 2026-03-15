package poll

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kuromii5/poller/internal/domain"
)

type voteRequest struct {
	OptionID string `json:"option_id"`
}

func (h *Handler) Vote(w http.ResponseWriter, r *http.Request) {
	pollID := r.PathValue("id")

	var req voteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OptionID == "" {
		writeError(w, http.StatusBadRequest, "option_id is required")
		return
	}

	ip := realIP(r)

	if err := h.svc.Vote(r.Context(), pollID, req.OptionID, ip); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			writeError(w, http.StatusNotFound, "poll not found")
		case errors.Is(err, domain.ErrExpired):
			writeError(w, http.StatusGone, "poll has expired")
		case errors.Is(err, domain.ErrAlreadyVoted):
			writeError(w, http.StatusConflict, "already voted")
		case errors.Is(err, domain.ErrInvalidOption):
			writeError(w, http.StatusBadRequest, "invalid option")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func realIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	if ip, _, found := strings.Cut(r.RemoteAddr, ":"); found {
		return ip
	}
	return r.RemoteAddr
}
