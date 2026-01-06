# Quick Start Guide

This guide will help you get the API Gateway up and running in minutes.

## Prerequisites

- Go 1.21 or higher installed
- Access to your microservices (or mock services for testing)

## Step 1: Install Dependencies

```bash
go mod download
```

Or use the Makefile:

```bash
make deps
```

## Step 2: Configure

Edit `config.yaml` to match your environment:

```yaml
server:
  port: 8080

auth:
  jwt_secret: "change-this-to-a-secure-random-string"

services:
  - name: "my-service"
    base_path: "/api/my-service"
    target: "http://localhost:3000"
    methods: ["GET", "POST", "PUT", "DELETE"]
    rate_limit: 100
```

## Step 3: Run

```bash
go run cmd/main.go
```

Or use the Makefile:

```bash
make run
```

## Step 4: Test

Check health:

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"healthy","service":"api-gateway"}
```

## Testing Authentication

For testing JWT authentication, you'll need a JWT token. You can use this example:

1. Generate a test JWT token using a tool like [jwt.io](https://jwt.io)
2. Use the same secret in your config: `your-secret-key-change-this-in-production`
3. Make a request with the token:

```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
     http://localhost:8080/api/users
```

## Testing Rate Limiting

Try making multiple requests quickly to see rate limiting in action:

```bash
for i in {1..120}; do 
  curl http://localhost:8080/api/users &
done
wait
```

You should see some requests return `429 Too Many Requests` when the limit is exceeded.

## Building for Production

```bash
make build
```

This creates an `api-gateway` executable.

## Configuration Options

### Environment Variables

You can override the config path:

```bash
CONFIG_PATH=/path/to/custom-config.yaml ./api-gateway
```

### Running in Production

For production, consider:

1. Running as a service (systemd, supervisor, etc.)
2. Using a process manager (PM2, supervisor)
3. Setting up monitoring and alerting
4. Configuring log rotation
5. Using environment variables for sensitive data

## Next Steps

- Review the `README.md` for detailed documentation
- Customize middleware in `internal/middleware/`
- Add your microservices to `config.yaml`
- Implement distributed rate limiting with Redis
- Add monitoring and metrics

## Troubleshooting

### Port Already in Use

Change the port in `config.yaml`:
```yaml
server:
  port: 8081
```

### Service Not Found Errors

Check:
1. Your microservices are running
2. The `target` URLs in config are correct
3. The `base_path` matches your request path

### Authentication Errors

Verify:
1. JWT secret matches between gateway and your auth service
2. Token is not expired
3. Token format is correct (Bearer token)

## Support

For issues or questions, please open an issue on the repository.

