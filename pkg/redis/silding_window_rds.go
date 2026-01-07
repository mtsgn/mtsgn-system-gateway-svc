package rds

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtsgn/mtsgn-system-gateway-svc/pkg/config"
	"github.com/redis/go-redis/v9"
)

type RedisSlidingWindowLimiter struct {
	client  *redis.Client
	prefix  string
	limit   int
	window  time.Duration
	counter atomic.Uint64 // For generating unique members
}

func NewRedisSlidingWindowLimiter(cfg *config.RedisConfig, limit int, window time.Duration) (*RedisSlidingWindowLimiter, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       0,
	})
	if limit <= 0 {
		limit = 100
	}
	if window <= 0 {
		window = 30 * time.Second
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisSlidingWindowLimiter{
		client:  client,
		prefix:  "rate_limit",
		limit:   limit,
		window:  window,
		counter: atomic.Uint64{},
	}, nil
}

func (l *RedisSlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	// Validate input
	if key == "" {
		return false, fmt.Errorf("rate limit key cannot be empty")
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	now := time.Now().UnixMilli()
	windowStart := now - int64(l.window/time.Millisecond)

	redisKey := fmt.Sprintf("%s:%s", l.prefix, key)

	member := fmt.Sprintf("%d:%d:%d", now, time.Now().UnixNano()%1e6, l.counter.Add(1))

	pipe := l.client.Pipeline()

	pipe.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now),
		Member: member,
	})

	pipe.ZRemRangeByScore(ctx, redisKey, "-inf", fmt.Sprintf("%d", windowStart))

	countCmd := pipe.ZCard(ctx, redisKey)

	pipe.Expire(ctx, redisKey, l.window+time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("redis pipeline exec failed: %w", err)
	}

	count, err := countCmd.Result()
	if err != nil {
		return false, fmt.Errorf("redis zcard failed: %w", err)
	}

	return count <= int64(l.limit), nil
}

// -------------------- local rate limiter ---------------------------- //
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type TokenBucketLimiter struct {
	mu         sync.RWMutex
	buckets    map[string]*tokenBucket
	rate       float64 // tokens per second
	capacity   int
	cleanupTTL time.Duration
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

func NewTokenBucketLimiter(rps, burst int) *TokenBucketLimiter {
	limiter := &TokenBucketLimiter{
		buckets:    make(map[string]*tokenBucket),
		rate:       float64(rps),
		capacity:   burst,
		cleanupTTL: time.Hour,
	}
	go limiter.cleanupStaleBuckets()
	return limiter
}

func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	// Validate input
	if key == "" {
		return false, fmt.Errorf("rate limit key cannot be empty")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	bucket, exists := l.buckets[key]

	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(l.capacity) - 1,
			lastRefill: now,
		}
		l.buckets[key] = bucket
		return true, nil
	}

	// Refill tokens
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	refill := elapsed * l.rate
	bucket.tokens = min(float64(l.capacity), bucket.tokens+refill)
	bucket.lastRefill = now

	// Check if we can take a token
	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, nil
	}

	return false, nil
}

func (l *TokenBucketLimiter) cleanupStaleBuckets() {
	ticker := time.NewTicker(30 * time.Minute)
	for range ticker.C {
		l.mu.Lock()
		cutoff := time.Now().Add(-l.cleanupTTL)
		for key, bucket := range l.buckets {
			if bucket.lastRefill.Before(cutoff) {
				delete(l.buckets, key)
			}
		}
		l.mu.Unlock()
	}
}
