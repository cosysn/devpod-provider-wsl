package cmd

import (
	"github.com/spf13/cobra"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check WSL status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("status - to be implemented")
		return nil
	},
}
