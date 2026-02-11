package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cosysn/devpod-provider-wsl/pkg/agent"
	"github.com/cosysn/devpod-provider-wsl/pkg/tunnel"
	grpcClient "github.com/cosysn/devpod-provider-wsl/pkg/grpc"
	pb "github.com/cosysn/devpod-provider-wsl/pkg/grpc/proto"
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
	// 如果命令已经是 shell 格式，直接传递；否则使用 bash -c 包装
	wslArgs := []string{"-d", distro}
	if strings.HasPrefix(targetCommand, "bash -c") || strings.HasPrefix(targetCommand, "sh -c") {
		// 命令已包含 shell 包装，去掉内层的 bash -c 执行实际命令
		// 提取内层命令: "bash -c 'echo hi'" -> "echo hi"
		parts := strings.SplitN(targetCommand, "'", 3)
		if len(parts) >= 2 {
			// 执行内层命令
			innerCmd := parts[1]
			wslArgs = append(wslArgs, "--", "bash", "-c", innerCmd)
		} else {
			wslArgs = append(wslArgs, "--", "bash", "-c", targetCommand)
		}
	} else {
		wslArgs = append(wslArgs, "--", "bash", "-c", targetCommand)
	}

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

	// 转发 stdin 到 WSL（过滤 Windows 换行符 CR）
	go func() {
		buf := make([]byte, 4096)
		for {
			n, readErr := os.Stdin.Read(buf)
			if n > 0 {
				// 过滤掉 \r 字符
				filtered := filterCRBytes(buf[:n])
				if len(filtered) > 0 {
					stdin.Write(filtered)
				}
			}
			if readErr != nil {
				break
			}
		}
		stdin.Close()
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

	// 4. 使用 Exec RPC 进行交互式命令执行
	logs.Infof("Executing: %s", targetCommand)
	execClient, err := client.Exec(ctx)
	if err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}

	// 发送命令
	if err := execClient.Send(&pb.ExecRequest{
		Data: &pb.ExecRequest_Input{Input: targetCommand + "\n"},
	}); err != nil {
		return fmt.Errorf("send command failed: %w", err)
	}

	// 5. 并行处理：转发 stdin 和接收 stdout
	var wg sync.WaitGroup
	stdinDone := make(chan struct{})

	// Stdin 转发 goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				execClient.Send(&pb.ExecRequest{
					Data: &pb.ExecRequest_Input{Input: string(buf[:n])},
				})
			}
			if err == io.EOF {
				// 只有当 recv 循环结束时才发送 EOF
				select {
				case <-stdinDone:
					// 接收已经结束，发送 EOF 关闭 stdin
					execClient.Send(&pb.ExecRequest{
						Data: &pb.ExecRequest_Eof{},
					})
				default:
					// 等待接收结束
				}
				break
			}
			if err != nil {
				break
			}
		}
	}()

	// Stdout 接收循环
	stdinClosed := false
	for {
		resp, err := execClient.Recv()
		if err == io.EOF {
			// 标记接收已结束
			close(stdinDone)
			stdinClosed = true
			break
		}
		if err != nil {
			return fmt.Errorf("recv failed: %w", err)
		}

		if len(resp.Stdout) > 0 {
			os.Stdout.Write(resp.Stdout)
		}
		if len(resp.Stderr) > 0 {
			os.Stderr.Write(resp.Stderr)
		}
		if resp.Done {
			// 命令执行完成
			close(stdinDone)
			stdinClosed = true
			break
		}
	}

	// 如果 stdin 还没关闭，等待它
	if !stdinClosed {
		close(stdinDone)
	}

	// 等待 stdin goroutine 结束
	wg.Wait()

	return nil
}

// filterCRBytes 过滤字节数组中的 \r 字符
func filterCRBytes(input []byte) []byte {
	result := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
		if input[i] != '\r' {
			result = append(result, input[i])
		}
	}
	return result
}
