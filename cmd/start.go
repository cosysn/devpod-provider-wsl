package cmd

import (
	"github.com/spf13/cobra"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start WSL distribution",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("start - to be implemented")
		return nil
	},
}
