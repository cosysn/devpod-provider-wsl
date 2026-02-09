package cmd

import (
	"context"
	"fmt"

	"github.com/cosysn/devpod-provider-wsl/pkg/wsl"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
)

// StartCmd holds the cmd flags
type StartCmd struct{}

// NewStartCmd defines a start command
func NewStartCmd() *cobra.Command {
	cmd := &StartCmd{}
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start WSL distribution",
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

	return startCmd
}

// Run runs the start logic
func (cmd *StartCmd) Run(
	ctx context.Context,
	providerWsl *wsl.WslProvider,
	machine *provider.Machine,
	logs log.Logger,
) error {
	distro := providerWsl.Config.WSLDistro
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
		fmt.Printf("Distribution '%s' is already running\n", distro)
		return nil
	}

	// Start the distribution
	fmt.Printf("Starting distribution '%s'...\n", distro)
	if err := w.Start(); err != nil {
		return fmt.Errorf("start failed: %w", err)
	}

	fmt.Printf("Distribution '%s' started successfully\n", distro)
	return nil
}
