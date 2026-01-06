package rds

import (
	"context"
	"fmt"

	"gitea.solu-m.io/smart-pos/proposal-gateway-architect/pkg/config"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func InitRedis(cfg config.RedisConfig) (*RedisClient, error) {
	var client *redis.Client
	if cfg.Password == "" {
		client = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			DB:   0,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Password: cfg.Password,
			DB:       0,
		})
	}

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &RedisClient{
		client: client,
	}, nil
}

func (w *RedisClient) Close() error {
	return w.client.Close()
}
