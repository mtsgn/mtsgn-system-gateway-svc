package handlers

import (
	"context"
	"fmt"
	"net/http"

	"gitea.solu-m.io/smart-pos/proposal-gateway-architect/pkg/utils"
)

// ==================== Rate Limiting Middleware ====================
func (p *ProxyHandler) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ctx context.Context = r.Context()
		clientIP := utils.GetClientIP(r)
		path := r.URL.Path
		rateLimitKey := fmt.Sprintf("%s:%s", clientIP, path)

		allowed, err := p.rateLimiter.Allow(ctx, rateLimitKey)
		if err != nil {
			p.logger.Error(ctx, "Rate limiter error", "client_ip", clientIP, "path", path, "error", err)
			next(w, r)
			return
		}

		// p.logger.Info(ctx, "Rate limit check", "client_ip", clientIP, "path", path, "allowed", allowed)

		if !allowed {
			w.Header().Set("X-RateLimit-Limit", "Exceeded")
			w.Header().Set("Retry-After", "1")
			p.logger.Error(ctx, "Rate limit exceeded", "client_ip", clientIP, "path", path)

			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
