BINDIR ?= bin
APP    ?= volnixd

.PHONY: all build install tidy test proto-gen clean init start testnet dev-build dev-test check status help

all: build

build:
	@mkdir -p $(BINDIR)
	go build -ldflags "-s -w -X main.commit=$$(git rev-parse --short HEAD 2>/dev/null || echo dev)" -o $(BINDIR)/$(APP) ./cmd/volnixd

install:
	go install -ldflags "-s -w -X main.commit=$$(git rev-parse --short HEAD 2>/dev/null || echo dev)" ./cmd/volnixd

tidy:
	go mod tidy

test:
	go test ./...

proto-gen: buf-check
	cd proto && buf dep update && buf lint && buf generate

buf-check:
	@command -v buf >/dev/null 2>&1 || { echo "buf not found. Install from https://buf.build/docs/installation"; exit 1; }

clean:
	rm -rf $(BINDIR)

# Новые команды для полноценного запуска

init:
	@echo "🚀 Initializing Volnix node..."
	@if [ ! -f "$(BINDIR)/$(APP)" ]; then \
		echo "❌ Binary not found. Run 'make build' first."; \
		exit 1; \
	fi
	@./$(BINDIR)/$(APP) init volnix-node

start:
	@echo "📡 Starting Volnix node..."
	@if [ ! -f "$(BINDIR)/$(APP)" ]; then \
		echo "❌ Binary not found. Run 'make build' first."; \
		exit 1; \
	fi
	@if [ ! -d "$(HOME)/.volnix/config" ]; then \
		echo "❌ Node not initialized. Run 'make init' first."; \
		exit 1; \
	fi
	@./$(BINDIR)/$(APP) start

testnet:
	@echo "🌐 Starting Volnix testnet..."
	@if [ ! -f "$(BINDIR)/$(APP)" ]; then \
		echo "❌ Binary not found. Run 'make build' first."; \
		exit 1; \
	fi
	@cd testnet && ./start.sh

# Команды для разработки

dev-build:
	@echo "🔨 Building for development..."
	go build -race -o $(BINDIR)/$(APP) ./cmd/volnixd

dev-test:
	@echo "🧪 Running tests with race detection..."
	go test -race ./...

# Команды для проверки

check: tidy test build
	@echo "✅ All checks passed!"

status:
	@echo "📊 Volnix Protocol Status:"
	@echo "Binary: $(shell if [ -f "$(BINDIR)/$(APP)" ]; then echo "✅ Built"; else echo "❌ Not built"; fi)"
	@echo "Node: $(shell if [ -d "$(HOME)/.volnix/config" ]; then echo "✅ Initialized"; else echo "❌ Not initialized"; fi)"
	@echo "Process: $(shell if pgrep -f volnixd >/dev/null; then echo "✅ Running"; else echo "❌ Not running"; fi)"

test-current:
	@echo "🧪 Testing current functionality..."
	@if [ -f "./scripts/test_current_functionality.sh" ]; then \
		./scripts/test_current_functionality.sh; \
	else \
		echo "❌ Test script not found"; \
		exit 1; \
	fi

help:
	@echo "🚀 Волникс Протокол - Available Commands:"
	@echo ""
	@echo "Build & Install:"
	@echo "  build        - Build the binary"
	@echo "  install      - Install binary to GOPATH"
	@echo "  clean        - Remove built binaries"
	@echo ""
	@echo "Development:"
	@echo "  tidy         - Tidy Go modules"
	@echo "  test         - Run tests"
	@echo "  proto-gen    - Generate protobuf code"
	@echo "  dev-build    - Build with race detection"
	@echo "  dev-test     - Test with race detection"
	@echo ""
	@echo "Node Management:"
	@echo "  init         - Initialize a new node"
	@echo "  start        - Start the node"
	@echo "  testnet      - Start testnet"
	@echo ""
	@echo "Testing:"
	@echo "  test-current - Test current ABCI server functionality"
	@echo ""
	@echo "Utilities:"
	@echo "  check        - Run all checks (tidy, test, build)"
	@echo "  status       - Show current status"
	@echo "  help         - Show this help message"


