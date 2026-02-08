package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/cosysn/devpod-provider-wsl/pkg/wsl"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check WSL status",
	Long: `Check the status of the specified WSL distribution.
Returns the current state (Running or Stopped).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		distro := os.Getenv("WSL_DISTRO")
		if distro == "" {
			return fmt.Errorf("WSL_DISTRO environment variable is required")
		}

		w := wsl.WSL{Distro: distro}

		status := w.Status()
		fmt.Println(status)
		return nil
	},
}
