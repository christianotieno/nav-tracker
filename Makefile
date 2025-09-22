BINARY_NAME=nav-tracker
BUILD_DIR=build
PORT=8080

.PHONY: all help build clean test test-coverage lint run run-dev docker-build docker-run fmt deps version

clean: ## Remove build artifacts and coverage files
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html

fmt: ## Format and vet the code
	@echo "Formatting and vetting..."
	go fmt ./...
	go vet ./...

deps: ## Sync Go module dependencies
	@echo "Tidying and downloading deps..."
	go mod tidy
	go mod download

version: ## Print version info
	@echo "Version: $$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')"

docker-run: ## Run the Docker image locally
	@echo "Running Docker image on port $(PORT)..."
	docker run --rm -p $(PORT):$(PORT) --name $(BINARY_NAME) $(BINARY_NAME)

all: build test lint ## Build, test, and lint

help:
	@echo "Navigation Tracker - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✓ Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

test: ## Run all tests
	@echo "Running tests..."
	go test -v ./...
	@echo "✓ Tests completed"

test-coverage: ## Run tests with coverage and generate HTML report
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
docker-build: ## Build the Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME) .
	@echo "✓ Docker image built: $(BINARY_NAME)"
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  WARNING: golangci-lint not installed, skipping linting"; \
		echo "   Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

run: build ## Build and run the application
	@echo "Starting $(BINARY_NAME) on port $(PORT)..."
	@echo "Press Ctrl+C to stop"
	./$(BUILD_DIR)/$(BINARY_NAME)

run-dev: ## Run the application in development mode
	@echo "Starting $(BINARY_NAME) in development mode..."
	go run .

docker-build: ## Build the Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME) .
	@echo "✓