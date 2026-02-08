package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/cosysn/devpod-provider-wsl/pkg/wsl"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start WSL distribution",
	Long: `Start the specified WSL distribution.
This command starts the WSL virtual machine if it's not already running.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		distro := os.Getenv("WSL_DISTRO")
		if distro == "" {
			return fmt.Errorf("WSL_DISTRO environment variable is required")
		}

		w := wsl.WSL{Distro: distro}

		// Check if distribution exists
		if !w.Exists() {
			return fmt.Errorf("distribution '%s' not found", distro)
		}

		// Check if already running
		status := w.Status()
		if status == "Running" {
			cmd.Printf("Distribution '%s' is already running\n", distro)
			return nil
		}

		// Start the distribution
		cmd.Printf("Starting distribution '%s'...\n", distro)
		if err := w.Start(); err != nil {
			return fmt.Errorf("start failed: %w", err)
		}

		cmd.Printf("Distribution '%s' started successfully\n", distro)
		return nil
	},
}
