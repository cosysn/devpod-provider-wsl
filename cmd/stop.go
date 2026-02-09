package cmd

import (
	"fmt"

	"context"

	"github.com/cosysn/devpod-provider-wsl/pkg/wsl"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
)

// StopCmd holds the cmd flags
type StopCmd struct{}

// NewStopCmd defines a stop command
func NewStopCmd() *cobra.Command {
	cmd := &StopCmd{}
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop WSL distribution",
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

	return stopCmd
}

// Run runs the command logic
func (cmd *StopCmd) Run(
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

	// Check if running
	status := w.Status()
	if status != "Running" {
		fmt.Printf("Distribution '%s' is not running\n", distro)
		return nil
	}

	// Stop the distribution
	fmt.Printf("Stopping distribution '%s'...\n", distro)
	if err := w.Stop(); err != nil {
		return fmt.Errorf("stop failed: %w", err)
	}

	fmt.Printf("Distribution '%s' stopped successfully\n", distro)
	return nil
}
