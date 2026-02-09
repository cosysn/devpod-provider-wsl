# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DevPod WSL Provider is a native WSL (Windows Subsystem for Linux) development environment provider for DevPod. It establishes persistent named pipe communication channels between Windows and WSL, enabling DevPod to manage development workspaces directly within WSL.

## Build Commands

```bash
# Basic build
go build -o devpod-provider-wsl.exe .

# Release build with version
./hack/build.sh v0.0.1

# Run all tests
go test -v ./...

# Run specific package tests
go test -v ./cmd
go test -v ./pkg/wsl
go test -v ./pkg/pipe
```

## Architecture

### Core Components

| Component | File | Purpose |
|-----------|------|---------|
| WSL Operations | `pkg/wsl/wsl.go` | WSL version check, start, stop, status, disk space, tool verification |
| Named Pipe | `pkg/pipe/pipe.go` | Windows named pipe server creation |
| Options | `pkg/options/options.go` | Environment variable parsing (WSL_DISTRO, IDLE_TIMEOUT) |

### CLI Commands (cmd/)

| Command | Purpose |
|---------|---------|
| `init` | Validate WSL environment (version 2+, distro exists, disk space, tools) |
| `command` | **Core** - Establish persistent named pipe for Windows-WSL communication |
| `start` | Start the specified WSL distribution |
| `stop` | Stop the specified WSL distribution |
| `status` | Check if WSL distribution is running |

### Communication Mechanism

**Named Pipe + 9p Filesystem**:
- Windows pipe: `\\.\pipe\devpod-wsl-<DISTRO>`
- WSL accesses via 9p: `/mnt/wsl$/<DISTRO>/pipe/devpod-wsl`
- Uses `socat` to bridge named pipe to stdin/stdout in WSL

**Command Execution Flow**:
1. Provider creates named pipe server on Windows
2. WSL内的socat connects to the 9p-mounted pipe
3. Commands passed via stdin pipe (avoids shell escaping issues)
4. Uses `--noprofile --norc` flags and `stsl -echo` to prevent echo pollution

## Key Configuration

**Provider options** (`provider.yaml`):
- `WSL_DISTRO` (required): WSL distribution name (e.g., "Ubuntu-22.04")
- `IDLE_TIMEOUT` (default: "30"): Minutes before auto-stop

**Environment Variables**:
- `WSL_DISTRO`: Target WSL distribution
- `IDLE_TIMEOUT`: Auto-stop timeout
- `COMMAND`: Command to execute in WSL (set by DevPod)
- `DEVPOD`: DevPod CLI path (set by DevPod)

## Testing

Go unit tests use table-driven patterns in `*_test.go` files. Python E2E tests are in `tests/test_wsl_provider.py` for integration testing.
