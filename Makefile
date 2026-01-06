.PHONY: build run clean test deps

# Build the application
build:
	@echo "Building API Gateway..."
	@go build -o api-gateway ./cmd/main.go

# Run the application
run:
	@echo "Running API Gateway..."
	@go run cmd/main.go

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f api-gateway

# Format code
fmt:
	@go fmt ./...

# Lint code
lint:
	@go vet ./...

# Get dependencies
tidy:
	@go mod tidy

.DEFAULT_GOAL := build

