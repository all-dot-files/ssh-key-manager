#!/bin/bash
set -e

# Build script for SKM

# Ensure we are in the project root
cd "$(git rev-parse --show-toplevel)"

VERSION=${VERSION:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

LDFLAGS="-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

echo "Building SKM..."
echo "  Version: ${VERSION}"
echo "  Commit: ${COMMIT}"
echo "  Build Time: ${BUILD_TIME}"
echo ""

# Build client
echo "Building client (skm)..."
go build -ldflags "${LDFLAGS}" -o bin/skm ./main.go

# Build server
echo "Building server (skm-server)..."
go build -ldflags "${LDFLAGS}" -o bin/skm-server ./cmd/skm-server/main.go

echo ""
echo "✓ Build complete!"
echo "  Client: bin/skm"

echo "  Server: bin/skm-server"
echo ""
echo "To install:"
echo "  sudo cp bin/skm /usr/local/bin/"
echo "  sudo cp bin/skm-server /usr/local/bin/"
