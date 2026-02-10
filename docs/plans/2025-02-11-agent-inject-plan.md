# Agent 注入功能实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标:** 在 WSL Provider 中实现 agent 注入功能，包括 Linux agent binary 嵌入、检查版本、注入到 WSL。

**架构:** 使用 Go 1.16+ embed.FS 嵌入 Linux binary，在 command 命令执行时检查并注入 agent 到 `/tmp/devpod-agent`。

**技术栈:** Go 1.24+, embed.FS, Cobra, WSL

---

## 第一阶段：创建 pkg/agent 包

### Task 1: 创建 pkg/agent 目录结构

**Files:**
- Create: `pkg/agent/embed.go`
- Create: `pkg/agent/install.go`
- Create: `pkg/agent/agent.go`
- Create: `pkg/agent/embed_test.go`

**Step 1: 创建目录**

```bash
mkdir -p pkg/agent
```

**Step 2: 创建 embed.go**

```go
package agent

import "embed"

//go:embed agent-linux
var Agent embed.FS

func GetAgent() ([]byte, error) {
	return Agent.ReadFile("agent-linux")
}
```

**Step 3: 创建 install.go**

```go
package agent

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	AgentPath    = "/tmp/devpod-agent"
	AgentVersion = "v0.0.1" // 硬编码版本号
)

func InstallAgent(data []byte, distro string) error {
	// 检查是否需要升级
	if needsUpgrade(distro) {
		// 删除旧版本
		if err := removeAgent(distro); err != nil {
			return fmt.Errorf("remove old agent: %w", err)
		}
	}

	// 写入文件到 WSL
	if err := writeAgent(data, distro); err != nil {
		return fmt.Errorf("write agent: %w", err)
	}

	// 设置可执行权限
	if err := chmodAgent(distro); err != nil {
		return fmt.Errorf("chmod agent: %w", err)
	}

	return nil
}

func needsUpgrade(distro string) bool {
	cmd := exec.Command("wsl.exe", "-d", distro, "-e", "sh", "-c",
		fmt.Sprintf("[ -f %s ] && %s --version 2>/dev/null || echo 'not found'", AgentPath, AgentPath))
	output, _ := cmd.Output()
	return !strings.Contains(string(output), AgentVersion)
}

func removeAgent(distro string) error {
	cmd := exec.Command("wsl.exe", "-d", distro, "-e", "rm", "-f", AgentPath)
	return cmd.Run()
}

func writeAgent(data []byte, distro string) error {
	// 通过 stdin 传入 WSL
	cmd := exec.Command("wsl.exe", "-d", distro, "-e", "sh", "-c",
		fmt.Sprintf("cat > %s", AgentPath))
	cmd.Stdin = strings.NewReader(string(data))
	return cmd.Run()
}

func chmodAgent(distro string) error {
	cmd := exec.Command("wsl.exe", "-d", distro, "-e", "chmod", "+x", AgentPath)
	return cmd.Run()
}
```

**Step 4: 创建 agent.go**

```go
package agent

import "fmt"

const (
	AgentPath    = "/tmp/devpod-agent"
	AgentVersion = "v0.0.1"
)

func RunAgent() error {
	// agent 启动逻辑，后续实现
	return fmt.Errorf("agent not implemented yet")
}
```

**Step 5: 创建 embed_test.go**

```go
package agent

import "testing"

func TestGetAgent(t *testing.T) {
	data, err := GetAgent()
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Agent binary is empty")
	}
}
```

**Step 6: 运行测试**

```bash
go test ./pkg/agent/... -v
Expected: FAIL (embed file not found - expected)
```

**Step 7: 提交**

```bash
git add pkg/agent/
git commit -m "feat: create pkg/agent package structure"
```

---

## 第二阶段：更新构建脚本

### Task 2: 更新 hack/build.sh 为两步构建

**Files:**
- Modify: `hack/build.sh`

**Step 1: 备份并更新构建脚本**

```bash
cat > hack/build.sh << 'EOF'
#!/bin/bash
set -e

cd "$(dirname "$0")/.."

VERSION=${1:-"v0.0.1"}
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "Building devpod-provider-wsl ${VERSION}..."
echo ""

# Step 1: 构建 Linux agent
echo "[1/3] Building Linux agent..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o agent-linux .

# Step 2: 构建 Windows provider (嵌入 agent)
echo "[2/3] Building Windows provider..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o release/devpod-provider-wsl-amd64.exe .

# 清理临时文件
rm -f agent-linux

# Step 3: 生成 provider.yaml
echo "[3/3] Generating provider.yaml..."
go run ./hack/provider/main.go ${VERSION} > provider.yaml

echo ""
echo "========================================"
echo "Build complete!"
echo "========================================"
echo ""
echo "Binary: release/devpod-provider-wsl-amd64.exe"
echo "Provider: provider.yaml"
EOF
```

**Step 2: 运行测试**

```bash
./hack/build.sh v0.0.1
Expected: 构建成功，release/ 目录生成 exe
```

**Step 3: 提交**

```bash
git add hack/build.sh
git commit -m "chore: update build.sh for two-step build"
```

---

## 第三阶段：集成到 command 命令

### Task 3: 更新 cmd/command.go 调用注入逻辑

**Files:**
- Modify: `cmd/command.go`

**Step 1: 更新 import 和 RunCommand**

```go
package cmd

import (
	"github.com/cosysn/devpod-provider-wsl/pkg/agent"
	"github.com/cosysn/devpod-provider-wsl/pkg/options"
	"github.com/cosysn/devpod-provider-wsl/pkg/pipe"
	"github.com/cosysn/devpod-provider-wsl/pkg/wsl"
	"github.com/spf13/cobra"
)

var commandCmd = &cobra.Command{
	Use:   "command",
	Short: "Establish communication pipe with WSL",
	RunE: func(cmd *cobra.Command, args []string) error {
		distro, err := options.GetWDLDistro()
		if err != nil {
			return err
		}

		// 获取嵌入的 agent
		agentData, err := agent.GetAgent()
		if err != nil {
			return fmt.Errorf("get embedded agent: %w", err)
		}

		// 注入 agent 到 WSL
		if err := agent.InstallAgent(agentData, distro); err != nil {
			return fmt.Errorf("install agent: %w", err)
		}

		// 启动管道通信
		return pipe.RunPipeServer(distro)
	},
}
```

**Step 2: 运行测试**

```bash
go build -o devpod-provider-wsl.exe .
Expected: 编译成功
```

**Step 3: 提交**

```bash
git add cmd/command.go
git commit -m "feat: integrate agent injection into command"
```

---

## 第四阶段：测试验证

### Task 4: 端到端测试

**Step 1: 构建**

```bash
./hack/build.sh v0.0.1
```

**Step 2: 测试注入**

```bash
# 清除旧版本
wsl.exe -d Ubuntu-22.04 -e rm -f /tmp/devpod-agent

# 执行 init 命令测试
./release/devpod-provider-wsl-amd64.exe init

# 检查文件是否存在
wsl.exe -d Ubuntu-22.04 -e ls -la /tmp/devpod-agent
Expected: 文件存在且可执行
```

**Step 3: 提交**

```bash
git add -A
git commit -m "feat: implement agent injection (closes #N)"
```

---

## 第五阶段：更新 Windows 构建脚本

### Task 5: 更新 hack/build.bat

**Files:**
- Modify: `hack/build.bat`

**Step 1: 更新 bat 脚本**

```bat
@echo off
setlocal enabledelayedexpansion

REM Build script for devpod-provider-wsl
REM Usage: .\hack\build.bat [version]
REM Example: .\hack\build.bat v0.0.1

if "%1"=="" (
    set VERSION=v0.0.1
) else (
    set VERSION=%1
)

set SCRIPT_DIR=%~dp0
set ROOT_DIR=%SCRIPT_DIR%..

echo Building devpod-provider-wsl %VERSION%...
echo.

REM Create release directory
if not exist "%ROOT_DIR%\release" mkdir "%ROOT_DIR%\release"

set LDFLAGS=-s -w -X main.version=%VERSION%

REM Step 1: Build Linux agent
echo [1/3] Building Linux agent...
cd /d "%ROOT_DIR%"
set GOOS=linux
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o agent-linux .

REM Step 2: Build Windows provider
echo [2/3] Building Windows provider...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o "release/devpod-provider-wsl-amd64.exe" .

REM Cleanup
del agent-linux

REM Step 3: Generate provider.yaml
echo [3/3] Generating provider.yaml...
cd /d "%ROOT_DIR%"
go run ./hack/provider/main.go %VERSION% > provider.yaml

REM Generate SHA256 checksum
echo Generating SHA256 checksum...
cd /d "%ROOT_DIR%\release"
for /f "skip=1 tokens=*" %%a in ('certutil -hashfile devpod-provider-wsl-amd64.exe SHA256') do (
    echo %%a > devpod-provider-wsl-amd64.exe.sha256
    goto checksum_done
)
:checksum_done

REM Create zip package
echo Packaging...
cd /d "%ROOT_DIR%\release"
del devpod-provider-wsl-%VERSION%.zip 2>nul
powershell -Command "Compress-Archive -Path 'devpod-provider-wsl-amd64.exe', 'devpod-provider-wsl-amd64.exe.sha256' -DestinationPath 'devpod-provider-wsl-%VERSION%.zip' -Force"

echo.
echo ========================================
echo Build complete!
echo ========================================
echo.
echo Binary: release/devpod-provider-wsl-amd64.exe
echo SHA256: release/devpod-provider-wsl-amd64.exe.sha256
echo Zip:    release/devpod-provider-wsl-%VERSION%.zip
```

**Step 2: 测试 Windows 构建**

```cmd
.\hack\build.bat v0.0.1
```

**Step 3: 提交**

```bash
git add hack/build.bat
git commit -m "chore: update build.bat for two-step build"
```

---

## 任务清单

- [ ] Task 1: 创建 pkg/agent 目录结构
- [ ] Task 2: 更新 hack/build.sh
- [ ] Task 3: 更新 cmd/command.go
- [ ] Task 4: 端到端测试
- [ ] Task 5: 更新 hack/build.bat

---

**计划完成，保存到 `docs/plans/2025-01-20-agent-inject-plan.md`**

**执行方式选择：**

1. **Subagent-Driven (本会话)** - 每个任务由子 agent 执行，我review后继续
2. **Parallel Session (新会话)** - 新开 session 使用 executing-plans 批量执行

选择哪种方式？
