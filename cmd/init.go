package cmd

import (
	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Check WSL environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("init command - to be implemented")
		return nil
	},
}
