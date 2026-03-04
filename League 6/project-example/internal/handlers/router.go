package handlers

import (
	"net/http"

	"github.com/kuromii5/snapbin/internal/handlers/health"
	"github.com/kuromii5/snapbin/internal/handlers/middleware"
	pollhandler "github.com/kuromii5/snapbin/internal/handlers/poll"
)

func NewRouter(pollH *pollhandler.Handler, healthH *health.Handler, limiter middleware.Limiter) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", healthH.Health)

	rateLimiterMW := middleware.RateLimit(limiter)
	mux.Handle("POST /api/polls",
		rateLimiterMW(http.HandlerFunc(pollH.Create)),
	)

	mux.HandleFunc("GET /api/polls/{id}", pollH.Get)
	mux.HandleFunc("POST /api/polls/{id}/vote", pollH.Vote)
	mux.HandleFunc("DELETE /api/polls/{id}", pollH.Delete)

	return middleware.RequestID(middleware.Logger(mux))
}
