package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/cosysn/devpod-provider-wsl/pkg/pipe"
)

var CommandCmd = &cobra.Command{
	Use:   "command",
	Short: "Establish persistent pipe to WSL",
	Long: `Establish a persistent communication pipe between Windows and WSL.
This command is used by DevPod to communicate with the WSL environment.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		distro := os.Getenv("WSL_DISTRO")
		if distro == "" {
			return fmt.Errorf("WSL_DISTRO environment variable is required")
		}

		return runCommand(distro, cmd)
	},
}

func runCommand(distro string, cmd *cobra.Command) error {
	pipeName := pipe.GeneratePipeName(distro)

	// Create named pipe server
	cmd.Printf("Creating named pipe: %s\n", pipeName)
	listener, err := pipe.CreateNamedPipe(pipeName)
	if err != nil {
		return fmt.Errorf("create pipe: %w", err)
	}
	defer listener.Close()

	cmd.Println("Pipe server started, waiting for WSL connection...")

	// Start WSL with socat to connect to the named pipe
	wslCmd := exec.Command("wsl.exe", "-d", distro, "--", "socat",
		fmt.Sprintf("UNIX-CONNECT:/mnt/wsl$/%s/pipe/%s", distro, pipeName),
		"STDIO")
	wslCmd.Stdin = os.Stdin
	wslCmd.Stdout = os.Stdout
	wslCmd.Stderr = os.Stderr

	if err := wslCmd.Start(); err != nil {
		return fmt.Errorf("start wsl socat: %w", err)
	}

	// Wait for client connection with timeout
	conn, err := listener.Accept()
	if err != nil {
		wslCmd.Wait()
		return fmt.Errorf("accept pipe connection: %w", err)
	}

	cmd.Println("WSL connected, starting relay...")

	// Set connection timeout
	deadline := time.Now().Add(30 * time.Second)
	conn.SetDeadline(deadline)

	// Relay data between stdin/stdout and the pipe
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(conn, os.Stdin)
		if err != nil {
			cmd.Printf("stdin copy error: %v\n", err)
		}
	}()

	go func() {
		defer wg.Done()
		_, err := io.Copy(os.Stdout, conn)
		if err != nil {
			cmd.Printf("stdout copy error: %v\n", err)
		}
	}()

	wg.Wait()

	// Clean up
	conn.Close()
	wslCmd.Wait()

	return nil
}
