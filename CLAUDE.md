# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DevPod WSL Provider is a native WSL (Windows Subsystem for Linux) development environment provider for DevPod. It establishes communication channels between Windows and WSL, enabling DevPod to manage development workspaces directly within WSL.

## Build Commands

```bash
# Basic build (without embedded agent)
go build -o devpod-provider-wsl.exe .

# Release build with version and embedded agent
./hack/build.sh v0.0.1

# Run all tests
go test -v ./...

# Run specific package tests
go test -v ./cmd
go test -v ./pkg/wsl
go test -v ./pkg/pipe

# Run agent tests (requires embed build tag)
go test -v -tags=embed ./pkg/agent
```

## Architecture

### Core Components

| Component | File | Purpose |
|-----------|------|---------|
| WSL Operations | `pkg/wsl/wsl.go` | WSL version check, start, stop, status, disk space, tool verification |
| Named Pipe | `pkg/pipe/pipe.go` | Windows named pipe server creation |
| Options | `pkg/options/options.go` | Environment variable parsing (WSL_DISTRO) |
| Agent Injection | `pkg/agent/*.go` | Embeds and injects agent binary into WSL |
| Tunnel | `pkg/tunnel/*.go` | Unix socket + Yamux multiplexing (Linux testing) |
| gRPC | `pkg/grpc/*.go` | gRPC server/client for command execution |

### CLI Commands (cmd/)

| Command | Purpose |
|---------|---------|
| `init` | Validate WSL environment (version 2+, distro exists, disk space, tools) |
| `create` | Create a new WSL instance |
| `delete` | Delete the WSL instance |
| `command` | **Core** - Execute commands in WSL via named pipe or tunnel |
| `start` | Start the specified WSL distribution |
| `stop` | Stop the specified WSL distribution |
| `status` | Check if WSL distribution is running |

### Communication Mechanisms

**Windows (Default): Named Pipe + 9p Filesystem**
- Windows pipe: `\\.\pipe\devpod-wsl-<DISTRO>`
- WSL accesses via 9p: `/mnt/wsl$/<DISTRO>/pipe/devpod-wsl`
- Uses `socat` to bridge named pipe to stdin/stdout in WSL

**Linux (Testing): Unix Socket + Yamux + gRPC**
- Unix socket: `/var/tmp/devpod.sock`
- Yamux for stream multiplexing
- gRPC for command execution (Start, Stop, Exec, Stdin, Stdout, Stderr, Status, Upload)
- Enables testing provider on native Linux without WSL

**Command Execution Flow (Windows)**:
1. Agent binary is injected to WSL (if embedded)
2. WSL is started with `wsl.exe -d <distro>`
3. `stdin` pipe passes commands to bash with `--noprofile --norc` flags
4. `stty -echo` prevents terminal echo, `TERM=dumb` suppresses ANSI codes

**Command Execution Flow (Linux/Tunnel)**:
1. Agent binary is installed locally
2. Agent starts Unix socket server with gRPC
3. Provider connects via gRPC client
4. Provider calls Start RPC to execute commands

### Agent Injection (Build Tags)

The agent is embedded using Go build tags:
- **With `embed` tag**: Agent binary is embedded in the Windows binary via `pkg/agent/embed.go` (FS)
- **Without `embed` tag**: `pkg/agent/agent_stub.go` returns nil (used during development)
- Agent is injected to `/var/tmp/devpod-agent` in WSL on first run or upgrade
- Agent version is hardcoded in `pkg/agent/install.go` (AgentVersion constant)

**Standalone Agent** (`agent/main.go`):
- Separate binary entry point for Linux testing
- Listens on Unix socket, provides gRPC services
- Run with: `go run ./agent -socket /var/tmp/devpod.sock`

## Key Configuration

**Provider options** (`provider.yaml`):
- `WSL_DISTRO` (required): WSL distribution name (e.g., "Ubuntu-22.04")

**Environment Variables**:
- `WSL_DISTRO`: Target WSL distribution
- `COMMAND`: Command to execute in WSL (set by DevPod)
- `DEVPOD`: DevPod CLI path (set by DevPod)
- `WSL_UTF8`: Forces UTF-8 encoding in WSL
- `WSL_PROXY`: Suppresses automatic proxy warnings (WSL_PROXY=0)
- `DONT_SET_WSL_PROXY`: Suppresses network-related warnings

## Testing

**Go unit tests**: Table-driven patterns in `*_test.go` files.

**Python E2E tests**: `tests/test_wsl_provider.py` for Windows/WSL integration testing.

**Linux testing** (development/testing without WSL):
```bash
# Build and run agent
go build -o /tmp/devpod-agent ./agent
/tmp/agent -socket /var/tmp/devpod.sock &

# Test gRPC with grpcurl
grpcurl -unix-socket /var/tmp/devpod.sock tunnel.DevPodWSLService/Status
grpcurl -unix-socket /var/tmp/devpod.sock -d '{"command":"echo hello"}' tunnel.DevPodWSLService/Start
```

## Tech Stack

- Go 1.24
- hashicorp/yamux v0.1.1 (stream multiplexing for tunnel)
- google.golang.org/grpc v1.68.1 + protobuf v1.36.5 (command RPC)
- cobra (CLI framework)
- lofthq/devpod (provider interface)
