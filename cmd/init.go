package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/cosysn/devpod-provider-wsl/pkg/wsl"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Check WSL environment",
	Long: `Check if WSL environment is properly configured.
Verifies WSL version, distribution, disk space, and required tools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		distro := os.Getenv("WSL_DISTRO")
		if distro == "" {
			return fmt.Errorf("WSL_DISTRO environment variable is required")
		}

		w := wsl.WSL{Distro: distro}

		// 1. Check WSL version
		cmd.Println("Checking WSL version...")
		version, err := w.Version()
		if err != nil {
			return fmt.Errorf("wsl not available: %w", err)
		}
		if version < 2 {
			return fmt.Errorf("wsl 2 is required, got version %d", version)
		}
		cmd.Printf("  WSL version: %d\n", version)

		// 2. Check distribution exists
		cmd.Println("Checking distribution...")
		if !w.Exists() {
			return fmt.Errorf("distribution '%s' not found", distro)
		}
		cmd.Printf("  Distribution '%s' found\n", distro)

		// 3. Check disk space
		cmd.Println("Checking disk space...")
		if err := w.CheckDiskSpace(5); err != nil {
			return fmt.Errorf("disk space check failed: %w", err)
		}
		cmd.Println("  Disk space: OK (>= 5GB)")

		// 4. Check required tools
		cmd.Println("Checking required tools...")
		requiredTools := []string{"git", "curl"}
		if err := w.CheckTools(requiredTools); err != nil {
			return fmt.Errorf("tool check failed: %w", err)
		}
		cmd.Printf("  Tools: OK (%v)\n", requiredTools)

		cmd.Println("\nWSL environment check passed!")
		return nil
	},
}
