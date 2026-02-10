package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/cosysn/devpod-provider-wsl/pkg/agent"
	"github.com/cosysn/devpod-provider-wsl/pkg/wsl"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
)

// CommandCmd holds the cmd flags
type CommandCmd struct{}

// NewCommandCmd defines a command
func NewCommandCmd() *cobra.Command {
	cmd := &CommandCmd{}
	commandCmd := &cobra.Command{
		Use:   "command",
		Short: "Command an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			wslProvider, err := wsl.NewProvider(context.Background(), log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(
				context.Background(),
				wslProvider,
				provider.FromEnvironment(),
				log.Default,
			)
		},
	}

	return commandCmd
}

// Run 负责处理 devpod-provider-wsl.exe command 的核心逻辑
func (cmd *CommandCmd) Run(
	ctx context.Context,
	providerWsl *wsl.WslProvider,
	machine *provider.Machine,
	logs log.Logger,
) error {
	distro := providerWsl.Config.WSLDistro

	// 注入 agent 到 WSL
	agentData, err := agent.GetAgent()
	if err != nil {
		return fmt.Errorf("get embedded agent: %w", err)
	}
	if len(agentData) > 0 {
		if err := agent.InstallAgent(agentData, distro); err != nil {
			return fmt.Errorf("install agent: %w", err)
		}
	}

	// 1. 获取 DevPod 传入的原始指令
	targetCommand := os.Getenv("COMMAND")
	if targetCommand == "" {
		logs.Errorf("COMMAND environment variable is required")
		os.Exit(1)
	}

	// 2. 净化环境：通过环境变量压制 WSL 的系统级警告（如代理提示、编码提示）
	// WSL_UTF8=1: 强制 UTF-8 编码
	// WSL_PROXY=0: 尝试压制自动代理警告（部分版本支持）
	// DONT_SET_WSL_PROXY: 压制某些特定版本的网络警告
	os.Setenv("WSL_UTF8", "1")
	os.Setenv("WSL_PROXY", "0")
	os.Setenv("DONT_SET_WSL_PROXY", "1")

	// 3. 构建 WSL 启动参数
	// -d: 指定分发版
	// --: 停止 wsl.exe 参数解析，后续全传给 bash
	// --noprofile, --norc: 核心！跳过 .bashrc, 杜绝欢迎词污染 Stdout
	// -s: 让 bash 从标准输入读取指令
	wslArgs := []string{"-d", distro, "--", "bash", "--noprofile", "--norc", "-s"}

	wslcmd := exec.CommandContext(ctx, "wsl.exe", wslArgs...)

	// 4. 核心：建立 Stdin 管道，用于无损传输 targetCommand
	stdin, err := wslcmd.StdinPipe()
	if err != nil {
		logs.Errorf("Failed to create stdin pipe: %v", err)
		return err
	}

	// 5. 流量绑定：将 WSL 的输出直接接到本地，但仅限 Stdout
	wslcmd.Stdout = os.Stdout
	// 将 Stderr 接到本地 Stderr，这样 wsl: 检测到代理 这种警告会显示在终端但不会干扰数据
	wslcmd.Stderr = os.Stderr

	// 6. 信号监听与平滑处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			if wslcmd.Process != nil {
				_ = wslcmd.Process.Kill()
			}
		case <-ctx.Done():
		}
	}()

	// 7. 启动 WSL 进程
	if err := wslcmd.Start(); err != nil {
		logs.Errorf("WSL execution error: %v", err)
		return err
	}

	// 8. 异步注入指令：解决转义和回显的关键
	go func() {
		defer stdin.Close()

		// A. 发送初始化序列：
		// - stty -echo: 彻底禁用终端回显（防止收到重复的指令字符串）
		// - export TERM=dumb: 告诉内部程序不要发 ANSI 颜色代码
		// - set +v / set +x: 确保执行过程完全静默
		setup := "stty -echo 2>/dev/null; export TERM=dumb; set +v; set +x\n"
		_, _ = stdin.Write([]byte(setup))

		// B. 写入真正的脚本内容
		// 因为是通过 Stdin 写入，所以 targetCommand 里的引号不再需要担心转义问题
		_, _ = stdin.Write([]byte(targetCommand + "\n"))

		// C. 强制收尾
		_, _ = stdin.Write([]byte("exit\n"))
	}()

	// 9. 等待 WSL 任务完成
	err = wslcmd.Wait()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// 将内部 Linux 进程的退出码原样返回给 DevPod
			os.Exit(exitError.ExitCode())
		}
		logs.Errorf("WSL process finished with error: %v", err)
		return err
	}

	return nil
}
