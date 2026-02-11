# DevPod WSL Tunnel 设计文档

## 概述

本文档描述 DevPod WSL Provider 的 tunnel 能力设计，用于在 Windows 和 WSL 之间建立双向通信通道。

## 背景

当前 DevPod WSL Provider 使用 stdin pipe 传递命令，存在以下限制：
- 只能单向传递命令
- 缺乏状态反馈能力
- 不支持多路复用

## 目标

设计一个分层的通信架构，支持：
- Windows 和 WSL 之间的双向通信
- 多路复用（Yamux）
- gRPC 协议支持
- 管道数据传输（stdin/stdout/stderr）
- Agent 跟随子进程生命周期

## 架构设计

### 分层架构

```
┌─────────────────────────────────────────────────────────┐
│  Layer 4: gRPC - DevPodWSLService                        │
│  - Exec (stream), Start, Stop                           │
│  - Status, Upload                                       │
│  - Stdin (stream), Stdout (stream), Stderr (stream)     │
├─────────────────────────────────────────────────────────┤
│  Layer 3: Net 接口                                      │
│  - TunnelListener (implements net.Listener)             │
│  - TunnelConn (implements net.Conn)                     │
├─────────────────────────────────────────────────────────┤
│  Layer 2: Yamux                                         │
│  - WSL 端管理 Yamux session                             │
│  - 动态创建 stream                                       │
├─────────────────────────────────────────────────────────┤
│  Layer 1: Unix Socket                                   │
│  - WSL server: /var/tmp/devpod.sock (可配置）           │
│  - Windows 通过 9p 访问                                  │
│  - 路径: /mnt/wsl$/<DISTRO>/<SOCKET_PATH>              │
└─────────────────────────────────────────────────────────┘
```

### 组件关系

```
┌─────────────────────────────────────────────────────────┐
│ Windows (Provider - command)                            │
│                                                         │
│  DevPod ──► command ──► gRPC Client                     │
│                            │                            │
│                            ▼                            │
│                     Unix Socket Client                   │
│                            │                            │
│                            │ 9p                         │
└────────────────────────────┼────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────┐
│ WSL (Agent 进程)                                        │
│                                                         │
│  1. 创建 Unix Socket Server (/var/tmp/devpod.sock)     │
│  2. 启动 Yamux Session                                  │
│  3. 启动 gRPC Server                                    │
│  4. 接收命令、执行、返回结果                             │
└─────────────────────────────────────────────────────────┘
```

## 详细设计

### Layer 1: Unix Socket

#### 组件

**WSL 端** (`pkg/tunnel/unix_server.go`):
- 监听 Unix socket
- 支持持久化（进程退出后 socket 文件保留）
- 支持客户端重连

**Windows 端** (`pkg/tunnel/unix_client.go`):
- 通过 9p 路径连接 Unix socket
- 路径格式: `/mnt/wsl$/<DISTRO>/<SOCKET_PATH>`

#### 配置

```yaml
# provider.yaml
WSL_DISTRO: Ubuntu-22.04
WSL_SOCKET_PATH: /var/tmp/devpod.sock  # 默认值
```

### Layer 2: Yamux

#### 设计

- WSL 端管理 Yamux session
- 动态创建 stream
- 支持多路复用

### Layer 3: Net 接口

#### TunnelListener

```go
type TunnelListener struct {
    session *yamux.Session
}

func (l *TunnelListener) Accept() (net.Conn, error)
func (l *TunnelListener) Addr() net.Addr
func (l *TunnelListener) Close() error
```

#### TunnelConn

封装 Yamux stream，提供 `net.Conn` 接口。

### Layer 4: gRPC

#### 服务定义

```protobuf
service DevPodWSLService {
    // Command 服务
    rpc Exec(stream Request) returns (stream Response);
    rpc Start(StartRequest) returns (StartResponse);
    rpc Stop(StopRequest) returns (StopResponse);

    // Agent 服务
    rpc Status(Empty) returns (AgentStatus);
    rpc Upload(stream Chunk) returns (UploadResponse);

    // 管道数据传输
    rpc Stdin(stream Data) returns (Empty);
    rpc Stdout(Empty) returns (stream Data);
    rpc Stderr(Empty) returns (stream Data);
}
```

#### RPC 说明

| RPC | 方向 | 说明 |
|-----|------|------|
| Exec | 双向流 | 执行命令，双向传递输入输出 |
| Start | 请求-响应 | 启动进程 |
| Stop | 请求-响应 | 停止进程 |
| Status | 请求-响应 | 获取 agent 状态 |
| Upload | 双向流 | 上传文件到 WSL |
| Stdin | 客户端流 | 发送 stdin 数据 |
| Stdout | 服务端流 | 接收 stdout 数据 |
| Stderr | 服务端流 | 接收 stderr 数据 |

## 执行方式

### Windows 执行方式

```
┌─────────────────────────────────────────────────────────┐
│ Windows (Provider - command)                            │
│                                                         │
│  DevPod ──► command ──► gRPC Client                     │
│                            │                            │
│                            ▼                            │
│                     Unix Socket Client                   │
│                            │                            │
│                            │ 9p                         │
└────────────────────────────┼────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────┐
│ WSL (Agent 进程)                                        │
│                                                         │
│  1. 创建 Unix Socket Server (/var/tmp/devpod.sock)     │
│  2. 启动 Yamux Session                                  │
│  3. 启动 gRPC Server                                    │
│  4. 接收命令、执行、返回结果                             │
└─────────────────────────────────────────────────────────┘
```

### Linux 执行方式（测试）

```
┌─────────────────────────────────────────────────────────┐
│ Linux (Provider - command)                              │
│                                                         │
│  DevPod ──► command ──► gRPC Client                     │
│                            │                            │
│                            ▼                            │
│                     Unix Socket Client                   │
│                            │                            │
│                            │ 本地连接                   │
└────────────────────────────┼────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────┐
│ Linux (Agent 进程)                                       │
│                                                         │
│  1. 创建 Unix Socket Server (/var/tmp/devpod.sock)     │
│  2. 启动 Yamux Session                                  │
│  3. 启动 gRPC Server                                    │
│  4. 接收命令、执行、返回结果                             │
└─────────────────────────────────────────────────────────┘
```

### 路径一致性

| 路径 | Windows | Linux |
|-----|---------|-------|
| Agent 路径 | `/var/tmp/devpod-agent` | `/var/tmp/devpod-agent` |
| Socket 路径 | `/var/tmp/devpod.sock` | `/var/tmp/devpod.sock` |

### 环境检测

```go
func isWindows() bool {
    return runtime.GOOS == "windows"
}
```

- **Windows**: 通过 9p 访问 WSL 内的 Unix socket
- **Linux**: 直接访问本地 Unix socket

## 测试方案

### Linux 本地测试

1. 编译 provider（Linux 版本）
2. 注入 agent 到 `/var/tmp/devpod-agent`
3. 启动 agent（本地 Unix socket server）
4. command 连接 socket，执行 gRPC 调用
5. 验证：连接、数据传输、命令执行

## 文件结构

```
pkg/
├── tunnel/
│   ├── unix_server.go    # Unix socket server (WSL)
│   ├── unix_client.go    # Unix socket client (Windows)
│   ├── listener.go       # TunnelListener
│   ├── conn.go           # TunnelConn
│   └── tunnel.go         # 公共接口
├── grpc/
│   ├── proto/
│   │   └── tunnel.proto  # 服务定义
│   ├── client.go         # gRPC 客户端
│   └── server.go         # gRPC 服务端
└── options/
    └── options.go        # 配置（新增 WSL_SOCKET_PATH）
cmd/
└── command.go           # 修改为使用 tunnel
agent/
└── agent.go             # 内置 gRPC server
```

## 实施顺序

1. Layer 1: Unix Socket 层
2. Layer 2: Yamux 层
3. Layer 3: Net 接口层
4. Layer 4: gRPC 层
5. Agent 集成
6. 集成测试
