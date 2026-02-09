package cmd

import (
	"context"

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

// Run runs the command logic
func (cmd *CommandCmd) Run(
	ctx context.Context,
	providerWsl *wsl.WslProvider,
	machine *provider.Machine,
	logs log.Logger,
) error {
	logs.Infof("Commanding workspace in WSL distribution '%s'...", providerWsl.Config.WSLDistro)
	return nil
}
