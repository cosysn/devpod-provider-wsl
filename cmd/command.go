package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/cosysn/devpod-provider-wsl/pkg/agent"
	"github.com/cosysn/devpod-provider-wsl/pkg/tunnel"
	grpcClient "github.com/cosysn/devpod-provider-wsl/pkg/grpc"
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

// isWindows 检测当前是否运行在 Windows 上
func isWindows() bool {
	return runtime.GOOS == "windows"
}

// Run 负责处理 devpod-provider-wsl command 的核心逻辑
func (cmd *CommandCmd) Run(
	ctx context.Context,
	providerWsl *wsl.WslProvider,
	machine *provider.Machine,
	logs log.Logger,
) error {
	distro := providerWsl.Config.WSLDistro

	// 获取原始指令
	targetCommand := os.Getenv("COMMAND")
	if targetCommand == "" {
		logs.Errorf("COMMAND environment variable is required")
		os.Exit(1)
	}

	if isWindows() {
		return cmd.runOnWindows(ctx, distro, targetCommand, logs)
	}
	return cmd.runOnLinux(ctx, distro, targetCommand, logs)
}

// runOnWindows Windows 环境下使用 stdin pipe
func (cmd *CommandCmd) runOnWindows(
	ctx context.Context,
	distro, targetCommand string,
	logs log.Logger,
) error {
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

	// 净化环境
	os.Setenv("WSL_UTF8", "1")
	os.Setenv("WSL_PROXY", "0")
	os.Setenv("DONT_SET_WSL_PROXY", "1")

	// 构建 WSL 启动参数
	wslArgs := []string{"-d", distro, "--", "bash", "--noprofile", "--norc", "-s"}

	wslcmd := exec.CommandContext(ctx, "wsl.exe", wslArgs...)

	stdin, err := wslcmd.StdinPipe()
	if err != nil {
		logs.Errorf("Failed to create stdin pipe: %v", err)
		return err
	}

	wslcmd.Stdout = os.Stdout
	wslcmd.Stderr = os.Stderr

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

	if err := wslcmd.Start(); err != nil {
		logs.Errorf("WSL execution error: %v", err)
		return err
	}

	go func() {
		defer stdin.Close()
		setup := "stty -echo 2>/dev/null; export TERM=dumb; set +v; set +x\n"
		_, _ = stdin.Write([]byte(setup))
		_, _ = stdin.Write([]byte(targetCommand + "\n"))
		_, _ = stdin.Write([]byte("exit\n"))
	}()

	err = wslcmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		logs.Errorf("WSL process finished with error: %v", err)
		return err
	}

	return nil
}

// runOnLinux Linux 环境下使用 tunnel (Unix socket + gRPC)
func (cmd *CommandCmd) runOnLinux(
	ctx context.Context,
	distro, targetCommand string,
	logs log.Logger,
) error {
	socketPath := tunnel.DefaultSocketPath

	// 1. 注入 agent 到本地
	agentData, err := agent.GetAgent()
	if err != nil {
		return fmt.Errorf("get embedded agent: %w", err)
	}
	if len(agentData) > 0 {
		if err := agent.InstallAgentLocal(agentData); err != nil {
			return fmt.Errorf("install agent: %w", err)
		}
		logs.Infof("Agent installed to %s", agent.AgentPath)
	}

	// 2. 启动 agent
	logs.Infof("Starting agent...")
	agentCmd := exec.CommandContext(ctx, agent.AgentPath)
	agentCmd.Stdout = os.Stdout
	agentCmd.Stderr = os.Stderr
	if err := agentCmd.Start(); err != nil {
		return fmt.Errorf("start agent: %w", err)
	}

	// 3. 连接 gRPC
	logs.Infof("Connecting to %s...", socketPath)
	time.Sleep(1 * time.Second) // 等待 agent 启动

	client, err := grpcClient.NewClient(socketPath, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect to agent: %w", err)
	}
	defer client.Close()

	// 4. 执行命令
	logs.Infof("Executing command: %s", targetCommand)
	resp, err := client.Start(ctx, targetCommand, "", nil)
	if err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	logs.Infof("Process started with PID: %d", resp.Pid)

	// 5. 转发 stdin 到进程
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				client.SendStdin(ctx, resp.Pid, buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// 6. 等待进程退出
	time.Sleep(100 * time.Millisecond) // 等待 goroutine 启动
	_, _ = client.Stop(ctx, resp.Pid)

	return nil
}
