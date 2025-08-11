BINDIR ?= bin
APP    ?= helvetiad

.PHONY: all build install tidy test proto-gen clean

all: build

build:
	@mkdir -p $(BINDIR)
	go build -ldflags "-s -w -X main.commit=$$(git rev-parse --short HEAD 2>/dev/null || echo dev)" -o $(BINDIR)/$(APP) ./cmd/helvetiad

install:
	go install -ldflags "-s -w -X main.commit=$$(git rev-parse --short HEAD 2>/dev/null || echo dev)" ./cmd/helvetiad

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


