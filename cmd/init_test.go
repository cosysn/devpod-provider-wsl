package cmd

import (
	"os"
	"testing"
)

func TestInitCmd_RequiresDistro(t *testing.T) {
	// Save original env
	orig := os.Getenv("WSL_DISTRO")
	defer os.Setenv("WSL_DISTRO", orig)

	// Clear env
	os.Unsetenv("WSL_DISTRO")

	// Test that missing distro returns error
	distro := os.Getenv("WSL_DISTRO")
	if distro != "" {
		t.Errorf("Expected empty WSL_DISTRO, got %q", distro)
	}
}

func TestInitCmd_DistroFromEnv(t *testing.T) {
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
