#!/bin/bash
set -e

SOCKET_PATH="/tmp/devpod-test.sock"
AGENT_PATH="/tmp/devpod-agent"

# Cleanup
rm -f "$SOCKET_PATH" "$AGENT_PATH"

export PATH=$PATH:/usr/local/go/bin

echo "=== Building agent ==="
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
echo "=== Socket created at $SOCKET_PATH ==="

# Check agent is running
if kill -0 $AGENT_PID 2>/dev/null; then
    echo "=== Agent is running ==="
else
    echo "ERROR: Agent died"
    exit 1
fi

echo "=== Cleanup ==="
kill $AGENT_PID 2>/dev/null || true
rm -f "$SOCKET_PATH" "$AGENT_PATH"

echo "=== Agent test passed ==="
