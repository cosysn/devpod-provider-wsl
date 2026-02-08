#!/bin/bash
set -e

cd "$(dirname "$0")/.."

# Build parameters
VERSION=${1:-"v0.0.1"}
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "Building devpod-provider-wsl ${VERSION}..."
echo ""

# Create release directory
mkdir -p release

# Build for Windows AMD64
echo "[1/2] Building for Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" \
    -o release/devpod-provider-wsl-windows-amd64.exe \
    .

# Generate provider.yaml
echo "[2/2] Generating provider.yaml..."
go run ./hack/provider/main.go ${VERSION} > provider.yaml

echo ""
echo "========================================"
echo "Build complete!"
echo "========================================"
echo ""
echo "Binary: release/devpod-provider-wsl-windows-amd64.exe"
echo "Provider: provider.yaml"
echo ""
echo "Usage:"
echo "  ./hack/build.sh v0.0.1"
echo ""
echo "Add to DevPod:"
echo "  devpod provider add ./provider.yaml"
echo ""
echo "Configure provider:"
echo "  devpod provider option set devpod-provider-wsl WSL_DISTRO Ubuntu-22.04"
