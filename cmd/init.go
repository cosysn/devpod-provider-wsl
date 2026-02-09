package cmd

import (
	"fmt"
	"os"
	"os/exec"

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
	},
}

// RunInitCommand is used by DevPod to run init via exec.Command
func RunInitCommand(distro string) error {
	cmd := exec.Command(os.Args[0], "init")
	cmd.Env = append(os.Environ(), "WSL_DISTRO="+distro)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
