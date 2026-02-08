#!/bin/bash
set -e

cd "$(dirname "$0")/.."

# Build parameters
VERSION=${1:-"v0.0.1"}
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "Building devpod-provider-wsl ${VERSION}..."

# Create release directory
mkdir -p release

# Build for Windows AMD64
echo "Building for Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" \
    -o release/devpod-provider-wsl-windows-amd64.exe \
    .

# Generate provider.yaml
echo "Generating provider.yaml..."
go run ./hack/provider/main.go ${VERSION} > provider.yaml

echo ""
echo "Build complete!"
echo "  Binary: release/devpod-provider-wsl-windows-amd64.exe"
echo "  Provider: provider.yaml"
echo ""
echo "To add the provider to DevPod:"
echo "  devpod provider add ./provider.yaml"
