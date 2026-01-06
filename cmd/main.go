package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"gitea.solu-m.io/smart-pos/proposal-gateway-architect/internal/handlers"
	"gitea.solu-m.io/smart-pos/proposal-gateway-architect/internal/middleware"
	"gitea.solu-m.io/smart-pos/proposal-gateway-architect/pkg/config"
	rds "gitea.solu-m.io/smart-pos/proposal-gateway-architect/pkg/redis"
	"gitea.solu-m.io/smart-pos/sp-system-common-svc/common/logger"
)

var (
	confPath = flag.String("config", "./config/app.development.yaml", "config file path")
)

func main() {

	flag.Parse()
	ctx := context.Background()
	// Load configuration
	cfg, err := config.LoadConfig(*confPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// --------------- Logger ---------------------------- //
	zeroLogger, err := logger.NewLogger(logger.Config{
		Level:         logger.Level(cfg.LogLevel),
		Service:       "sp-gateway-architect",
		DisableCaller: true,
	})
	if err != nil {
		log.Println("Failed to initialize zerolog", err)
		return
	}

	// --------------- Setup Local Rate Limiter ---------------------------- //
	localLimiter := rds.NewTokenBucketLimiter(
		cfg.RateLimit.RequestsPerSecond,
		cfg.RateLimit.Burst,
	)
	var redisLimiter *rds.RedisSlidingWindowLimiter = nil
	var rateLimiter rds.RateLimiter = localLimiter
	// Try to initialize Redis limiter if configured
	if cfg.Redis.Host != "" {
		redisLimiter, err = rds.NewRedisSlidingWindowLimiter(
			&cfg.Redis,
			cfg.RateLimit.RequestsPerSecond,
			time.Second,
		)
		if err != nil {
			zeroLogger.Error(ctx, "Failed to connect to Redis, using local limiter", err)
		} else {
			rateLimiter = redisLimiter
		}
	}

	zeroLogger.Info(ctx, "API Gateway initialized", "rate_limiter", fmt.Sprintf("%T", rateLimiter))
	// Initialize handlers
	proxyHandler := handlers.NewProxyHandler(cfg, rateLimiter, redisLimiter, *zeroLogger)

	// Setup HTTP server with middlewares
	mux := http.NewServeMux()

	// Apply global middleware
	handler := middleware.Logger(
		middleware.CORS(
			proxyHandler,
		),
		*zeroLogger,
	)

	// Register routes
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"api-gateway"}`))
	})

	mux.Handle("/", handler)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	zeroLogger.Info(ctx, "API Gateway starting on", "addr", addr)
	zeroLogger.Info(ctx, "Configured services", "count", len(cfg.Services))
	if err := http.ListenAndServe(addr, mux); err != nil {
		zeroLogger.Error(ctx, "Server failed to start", "error", err)
	}
}
