package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create workspace in WSL",
	Long: `Create a new workspace in the WSL distribution.
This command is called by DevPod when creating a new workspace.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		distro := os.Getenv("WSL_DISTRO")
		if distro == "" {
			return fmt.Errorf("WSL_DISTRO environment variable is required")
		}

		cmd.Printf("Creating workspace in WSL distribution '%s'...\n", distro)
		cmd.Println("Workspace creation delegated to DevPod agent.")
		return nil
	},
}

var DeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete workspace in WSL",
	Long: `Delete a workspace in the WSL distribution.
This command is called by DevPod when deleting a workspace.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		distro := os.Getenv("WSL_DISTRO")
		if distro == "" {
			return fmt.Errorf("WSL_DISTRO environment variable is required")
		}

		cmd.Printf("Deleting workspace in WSL distribution '%s'...\n", distro)
		cmd.Println("Workspace deletion delegated to DevPod agent.")
		return nil
	},
}
