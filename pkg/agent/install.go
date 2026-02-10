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
