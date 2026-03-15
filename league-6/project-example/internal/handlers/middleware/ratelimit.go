package middleware

import (
	"context"
	"net/http"
	"strings"
)

type Limiter interface {
	Allow(ctx context.Context, ip string) bool
}

// RateLimit wraps a handler and rejects requests that exceed the rate limit.
func RateLimit(l Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			if !l.Allow(r.Context(), ip) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded — max 10 pastes per minute"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// realIP extracts the client IP, respecting X-Forwarded-For for proxied requests.
func realIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	if ip, _, found := strings.Cut(r.RemoteAddr, ":"); found {
		return ip
	}
	return r.RemoteAddr
}
