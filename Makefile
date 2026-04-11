.PHONY: build build-ui build-backend build-docker clean help dev test lint

VERSION ?= dev
LDFLAGS := -X 'main.version=$(VERSION)'

help:
	@echo "Billax Makefile commands:"
	@echo "  make build          - Build UI and backend binary"
	@echo "  make build-ui       - Build React UI only"
	@echo "  make build-backend  - Build Go binary only"
	@echo "  make build-docker   - Build Docker image"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make dev            - Run development server (requires Go and Node)"
	@echo "  make test           - Run Go tests"
	@echo "  make lint           - Run linters"

# Build everything: UI + backend binary
build: build-ui build-backend
	@echo "✓ Build complete"

# Build the React UI
build-ui:
	@echo "Building UI..."
	cd ui && npm install && npm run build
	@echo "✓ UI built to ui/dist"

# Build the Go backend binary
build-backend: build-ui
	@echo "Building backend..."
	go build -ldflags "$(LDFLAGS)" -o payd ./cmd/payd
	@echo "✓ Binary: ./payd"

# Build Docker image (requires Docker)
build-docker: build
	@echo "Building Docker image..."
	docker build --build-arg VERSION=$(VERSION) -t billax:$(VERSION) .
	@echo "✓ Docker image: billax:$(VERSION)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf ui/dist ui/node_modules payd
	go clean ./...
	@echo "✓ Clean complete"

# Development: run Go server with hot-reload (Node dev server proxies to :8080)
dev:
	@echo "Starting development environment..."
	@echo "  Backend: http://localhost:8080"
	@echo "  Frontend: http://localhost:5173 (dev server)"
	@echo ""
	cd ui && npm run dev &
	go run ./cmd/payd

# Run tests
test:
	go test -v ./...

# Run linters
lint:
	go vet ./...
	golangci-lint run ./... || true
	cd ui && npm run lint || true
