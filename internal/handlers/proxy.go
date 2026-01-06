package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitea.solu-m.io/smart-pos/proposal-gateway-architect/internal/server"
	"gitea.solu-m.io/smart-pos/proposal-gateway-architect/pkg/config"
	rds "gitea.solu-m.io/smart-pos/proposal-gateway-architect/pkg/redis"
	"gitea.solu-m.io/smart-pos/sp-system-common-svc/common/logger"
)

type ProxyHandler struct {
	config       *config.Config
	logger       logger.ZeroLogger
	httpClient   *http.Client
	router       *server.PriorityRouter
	rateLimiter  rds.RateLimiter
	redisLimiter *rds.RedisSlidingWindowLimiter
}

func NewProxyHandler(cfg *config.Config, rateLimiter rds.RateLimiter, redisLimiter *rds.RedisSlidingWindowLimiter, logger logger.ZeroLogger) *ProxyHandler {
	router := server.NewPriorityRouter()
	for _, service := range cfg.Services {
		serviceConfig := &server.ServiceConfig{
			Name:     service.Name,
			Target:   service.Target,
			Methods:  service.Methods,
			SkipAuth: service.SkipAuth,
		}
		router.AddRoute(service.BasePath, serviceConfig)
		logger.Info(context.Background(), "Registered service", "base_path", service.BasePath, "target", serviceConfig.Target, "name", serviceConfig.Name)
	}

	timeout := 30
	if cfg.Server.Timeout > 0 {
		timeout = cfg.Server.Timeout
	}
	return &ProxyHandler{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		router:       router,
		rateLimiter:  rateLimiter,
		redisLimiter: redisLimiter,
		logger:       logger,
	}
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := p.rateLimitMiddleware(http.HandlerFunc(p.forwardRequest))
	handler(w, r)
}

func (p *ProxyHandler) isMethodAllowed(method string, allowedMethods []string) bool {
	if len(allowedMethods) == 0 {
		return true // If no methods specified, allow all
	}

	for _, allowedMethod := range allowedMethods {
		if strings.EqualFold(method, allowedMethod) {
			return true
		}
	}
	return false
}

// func (p *ProxyHandler) routeRequest(w http.ResponseWriter, r *http.Request) {
// 	// Find the best matching service
// 	ctx := context.Background()
// 	service := p.router.FindBestMatch(r.URL.Path)

// 	if service == nil {
// 		p.logger.Error(ctx, "No route found", "path", r.URL.Path)
// 		http.NotFound(w, r)
// 		return
// 	}

// 	// Check authorization
// 	if err := p.authorizationMiddleware(w, r, p.config, service); err != nil {
// 		p.logger.Error(ctx, "Authorization failed", "error", err)
// 		http.Error(w, err.Error(), http.StatusUnauthorized)
// 		return
// 	}

// 	// Check if the HTTP method is allowed
// 	if !p.isMethodAllowed(r.Method, service.Methods) {
// 		p.logger.Error(ctx, "Method not allowed", "path", r.URL.Path, "method", r.Method)
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	p.logger.Info(ctx, "Routing request", "path", r.URL.Path, "service", service.Name, "priority", service.Priority)

// 	// Check service-specific rate limit if configured
// 	// if service.RateLimit != nil && service.RateLimit.Enabled {
// 	//     clientIP := getClientIP(r)
// 	//     serviceKey := fmt.Sprintf("service:%s:%s", service.Name, clientIP)

// 	//     allowed, err := g.rateLimiter.Allow(serviceKey)
// 	//     if err != nil || !allowed {
// 	//         g.logger.Error("Service rate limit exceeded",
// 	//             zap.String("service", service.Name),
// 	//             zap.String("client_ip", clientIP))

// 	//         w.Header().Set("X-Service-RateLimit", "Exceeded")
// 	//         http.Error(w, "Service rate limit exceeded", http.StatusTooManyRequests)
// 	//         return
// 	//     }
// 	// }

// 	// Proxy the request to the target service
// 	target, err := url.Parse(service.Target)
// 	if err != nil {
// 		p.logger.Error(ctx, "Invalid target URL", "service", service.Name, "target", service.Target, "error", err)
// 		http.Error(w, "Internal server error", http.StatusInternalServerError)
// 		return
// 	}

// 	// Create reverse proxy
// 	proxy := httputil.NewSingleHostReverseProxy(target)

// 	// Modify request
// 	originalDirector := proxy.Director
// 	proxy.Director = func(req *http.Request) {
// 		originalDirector(req)

// 		// Add gateway headers
// 		req.Header.Set("X-Forwarded-For", utils.GetClientIP(r))
// 		req.Header.Set("X-Forwarded-Host", req.Host)
// 		req.Header.Set("X-Forwarded-Proto", "http")
// 		if r.TLS != nil {
// 			req.Header.Set("X-Forwarded-Proto", "https")
// 		}
// 		req.Header.Set("X-Gateway-Service", service.Name)

// 		p.logger.Debug(ctx, "Forwarding request", "to", target.String(), "path", req.URL.Path)
// 	}

// 	// Add error handling
// 	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
// 		p.logger.Error(ctx, "Proxy error", "service", service.Name, "target", target.String(), "error", err)
// 		http.Error(w, "Service unavailable", http.StatusBadGateway)
// 	}

// 	// Add response modification
// 	proxy.ModifyResponse = func(resp *http.Response) error {
// 		// Add gateway headers to response
// 		resp.Header.Set("X-Gateway-Service", service.Name)
// 		resp.Header.Set("X-Gateway-Request-ID", r.Header.Get("X-Request-ID"))

// 		// Log response status
// 		p.logger.Debug(ctx, "Service response", "service", service.Name, "status", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"))
// 		return nil
// 	}

// 	// Serve the request
// 	proxy.ServeHTTP(w, r)
// }

func (p *ProxyHandler) forwardRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	service := p.router.FindBestMatch(r.URL.Path)

	if service == nil {
		p.logger.Error(ctx, "No route found", "path", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	// Check authorization
	if err := p.authorizationMiddleware(w, r, p.config, service); err != nil {
		p.logger.Error(ctx, "Authorization failed", "error", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Check if the HTTP method is allowed
	if !p.isMethodAllowed(r.Method, service.Methods) {
		p.logger.Error(ctx, "Method not allowed", "path", r.URL.Path, "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Parse target URL
	target, err := url.Parse(service.Target)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid target URL: %v", err), http.StatusInternalServerError)
		return
	}

	// Create new request
	proxyURL := *r.URL
	proxyURL.Scheme = target.Scheme
	proxyURL.Host = target.Host

	// Create the request
	req, err := http.NewRequest(r.Method, proxyURL.String(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating request: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error forwarding request: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		fmt.Printf("Error copying response body: %v\n", err)
	}
}
