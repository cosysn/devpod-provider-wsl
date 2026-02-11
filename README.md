# DevPod WSL Provider with Tunnel Support

DevPod WSL Provider 支持两种通信模式：

1. **Windows 模式**: 使用 stdin pipe + 9p 文件系统（生产环境）
2. **Linux 模式**: 使用 Unix Socket + gRPC（开发测试）

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│ Layer 4: gRPC - DevPodWSLService                         │
│  - Start/Stop/Exec (commands)                           │
│  - Stdin/Stdout/Stderr (streaming)                       │
├─────────────────────────────────────────────────────────┤
│ Layer 3: Yamux (stream multiplexing)                      │
├─────────────────────────────────────────────────────────┤
│ Layer 2: Unix Socket                                    │
│  - Linux: /var/tmp/devpod.sock                         │
│  - Windows: /mnt/wsl$/<DISTRO>/var/tmp/devpod.sock     │
└─────────────────────────────────────────────────────────┘
```

## Build

### Prerequisites

- Go 1.24+
- protobuf compiler (`protoc`)
- `protoc-gen-go` and `protoc-gen-go-grpc`

### Install tools

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
```

### Build Release (Windows with embedded agent)

```bash
./hack/build.sh v0.0.1
```

Output:
- `release/devpod-provider-wsl-amd64.exe` - Windows binary
- `provider.yaml` - Provider configuration

### Build for Linux Testing

```bash
# Build agent
cd agent && go build -o devpod-agent .

# Build provider (without embed tag)
go build -o devpod-provider-wsl .
```

## Usage

### Linux Testing

```bash
# Start agent with Unix socket server
/var/tmp/devpod-agent -socket /var/tmp/devpod.sock &

# Test with gRPC client
grpcurl -unix-socket /var/tmp/devpod.sock \
  -proto pkg/grpc/proto/tunnel.proto \
  -plaintext \
  tunnel.DevPodWSLService/Status
```

### Integration Test

```bash
# Run full integration test
./test_integration.sh
```

This test:
1. Builds agent binary
2. Starts agent
3. Tests Status RPC
4. Tests Start RPC
5. Tests Stop RPC
6. Cleans up

### Run Tests

```bash
# All tests
go test -v ./pkg/grpc ./pkg/tunnel

# Specific tests
go test -v ./pkg/grpc -run TestServer_Start
go test -v ./pkg/tunnel -run TestUnixServer
```

## gRPC Service API

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `Status` | Empty | AgentStatus | Get agent running status |
| `Start` | StartRequest | StartResponse | Start a command process |
| `Stop` | StopRequest | StopResponse | Stop a running process |
| `Exec` | stream ExecRequest | stream ExecResponse | Interactive command execution |
| `Upload` | stream Chunk | UploadResponse | Upload files to WSL |

### Message Types

```protobuf
message StartRequest {
    string command = 1;
    string workdir = 2;
    map<string, string> env = 3;
}

message StartResponse {
    int32 pid = 1;
}

message StopRequest {
    int32 pid = 1;
}

message StopResponse {
    int32 exit_code = 1;
}
```

## Development

### Project Structure

```
devpod-provider-wsl/
├── agent/                 # Agent binary entry point
│   └── main.go           # Agent with socket server
├── pkg/
│   ├── tunnel/           # Unix socket + Yamux layer
│   │   ├── unix_server.go
│   │   ├── unix_client.go
│   │   ├── session.go
│   │   └── listener.go
│   ├── grpc/             # gRPC service
│   │   ├── client.go
│   │   ├── server.go
│   │   └── proto/
│   │       └── tunnel.proto
│   └── agent/            # Agent installation
├── cmd/
│   └── command.go        # command subcommand
├── hack/
│   └── build.sh          # Release build script
└── docs/
    └── plans/            # Design documents
```

### Protocol Buffers

To regenerate gRPC code:

```bash
protoc \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  pkg/grpc/proto/tunnel.proto
```

## Troubleshooting

### Agent won't start

Check socket path permissions:
```bash
ls -la /var/tmp/devpod.sock
```

### Connection refused

Ensure agent is running:
```bash
pgrep -f devpod-agent
```

### gRPC reflection not available

Use `-proto` flag with grpcurl:
```bash
grpcurl -proto pkg/grpc/proto/tunnel.proto ...
```
