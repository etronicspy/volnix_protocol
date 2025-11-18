# Volnix Protocol Makefile

# Build variables
BINARY_NAME=volnixd
VERSION=0.1.0-alpha
BUILD_DIR=./build
GO_VERSION=1.21

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
PURPLE=\033[0;35m
CYAN=\033[0;36m
NC=\033[0m # No Color

.PHONY: help build build-standalone install test clean run init start status keys version

# Default target
all: build build-standalone

help: ## Show this help message
	@echo "$(CYAN)🚀 Volnix Protocol - Build Commands$(NC)"
	@echo "$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "$(YELLOW)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the volnixd binary
	@echo "$(GREEN)🔨 Building Volnix Protocol...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/volnixd
	@echo "$(GREEN)✅ Build completed: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-standalone: ## Build the volnixd-standalone binary
	@echo "$(GREEN)🔨 Building Volnix Protocol Standalone...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/volnixd-standalone ./cmd/volnixd-standalone
	@echo "$(GREEN)✅ Standalone build completed: $(BUILD_DIR)/volnixd-standalone$(NC)"

build-linux: ## Build for Linux
	@echo "$(GREEN)🔨 Building for Linux...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./cmd/volnixd
	@echo "$(GREEN)✅ Linux build completed: $(BUILD_DIR)/$(BINARY_NAME)-linux$(NC)"

build-windows: ## Build for Windows
	@echo "$(GREEN)🔨 Building for Windows...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME).exe ./cmd/volnixd
	@echo "$(GREEN)✅ Windows build completed: $(BUILD_DIR)/$(BINARY_NAME).exe$(NC)"

build-darwin: ## Build for macOS
	@echo "$(GREEN)🔨 Building for macOS...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin ./cmd/volnixd
	@echo "$(GREEN)✅ macOS build completed: $(BUILD_DIR)/$(BINARY_NAME)-darwin$(NC)"

build-all: build-linux build-windows build-darwin ## Build for all platforms
	@echo "$(GREEN)🎉 All platform builds completed!$(NC)"

install: build ## Install the binary to GOPATH/bin
	@echo "$(GREEN)📦 Installing $(BINARY_NAME)...$(NC)"
	@go install ./cmd/volnixd
	@echo "$(GREEN)✅ Installation completed$(NC)"

test: ## Run all tests
	@echo "$(BLUE)🧪 Running tests...$(NC)"
	@go test ./... -v

test-unit: ## Run unit tests only
	@echo "$(BLUE)🧪 Running unit tests...$(NC)"
	@go test ./x/*/keeper -v
	@go test ./x/*/types -v

test-integration: ## Run integration tests
	@echo "$(BLUE)🧪 Running integration tests...$(NC)"
	@go test ./tests -v -run Integration

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)🧪 Running tests with coverage...$(NC)"
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report generated: coverage.html$(NC)"
	@echo "$(CYAN)📊 See TEST_COVERAGE_REPORT.md for detailed analysis$(NC)"

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f volnixd-standalone volnixd-standalone.exe
	@echo "$(GREEN)✅ Clean completed$(NC)"

deps: ## Download and tidy dependencies
	@echo "$(BLUE)📦 Managing dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

fmt: ## Format Go code
	@echo "$(BLUE)🎨 Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✅ Code formatted$(NC)"

lint: ## Run linter
	@echo "$(BLUE)🔍 Running linter...$(NC)"
	@golangci-lint run
	@echo "$(GREEN)✅ Linting completed$(NC)"

# Node management commands
init: build ## Initialize a new node
	@echo "$(PURPLE)🚀 Initializing Volnix node...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME) init testnode

start: build ## Start the node
	@echo "$(PURPLE)🚀 Starting Volnix node...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME) start

status: build ## Show node status
	@echo "$(PURPLE)📊 Checking node status...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME) status

version: build ## Show version information
	@$(BUILD_DIR)/$(BINARY_NAME) version

keys-add: build ## Add a new key (usage: make keys-add NAME=mykey)
	@echo "$(PURPLE)🔑 Adding new key: $(NAME)$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME) keys add $(NAME)

keys-list: build ## List all keys
	@echo "$(PURPLE)🔑 Listing keys...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME) keys list

# Development commands
dev-setup: deps fmt ## Setup development environment
	@echo "$(GREEN)🛠️  Development environment setup completed$(NC)"

dev-test: fmt test ## Format code and run tests
	@echo "$(GREEN)✅ Development testing completed$(NC)"

dev-build: fmt build ## Format code and build
	@echo "$(GREEN)✅ Development build completed$(NC)"

# Testnet commands
testnet-start: build ## Start testnet (Windows)
	@echo "$(CYAN)🌐 Starting testnet...$(NC)"
	@cd testnet && start.bat

testnet-start-unix: build ## Start testnet (Linux/macOS)
	@echo "$(CYAN)🌐 Starting testnet...$(NC)"
	@cd testnet && chmod +x start.sh && ./start.sh

# Docker commands (future)
docker-build: ## Build Docker image
	@echo "$(BLUE)🐳 Building Docker image...$(NC)"
	@echo "$(YELLOW)⚠️  Docker support coming soon$(NC)"

docker-run: ## Run in Docker container
	@echo "$(BLUE)🐳 Running in Docker...$(NC)"
	@echo "$(YELLOW)⚠️  Docker support coming soon$(NC)"

# Release commands
release: clean build-all test ## Prepare release build
	@echo "$(GREEN)🎉 Release build completed!$(NC)"
	@echo "$(GREEN)📦 Binaries ready:$(NC)"
	@ls -la $(BUILD_DIR)/

# Info commands
info: ## Show project information
	@echo "$(CYAN)🚀 Volnix Protocol$(NC)"
	@echo "$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(YELLOW)Version:$(NC) $(VERSION)"
	@echo "$(YELLOW)Go Version:$(NC) $(shell go version)"
	@echo "$(YELLOW)Build Target:$(NC) $(BINARY_NAME)"
	@echo ""
	@echo "$(BLUE)🏗️  Architecture:$(NC)"
	@echo "  • Cosmos SDK v0.53.x"
	@echo "  • CometBFT v0.38.x"
	@echo "  • GoLevelDB storage"
	@echo ""
	@echo "$(BLUE)📦 Modules:$(NC)"
	@echo "  • ident - Identity & ZKP verification"
	@echo "  • lizenz - LZN license management"
	@echo "  • anteil - ANT internal market"
	@echo "  • consensus - PoVB consensus"
	@echo ""
	@echo "$(BLUE)🌟 Features:$(NC)"
	@echo "  • Hybrid PoVB Consensus"
	@echo "  • ZKP Identity Verification"
	@echo "  • Three-tier Economy (WRT/LZN/ANT)"
	@echo "  • High Performance (10,000+ TPS)"
	@echo ""
	@echo "$(BLUE)🧪 Test Coverage:$(NC)"
	@echo "  • 97 unit tests (89% passing)"
	@echo "  • 1,870+ lines of test code"
	@echo "  • Consensus: 100% ✅"
	@echo "  • Lizenz: 92% ✅"
	@echo "  • Ident: 83% 🟡"
	@echo "  • Anteil: 83% 🟡"

# Quick commands
quick-start: build init ## Quick start: build and initialize
	@echo "$(GREEN)🎉 Quick start completed! Run 'make start' to begin$(NC)"

quick-test: fmt test-unit ## Quick test: format and run unit tests
	@echo "$(GREEN)✅ Quick testing completed$(NC)"