package poll

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kuromii5/snapbin/internal/domain"
	"github.com/kuromii5/snapbin/internal/service/poll"
)

type PollService interface {
	Create(ctx context.Context, in poll.CreateInput) (*poll.CreateResult, error)
	Get(ctx context.Context, id string) (*domain.PollResult, error)
	Vote(ctx context.Context, pollID, optionID, ip string) error
	Delete(ctx context.Context, id string) error
}

type Handler struct {
	svc PollService
}

func NewHandler(svc PollService) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
