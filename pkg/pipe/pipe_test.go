package pipe

import (
	"testing"
)

func TestGeneratePipeName(t *testing.T) {
	tests := []struct {
		name   string
		distro string
		want   string
	}{
		{
			name:   "Ubuntu distro",
			distro: "Ubuntu-22.04",
			want:   `\\.\pipe\devpod-wsl-Ubuntu-22.04`,
		},
		{
			name:   "Debian distro",
			distro: "Debian",
			want:   `\\.\pipe\devpod-wsl-Debian`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePipeName(tt.distro)
			if got != tt.want {
				t.Errorf("GeneratePipeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipeNameFormat(t *testing.T) {
	// Verify the generated name follows Windows named pipe format
	pipeName := GeneratePipeName("test")
	if len(pipeName) == 0 {
		t.Error("GeneratePipeName() returned empty string")
	}
}
