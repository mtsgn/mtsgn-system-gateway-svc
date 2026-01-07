package rds

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/mtsgn/mtsgn-system-gateway-svc/pkg/config"
	"github.com/redis/go-redis/v9"
)

// parsePort converts a port string to int
func parsePort(portStr string) int {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 6379 // default
	}
	return port
}

// setupTestRedis creates a miniredis instance and returns a limiter configured to use it
func setupTestRedis(t *testing.T) (*RedisSlidingWindowLimiter, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to ping redis: %v", err)
	}

	limiter := &RedisSlidingWindowLimiter{
		client: client,
		prefix: "rate_limit",
		limit:  5,
		window: 1 * time.Second,
	}

	return limiter, mr
}

func TestNewRedisSlidingWindowLimiter(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	tests := []struct {
		name    string
		cfg     *config.RedisConfig
		limit   int
		window  time.Duration
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "",
			},
			limit:   10,
			window:  1 * time.Second,
			wantErr: false,
		},
		{
			name: "zero limit defaults to 100",
			cfg: &config.RedisConfig{
				Host: mr.Host(),
				Port: parsePort(mr.Port()),
			},
			limit:   0,
			window:  1 * time.Second,
			wantErr: false,
		},
		{
			name: "zero window defaults to 30s",
			cfg: &config.RedisConfig{
				Host: mr.Host(),
				Port: parsePort(mr.Port()),
			},
			limit:   10,
			window:  0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override with miniredis address for valid configs
			if tt.name == "valid config" {
				tt.cfg.Host = mr.Host()
				tt.cfg.Port = parsePort(mr.Port())
			}

			limiter, err := NewRedisSlidingWindowLimiter(tt.cfg, tt.limit, tt.window)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRedisSlidingWindowLimiter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && limiter == nil {
				t.Error("NewRedisSlidingWindowLimiter() returned nil limiter")
				return
			}

			if !tt.wantErr {
				// Check defaults
				if tt.limit == 0 && limiter.limit != 100 {
					t.Errorf("Expected default limit 100, got %d", limiter.limit)
				}
				if tt.window == 0 && limiter.window != 30*time.Second {
					t.Errorf("Expected default window 30s, got %v", limiter.window)
				}

				limiter.client.Close()
			}
		})
	}
}

func TestRedisSlidingWindowLimiter_Allow(t *testing.T) {
	limiter, mr := setupTestRedis(t)
	defer mr.Close()
	defer limiter.client.Close()

	t.Run("allows requests within limit", func(t *testing.T) {
		key := "test_key_1"
		// Make 5 requests (limit is 5)
		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow(context.Background(), key)
			if err != nil {
				t.Fatalf("Allow() error = %v", err)
			}
			if !allowed {
				t.Errorf("Allow() = false, want true for request %d", i+1)
			}
			// Small delay to ensure different timestamps
			time.Sleep(1 * time.Millisecond)
		}
	})

	t.Run("rejects requests over limit", func(t *testing.T) {
		key := "test_key_2"
		// Make 5 requests (all should be allowed)
		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow(context.Background(), key)
			if err != nil {
				t.Fatalf("Allow() error = %v", err)
			}
			if !allowed {
				t.Errorf("Allow() = false, want true for request %d", i+1)
			}
			// Small delay to ensure different timestamps
			time.Sleep(1 * time.Millisecond)
		}

		// 6th request should be rejected
		time.Sleep(1 * time.Millisecond)
		allowed, err := limiter.Allow(context.Background(), key)
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if allowed {
			t.Error("Allow() = true, want false for request over limit")
		}
	})

	t.Run("different keys have separate limits", func(t *testing.T) {
		key1 := "test_key_3"
		key2 := "test_key_4"

		// Fill up key1
		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow(context.Background(), key1)
			if err != nil {
				t.Fatalf("Allow() error = %v", err)
			}
			if !allowed {
				t.Errorf("Allow() = false, want true for key1 request %d", i+1)
			}
			time.Sleep(1 * time.Millisecond)
		}

		// key2 should still have full limit
		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow(context.Background(), key2)
			if err != nil {
				t.Fatalf("Allow() error = %v", err)
			}
			if !allowed {
				t.Errorf("Allow() = false, want true for key2 request %d", i+1)
			}
			time.Sleep(1 * time.Millisecond)
		}
	})

	t.Run("sliding window allows requests after window expires", func(t *testing.T) {
		key := "test_key_5"
		// Fill up the limit
		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow(context.Background(), key)
			if err != nil {
				t.Fatalf("Allow() error = %v", err)
			}
			if !allowed {
				t.Errorf("Allow() = false, want true for request %d", i+1)
			}
			time.Sleep(1 * time.Millisecond)
		}

		// 6th request should be rejected
		time.Sleep(1 * time.Millisecond)
		allowed, err := limiter.Allow(context.Background(), key)
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if allowed {
			t.Error("Allow() = true, want false for request over limit")
		}

		// Wait for window to expire (window is 1 second, wait 1.1 seconds)
		// Fast-forward Redis time and wait real time
		mr.FastForward(1100 * time.Millisecond)
		time.Sleep(1100 * time.Millisecond)

		// Now requests should be allowed again (old entries expired)
		allowed, err = limiter.Allow(context.Background(), key)
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if !allowed {
			t.Error("Allow() = false, want true after window expired")
		}
	})

	t.Run("concurrent requests", func(t *testing.T) {
		key := "test_key_6"
		const numGoroutines = 10
		const requestsPerGoroutine = 1

		results := make(chan bool, numGoroutines*requestsPerGoroutine)
		var wg sync.WaitGroup

		// Use a barrier to start all goroutines at roughly the same time
		start := make(chan struct{})

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				<-start // Wait for start signal
				// Add small delay based on ID to ensure different timestamps
				time.Sleep(time.Duration(id) * time.Millisecond)
				allowed, err := limiter.Allow(context.Background(), key)
				if err != nil {
					t.Errorf("Allow() error = %v", err)
					return
				}
				results <- allowed
			}(i)
		}

		// Start all goroutines
		close(start)
		wg.Wait()
		close(results)

		allowedCount := 0
		for allowed := range results {
			if allowed {
				allowedCount++
			}
		}

		// With limit of 5, we should have at most 5 allowed requests
		// (10 total requests, but only 5 should be allowed)
		if allowedCount > 5 {
			t.Errorf("Expected at most 5 allowed requests, got %d", allowedCount)
		}
		// At least some requests should be allowed
		if allowedCount == 0 {
			t.Error("Expected at least some allowed requests, got 0")
		}
	})
}

func TestRedisSlidingWindowLimiter_Allow_EdgeCases(t *testing.T) {
	limiter, mr := setupTestRedis(t)
	defer mr.Close()
	defer limiter.client.Close()

	t.Run("empty key", func(t *testing.T) {
		allowed, err := limiter.Allow(context.Background(), "")
		if err == nil {
			t.Error("Allow() expected error for empty key, got nil")
		}
		if allowed {
			t.Error("Allow() = true, want false for empty key")
		}
		if err != nil && err.Error() != "rate limit key cannot be empty" {
			t.Errorf("Allow() error = %v, want 'rate limit key cannot be empty'", err)
		}
	})

	t.Run("special characters in key", func(t *testing.T) {
		key := "test:key:with:colons"
		allowed, err := limiter.Allow(context.Background(), key)
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if !allowed {
			t.Error("Allow() = false, want true for key with special characters")
		}
	})

	t.Run("very long key", func(t *testing.T) {
		key := make([]byte, 1000)
		for i := range key {
			key[i] = 'a'
		}
		allowed, err := limiter.Allow(context.Background(), string(key))
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if !allowed {
			t.Error("Allow() = false, want true for very long key")
		}
	})
}

func TestRedisSlidingWindowLimiter_Allow_RedisFailure(t *testing.T) {
	limiter, mr := setupTestRedis(t)
	defer limiter.client.Close()

	// Close Redis to simulate failure
	mr.Close()

	allowed, err := limiter.Allow(context.Background(), "test_key")
	if err == nil {
		t.Error("Allow() error = nil, want error when Redis is down")
	}
	if allowed {
		t.Error("Allow() = true, want false when Redis is down")
	}
}

func BenchmarkRedisSlidingWindowLimiter_Allow(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0,
	})
	defer client.Close()

	limiter := &RedisSlidingWindowLimiter{
		client: client,
		prefix: "rate_limit",
		limit:  1000,
		window: 1 * time.Second,
	}

	key := "bench_key"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = limiter.Allow(context.Background(), key)
		}
	})
}
