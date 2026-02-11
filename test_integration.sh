#!/bin/bash
set -e

SOCKET_PATH="/tmp/test.sock"
AGENT_PATH="/tmp/devpod-agent"

# Cleanup old agent
pkill -9 -f "devpod-agent" 2>/dev/null || true
rm -f "$SOCKET_PATH" "$AGENT_PATH"

export PATH=$PATH:/usr/local/go/bin

echo "=== Building agent binary ==="
go build -o "$AGENT_PATH" ./agent

echo "=== Starting agent ==="
"$AGENT_PATH" -socket "$SOCKET_PATH" &
AGENT_PID=$!
sleep 2

if [ ! -S "$SOCKET_PATH" ]; then
    echo "ERROR: Agent socket not created"
    kill $AGENT_PID 2>/dev/null || true
    exit 1
fi

echo "=== Agent started (PID: $AGENT_PID) ==="

echo "=== Running integration test ==="
go run test_integration.go

echo "=== Cleanup ==="
kill $AGENT_PID 2>/dev/null || true
rm -f "$SOCKET_PATH" "$AGENT_PATH"

echo "=== Full integration test passed ==="
