package cmd

import (
	"github.com/spf13/cobra"
)

var CommandCmd = &cobra.Command{
	Use:   "command",
	Short: "Establish persistent pipe to WSL",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("command - to be implemented")
		return nil
	},
}
