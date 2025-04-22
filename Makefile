.PHONY: build test clean run-agent run-api run-cli format lint proto help dev-help install-cli install-agent release-binaries run check docs dev-init dev-env install-dev test-agent test-cli stop-dev-env dev-cli

GO_BUILD_FLAGS := -v
GO_TEST_FLAGS := -v -race
MODULE := github.com/martinshumberto/sync-manager
BINARY_DIR := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d %H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X 'main.BuildTime=$(BUILD_TIME)'"

# Detect OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	OPEN_CMD := open
	PLATFORM := darwin
else ifeq ($(UNAME_S),Linux)
	OPEN_CMD := xdg-open
	PLATFORM := linux
else
	PLATFORM := windows
	OPEN_CMD := start
endif

# Default target
all: build

# Main help target (for end users)
help:
	@echo "Sync Manager - Usage:"
	@echo "========================================"
	@echo "User Commands:"
	@echo "  build           Build all components"
	@echo "  build-agent     Build only the agent"
	@echo "  build-cli       Build only the CLI"
	@echo "  clean           Clean build artifacts"
	@echo "  run-agent       Run the agent"
	@echo "  run-cli         Run the CLI"
	@echo "  dev-cli         Run the CLI with hot reload for development"
	@echo "  install-cli     Install CLI to your system"
	@echo "  install-agent   Install agent to your system"
	@echo "  check           Run basic checks before usage"
	@echo "  release-binaries Build release binaries for all platforms"
	@echo "  docs            View documentation"
	@echo ""
	@echo "CLI Commands (after building or installing):"
	@echo "  sync-cli version         Show version information"
	@echo "  sync-cli status          Show sync status of monitored folders"
	@echo "  sync-cli start           Start the sync agent"
	@echo "  sync-cli stop            Stop the sync agent"
	@echo "  sync-cli wizard          Start interactive configuration wizard"
	@echo "  sync-cli config [get|set|reset] Manage configuration settings"
	@echo "  sync-cli folder [add|list|remove] Manage sync folders"
	@echo "  sync-cli sync [start|stop|now] Control synchronization process"
	@echo "  sync-cli monitor         Monitor sync activity"
	@echo "  sync-cli init            Initialize a new sync configuration"
	@echo ""
	@echo "For development commands, use: make dev-help"

# Developer help target
dev-help:
	@echo "Sync Manager - Developer Commands:"
	@echo "========================================"
	@echo "Development Commands:"
	@echo "  dev-init        Initialize dev environment (install dependencies & set up dev environment)"
	@echo "  dev-env         Start development environment (Docker)"
	@echo "  dev-cli         Run CLI with hot reload for development"
	@echo "  stop-dev-env    Stop development environment"
	@echo "  install-dev     Install development dependencies"
	@echo "  build-api       Build only the API"
	@echo "  run-api         Run the API"
	@echo "  format          Format code"
	@echo "  lint            Run linters"
	@echo "  test            Run tests"
	@echo "  test-agent      Run agent tests with coverage"
	@echo "  test-cli        Run CLI tests with coverage"
	@echo "  proto           Generate protobuf files"
	@echo ""
	@echo "For user commands, use: make help"

# Initialize the development project
dev-init: install-dev dev-env
	@echo "Development environment initialized successfully."
	@echo "You can now build the project with 'make build'"

# Build all components
build: build-agent build-cli

# Build the agent
build-agent:
	@echo "Building agent..."
	@mkdir -p $(BINARY_DIR)
	@go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/sync-agent ./agent/cmd

# Build the API (development only)
build-api:
	@echo "Building API..."
	@mkdir -p $(BINARY_DIR)
	@go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/sync-api ./api/cmd

# Build the CLI
build-cli:
	@echo "Building CLI..."
	@mkdir -p $(BINARY_DIR)
	@go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/sync-cli ./cli/cmd

# Run tests
test:
	@echo "Running tests..."
	@go test $(GO_TEST_FLAGS) ./...

# Run agent tests with coverage
test-agent:
	@echo "Running agent tests with coverage..."
	@mkdir -p reports
	@go test $(GO_TEST_FLAGS) -coverprofile=reports/coverage-agent.out ./agent/...
	@go tool cover -html=reports/coverage-agent.out -o reports/coverage-agent.html
	@echo "Coverage report generated at reports/coverage-agent.html"
	@$(OPEN_CMD) reports/coverage-agent.html 2>/dev/null || true

# Run CLI tests with coverage
test-cli:
	@echo "Running CLI tests with coverage..."
	@mkdir -p reports
	@go test $(GO_TEST_FLAGS) -coverprofile=reports/coverage-cli.out ./cli/...
	@go tool cover -html=reports/coverage-cli.out -o reports/coverage-cli.html
	@echo "Coverage report generated at reports/coverage-cli.html"
	@$(OPEN_CMD) reports/coverage-cli.html 2>/dev/null || true

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BINARY_DIR)
	@rm -rf reports
	@rm -rf dist

# Start development environment
dev-env:
	@echo "Starting development environment..."
	@docker-compose up -d

# Stop development environment
stop-dev-env:
	@echo "Stopping development environment..."
	@docker-compose down

# Install development dependencies
install-dev:
	@echo "Installing development dependencies..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Development dependencies installed successfully."

# Format code
format:
	@echo "Formatting code..."
	@goimports -w -local $(MODULE) .
	@go fmt ./...

# Run linters
lint:
	@echo "Running linters..."
	@go vet ./...
	@golangci-lint run ./ || echo "golangci-lint check failed"

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	@protoc --go_out=. --go-grpc_out=. ./common/proto/*.proto

# Run the agent
run-agent:
	@echo "Running agent..."
	@go run $(LDFLAGS) ./agent/cmd/main.go

# Run the API
run-api:
	@echo "Running API..."
	@go run $(LDFLAGS) ./api/cmd/main.go

# Run the CLI
run-cli:
	@echo "Running CLI..."
	@go run $(LDFLAGS) ./cli/cmd/main.go

# Run the CLI with hot reload for development
dev-cli:
	@echo "Running CLI with hot reload..."
	@cd cli && ./run.sh $(ARGS)

# Simple run - runs the CLI
run: build-cli
	@echo "Running Sync CLI..."
	@$(BINARY_DIR)/sync-cli $(ARGS)

# Install CLI to your system
install-cli: build-cli
	@echo "Installing Sync CLI to your system..."
	@cp $(BINARY_DIR)/sync-cli $(GOPATH)/bin/
	@echo "Sync CLI installed. You can now use 'sync-cli' from anywhere."

# Install agent to your system
install-agent: build-agent
	@echo "Installing Sync agent to your system..."
	@mkdir -p $(HOME)/.sync-manager/bin
	@cp $(BINARY_DIR)/sync-agent $(HOME)/.sync-manager/bin/
	@echo "Sync agent installed to $(HOME)/.sync-manager/bin/"
	@echo "You may want to set up a startup service to run the agent automatically."

# Build release binaries for all platforms
release-binaries:
	@echo "Building release binaries..."
	@mkdir -p dist
	
	@echo "Building for Linux (amd64)..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/sync-agent-linux-amd64 ./agent/cmd
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/sync-cli-linux-amd64 ./cli/cmd
	
	@echo "Building for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/sync-agent-macos-amd64 ./agent/cmd
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/sync-cli-macos-amd64 ./cli/cmd
	
	@echo "Building for macOS (arm64)..."
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/sync-agent-macos-arm64 ./agent/cmd
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/sync-cli-macos-arm64 ./cli/cmd
	
	@echo "Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/sync-agent-windows-amd64.exe ./agent/cmd
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/sync-cli-windows-amd64.exe ./cli/cmd
	
	@echo "Release binaries built in the dist/ directory"

# Run checks (simplified version for users)
check: 
	@echo "Running basic checks..."
	@go vet ./...
	@echo "Basic checks passed!"

# Developer version of check - more thorough
dev-check: format lint test
	@echo "All development checks passed!"

# Generate and open documentation (placeholder - would use godoc or similar)
docs:
	@echo "Viewing documentation..."
	@mkdir -p docs
	@echo "Documentation would be displayed here in a real implementation."
	@echo "For now, refer to the README.md and other markdown files." 