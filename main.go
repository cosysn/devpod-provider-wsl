package main

import (
	"os"

	"github.com/cosysn/devpod-provider-wsl/cmd"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "devpod-provider-wsl",
		Short:   "DevPod provider for WSL environments",
		Version: version,
	}

	rootCmd.AddCommand(
		cmd.InitCmd,
		cmd.CommandCmd,
		cmd.StartCmd,
		cmd.StopCmd,
		cmd.StatusCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
