package cmd

import (
	"github.com/spf13/cobra"
)

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop WSL distribution",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("stop - to be implemented")
		return nil
	},
}
