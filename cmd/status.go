package cmd

import (
	"fmt"

	"context"

	"github.com/cosysn/devpod-provider-wsl/pkg/wsl"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
)

// StatusCmd holds the cmd flags
type StatusCmd struct{}

// NewStatusCmd defines a status command
func NewStatusCmd() *cobra.Command {
	cmd := &StatusCmd{}
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check WSL status",
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

	return statusCmd
}

// Run runs the command logic
func (cmd *StatusCmd) Run(
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

	status := w.Status()
	logs.Infof(status)
	return nil
}
