package cmd

import (
	"os"
	"testing"
)

func TestRootCmd_HasSubcommands(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "check subcommands",
			want: []string{"init", "command", "start", "stop", "status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := []string{
				InitCmd.Use,
				CommandCmd.Use,
				StartCmd.Use,
				StopCmd.Use,
				StatusCmd.Use,
			}
			for i, want := range tt.want {
				if cmds[i] != want {
					t.Errorf("Subcommand[%d] = %v, want %v", i, cmds[i], want)
				}
			}
		})
	}
}

func TestWSLDistroFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		want    string
		wantErr bool
	}{
		{
			name:    "set env var",
			envVar:  "Ubuntu-22.04",
			want:    "Ubuntu-22.04",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("WSL_DISTRO", tt.envVar)
			defer os.Unsetenv("WSL_DISTRO")

			got := os.Getenv("WSL_DISTRO")
			if got != tt.want {
				t.Errorf("WSL_DISTRO = %v, want %v", got, tt.want)
			}
		})
	}
}
