#!/bin/bash
set -e

cd "$(dirname "$0")/.."

VERSION=${1:-"v0.0.1"}
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "Building devpod-provider-wsl ${VERSION}..."
echo ""

# Step 1: 构建 Linux agent
echo "[1/3] Building Linux agent..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o agent-linux .

# Step 2: 构建 Windows provider (嵌入 agent)
echo "[2/3] Building Windows provider..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o release/devpod-provider-wsl-amd64.exe .

# 清理临时文件
rm -f agent-linux

# Step 3: 生成 provider.yaml
echo "[3/3] Generating provider.yaml..."
go run ./hack/provider/main.go ${VERSION} > provider.yaml

echo ""
echo "========================================"
echo "Build complete!"
echo "========================================"
echo ""
echo "Binary: release/devpod-provider-wsl-amd64.exe"
echo "Provider: provider.yaml"
