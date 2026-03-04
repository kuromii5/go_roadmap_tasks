package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	rateLimitWindow = time.Minute
	rateLimitMax    = 10
)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

// Allow returns true if the IP is within the rate limit, false otherwise.
func (r *RateLimiter) Allow(ctx context.Context, ip string) bool {
	key := "ratelimit:" + ip
	count, err := r.rdb.Incr(ctx, key).Result()
	if err != nil {
		// On Redis failure, allow the request (fail-open)
		return true
	}
	if count == 1 {
		r.rdb.Expire(ctx, key, rateLimitWindow)
	}
	return count <= rateLimitMax
}
