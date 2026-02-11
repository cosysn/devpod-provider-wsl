package tunnel

import (
	"net"
	"os"
	"testing"
	"time"
)

func TestUnixServer_ListenAndAccept(t *testing.T) {
	// Clean up any existing socket
	socketPath := "/tmp/test-devpod.sock"
	os.Remove(socketPath)
	defer os.Remove(socketPath)

	// Create server
	server := NewUnixServer(socketPath)
	if err := server.Listen(); err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer server.Close()

	// Connect client
	client, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer client.Close()

	// Server should accept
	go func() {
		server.Accept()
	}()

	// Give server time to accept
	time.Sleep(100 * time.Millisecond)

	t.Log("Unix socket test passed")
}

func TestUnixClient_Dial(t *testing.T) {
	socketPath := "/tmp/test-devpod-client.sock"
	os.Remove(socketPath)
	defer os.Remove(socketPath)

	// Create server first
	server := NewUnixServer(socketPath)
	if err := server.Listen(); err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer server.Close()

	// Accept connection in background
	go func() {
		server.Accept()
	}()

	time.Sleep(50 * time.Millisecond)

	// Create client and dial
	client := NewUnixClient(socketPath)
	conn, err := client.Dial()
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	t.Log("Unix client dial test passed")
}
