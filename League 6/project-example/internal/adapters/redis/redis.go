package redis

import (
	"github.com/redis/go-redis/v9"

	"github.com/kuromii5/poller/config"
)

func NewClient(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}
