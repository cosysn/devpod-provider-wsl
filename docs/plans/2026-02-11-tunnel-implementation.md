# DevPod WSL Tunnel 实现计划

## 概述

本文档描述 tunnel 能力的详细实现步骤。

## 依赖

- `github.com/hashicorp/yamux` - 多路复用库
- `google.golang.org/grpc` - gRPC 框架
- `google.golang.org/protobuf` - Protocol Buffers

## 实施步骤

### Step 1: 创建 protobuf 定义

**文件**: `pkg/grpc/proto/tunnel.proto`

```protobuf
syntax = "proto3";

package tunnel;

option go_package = "github.com/cosysn/devpod-provider-wsl/pkg/grpc/proto";

service DevPodWSLService {
    rpc Start(StartRequest) returns (StartResponse);
    rpc Stop(StopRequest) returns (StopResponse);
    rpc Exec(stream ExecRequest) returns (stream ExecResponse);
    rpc Stdin(stream Data) returns (Empty);
    rpc Stdout(Empty) returns (stream Data);
    rpc Stderr(Empty) returns (stream Data);
    rpc Status(Empty) returns (AgentStatus);
    rpc Upload(stream Chunk) returns (UploadResponse);
}

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

message ExecRequest {
    oneof data {
        string input = 1;
        bool eof = 2;
    }
}

message ExecResponse {
    bytes stdout = 1;
    bytes stderr = 2;
    int32 exit_code = 3;
    bool done = 4;
}

message Data {
    bytes content = 1;
}

message Empty {}

message AgentStatus {
    bool running = 1;
    int32 pid = 2;
}

message Chunk {
    string path = 1;
    bytes content = 2;
    bool eof = 3;
}

message UploadResponse {
    bool success = 1;
}
```

**命令**: 生成 Go 代码
```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  pkg/grpc/proto/tunnel.proto
```

---

### Step 2: Unix Socket 层

**文件**: `pkg/tunnel/unix_server.go`

```go
package tunnel

import (
    "net"
    "os"
    "path/filepath"
)

const DefaultSocketPath = "/var/tmp/devpod.sock"

type UnixServer struct {
    socketPath string
    listener   net.Listener
}

func NewUnixServer(socketPath string) *UnixServer {
    if socketPath == "" {
        socketPath = DefaultSocketPath
    }
    return &UnixServer{socketPath: socketPath}
}

func (s *UnixServer) Listen() error {
    // 确保目录存在
    dir := filepath.Dir(s.socketPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    // 删除已存在的 socket 文件
    os.Remove(s.socketPath)

    listener, err := net.Listen("unix", s.socketPath)
    if err != nil {
        return err
    }
    s.listener = listener
    return nil
}

func (s *UnixServer) Accept() (net.Conn, error) {
    return s.listener.Accept()
}

func (s *UnixServer) Close() error {
    return s.listener.Close()
}

func (s *UnixServer) Addr() net.Addr {
    return &net.UnixAddr{Name: s.socketPath, Net: "unix"}
}
```

**文件**: `pkg/tunnel/unix_client.go`

```go
package tunnel

import "net"

type UnixClient struct {
    socketPath string
}

func NewUnixClient(socketPath string) *UnixClient {
    return &UnixClient{socketPath: socketPath}
}

func (c *UnixClient) Dial() (net.Conn, error) {
    return net.Dial("unix", c.socketPath)
}
```

---

### Step 3: Yamux 层

**文件**: `pkg/tunnel/session.go`

```go
package tunnel

import (
    "net"
    "sync"

    "github.com/hashicorp/yamux"
)

type YamuxSession struct {
    session *yamux.Session
    mu      sync.Mutex
}

func NewYamuxSession(conn net.Conn) *YamuxSession {
    session, _ := yamux.Server(conn, nil)
    return &YamuxSession{session: session}
}

func (s *YamuxSession) Accept() (net.Conn, error) {
    return s.session.Accept()
}

func (s *YamuxSession) Open() (net.Conn, error) {
    return s.session.Open()
}

func (s *YamuxSession) Close() error {
    return s.session.Close()
}
```

---

### Step 4: Net 接口层

**文件**: `pkg/tunnel/listener.go`

```go
package tunnel

import (
    "net"

    "github.com/hashicorp/yamux"
)

type TunnelListener struct {
    session *yamux.Session
}

func NewTunnelListener(session *yamux.Session) *TunnelListener {
    return &TunnelListener{session: session}
}

func (l *TunnelListener) Accept() (net.Conn, error) {
    return l.session.Accept()
}

func (l *TunnelListener) Addr() net.Addr {
    return &net.UnixAddr{Name: "", Net: "unix"}
}

func (l *TunnelListener) Close() error {
    return l.session.Close()
}
```

---

### Step 5: gRPC 客户端

**文件**: `pkg/grpc/client.go`

```go
package grpc

import (
    "context"
    "io"
    "net"

    "github.com/cosysn/devpod-provider-wsl/pkg/tunnel"
    pb "github.com/cosysn/devpod-provider-wsl/pkg/grpc/proto"
    "google.golang.org/grpc"
)

type Client struct {
    conn   *grpc.ClientConn
    client pb.DevPodWSLServiceClient
}

func NewClient(socketPath string) (*Client, error) {
    conn, err := grpc.Dial(
        "passthrough:///unix://"+socketPath,
        grpc.WithInsecure(),
        grpc.WithBlock(),
    )
    if err != nil {
        return nil, err
    }
    return &Client{
        conn:   conn,
        client: pb.NewDevPodWSLServiceClient(conn),
    }, nil
}

func (c *Client) Start(ctx context.Context, command, workdir string, env map[string]string) (*pb.StartResponse, error) {
    return c.client.Start(ctx, &pb.StartRequest{
        Command: command,
        Workdir: workdir,
        Env:     env,
    })
}

func (c *Client) Stop(ctx context.Context, pid int32) (*pb.StopResponse, error) {
    return c.client.Stop(ctx, &pb.StopRequest{Pid: pid})
}

func (c *Client) Exec(ctx context.Context) (pb.DevPodWSLService_ExecClient, error) {
    return c.client.Exec(ctx)
}

func (c *Client) Close() error {
    return c.conn.Close()
}

// Helper: 连接到 Yamux session
func DialYamux(socketPath string) (net.Conn, error) {
    dialer := &net.Dialer{}
    conn, err := dialer.Dial("unix", socketPath)
    if err != nil {
        return nil, err
    }
    muxConfig := yamux.ClientConfig{}
    session, err := yamux.Client(conn, &muxConfig)
    if err != nil {
        conn.Close()
        return nil, err
    }
    return session.Open()
}
```

---

### Step 6: gRPC 服务端

**文件**: `pkg/grpc/server.go`

```go
package grpc

import (
    "context"
    "io"
    "os/exec"
    "sync"

    pb "github.com/cosysn/devpod-provider-wsl/pkg/grpc/proto"
    "google.golang.org/grpc"
)

type Server struct {
    pb.UnimplementedDevPodWSLServiceServer
    processes map[int32]*exec.Cmd
    mu        sync.Mutex
}

func NewServer() *Server {
    return &Server{
        processes: make(map[int32]*exec.Cmd),
    }
}

func (s *Server) Start(ctx context.Context, req *pb.StartRequest) (*pb.StartResponse, error) {
    cmd := exec.CommandContext(ctx, "/bin/sh", "-c", req.Command)
    cmd.Dir = req.Workdir
    // 设置环境变量...

    if err := cmd.Start(); err != nil {
        return nil, err
    }

    s.mu.Lock()
    s.processes[cmd.Process.Pid] = cmd
    s.mu.Unlock()

    return &pb.StartResponse{Pid: cmd.Process.Pid}, nil
}

func (s *Server) Stop(ctx context.Context, req *pb.StopRequest) (*pb.StopResponse, error) {
    s.mu.Lock()
    cmd, ok := s.processes[req.Pid]
    s.mu.Unlock()

    if !ok {
        return nil, nil
    }

    cmd.Process.Kill()
    cmd.Wait()

    s.mu.Lock()
    delete(s.processes, req.Pid)
    s.mu.Unlock()

    return &pb.StopResponse{ExitCode: 0}, nil
}

func (s *Server) Exec(stream pb.DevPodWSLService_ExecServer) error {
    // 实现双向流执行...
    return nil
}

func (s *Server) Stdin(stream pb.DevPodWSLService_StdinServer) error {
    return nil
}

func (s *Server) Stdout(req *pb.Empty, stream pb.DevPodWSLService_StdoutServer) error {
    return nil
}

func (s *Server) Stderr(req *pb.Empty, stream pb.DevPodWSLService_StderrServer) error {
    return nil
}

func (s *Server) Status(ctx context.Context, req *pb.Empty) (*pb.AgentStatus, error) {
    return &pb.AgentStatus{Running: true}, nil
}

func (s *Server) Upload(stream pb.DevPodWSLService_UploadServer) error {
    return nil
}
```

---

### Step 7: 修改 command.go

**文件**: `cmd/command.go`

主要修改：
1. 添加环境检测 `isWindows()`
2. Linux 下：注入 agent → 启动 agent → 连接 Unix socket → gRPC 调用
3. Windows 下：保持现有 stdin pipe 逻辑

```go
import (
    "runtime"
)

func isWindows() bool {
    return runtime.GOOS == "windows"
}

// 修改 Run 函数
func (cmd *CommandCmd) Run(...) error {
    if isWindows() {
        return cmd.runOnWindows(...)
    }
    return cmd.runOnLinux(...)
}

func (cmd *CommandCmd) runOnLinux(...) error {
    // 1. 注入 agent
    agentData, err := agent.GetAgent()
    if err != nil {
        return err
    }
    if len(agentData) > 0 {
        if err := agent.InstallAgentLocal(agentData); err != nil {
            return err
        }
    }

    // 2. 启动 agent
    // 3. 连接 Unix socket
    // 4. gRPC 调用
}
```

---

### Step 8: 修改 agent/install.go

添加 Linux 版本的安装函数：

```go
// InstallAgentLocal 安装 agent 到本地 Linux
func InstallAgentLocal(data []byte) error {
    if needsUpgradeLocal() {
        removeAgentLocal()
    }
    writeAgentLocal(data)
    chmodAgentLocal()
    return nil
}
```

---

### Step 9: 创建 agent 主函数

**文件**: `agent/main.go`

```go
package main

import (
    "log"
    "net"

    "github.com/cosysn/devpod-provider-wsl/pkg/tunnel"
    "github.com/cosysn/devpod-provider-wsl/pkg/grpc"
    "github.com/hashicorp/yamux"
)

func main() {
    // 1. 创建 Unix socket server
    server := tunnel.NewUnixServer("/var/tmp/devpod.sock")
    if err := server.Listen(); err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    // 2. 等待连接
    conn, err := server.Accept()
    if err != nil {
        log.Fatalf("Failed to accept: %v", err)
    }

    // 3. Yamux session
    yamuxSession := tunnel.NewYamuxSession(conn)

    // 4. gRPC server
    grpcServer := grpc.NewServer()
    pb.RegisterDevPodWSLServiceServer(grpcServer, &grpc.Server{})

    // 5. 启动 gRPC
    listener := tunnel.NewTunnelListener(yamuxSession)
    grpcServer.Serve(listener)
}
```

---

## 文件结构

```
pkg/
├── tunnel/
│   ├── unix_server.go    # Unix socket server
│   ├── unix_client.go    # Unix socket client
│   ├── session.go        # Yamux session 封装
│   └── listener.go       # TunnelListener
├── grpc/
│   ├── proto/
│   │   ├── tunnel.proto
│   │   ├── tunnel.pb.go
│   │   └── tunnel_grpc.pb.go
│   ├── client.go         # gRPC 客户端
│   └── server.go         # gRPC 服务端
└── agent/
    ├── install.go        # agent 安装（Windows + Linux）
    └── main.go           # agent 入口
cmd/
└── command.go           # 修改为支持 tunnel
```

## 测试

```bash
# 编译
go build -o devpod-provider-wsl .

# Linux 测试
export WSL_DISTRO=TestDistro
export COMMAND="echo hello"
./devpod-provider-wsl command
```

## 注意事项

1. Yamux 依赖版本兼容性
2. Unix socket 权限问题
3. Agent 进程生命周期管理
4. gRPC 流式传输的 EOF 处理
