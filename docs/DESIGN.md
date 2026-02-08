# DevPod WSL Provider 设计文档

## 概述

DevPod WSL Provider 是一个原生的 WSL 开发环境 provider，旨在替代官方基于 Docker 的 WSL 支持方案。该 provider 通过命名管道在 Windows 和 WSL 之间建立持久通信通道，让 DevPod 能够直接在 WSL 内管理开发工作区。

## 背景与动机

官方 Docker provider 在 WSL 环境下存在以下问题：

1. **配置复杂**：需要在 WSL 内开启 Docker 代理，Windows 端也需要配置连接代理
2. **不支持自动管理 WSL 生命周期**：长时间不使用时无法自动关闭 WSL
3. **文件系统性能差**：代码下载到 Windows 文件系统再挂载到 WSL，性能不佳
4. **编码问题**：Windows 和 Linux 编码不一致，需要转换

WSL Provider 的解决方案：

1. **简化配置**：直接使用 WSL 原生环境，无需 Docker 代理
2. **自动生命周期管理**：自动启动/关闭 WSL，支持空闲超时
3. **高性能**：代码直接存放在 WSL 文件系统内
4. **原生体验**：与原生 Linux 开发环境一致

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Windows                                   │
│                                                                 │
│   ┌──────────────┐     SSH 复用同一管道      ┌────────────────┐│
│   │ Named Pipe   │◀─────── tunnel ────────▶  │ SSH (workspace)││
│   │ Server       │                          │                ││
│   └──────────────┘                          │ ssh workspace  ││
│        │                                    │ scp workspace  ││
│        │ wsl.exe                            │ rsync          ││
│        │ (生命周期管理)                      └────────────────┘│
│        ▼                                                         │
└─────────────────────────────────────────────────────────────────┘
         │
         │ (9p 协议 - 本地访问)
         ▼
┌─────────────────────────────────────────────────────────────────┐
│                     WSL (Ubuntu 等)                              │
│                                                                 │
│   ┌──────────────┐                                               │
│   │ DevPod       │  ←── ${COMMAND}                                │
│   │ Agent        │                                               │
│   └──────────────┘                                               │
│         │                                                        │
│         ▼                                                        │
│   ┌──────────────┐                                               │
│   │ Workspace    │  ←── 操作容器                                  │
│   │ Container     │                                               │
│   └──────────────┘                                               │
└─────────────────────────────────────────────────────────────────┘
```

### 核心组件

| 组件 | 职责 |
|------|------|
| Provider (Windows) | 提供命名管道服务器，管理 WSL 生命周期 |
| Named Pipe Server | 建立并保持持久管道，透传数据 |
| WSL socat | 桥接命名管道到 stdin/stdout |
| DevPod Agent | 注入到 WSL 内，管理 workspace 容器 |

### 通信机制

**命名管道 + 9p 挂载**

- Windows 端创建命名管道：`\\.\pipe\devpod-wsl`
- WSL 2 内通过 9p 协议直接访问：`/mnt/wsl$/Ubuntu/pipe/devpod-wsl`
- 使用 socat 桥接到标准输入输出

**多会话复用**

- 一个命名管道实例对应一个 workspace
- 多个 SSH/SCP/RSYNC 会话复用同一管道
- SSH 配置使用 `ProxyCommand` 通过管道传输

## Provider 命令

| 命令 | 类型 | 职责 |
|------|------|------|
| `init` | 必需 | 检查 WSL 环境（版本、发行版、磁盘空间、工具） |
| `command` | **核心** | 建立持久管道，透传数据 |
| `start` | 必需 | 启动指定 WSL 发行版 |
| `stop` | 必需 | 关闭指定 WSL 发行版 |
| `status` | 必需 | 检查 WSL 运行状态 |

## Provider 配置项

```yaml
options:
  WSL_DISTRO:
    description: "WSL 发行版名称（如 Ubuntu-22.04）"
    required: true
  IDLE_TIMEOUT:
    description: "空闲超时时间（分钟），超时后自动关闭 WSL"
    default: "30"
```

### 环境变量

Provider 通过环境变量接收配置：

| 变量 | 来源 | 说明 |
|------|------|------|
| `WSL_DISTRO` | Provider 选项 | 要连接的 WSL 发行版 |
| `IDLE_TIMEOUT` | Provider 选项 | 空闲超时时间 |
| `DEVPOD` | DevPod 内置 | DevPod CLI 路径 |
| `${COMMAND}` | DevPod | 需要在 WSL 内执行的命令 |

## 工作流程

### 初始化 (init)

```
1. 检查 wsl.exe 是否可用
2. 检查 WSL 版本是否为 2
3. 检查指定发行版是否已安装
4. 检查磁盘空间（至少 5GB）
5. 检查必要工具（git, curl）
```

### 建立管道 (command)

```
1. Provider 创建命名管道服务器
2. 等待 WSL 内 socat 连接
3. 启动 WSL 内的 socat 进程
4. 建立双向数据透传通道
5. 保持连接不退出
```

### WSL 生命周期

| 操作 | 命令 |
|------|------|
| 启动 WSL | `wsl.exe -d ${WSL_DISTRO}` |
| 关闭 WSL | `wsl.exe --terminate ${WSL_DISTRO}` |
| 检查状态 | `wsl.exe -d ${WSL_DISTRO} -e echo running` |

## 项目结构

```
devpod-provider-wsl/
├── .devcontainer/
│   └── devcontainer.json          # VSCode DevContainer 配置
├── cmd/
│   ├── init.go                    # init 命令实现
│   ├── command.go                  # command 命令实现（核心）
│   ├── start.go                    # start 命令实现
│   ├── stop.go                     # stop 命令实现
│   └── status.go                   # status 命令实现
├── pkg/
│   ├── wsl/
│   │   └── wsl.go                  # WSL 操作封装
│   └── pipe/
│       └── pipe.go                 # 命名管道封装
├── hack/
│   ├── build.sh                    # 构建脚本
│   └── provider/
│       ├── main.go                 # provider.yaml 生成器
│       └── provider.yaml           # provider.yaml 模板
├── docs/
│   └── DESIGN.md                   # 本设计文档
├── main.go                         # 程序入口
└── go.mod                          # Go 模块文件
```

## 实现细节

### 命名管道服务器

```go
func runPipeServer(distro string) error {
    pipeName := `\\.\pipe\devpod-wsl`

    // 创建命名管道服务器
    listener, err := CreateNamedPipe(pipeName)
    if err != nil {
        return fmt.Errorf("create pipe: %w", err)
    }
    defer listener.Close()

    fmt.Println("Pipe server started, waiting for WSL connection...")

    // 启动 WSL 内的 socat
    wslCmd := exec.Command("wsl.exe", "-d", distro, "--", "socat",
        "UNIX-CONNECT:/mnt/wsl$/"+distro+"/pipe/devpod-wsl", "STDIO")

    // 等待管道连接
    conn, err := listener.Accept()
    if err != nil {
        return fmt.Errorf("accept pipe: %w", err)
    }
    defer conn.Close()

    // 透传数据
    io.Copy(conn, os.Stdin)
    io.Copy(os.Stdout, conn)

    return nil
}
```

### WSL 内 socat 启动

```bash
# 在 WSL 内执行
socat UNIX-CONNECT:/mnt/wsl$/Ubuntu/pipe/devpod-wsl STDIO
```

## Agent 配置

```yaml
agent:
  path: ${DEVPOD}
  inactivityTimeout: ${IDLE_TIMEOUT}m
```

- `path`: DevPod agent 路径
- `inactivityTimeout`: 空闲超时时间，由 agent 负责检测并触发关闭

## 构建与发布

### 构建脚本 (hack/build.sh)

```bash
#!/bin/bash
set -e

VERSION=${1:-"v0.0.1"}
LDFLAGS="-s -w -X main.version=${VERSION}"

# 构建 Windows AMD64
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" \
    -o release/devpod-provider-wsl-windows-amd64.exe .

# 生成 provider.yaml
go run ./hack/provider/main.go ${VERSION} > provider.yaml
```

### 发布文件

| 文件 | 说明 |
|------|------|
| `release/devpod-provider-wsl-windows-amd64.exe` | Windows 可执行文件 |
| `provider.yaml` | Provider 配置（含校验和） |

## 使用方法

### 添加 Provider

```bash
devpod provider add ./path/to/provider.yaml
```

### 配置 Provider

```bash
# 设置 WSL 发行版
devpod provider option set devpod-provider-wsl WSL_DISTRO Ubuntu-22.04

# 设置空闲超时（可选，默认 30 分钟）
devpod provider option set devpod-provider-wsl IDLE_TIMEOUT 60
```

### 启动 Workspace

```bash
devpod up github.com/example/my-project
```

## 限制与约束

1. **WSL 版本**：仅支持 WSL 2
2. **操作系统**：仅支持 Windows
3. **命名管道访问**：WSL 内需要支持 9p 文件系统挂载
4. **发行版**：需要预先安装并配置好 WSL 发行版

## 未来优化方向

1. **并行多 workspace 支持**：每个 workspace 使用独立命名管道
2. **磁盘空间监控**：自动清理临时文件
3. **性能优化**：减少管道通信延迟
4. **错误恢复**：自动重连断开的管道
