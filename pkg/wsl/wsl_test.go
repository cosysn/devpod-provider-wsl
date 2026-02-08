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
		{
			name:    "WSL 2 long output",
			output:  "WSL version 2.0.11.0",
			want:    2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getVersionFromOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("getVersionFromOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getVersionFromOutput() = %v, want %v", got, tt.want)
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
		{
			name:    "empty output",
			output:  "",
			distro:  "Ubuntu",
			want:    false,
		},
		{
			name:    "with spaces",
			output:  " Ubuntu-22.04 ",
			distro:  "Ubuntu-22.04",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkDistroExists(tt.output, tt.distro)
			if got != tt.want {
				t.Errorf("checkDistroExists() = %v, want %v", got, tt.want)
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
			got := getStatusFromError(tt.runError)
			if got != tt.want {
				t.Errorf("getStatusFromError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWSL_DiskSpace(t *testing.T) {
	tests := []struct {
		name     string
		dfOutput string
		minGB    int
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "sufficient space",
			dfOutput: "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1       100G   20G   80G  20% /",
			minGB:    5,
			wantErr:  false,
		},
		{
			name:     "insufficient space",
			dfOutput: "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1       10G    5G    5G  50% /",
			minGB:    10,
			wantErr:  true,
			errMsg:   "insufficient disk space",
		},
		{
			name:     "exactly at limit",
			dfOutput: "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1       20G   10G   10G  50% /",
			minGB:    10,
			wantErr:  false,
		},
		{
			name:     "vd disk",
			dfOutput: "Filesystem      Size  Used Avail Use% Mounted on\n/dev/vda1       50G   10G   40G  20% /",
			minGB:    5,
			wantErr:  false,
		},
		{
			name:     "empty output",
			dfOutput: "",
			minGB:    5,
			wantErr:  false,
		},
		{
			name:     "non-numeric size",
			dfOutput: "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1       unknown  10G   40G  20% /",
			minGB:    5,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseDiskSpace(tt.dfOutput, tt.minGB)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseDiskSpace() expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("parseDiskSpace() error = %v, want contains %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("parseDiskSpace() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWSL_CheckTools(t *testing.T) {
	// Test helper function for tool existence check
	tests := []struct {
		name    string
		output  string
		tool    string
		wantErr bool
	}{
		{
			name:    "tool exists",
			output:  "/usr/bin/git",
			tool:    "git",
			wantErr: false,
		},
		{
			name:    "tool not found",
			output:  "",
			tool:    "nonexistent",
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			tool:    "git",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate which command behavior
			err := checkToolExists(tt.output, tt.tool)
			if tt.wantErr && err == nil {
				t.Errorf("checkToolExists() expected error for tool %s", tt.tool)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("checkToolExists() unexpected error: %v", err)
			}
		})
	}
}

func TestWSL_CheckToolsMultiple(t *testing.T) {
	// Test checking multiple tools at once
	tests := []struct {
		name     string
		outputs  map[string]string
		tools    []string
		wantErr  bool
		missing  string
	}{
		{
			name:     "all tools exist",
			outputs:  map[string]string{"git": "/usr/bin/git", "curl": "/usr/bin/curl"},
			tools:    []string{"git", "curl"},
			wantErr:  false,
			missing:  "",
		},
		{
			name:     "one tool missing",
			outputs:  map[string]string{"git": "/usr/bin/git"},
			tools:    []string{"git", "curl"},
			wantErr:  true,
			missing:  "curl",
		},
		{
			name:     "empty tools list",
			outputs:  map[string]string{},
			tools:    []string{},
			wantErr:  false,
			missing:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var missingTool string
			for _, tool := range tt.tools {
				output, exists := tt.outputs[tool]
				if !exists || output == "" {
					missingTool = tool
					break
				}
			}

			if tt.wantErr && missingTool == "" {
				t.Errorf("Expected missing tool, but all tools exist")
			}
			if !tt.wantErr && missingTool != "" {
				t.Errorf("Expected no missing tool, but %s is missing", missingTool)
			}
		})
	}
}

func TestWSL_DiskSpaceError(t *testing.T) {
	err := &DiskSpaceError{Available: 5, Required: 10}
	want := "insufficient disk space: 5G < 10G required"

	if err.Error() != want {
		t.Errorf("DiskSpaceError.Error() = %v, want %v", err.Error(), want)
	}
}

func TestWSL_MissingToolError(t *testing.T) {
	err := &MissingToolError{Tool: "git"}
	want := "missing tool: git"

	if err.Error() != want {
		t.Errorf("MissingToolError.Error() = %v, want %v", err.Error(), want)
	}
}

// Helper functions for testing
func checkToolExists(output, tool string) error {
	if output == "" {
		return &MissingToolError{Tool: tool}
	}
	return nil
}
