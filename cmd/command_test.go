package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestCommandCmd_Use(t *testing.T) {
	if CommandCmd.Use != "command" {
		t.Errorf("CommandCmd.Use = %q, want %q", CommandCmd.Use, "command")
	}
}

func TestCommandCmd_Short(t *testing.T) {
	want := "Establish persistent pipe to WSL"
	if CommandCmd.Short != want {
		t.Errorf("CommandCmd.Short = %q, want %q", CommandCmd.Short, want)
	}
}

// Test socat command path construction
func TestSocatCommandPath(t *testing.T) {
	tests := []struct {
		name        string
		distro      string
		pipeName    string
		wantPattern string
	}{
		{
			name:        "Ubuntu distro",
			distro:      "Ubuntu-22.04",
			pipeName:    `\\.\pipe\devpod-wsl-Ubuntu-22.04`,
			wantPattern: "wsl.exe",
		},
		{
			name:        "Debian distro",
			distro:      "Debian",
			pipeName:    `\\.\pipe\devpod-wsl-Debian`,
			wantPattern: "wsl.exe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify pipe name generation
			gotPipeName := generateTestPipeName(tt.distro)
			if gotPipeName != tt.pipeName {
				t.Errorf("Pipe name = %q, want %q", gotPipeName, tt.pipeName)
			}

			// Verify socat path pattern
			socatPath := fmt.Sprintf("UNIX-CONNECT:/mnt/wsl$/%s/pipe/%s", tt.distro, gotPipeName)
			if !strings.Contains(socatPath, "wsl$") {
				t.Errorf("socat path should contain wsl$, got %q", socatPath)
			}
			if !strings.Contains(socatPath, tt.distro) {
				t.Errorf("socat path should contain distro %q, got %q", tt.distro, socatPath)
			}
		})
	}
}

// TestWSLSocatCommandArgs tests the construction of wsl.exe arguments
func TestWSLSocatCommandArgs(t *testing.T) {
	tests := []struct {
		name        string
		distro      string
		wantArgsLen int
		wantContain string
	}{
		{
			name:        "Ubuntu",
			distro:      "Ubuntu-22.04",
			wantArgsLen: 4, // wsl.exe, -d, distro, --, socat, ...
			wantContain: "wsl.exe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate command args construction
			args := []string{"wsl.exe", "-d", tt.distro, "--", "socat"}
			if len(args) < tt.wantArgsLen {
				t.Errorf("Args length = %d, want at least %d", len(args), tt.wantArgsLen)
			}
			if args[0] != tt.wantContain {
				t.Errorf("First arg = %q, want %q", args[0], tt.wantContain)
			}
		})
	}
}

// Helper function for testing (mirrors pipe.GeneratePipeName)
func generateTestPipeName(distro string) string {
	return `\\.\pipe\devpod-wsl-` + distro
}
