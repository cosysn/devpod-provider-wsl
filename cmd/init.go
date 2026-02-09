package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/cosysn/devpod-provider-wsl/pkg/options"
	"github.com/cosysn/devpod-provider-wsl/pkg/wsl"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
)

// InitCmd holds the cmd flags
type InitCmd struct{}

// NewInitCmd defines a init
func NewInitCmd() *cobra.Command {
	cmd := &InitCmd{}
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Init account",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(
				context.Background(),
				provider.FromEnvironment(),
				log.Default,
			)
		},
	}

	return initCmd
}

// Run runs the init logic
func (cmd *InitCmd) Run(
	ctx context.Context,
	machine *provider.Machine,
	logs log.Logger,
) error {
	config, err := options.FromEnv(true, true)
	if err != nil {
		return err
	}

	distro := config.WSLDistro
	if distro == "" {
		return fmt.Errorf("WSL_DISTRO environment variable is required")
	}

	w := wsl.WSL{Distro: distro}

	// 1. Check WSL version
	fmt.Fprintln(os.Stdout, "Checking WSL version...")
	version, err := w.Version()
	if err != nil {
		return fmt.Errorf("wsl not available: %w", err)
	}
	if version < 2 {
		return fmt.Errorf("wsl 2 is required, got version %d", version)
	}
	fmt.Fprintf(os.Stdout, "  WSL version: %d\n", version)

	// 2. Check distribution exists
	fmt.Fprintln(os.Stdout, "Checking distribution...")
	if !w.Exists() {
		return fmt.Errorf("distribution '%s' not found", distro)
	}
	fmt.Fprintf(os.Stdout, "  Distribution '%s' found\n", distro)

	// 3. Check disk space
	fmt.Fprintln(os.Stdout, "Checking disk space...")
	if err := w.CheckDiskSpace(5); err != nil {
		return fmt.Errorf("disk space check failed: %w", err)
	}
	fmt.Fprintln(os.Stdout, "  Disk space: OK (>= 5GB)")

	// 4. Check required tools
	fmt.Fprintln(os.Stdout, "Checking required tools...")
	requiredTools := []string{"git", "curl"}
	if err := w.CheckTools(requiredTools); err != nil {
		return fmt.Errorf("tool check failed: %w", err)
	}
	fmt.Fprintf(os.Stdout, "  Tools: OK (%v)\n", requiredTools)

	fmt.Fprintln(os.Stdout, "\nWSL environment check passed!")
	return nil
}
