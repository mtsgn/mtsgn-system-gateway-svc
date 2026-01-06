# Build stage
FROM golang:1.23-bookworm AS builder

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN --mount=type=secret,id=read_repo_user,target=/run/secrets/READ_REPO_USER \
    --mount=type=secret,id=read_repo_token,target=/run/secrets/READ_REPO_TOKEN \
    \
    READ_REPO_USER=$(cat /run/secrets/READ_REPO_USER) && \
    READ_REPO_TOKEN=$(cat /run/secrets/READ_REPO_TOKEN) && \
    \
    git config --global \
      url."https://${READ_REPO_USER}:${READ_REPO_TOKEN}@gitea.solu-m.io".insteadOf \
      "https://gitea.solu-m.io" && \
    \
    go env -w GOPRIVATE=gitea.solu-m.io/*
    
# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" -trimpath -o /app/build/bin/main ./cmd/main.go

# Final stage
FROM alpine:latest

# Copy ONLY the CA bundle (not the whole /etc/ssl)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app

# Copy the built binary
COPY --from=builder /app/build/bin/main main
COPY --from=builder /app/config ./config

# Expose the app port
EXPOSE 8080

# Run the binary
CMD ["/app/main"]