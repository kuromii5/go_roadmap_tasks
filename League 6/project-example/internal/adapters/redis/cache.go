package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kuromii5/poller/internal/domain"
)

type Cache struct {
	rdb *redis.Client
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

func cacheKey(id string) string { return "poll:" + id }

func (c *Cache) Get(ctx context.Context, id string) (*domain.PollResult, error) {
	data, err := c.rdb.Get(ctx, cacheKey(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var result domain.PollResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal cached poll: %w", err)
	}
	return &result, nil
}

func (c *Cache) Set(ctx context.Context, result *domain.PollResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal poll: %w", err)
	}
	ttl := time.Until(result.Poll.ExpiresAt)
	if ttl <= 0 {
		return nil
	}
	if err := c.rdb.Set(ctx, cacheKey(result.Poll.ID), data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func (c *Cache) Delete(ctx context.Context, id string) error {
	if err := c.rdb.Del(ctx, cacheKey(id)).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}
