# Tunnel Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement Unix socket + Yamux + gRPC tunnel for Windows-WSL communication, supporting Linux testing.

**Architecture:**
- Layer 1: Unix socket server/client for WSL/Linux communication
- Layer 2: Yamux multiplex on top of Unix socket
- Layer 3: net.Listener interface for gRPC
- Layer 4: gRPC service (DevPodWSLService) for command execution
- Provider runs on Windows, Agent runs in WSL (or Linux for testing)

**Tech Stack:**
- Go 1.24
- hashicorp/yamux v0.1.1
- google.golang.org/grpc v1.68.1
- google.golang.org/protobuf v1.36.5

---

## Files Created (Already Done)

**New Files:**
- `pkg/grpc/proto/tunnel.proto` - gRPC service definition
- `pkg/grpc/proto/tunnel.pb.go` - protobuf generated code
- `pkg/grpc/proto/tunnel_grpc.pb.go` - gRPC generated code
- `pkg/tunnel/unix_server.go` - Unix socket server
- `pkg/tunnel/unix_client.go` - Unix socket client
- `pkg/tunnel/session.go` - Yamux session wrapper
- `pkg/tunnel/listener.go` - TunnelListener interface
- `pkg/grpc/client.go` - gRPC client
- `pkg/grpc/server.go` - gRPC server stub
- `agent/main.go` - Agent entry point
- `docs/plans/2026-02-11-tunnel-design.md` - Design document
- `docs/plans/2026-02-11-tunnel-implementation.md` - Implementation guide

**Modified Files:**
- `cmd/command.go` - Added Windows/Linux environment detection
- `go.mod` - Added dependencies (yamux, grpc, protobuf)
- `pkg/agent/install.go` - Added Linux version functions

---

## Task 1: Fix Build Errors

**Goal:** Fix compilation errors and ensure project builds successfully.

**Step 1: Run build to identify errors**

```bash
export PATH=$PATH:/usr/local/go/bin && go build -o devpod-provider-wsl . 2>&1
```

Expected: Compilation errors (unknown commands, type mismatches, etc.)

**Step 2: Fix identified errors**

Fix any compilation errors found in Step 1.

**Step 3: Verify build succeeds**

```bash
export PATH=$PATH:/usr/local/go/bin && go build -o devpod-provider-wsl . 2>&1
```

Expected: SUCCESS (no output)

**Step 4: Commit**

```bash
git add cmd/command.go pkg/agent/install.go go.mod
git commit -m "fix: resolve build errors"
```

---

## Task 2: Test Unix Socket Layer

**Goal:** Verify Unix socket server and client work correctly.

**Files:**
- Test: `pkg/tunnel/unix_server_test.go` (new file)
- Create: `pkg/tunnel/unix_server_test.go`

**Step 1: Write failing test**

```go
package tunnel

import (
    "net"
    "os"
    "testing"
    "time"
)

func TestUnixServer_ListenAndAccept(t *testing.T) {
    // Clean up any existing socket
    socketPath := "/tmp/test-devpod.sock"
    os.Remove(socketPath)
    defer os.Remove(socketPath)

    // Create server
    server := NewUnixServer(socketPath)
    if err := server.Listen(); err != nil {
        t.Fatalf("Failed to listen: %v", err)
    }
    defer server.Close()

    // Connect client
    client, err := net.Dial("unix", socketPath)
    if err != nil {
        t.Fatalf("Failed to dial: %v", err)
    }
    defer client.Close()

    // Server should accept
    go func() {
        server.Accept()
    }()

    // Give server time to accept
    time.Sleep(100 * time.Millisecond)

    t.Log("Unix socket test passed")
}
```

**Step 2: Run test to verify it fails**

```bash
export PATH=$PATH:/usr/local/go/bin && go test -v ./pkg/tunnel -run TestUnixServer 2>&1
```

Expected: File not found error

**Step 3: Write minimal implementation (already done, just verify)**

```bash
export PATH=$PATH:/usr/local/go/bin && go test -v ./pkg/tunnel -run TestUnixServer 2>&1
```

Expected: PASS

**Step 4: Commit**

```bash
git add pkg/tunnel/unix_server_test.go
git commit -m "test: add Unix socket tests"
```

---

## Task 3: Implement gRPC Server Start Command

**Goal:** Implement Start RPC to spawn subprocess in agent.

**Files:**
- Modify: `pkg/grpc/server.go:45-72`

**Step 1: Write failing test**

```go
package grpc

import (
    "context"
    "testing"

    pb "github.com/cosysn/devpod-provider-wsl/pkg/grpc/proto"
)

func TestServer_Start(t *testing.T) {
    server := NewServer()

    resp, err := server.Start(context.Background(), &pb.StartRequest{
        Command: "echo hello",
        Workdir: "",
        Env:     map[string]string{},
    })

    if err != nil {
        t.Fatalf("Start failed: %v", err)
    }

    if resp.Pid <= 0 {
        t.Fatalf("Invalid PID: %d", resp.Pid)
    }

    t.Logf("Started process with PID: %d", resp.Pid)
}
```

**Step 2: Run test to verify it fails**

```bash
export PATH=$PATH:/usr/local/go/bin && go test -v ./pkg/grpc -run TestServer_Start 2>&1
```

Expected: FAIL (function not returning correct value)

**Step 3: Implement minimal Start command**

Modify `pkg/grpc/server.go` to properly start subprocess:

```go
func (s *Server) Start(ctx context.Context, req *pb.StartRequest) (*pb.StartResponse, error) {
    cmd := exec.CommandContext(ctx, "/bin/sh", "-c", req.Command)
    cmd.Dir = req.Workdir

    // Set environment variables
    for k, v := range req.Env {
        cmd.Env = append(cmd.Env, k+"="+v)
    }

    // Capture stdout/stderr
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }
    stderr, err := cmd.StderrPipe()
    if err != nil {
        return nil, err
    }

    if err := cmd.Start(); err != nil {
        return nil, err
    }

    // Async copy output
    go func() {
        io.Copy(os.Stdout, stdout)
    }()
    go func() {
        io.Copy(os.Stderr, stderr)
    }()

    s.mu.Lock()
    s.processes[cmd.Process.Pid] = cmd
    s.mu.Unlock()

    return &pb.StartResponse{Pid: cmd.Process.Pid}, nil
}
```

**Step 4: Run test to verify it passes**

```bash
export PATH=$PATH:/usr/local/go/bin && go test -v ./pkg/grpc -run TestServer_Start 2>&1
```

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/grpc/server.go pkg/grpc/server_test.go
git commit -m "feat: implement gRPC Start command"
```

---

## Task 4: Implement gRPC Server Stop Command

**Goal:** Implement Stop RPC to terminate subprocess.

**Files:**
- Modify: `pkg/grpc/server.go:74-95`

**Step 1: Write failing test**

```go
func TestServer_Stop(t *testing.T) {
    server := NewServer()

    // Start a long-running process
    startResp, err := server.Start(context.Background(), &pb.StartRequest{
        Command: "sleep 60",
        Workdir: "",
        Env:     map[string]string{},
    })
    if err !=.Fatalf("Start failed: %v nil {
        t", err)
    }

    // Stop it
    stopResp, err := server.Stop(context.Background(), &pb.StopRequest{
        Pid: startResp.Pid,
    })
    if err != nil {
        t.Fatalf("Stop failed: %v", err)
    }

    t.Logf("Stopped process %d with exit code: %d", startResp.Pid, stopResp.ExitCode)
}
```

**Step 2: Run test to verify it fails**

```bash
export PATH=$PATH:/usr/local/go/bin && go test -v ./pkg/grpc -run TestServer_Stop 2>&1
```

Expected: FAIL

**Step 3: Implement Stop command**

```go
func (s *Server) Stop(ctx context.Context, req *pb.StopRequest) (*pb.StopResponse, error) {
    s.mu.Lock()
    cmd, ok := s.processes[req.Pid]
    s.mu.Unlock()

    if !ok {
        return &pb.StopResponse{ExitCode: 0}, nil
    }

    cmd.Process.Kill()
    cmd.Wait()

    s.mu.Lock()
    delete(s.processes, req.Pid)
    s.mu.Unlock()

    return &pb.StopResponse{ExitCode: 0}, nil
}
```

**Step 4: Run test to verify it passes**

```bash
export PATH=$PATH:/usr/local/go/bin && go test -v ./pkg/grpc -run TestServer_Stop 2>&1
```

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/grpc/server.go pkg/grpc/server_test.go
git commit -m "feat: implement gRPC Stop command"
```

---

## Task 5: Implement Agent Main with Socket Server

**Goal:** Complete agent/main.go to properly start gRPC server.

**Files:**
- Modify: `agent/main.go:36-47`

**Step 1: Write integration test**

```bash
#!/bin/bash
# test_agent.sh

# Clean up
rm -f /var/tmp/devpod.sock /var/tmp/devpod-agent

# Build agent
export PATH=$PATH:/usr/local/go/bin
go build -o /var/tmp/devpod-agent ./agent

# Start agent in background
/var/tmp/devpod-agent &
AGENT_PID=$!
sleep 2

# Check socket exists
if [ -S /var/tmp/devpod.sock ]; then
    echo "Socket created successfully"
else
    echo "ERROR: Socket not created"
    kill $AGENT_PID 2>/dev/null
    exit 1
fi

# Test with grpcurl or netcat
echo "Agent test passed"

# Cleanup
kill $AGENT_PID 2>/dev/null
```

**Step 2: Run test to verify it fails**

```bash
chmod +x test_agent.sh && ./test_agent.sh 2>&1
```

Expected: FAIL (agent doesn't start properly)

**Step 3: Fix agent/main.go**

Fix the gRPC server registration and tunnel listener:

```go
// agent/main.go
func main() {
    socketPath := flag.String("socket", tunnel.DefaultSocketPath, "Unix socket path")
    flag.Parse()

    // Create Unix socket server
    server := tunnel.NewUnixServer(*socketPath)
    if err := server.Listen(); err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    // Create gRPC server
    grpcServer := grpc.NewServer()
    pb.RegisterDevPodWSLServiceServer(grpcServer, grpc.NewServer())

    // Create tunnel listener that wraps Unix socket
    listener := &tunnelAdapter{server}

    // Start gRPC server in goroutine
    go func() {
        if err := grpcServer.Serve(listener); err != nil {
            log.Printf("gRPC server error: %v", err)
        }
    }()

    log.Printf("Agent started, listening on %s", *socketPath)

    // Wait for shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    grpcServer.GracefulStop()
    server.Close()
}

// tunnelAdapter adapts net.Listener to work with gRPC
type tunnelAdapter struct {
    *tunnel.UnixServer
}

func (t *tunnelAdapter) Accept() (net.Conn, error) {
    return t.UnixServer.Accept()
}
```

**Step 4: Run test to verify it passes**

```bash
./test_agent.sh 2>&1
```

Expected: PASS

**Step 5: Commit**

```bash
git add agent/main.go test_agent.sh
git commit -m "feat: complete agent main with socket server"
```

---

## Task 6: Test Full Integration on Linux

**Goal:** Test complete flow: inject agent -> start agent -> connect gRPC -> execute command.

**Files:**
- Test: `test_integration.sh` (new file)

**Step 1: Write integration test**

```bash
#!/bin/bash
# test_integration.sh

set -e

SOCKET_PATH="/var/tmp/devpod.sock"
AGENT_PATH="/var/tmp/devpod-agent"

# Cleanup
rm -f "$SOCKET_PATH" "$AGENT_PATH"

export PATH=$PATH:/usr/local/go/bin

echo "=== Building ==="
go build -o "$AGENT_PATH" ./agent
go build -o ./devpod-provider-wsl .

echo "=== Starting Agent ==="
"$AGENT_PATH" &
AGENT_PID=$!
sleep 2

if [ ! -S "$SOCKET_PATH" ]; then
    echo "ERROR: Agent socket not created"
    kill $AGENT_PID 2>/dev/null || true
    exit 1
fi

echo "=== Agent started (PID: $AGENT_PID) ==="

echo "=== Testing with grpcurl ==="
# Install grpcurl if not available
if ! command -v grpcurl &> /dev/null; then
    go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
fi

# Test Status RPC
grpcurl -unix-socket "$SOCKET_PATH" tunnel.DevPodWSLService/Status

echo "=== Testing Start RPC ==="
# This would test actual command execution
# For now, just verify connection works

echo "=== Cleanup ==="
kill $AGENT_PID 2>/dev/null || true
rm -f "$SOCKET_PATH" "$AGENT_PATH"

echo "=== Integration test passed ==="
```

**Step 2: Run integration test**

```bash
chmod +x test_integration.sh && ./test_integration.sh 2>&1
```

Expected: Integration test completes successfully

**Step 3: Commit**

```bash
git add test_integration.sh
git commit -m "test: add integration tests"
```

---

## Task 7: Update Build Script

**Goal:** Update hack/build.sh to include new agent binary.

**Files:**
- Modify: `hack/build.sh`

**Step 1: Review current build script**

```bash
cat hack/build.sh
```

**Step 2: Update to build agent binary**

```bash
# Add after Step 1:
echo "[1.5] Building Linux agent binary..."
go build -ldflags="${LDFLAGS}" -o "${AGENT_PATH}" ./agent

# Update Step 2 to copy agent binary
cp "${AGENT_PATH}" pkg/agent/agent-linux
```

**Step 3: Test build script**

```bash
./hack/build.sh v0.0.1 2>&1
```

Expected: Build completes successfully

**Step 4: Commit**

```bash
git add hack/build.sh
git commit -m "build: update build script for agent binary"
```

---

## Execution Options

**Plan complete and saved to `docs/plans/2026-02-11-tunnel-implementation-plan.md`. Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
