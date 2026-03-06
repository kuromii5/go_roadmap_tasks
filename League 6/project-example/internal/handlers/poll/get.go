package poll

import (
	"errors"
	"net/http"

	"github.com/kuromii5/poller/internal/domain"
)

type optionResponse struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Votes int    `json:"votes"`
}

type getResponse struct {
	ID        string           `json:"id"`
	Question  string           `json:"question"`
	Options   []optionResponse `json:"options"`
	ExpiresAt string           `json:"expires_at"`
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	result, err := h.svc.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			writeError(w, http.StatusNotFound, "poll not found")
		case errors.Is(err, domain.ErrExpired):
			writeError(w, http.StatusGone, "poll has expired")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	opts := make([]optionResponse, len(result.Options))
	for i, o := range result.Options {
		opts[i] = optionResponse{ID: o.ID, Text: o.Text, Votes: o.Votes}
	}

	writeJSON(w, http.StatusOK, getResponse{
		ID:        result.Poll.ID,
		Question:  result.Poll.Question,
		Options:   opts,
		ExpiresAt: result.Poll.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}
