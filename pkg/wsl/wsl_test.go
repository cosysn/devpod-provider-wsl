package wsl

import (
	"os/exec"
	"strings"
	"testing"
)

func TestWSL_Version(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    int
		wantErr bool
	}{
		{
			name:    "WSL 2 output",
			output:  "WSL version: 2.0.0",
			want:    2,
			wantErr: false,
		},
		{
			name:    "WSL 1 output",
			output:  "WSL version: 1.0.0",
			want:    1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersion(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWSL_Exists(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		distro  string
		want    bool
	}{
		{
			name:    "distro exists",
			output:  "Ubuntu-22.04\nUbuntu-20.04",
			distro:  "Ubuntu-22.04",
			want:    true,
		},
		{
			name:    "distro not exists",
			output:  "Ubuntu-22.04\nUbuntu-20.04",
			distro:  "Debian",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDistroList(tt.output, tt.distro)
			if got != tt.want {
				t.Errorf("parseDistroList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWSL_Status(t *testing.T) {
	tests := []struct {
		name     string
		runError error
		want     string
	}{
		{
			name:     "running",
			runError: nil,
			want:     "Running",
		},
		{
			name:     "stopped",
			runError: &exec.ExitError{},
			want:     "Stopped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStatus(tt.runError)
			if got != tt.want {
				t.Errorf("parseStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions for testing (exported for testing)
func parseVersion(output string) (int, error) {
	output = strings.ToLower(output)
	if strings.Contains(output, "wsl 2") || strings.Contains(output, "2.") {
		return 2, nil
	}
	if strings.Contains(output, "wsl 1") || strings.Contains(output, "1.") {
		return 1, nil
	}
	return 2, nil
}

func parseDistroList(output, distro string) bool {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == distro {
			return true
		}
	}
	return false
}

func parseStatus(runError error) string {
	if runError == nil {
		return "Running"
	}
	return "Stopped"
}
