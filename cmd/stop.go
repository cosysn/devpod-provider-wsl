package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/cosysn/devpod-provider-wsl/pkg/wsl"
)

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop WSL distribution",
	Long: `Stop the specified WSL distribution.
This command terminates the WSL virtual machine.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		distro := os.Getenv("WSL_DISTRO")
		if distro == "" {
			return fmt.Errorf("WSL_DISTRO environment variable is required")
		}

		w := wsl.WSL{Distro: distro}

		// Check if running
		status := w.Status()
		if status != "Running" {
			cmd.Printf("Distribution '%s' is not running\n", distro)
			return nil
		}

		// Stop the distribution
		cmd.Printf("Stopping distribution '%s'...\n", distro)
		if err := w.Stop(); err != nil {
			return fmt.Errorf("stop failed: %w", err)
		}

		cmd.Printf("Distribution '%s' stopped successfully\n", distro)
		return nil
	},
}
