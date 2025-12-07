# Makefile for SSH Key Manager

.PHONY: all build clean test install docker help

# Variables
BINARY_NAME=skm
SERVER_NAME=skm-server
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

# Default target
all: build

## build: Build the binaries
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p bin
	@go build ${LDFLAGS} -o bin/${BINARY_NAME} ./main.go
	@echo "Building ${SERVER_NAME}..."
	@go build ${LDFLAGS} -o bin/${SERVER_NAME} ./cmd/skm-server/main.go
	@echo "✓ Build complete!"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf dist/
	@go clean
	@echo "✓ Clean complete!"

## test: Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Tests complete! Coverage report: coverage.html"

## install: Install binaries to /usr/local/bin
install: build
	@echo "Installing binaries..."
	@sudo cp bin/${BINARY_NAME} /usr/local/bin/
	@sudo cp bin/${SERVER_NAME} /usr/local/bin/
	@echo "✓ Installation complete!"

## docker: Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -f hack/Dockerfile -t ${BINARY_NAME}:${VERSION} .
	@echo "✓ Docker image built: ${BINARY_NAME}:${VERSION}"

## run: Run the client
run: build
	@./bin/${BINARY_NAME}

## run-server: Run the server
run-server: build
	@./bin/${SERVER_NAME} --addr :8080 --data ./data --jwt-secret "dev-secret-change-me"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated!"

## lint: Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run ./...
	@echo "✓ Linting complete!"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .
	@echo "✓ Formatting complete!"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet complete!"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
