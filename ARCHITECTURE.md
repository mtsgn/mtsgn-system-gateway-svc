# Architecture Overview

## System Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────┐
│      API Gateway (Port 8080)    │
│  ┌───────────────────────────┐  │
│  │   Middleware Stack        │  │
│  │  1. CORS                  │  │
│  │  2. Logger                │  │
│  │  3. Rate Limiter          │  │
│  │  4. Authorization (JWT)   │  │
│  └──────┬────────────────────┘  │
│         │                        │
│         ▼                        │
│  ┌─────────────┐                │
│  │   Proxy     │                │
│  │   Handler   │                │
│  └──────┬──────┘                │
└─────────┼────────────────────────┘
          │
          ├──────────┬──────────┐
          ▼          ▼          ▼
    ┌────────┐ ┌────────┐ ┌────────┐
    │ User   │ │ Order  │ │Product │
    │Service │ │Service │ │Service │
    │:8081   │ │:8082   │ │:8083   │
    └────────┘ └────────┘ └────────┘
```

## Component Details

### 1. Entry Point (`cmd/main.go`)

- Initializes configuration
- Sets up HTTP server
- Chains middleware in order:
  1. Logger
  2. Rate Limiter
  3. CORS
  4. Authorization
  5. Proxy Handler

### 2. Configuration (`config/config.go`)

Manages configuration using Viper:
- **Server config**: Port, host settings
- **Auth config**: JWT secret
- **Rate limit config**: Global limits and time windows
- **Service config**: Microservice routing rules

### 3. Middleware Stack

#### Logger (`internal/middleware/logger.go`)
- Logs all incoming requests
- Captures: method, path, remote address, status code, duration

#### Rate Limiter (`internal/middleware/ratelimit.go`)
- Per-client rate limiting using token bucket algorithm
- Configurable per-service or globally
- Automatic cleanup of stale entries
- Returns 429 when limit exceeded

#### CORS (`internal/middleware/cors.go`)
- Enables cross-origin requests
- Handles preflight OPTIONS requests
- Configurable headers and methods

#### Authorization (`internal/middleware/auth.go`)
- Validates JWT tokens
- Extracts user claims
- Adds user context to downstream requests

### 4. Proxy Handler (`internal/handlers/proxy.go`)

- Routes requests to appropriate microservice
- URL rewriting and path manipulation
- Method filtering per service
- Header forwarding
- Timeout handling (30s default)

## Request Flow

1. **Client** sends request to gateway
2. **CORS** middleware adds CORS headers
3. **Logger** captures request metadata
4. **Rate Limiter** checks and updates client rate limit
5. **Authorization** validates JWT (if required)
6. **Proxy Handler** routes to target microservice
7. **Response** flows back through middleware in reverse

## Configuration Schema

```yaml
server:
  port: 8080                    # Gateway listening port

auth:
  jwt_secret: "..."             # JWT validation key

rate_limit:
  default_limit: 100            # Requests per minute
  default_window: 60            # Time window (seconds)

services:
  - name: "service-name"        # Service identifier
    base_path: "/api/path"      # URL prefix for routing
    target: "http://host:port"  # Target microservice
    methods: ["GET", "POST"]    # Allowed methods
    rate_limit: 50              # Override global limit
```

## Security Features

### JWT Authentication
- HMAC-based signing (HS256)
- Token validation before proxying
- User context extraction and forwarding

### Rate Limiting
- Per-IP rate limiting
- Configurable limits per service
- Burst handling
- Protection against DDoS

### CORS
- Configurable origin policy
- Method whitelisting
- Header control

## Performance Considerations

### In-Memory Rate Limiting
- Fast, low-latency decisions
- Suitable for single-instance deployments
- Automatic garbage collection

### Future: Redis-Based Rate Limiting
- Distributed rate limiting
- Shared state across instances
- Better for multi-instance deployments

### Request Timeouts
- 30-second default timeout
- Prevents resource exhaustion
- Graceful error handling

## Extension Points

### Adding New Middleware

Create a new file in `internal/middleware/`:

```go
func MyMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Pre-processing
        next.ServeHTTP(w, r)
        // Post-processing
    })
}
```

Then chain in `cmd/main.go`:

```go
handler := middleware.MyMiddleware(
    middleware.Logger(proxyHandler)
)
```

### Adding New Service

Add to `config.yaml`:

```yaml
services:
  - name: "new-service"
    base_path: "/api/new"
    target: "http://localhost:8090"
    methods: ["GET", "POST"]
    rate_limit: 200
```

## Monitoring and Observability

### Current
- Request logging
- Health check endpoint
- Error responses

### Future Enhancements
- Prometheus metrics
- Distributed tracing (OpenTelemetry)
- Request/response size tracking
- Latency percentiles
- Error rate tracking

## Deployment Strategies

### Single Instance
- Direct deployment
- In-memory rate limiting
- Suitable for small scale

### Load Balanced
- Multiple gateway instances
- Shared rate limiting (Redis)
- Session affinity if needed

### Containerized
- Docker deployment
- Kubernetes support
- Service mesh integration

## Best Practices

1. **Configuration Management**
   - Use environment variables for secrets
   - Separate configs per environment
   - Version control non-sensitive configs

2. **Security**
   - Rotate JWT secrets regularly
   - Implement HTTPS/TLS
   - Monitor suspicious patterns

3. **Performance**
   - Adjust rate limits based on capacity
   - Implement caching where appropriate
   - Monitor resource usage

4. **Reliability**
   - Health checks for backend services
   - Circuit breaker pattern
   - Graceful degradation

5. **Monitoring**
   - Log aggregation
   - Alerting on errors
   - Traffic pattern analysis

