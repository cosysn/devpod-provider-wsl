---

## 配合 DevPod 的集成测试

### 背景

DevPod 通过 provider.yaml 配置和调用 WSL Provider。本节测试验证 provider 能正确与 DevPod 集成。

### TestDevPodInitProvider - DevPod 初始化 Provider

**测试目的**:
验证 DevPod 能正确初始化 WSL Provider

**测试步骤**:
```bash
# 1. 添加 provider
devpod provider add ./provider.yaml

# 2. 设置 WSL 发行版
devpod provider option set devpod-provider-wsl WSL_DISTRO Ubuntu-22.04

# 3. 初始化（调用 init 命令）
devpod provider init devpod-provider-wsl
```

**验证点**:
1. provider 添加成功
2. 选项设置成功
3. init 命令执行并输出检查结果

**预期输出**:
```
Checking WSL version...
  WSL version: 2
Distribution 'Ubuntu-22.04' found
Disk space: OK (>= 5GB)
Tools: OK ([git curl])

WSL environment check passed!
```

---

### TestDevPodUpWorkspace - DevPod up workspace

**测试目的**:
验证 `devpod up` 能通过 WSL Provider 创建 workspace

**前置条件**:
- provider 已配置
- WSL 发行版存在

**测试步骤**:
```bash
# 设置工作目录（可选）
export WORKSPACE_FOLDER=~/.devpod/workspaces

# 创建 workspace
devpod up github.com/example/my-project --provider devpod-provider-wsl
```

**验证点**:
1. DevPod 调用 provider command 命令
2. 管道建立成功
3. DevPod Agent 在 WSL 内启动
4. Workspace 容器创建成功
5. 返回可访问的 URL

**预期结果**:
```
> DevPod v0.x.x
> Provisioning workspace...
> Using WSL provider (Ubuntu-22.04)
> Starting DevPod agent...
> Agent started
> Creating container...
> Container ready
> Workspace URL: https://my-project.devpod.sh
```

---

### TestDevPodDownWorkspace - DevPod down workspace

**测试目的**:
验证 `devpod down` 能正确停止 workspace

**测试步骤**:
```bash
devpod down my-project --provider devpod-provider-wsl
```

**验证点**:
1. 容器停止
2. WSL 可能保持运行（或根据配置停止）

---

### TestDevPodSSHConnection - DevPod SSH 连接

**测试目的**:
验证通过 SSH 连接到 workspace

**测试步骤**:
```bash
# 直接 SSH 连接
devpod ssh my-project --provider devpod-provider-wSL

# 或使用 SSH 配置
ssh my-project.devpod
```

**验证点**:
1. SSH 通过管道连接到 WSL 内
2. 能执行命令
3. 连接稳定

---

### TestDevPodIDEConnection - DevPod IDE 连接

**测试目的**:
验证 VSCode 等 IDE 能通过 DevPod 连接到 workspace

**测试步骤**:
```bash
# 打开 VSCode
devpod up github.com/example/my-project --ide vscode --provider devpod-provider-wsl
```

**验证点**:
1. VSCode Remote SSH 连接建立
2. 源码同步完成
3. 终端功能正常

---

### TestDevPodMultipleWorkspaces - 多 Workspace 测试

**测试目的**:
验证能同时管理多个 workspace

**测试步骤**:
```bash
# 创建多个 workspace
devpod up project-a --provider devpod-provider-wsl
devpod up project-b --provider devpod-provider-wsl

# 验证两个都运行
devpod status
```

**验证点**:
1. 每个 workspace 有独立管道
2. 无冲突
3. 资源使用正常

---

### TestDevPodAgentLifecycle - Agent 生命周期

**测试目的**:
验证 DevPod Agent 在 WSL 内的生命周期管理

**测试步骤**:
```bash
# 启动 workspace
devpod up my-project

# 等待空闲超时
# 默认 30 分钟

# 验证自动停止
devpod status  # 应显示 Stopped

# 重新连接
devpod up my-project  # 应恢复
```

**验证点**:
1. Agent 正确检测空闲
2. 自动关闭 WSL（根据配置）
3. 状态恢复正确

---

### DevPod 集成测试配置

```yaml
# provider.yaml (DevPod 集成)
name: devpod-provider-wsl
version: v0.0.1
options:
  WSL_DISTRO:
    description: "WSL 发行版名称"
    required: true
    default: "Ubuntu-22.04"
  IDLE_TIMEOUT:
    description: "空闲超时时间（分钟）"
    default: "30"
agent:
  path: ${DEVPOD}
  inactivityTimeout: ${IDLE_TIMEOUT}m
  exec:
    shutdown: |-
      ${DEVPOD_PROVIDER_WSL} stop
binaries:
  DEVPOD_PROVIDER_WSL:
    - os: windows
      arch: amd64
      path: ./devpod-provider-wsl.exe
exec:
  init: ${DEVPOD_PROVIDER_WSL} init
  command: ${DEVPOD_PROVIDER_WSL} command
  start: ${DEVPOD_PROVIDER_WSL} start
  stop: ${DEVPOD_PROVIDER_WSL} stop
  status: ${DEVPOD_PROVIDER_WSL} status
```

---

### DevPod 测试命令速查

| 操作 | 命令 |
|------|------|
| 添加 provider | `devpod provider add ./provider.yaml` |
| 设置选项 | `devpod provider option set devpod-provider-wsl WSL_DISTRO Ubuntu-22.04` |
| 初始化 | `devpod provider init devpod-provider-wsl` |
| 创建 workspace | `devpod up <repo> --provider devpod-provider-wsl` |
| SSH 连接 | `devpod ssh <workspace>` |
| 停止 | `devpod down <workspace>` |
| 查看状态 | `devpod status` |
| 删除 workspace | `devpod delete <workspace>` |
| 删除 provider | `devpod provider delete devpod-provider-wsl` |

---

### CI 环境 DevPod 测试

```bash
# GitHub Actions 示例
name: DevPod WSL E2E Tests

on: [push, pull_request]

jobs:
  devpod-e2e:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build Provider
        run: |
          GOOS=windows go build -o devpod-provider-wsl.exe .

      - name: Add Provider
        run: |
          devpod provider add ./provider.yaml
          devpod provider option set devpod-provider-wsl WSL_DISTRO ${{ env.WSL_DISTRO }}

      - name: Init Provider
        run: devpod provider init devpod-provider-wsl

      - name: Run E2E Tests
        run: go test -tags e2e -v ./e2e/...
        env:
          WSL_DISTRO: ${{ env.WSL_DISTRO }}
```

---

## 测试覆盖总结

### 功能覆盖

| 分类 | 测试数 | P0 | P1 |
|------|--------|----|----|
| 基础命令 | 5 | 3 | 2 |
| 生命周期 | 2 | 2 | 0 |
| 错误处理 | 3 | 1 | 2 |
| 管道测试 | 2 | 1 | 1 |
| SSH/性能 | 3 | 0 | 3 |
| DevPod 集成 | 6 | 2 | 4 |
| **总计** | **21** | **9** | **12** |

### 测试优先级

**P0 (必须通过)**:
1. TestStatus
2. TestStartStop
3. TestInit
4. TestCommandTimeout
5. TestWSLNotRunningStart
6. TestFullWorkflow
7. TestDevPodInitProvider
8. TestDevPodUpWorkspace
9. TestMultiSSHClient

**P1 (应该通过)**:
- 其余测试
