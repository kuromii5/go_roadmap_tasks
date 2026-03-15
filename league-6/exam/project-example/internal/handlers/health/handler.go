package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	db  *sqlx.DB
	rdb *redis.Client
}

func NewHandler(db *sqlx.DB, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb}
}

type healthResponse struct {
	Status   string `json:"status"`
	Postgres string `json:"postgres"`
	Redis    string `json:"redis"`
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	res := healthResponse{Status: "ok", Postgres: "ok", Redis: "ok"}
	code := http.StatusOK

	if err := h.db.PingContext(ctx); err != nil {
		res.Postgres = "unavailable"
		res.Status = "degraded"
		code = http.StatusServiceUnavailable
	}
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		res.Redis = "unavailable"
		res.Status = "degraded"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(res)
}
