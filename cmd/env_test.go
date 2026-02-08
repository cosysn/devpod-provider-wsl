package cmd

import (
	"os"
	"testing"
)

func TestWSLDistroFromEnv_Empty(t *testing.T) {
	// Save original env
	orig := os.Getenv("WSL_DISTRO")
	defer os.Setenv("WSL_DISTRO", orig)

	// Clear env
	os.Unsetenv("WSL_DISTRO")

	// Test that missing distro returns empty
	distro := os.Getenv("WSL_DISTRO")
	if distro != "" {
		t.Errorf("Expected empty WSL_DISTRO, got %q", distro)
	}
}

func TestWSLDistroFromEnv_Set(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		want    string
		wantErr bool
	}{
		{
			name:    "valid distro",
			envVal:  "Ubuntu-22.04",
			want:    "Ubuntu-22.04",
			wantErr: false,
		},
		{
			name:    "empty distro",
			envVal:  "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("WSL_DISTRO", tt.envVal)
			defer os.Unsetenv("WSL_DISTRO")

			got := os.Getenv("WSL_DISTRO")
			if (got == "") != tt.wantErr {
				t.Errorf("WSL_DISTRO error state = %v, wantErr %v", got == "", tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("WSL_DISTRO = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWSLDistroFromEnv_SpecialChars(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		want   string
	}{
		{
			name:   "distro with version",
			envVal: "Ubuntu-22.04",
			want:   "Ubuntu-22.04",
		},
		{
			name:   "debian",
			envVal: "Debian",
			want:   "Debian",
		},
		{
			name:   "alpine",
			envVal: "Alpine-WSL",
			want:   "Alpine-WSL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("WSL_DISTRO", tt.envVal)
			defer os.Unsetenv("WSL_DISTRO")

			got := os.Getenv("WSL_DISTRO")
			if got != tt.want {
				t.Errorf("WSL_DISTRO = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIDLETimeoutFromEnv(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		want   string
	}{
		{
			name:   "default timeout",
			envVal: "30",
			want:   "30",
		},
		{
			name:   "custom timeout",
			envVal: "60",
			want:   "60",
		},
		{
			name:   "zero timeout",
			envVal: "0",
			want:   "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("IDLE_TIMEOUT", tt.envVal)
			defer os.Unsetenv("IDLE_TIMEOUT")

			got := os.Getenv("IDLE_TIMEOUT")
			if got != tt.want {
				t.Errorf("IDLE_TIMEOUT = %v, want %v", got, tt.want)
			}
		})
	}
}
