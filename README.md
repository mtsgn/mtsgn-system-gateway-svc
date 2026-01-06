# API Gateway Service

A powerful, production-ready API Gateway service built with Go for microservices architecture. This gateway provides essential features like authentication, rate limiting, logging, and request routing.

## Features

- 🔐 **JWT Authentication** - Secure token-based authentication
- 🚦 **Rate Limiting** - Configurable per-service rate limiting to prevent abuse
- 📝 **Request Logging** - Comprehensive logging of all API requests
- 🎯 **Request Routing** - Intelligent routing to backend microservices
- 🌐 **CORS Support** - Cross-origin resource sharing support
- ⚙️ **Configuration** - YAML-based configuration system
- 🏥 **Health Checks** - Built-in health check endpoint
- 🎨 **Method Filtering** - Restrict HTTP methods per service

## Architecture

```
Client → API Gateway → Microservices
         ├─ Auth
         ├─ Rate Limit
         ├─ Logging
         └─ Proxy
```

## Installation

1. Clone the repository:
```bash
git clone https://gitea.solu-m.io/smart-pos/proposal-gateway-architect
cd api-gateway
```

2. Install dependencies:
```bash
go mod download
```

3. Build the service:
```bash
go build -o api-gateway ./cmd/main.go
```

## Configuration

The API Gateway is configured via `config.yaml`. Here's an example:

```yaml
server:
  port: 8080

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

auth:
  jwt_secret: "your-secret-key-change-this-in-production"

rate_limit:
  default_limit: 100
  default_window: 60

services:
  - name: "user-service"
    base_path: "/api/users"
    target: "http://localhost:8081"
    methods: ["GET", "POST", "PUT", "DELETE", "PATCH"]
    rate_limit: 60
  
  - name: "order-service"
    base_path: "/api/orders"
    target: "http://localhost:8082"
    methods: ["GET", "POST", "PUT", "DELETE"]
    rate_limit: 30
```

### Configuration Fields

- **server.port**: The port the API Gateway will listen on
- **redis**: Redis configuration (for future distributed rate limiting)
- **auth.jwt_secret**: Secret key for JWT token validation
- **rate_limit.default_limit**: Default requests per minute
- **rate_limit.default_window**: Time window in seconds
- **services**: Array of backend services to route to
  - **name**: Service identifier
  - **base_path**: URL path prefix for routing
  - **target**: Target microservice URL
  - **methods**: Allowed HTTP methods
  - **rate_limit**: Per-service rate limit override

## Usage

### Starting the Service

```bash
./api-gateway
```

or

```bash
go run cmd/main.go
```

### Environment Variables

You can specify a custom config path:

```bash
CONFIG_PATH=/path/to/config.yaml ./api-gateway
```

## API Endpoints

### Health Check

```bash
GET /health

Response:
{
  "status": "healthy",
  "service": "api-gateway"
}
```

### Routing to Microservices

All requests to paths matching your configured `base_path` will be proxied to the corresponding microservice.

For example:
- `GET /api/users` → proxied to user-service
- `POST /api/orders` → proxied to order-service
- `GET /api/products` → proxied to product-service

## Authentication

The API Gateway uses JWT (JSON Web Tokens) for authentication.

### Protected Endpoints

Include the JWT token in the `Authorization` header:

```bash
curl -H "Authorization: Bearer <your-jwt-token>" \
     http://localhost:8080/api/users
```

### JWT Token Format

Your microservice should generate JWT tokens with the following claims:

```json
{
  "user_id": "123",
  "username": "john_doe",
  "exp": 1234567890
}
```

The gateway will add these headers to the proxied request:
- `X-User-ID`: User ID from token
- `X-Username`: Username from token

## Rate Limiting

Rate limiting is applied per-client (IP address). When the limit is exceeded, the gateway returns:

```
HTTP 429 Too Many Requests

Rate limit exceeded. Allow X requests per Y seconds with burst Z
```

### Example Rate Limit Responses

- Global: 100 requests per 60 seconds
- User service: 60 requests per 60 seconds
- Order service: 30 requests per 60 seconds

## Logging

All requests are logged in the following format:

```
[timestamp] METHOD PATH REMOTE_ADDR STATUS_CODE DURATION
```

Example:
```
2024/01/15 10:30:45 GET /api/users 192.168.1.1:12345 200 12.5ms
```

## CORS

CORS is enabled by default with the following settings:

- **Access-Control-Allow-Origin**: `*`
- **Access-Control-Allow-Methods**: `GET, POST, PUT, DELETE, OPTIONS, PATCH`
- **Access-Control-Allow-Headers**: `Content-Type, Authorization, X-Requested-With`
- **Access-Control-Max-Age**: `3600`

## Development

### Project Structure

```
.
├── cmd/
│   └── main.go              # Application entry point
├── config/
│   └── config.go            # Configuration management
├── internal/
│   ├── handlers/
│   │   └── proxy.go         # Proxy request handler
│   └── middleware/
│       ├── auth.go          # JWT authentication
│       ├── cors.go          # CORS handling
│       ├── logger.go        # Request logging
│       └── ratelimit.go    # Rate limiting
├── config.yaml              # Configuration file
├── go.mod                   # Go dependencies
└── README.md                # This file
```

### Adding a New Middleware

1. Create a new file in `internal/middleware/`
2. Implement the middleware function:

```go
func MyMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Your middleware logic
        next.ServeHTTP(w, r)
    })
}
```

3. Chain it in `cmd/main.go`:

```go
handler := middleware.MyMiddleware(
    middleware.Logger(proxyHandler)
)
```

## Testing

Run the service locally:

```bash
# Start backend services (simulated)
# You need to have your microservices running

# Start the API Gateway
go run cmd/main.go

# Test health endpoint
curl http://localhost:8080/health

# Test routing
curl http://localhost:8080/api/users
```

## Production Deployment

### Security Considerations

1. **Change JWT Secret**: Update the `jwt_secret` in production
2. **Use Environment Variables**: Consider loading sensitive config from environment
3. **Enable HTTPS**: Use TLS for production
4. **Monitor Rate Limits**: Adjust based on your traffic patterns
5. **Log Storage**: Implement log aggregation and monitoring

### Docker Deployment

Create a `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o api-gateway ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api-gateway .
COPY --from=builder /app/config.yaml .
CMD ["./api-gateway"]
```

Build and run:

```bash
docker build -t api-gateway .
docker run -p 8080:8080 api-gateway
```

## Future Enhancements

- [ ] Redis-based distributed rate limiting
- [ ] Load balancing across multiple service instances
- [ ] Circuit breaker pattern
- [ ] Request/response transformation
- [ ] API key authentication
- [ ] Metrics and monitoring (Prometheus integration)
- [ ] Distributed tracing support
- [ ] WebSocket support

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

gitea.solu-m.io/smart-pos/proposal-gateway-architectpham

