BINARY_NAME=nav-tracker
BUILD_DIR=build
PORT=8080

.PHONY: help build clean test run docker-build docker-run lint

help:
	@echo "Navigation Tracker - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✓ Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

test:
	@echo "Running tests..."
	go test -v ./...
	@echo "✓ Tests completed"

test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

lint:
	@echo "Running linter..."
	golangci-lint run || echo "Linter not installed, skipping..."

run: build
	@echo "Starting $(BINARY_NAME) on port $(PORT)..."
	@echo "Press Ctrl+C to stop"
	./$(BUILD_DIR)/$(BINARY_NAME)

run-dev:
	@echo "Starting $(BINARY_NAME) in development mode..."
	go run .

docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME) .
	@echo "✓ Docker image built: $(BINARY_NAME)"

docker-run: docker-build
	@echo "Running with Docker..."
	docker run -p $(PORT):8080 --rm $(BINARY_NAME)

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean
	@echo "✓ Clean completed"

fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Code formatted"

deps:
	@echo "Downloading dependencies..."
	go mod tidy
	go mod download
	@echo "✓ Dependencies downloaded"

version:
	@echo "Go version: $$(go version)"
	@echo "Git commit: $$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"

.DEFAULT_GOAL := help