// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cosysn/devpod-provider-wsl/pkg/grpc"
)

func main() {
	socketPath := "/tmp/test.sock"

	// Connect to agent
	client, err := grpc.NewClient(socketPath, 10*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test Status RPC
	fmt.Println("=== Testing Status RPC ===")
	status, err := client.Status(context.Background())
	if err != nil {
		log.Fatalf("Status failed: %v", err)
	}
	fmt.Printf("Agent running: %v, PID: %d\n", status.Running, status.Pid)

	// Test Start RPC
	fmt.Println("\n=== Testing Start RPC ===")
	startResp, err := client.Start(context.Background(), "echo hello_integration_test", "", nil)
	if err != nil {
		log.Fatalf("Start failed: %v", err)
	}
	fmt.Printf("Started process with PID: %d\n", startResp.Pid)

	// Test Stop RPC
	fmt.Println("\n=== Testing Stop RPC ===")
	stopResp, err := client.Stop(context.Background(), startResp.Pid)
	if err != nil {
		log.Fatalf("Stop failed: %v", err)
	}
	fmt.Printf("Stopped process with exit code: %d\n", stopResp.ExitCode)

	fmt.Println("\n=== Full integration test passed ===")
}
