package poll

import (
	"encoding/json"
	"net/http"

	"github.com/kuromii5/snapbin/internal/service/poll"
)

type createRequest struct {
	Question   string   `json:"question"`
	Options    []string `json:"options"`
	TTLMinutes int      `json:"ttl_minutes"`
}

type createResponse struct {
	ID        string `json:"id"`
	Link      string `json:"link"`
	ExpiresAt string `json:"expires_at"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Create(r.Context(), poll.CreateInput{
		Question:   req.Question,
		Options:    req.Options,
		TTLMinutes: req.TTLMinutes,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, createResponse{
		ID:        result.ID,
		Link:      result.Link,
		ExpiresAt: result.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}
