# Agent 注入设计文档

## 概述

DevPod WSL Provider 在执行 `command` 命令时，需要在 WSL 内部署一个 Linux 版本的 agent 进程。该 agent 负责监听 Unix socket，与 Windows 端的 Provider 进行双向通信。

## 架构设计

### 整体架构

```
Windows Provider                    WSL (Linux)
┌─────────────────┐               ┌─────────────────────┐
│ command 命令    │ ─── inject ──▶ │ /tmp/devpod-agent  │
│                 │               │ (Linux binary)      │
│                 │               │                     │
│ agent 命令      │               │ 监听 unix socket   │
│ (嵌入的Linux)   │ ◀──────────── │ 双向通信            │
└─────────────────┘               └─────────────────────┘
```

### 核心组件

| 组件 | 职责 |
|------|------|
| `pkg/agent/embed.go` | 使用 `embed.FS` 嵌入 Linux agent 二进制 |
| `pkg/agent/install.go` | 注入逻辑（检查、删除、写入、权限设置） |
| `cmd/command.go` | command 命令实现，调用注入逻辑 |

### 目录结构

```
pkg/
├── agent/
│   ├── embed.go      # embed.FS 嵌入 Linux binary
│   └── install.go    # 注入逻辑（检查、删除、写入、权限）
```

## 注入流程

### command 命令执行流程

```
1. 检查 /tmp/devpod-agent 是否存在
2. 检查版本号，不一致则删除旧版本
3. 注入 agent 到 /tmp/devpod-agent（设置可执行权限）
4. 启动 socat + agent
```

### 版本检查

- **版本号来源**：硬编码在代码中，后续可改为从 binary 提取
- **不一致处理**：先删除旧版本，再写入新版本（防止覆盖失败）

### 注入位置

- **路径**：`/tmp/devpod-agent`
- **权限**：设置可执行权限 (`chmod +x`)

## 构建流程

### 两步构建

```bash
# Step 1: 构建 Linux agent
GOOS=linux GOARCH=amd64 go build -o agent-linux .

# Step 2: 嵌入 + 构建 Windows provider
GOOS=windows GOARCH=amd64 go build -o devpod-provider-wsl.exe .
```

### 构建脚本更新

`hack/build.sh` 需要更新为两步构建流程：

```bash
#!/bin/bash
set -e

VERSION=${1:-"v0.0.1"}
LDFLAGS="-s -w -X main.version=${VERSION}"

# Step 1: 构建 Linux agent
echo "[1/3] Building Linux agent..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o agent-linux .

# Step 2: 嵌入 agent 并构建 Windows provider
echo "[2/3] Building Windows provider..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o release/devpod-provider-wsl-amd64.exe .

# 清理临时文件
rm -f agent-linux

# Step 3: 生成 provider.yaml
echo "[3/3] Generating provider.yaml..."
go run ./hack/provider/main.go ${VERSION} > provider.yaml
```

## 实现细节

### embed.go

```go
package agent

import "embed"

//go:embed agent-linux
var Agent embed.FS

func GetAgent() ([]byte, error) {
    return fs.ReadFile("agent-linux")
}
```

### install.go

```go
package agent

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

const (
    AgentPath     = "/tmp/devpod-agent"
    AgentVersion  = "v0.0.1"  // 硬编码，后续可修改
)

func InstallAgent(data []byte, distro string) error {
    // 检查版本
    if needsUpgrade(distro) {
        // 删除旧版本
        if err := removeAgent(distro); err != nil {
            return fmt.Errorf("remove old agent: %w", err)
        }
    }

    // 写入文件
    targetPath := filepath.Join("/tmp", "devpod-agent-"+distro)
    if err := writeAgentFile(data, targetPath); err != nil {
        return fmt.Errorf("write agent: %w", err)
    }

    // 设置可执行权限
    if err := chmodAgent(targetPath, distro); err != nil {
        return fmt.Errorf("chmod agent: %w", err)
    }

    return nil
}

func needsUpgrade(distro string) bool {
    // 检查文件是否存在且版本一致
    cmd := exec.Command("wsl.exe", "-d", distro, "-e", "sh", "-c",
        fmt.Sprintf("[ -f %s ] && %s --version 2>/dev/null || echo 'not found'", AgentPath, AgentPath))
    output, _ := cmd.Output()
    return !strings.Contains(string(output), AgentVersion)
}

func removeAgent(distro string) error {
    cmd := exec.Command("wsl.exe", "-d", distro, "-e", "rm", "-f", AgentPath)
    return cmd.Run()
}

func writeAgentFile(data []byte, targetPath string) error {
    // 通过 9p 挂载写入
    // 或者通过 stdin 传入
}

func chmodAgent(path, distro string) error {
    cmd := exec.Command("wsl.exe", "-d", distro, "-e", "chmod", "+x", path)
    return cmd.Run()
}
```

### command.go 调用

```go
func RunCommand(distro string) error {
    // 1. 获取嵌入的 agent 数据
    agentData, err := agent.GetAgent()
    if err != nil {
        return fmt.Errorf("get embedded agent: %w", err)
    }

    // 2. 注入 agent
    if err := agent.InstallAgent(agentData, distro); err != nil {
        return fmt.Errorf("install agent: %w", err)
    }

    // 3. 启动 socat + agent
    // ...
}
```

## 待完成事项

- [ ] 实现 `pkg/agent/embed.go`
- [ ] 实现 `pkg/agent/install.go`
- [ ] 更新 `cmd/command.go` 调用注入逻辑
- [ ] 更新 `hack/build.sh` 为两步构建
- [ ] 添加单元测试

## 后续工作

1. **通信协议设计**：定义 provider 和 agent 之间的消息格式
2. **Unix Socket 路径**：确定 Windows 和 WSL 共享访问的 socket 路径
3. **agent 命令完善**：实现 agent 的监听和通信逻辑
