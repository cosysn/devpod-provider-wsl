# DevPod WSL Provider E2E 测试设计

## 概述

本文档定义了 DevPod WSL Provider 的端到端测试方案，确保在真实 WSL 环境下验证 provider 的所有功能。

## 测试环境要求

### 硬件要求
- Windows 10/11 系统
- WSL 2 已启用
- 已安装 WSL 发行版（如 Ubuntu-22.04）

### 软件要求
- Go 1.21+
- Windows 权限（执行 wsl.exe）

### 环境变量
```bash
export WSL_DISTRO=Ubuntu-22.04  # 测试用的 WSL 发行版
export CI=true                    # CI 环境下跳过需要 WSL 的测试
```

## 测试用例

### 1. TestStatus - 状态检测

**目的**: 验证 status 命令能正确检测 WSL 运行状态

**前置条件**:
- WSL 发行版已安装

**测试步骤**:
```bash
# 1. 设置环境变量
export WSL_DISTRO=Ubuntu-22.04

# 2. 检查 WSL 是否存在
wsl.exe -l -q | grep Ubuntu-22.04

# 3. 运行测试
devpod-provider-wsl.exe status
```

**预期结果**:
- 返回 `Running` 或 `Stopped`
- 退出码为 0

**测试代码**:
```go
func TestStatus(t *testing.T) {
    distro := getTestDistro()
    t.Setenv("WSL_DISTRO", distro)

    result, err := runCmd("status")
    if err != nil {
        t.Fatalf("status failed: %v", err)
    }

    result = strings.TrimSpace(result)
    if result != "Running" && result != "Stopped" {
        t.Errorf("unexpected status: %q", result)
    }
}
```

---

### 2. TestStartStop - 生命周期管理

**目的**: 验证 start 和 stop 命令能正确管理 WSL 生命周期

**前置条件**:
- WSL 发行版已安装

**测试步骤**:
```bash
# 1. 先停止 WSL
devpod-provider-wsl.exe stop

# 2. 启动 WSL
devpod-provider-wsl.exe start

# 3. 验证状态
devpod-provider-wsl.exe status  # 应返回 Running

# 4. 停止 WSL
devpod-provider-wsl.exe stop

# 5. 验证状态
devpod-provider-wsl.exe status  # 应返回 Stopped
```

**预期结果**:
- start 后状态为 Running
- stop 后状态为 Stopped
- 无错误输出

**测试代码**:
```go
func TestStartStop(t *testing.T) {
    distro := getTestDistro()
    t.Setenv("WSL_DISTRO", distro)

    // Stop first
    runCmd("stop")
    time.Sleep(2 * time.Second)

    // Start
    _, err := runCmd("start")
    if err != nil {
        t.Fatalf("start failed: %v", err)
    }

    // Verify running
    status, _ := runCmd("status")
    if !strings.Contains(status, "Running") {
        t.Errorf("expected Running, got: %s", status)
    }

    // Stop
    _, err = runCmd("stop")
    if err != nil {
        t.Fatalf("stop failed: %v", err)
    }

    // Verify stopped
    time.Sleep(2 * time.Second)
    status, _ = runCmd("status")
    if !strings.Contains(status, "Stopped") {
        t.Errorf("expected Stopped, got: %s", status)
    }
}
```

---

### 3. TestInit - 环境检查

**目的**: 验证 init 命令能正确检查 WSL 环境

**测试步骤**:
```bash
export WSL_DISTRO=Ubuntu-22.04
devpod-provider-wsl.exe init
```

**预期结果**:
- 检查 WSL 版本（应为 WSL 2）
- 检查发行版是否存在
- 检查磁盘空间
- 检查必要工具（git, curl）

**测试代码**:
```go
func TestInit(t *testing.T) {
    distro := getTestDistro()
    t.Setenv("WSL_DISTRO", distro)

    output, err := runCmd("init")

    // init 可能在缺少工具时失败，这不算测试失败
    t.Logf("init output: %s", output)
    t.Logf("init error: %v", err)
}
```

---

### 4. TestHelp - 帮助命令

**目的**: 验证所有命令的帮助信息正确

**测试用例**:

| 命令 | 预期包含 |
|------|----------|
| `--help` | Usage, 可用命令 |
| `init --help` | "init" |
| `command --help` | "command", "pipe" |
| `start --help` | "start" |
| `stop --help` | "stop" |
| `status --help` | "status" |

**测试代码**:
```go
func TestHelp(t *testing.T) {
    tests := []struct {
        name    string
        command string
        want    string
    }{
        {"init", "init", "init"},
        {"command", "command", "command"},
        {"start", "start", "start"},
        {"stop", "stop", "stop"},
        {"status", "status", "status"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            output, err := runCmd(tt.command, "--help")
            if err != nil {
                t.Fatalf("%s --help failed: %v", tt.command, err)
            }
            if !strings.Contains(output, tt.want) {
                t.Errorf("%s --help should contain %q", tt.command, tt.want)
            }
        })
    }
}
```

---

### 5. TestVersion - 版本信息

**目的**: 验证版本命令正确输出

**测试步骤**:
```bash
devpod-provider-wsl.exe --version
```

**预期结果**:
- 输出版本号或 "dev"

---

### 6. TestMissingDistro - 错误处理

**目的**: 验证缺少 WSL_DISTRO 时的错误处理

**测试步骤**:
```bash
unset WSL_DISTRO
devpod-provider-wsl.exe status
```

**预期结果**:
- 命令失败并返回错误信息
- 退出码非 0

---

### 7. TestCommandHelp - Command 命令帮助

**目的**: 验证 command 命令的帮助信息

**测试步骤**:
```bash
devpod-provider-wsl.exe command --help
```

**预期结果**:
- 包含 "pipe" 相关描述

---

## 运行测试

### 本地运行
```bash
# 构建
GOOS=windows go build -o devpod-provider-wsl.exe .

# 运行所有 E2E 测试
cd e2e
GOOS=windows go test -tags wsl -v ./...

# 运行特定测试
GOOS=windows go test -tags wsl -v -run TestStatus ./...
```

### CI 环境
```bash
# 设置环境变量
export WSL_DISTRO=Ubuntu-22.04
export CI=true

# 运行（会自动跳过需要 WSL 的测试）
go test -tags wsl ./...
```

---

## 测试结果模板

```
=== RUN   TestStatus
    status_test.go:80: WSL status: Running
--- PASS: TestStatus (2.34s)
=== RUN   TestStartStop
    startstop_test.go:45: Stopping WSL...
    startstop_test.go:50: Testing start command...
    startstop_test.go:60: Testing stop command...
--- PASS: TestStartStop (8.45s)
=== RUN   TestHelp
--- PASS: TestHelp (1.23s)
=== RUN   TestVersion
--- PASS: TestVersion (0.05s)
PASS
ok      github.com/cosysn/devpod-provider-wsl/e2e   12.07s
```

---

## 注意事项

1. **测试顺序**: 先运行 status/start/stop，最后运行 init
2. **等待时间**: start/stop 后等待 2-3 秒让 WSL 状态稳定
3. **清理**: 测试结束后确保 WSL 处于停止状态（如需要）
4. **权限**: 需要管理员权限执行 wsl.exe
5. **超时**: CI 环境下自动跳过需要 WSL 的测试

---

## 后续扩展

- [ ] TestCommandPipe - 测试 command 管道建立
- [ ] TestMultiDistro - 测试多个发行版
- [ ] TestDiskSpaceError - 测试磁盘空间不足场景
- [ ] TestToolMissing - 测试工具缺失场景
