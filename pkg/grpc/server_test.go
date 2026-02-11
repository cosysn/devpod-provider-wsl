package grpc

import (
	"context"
	"testing"

	pb "github.com/cosysn/devpod-provider-wsl/pkg/grpc/proto"
)

func TestServer_Start(t *testing.T) {
	server := NewServer()

	resp, err := server.Start(context.Background(), &pb.StartRequest{
		Command: "echo hello",
		Workdir: "",
		Env:     map[string]string{},
	})

	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if resp.Pid <= 0 {
		t.Fatalf("Invalid PID: %d", resp.Pid)
	}

	t.Logf("Started process with PID: %d", resp.Pid)
}

func TestServer_StartWithEnv(t *testing.T) {
	server := NewServer()

	resp, err := server.Start(context.Background(), &pb.StartRequest{
		Command: "echo $TEST_VAR",
		Workdir: "",
		Env:     map[string]string{"TEST_VAR": "hello_world"},
	})

	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if resp.Pid <= 0 {
		t.Fatalf("Invalid PID: %d", resp.Pid)
	}

	t.Logf("Started process with PID: %d", resp.Pid)
}
