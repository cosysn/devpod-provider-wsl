package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var CommandCmd = &cobra.Command{
	Use:   "command",
	Short: "Establish persistent pipe to WSL",
	Long: `Establish a persistent communication pipe between Windows and WSL.
This command is used by DevPod to communicate with the WSL environment.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		distro := os.Getenv("WSL_DISTRO")
		if distro == "" {
			return fmt.Errorf("WSL_DISTRO environment variable is required")
		}

		return runCommand(distro, cmd)
	},
}

func runCommand(distro string, cobraCmd *cobra.Command) error {
	// Run a shell in WSL that connects to devpod agent via stdio
	// This allows DevPod to communicate with the WSL environment
	wslCmd := exec.Command("wsl.exe", "-d", distro)

	// Connect stdin/stdout/stderr for bidirectional communication
	wslCmd.Stdin = os.Stdin
	wslCmd.Stdout = os.Stdout
	wslCmd.Stderr = os.Stderr

	if err := wslCmd.Start(); err != nil {
		return fmt.Errorf("start wsl: %w", err)
	}

	return wslCmd.Wait()
}
