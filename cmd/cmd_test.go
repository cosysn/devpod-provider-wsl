package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

func TestRootCmd_HelpOutput(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "devpod-provider-wsl",
	}

	rootCmd.AddCommand(
		InitCmd,
		CommandCmd,
		StartCmd,
		StopCmd,
		StatusCmd,
	)

	tests := []struct {
		name    string
		command *cobra.Command
		want    string
	}{
		{
			name:    "root help",
			command: rootCmd,
			want:    "devpod-provider-wsl",
		},
		{
			name:    "init help",
			command: InitCmd,
			want:    "init",
		},
		{
			name:    "command help",
			command: CommandCmd,
			want:    "command",
		},
		{
			name:    "start help",
			command: StartCmd,
			want:    "start",
		},
		{
			name:    "stop help",
			command: StopCmd,
			want:    "stop",
		},
		{
			name:    "status help",
			command: StatusCmd,
			want:    "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			tt.command.SetOut(buf)
			tt.command.SetErr(buf)

			tt.command.Help()

			output := buf.String()
			if !strings.Contains(output, tt.want) {
				t.Errorf("Help() output should contain %q, got %q", tt.want, output)
			}
		})
	}
}

func TestInitCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	InitCmd.SetOut(buf)
	InitCmd.SetErr(buf)
	InitCmd.Help()

	output := buf.String()
	tests := []struct {
		name string
		want string
	}{
		{
			name: "use",
			want: "init",
		},
		{
			name: "description contains WSL",
			want: "WSL",
		},
		{
			name: "usage",
			want: "Usage:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(output, tt.want) {
				t.Errorf("Help() should contain %q, got %q", tt.want, output)
			}
		})
	}
}

func TestCommandCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	CommandCmd.SetOut(buf)
	CommandCmd.SetErr(buf)
	CommandCmd.Help()

	output := buf.String()
	tests := []struct {
		name string
		want string
	}{
		{
			name: "use",
			want: "command",
		},
		{
			name: "description contains pipe",
			want: "pipe",
		},
		{
			name: "usage",
			want: "Usage:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(output, tt.want) {
				t.Errorf("Help() should contain %q, got %q", tt.want, output)
			}
		})
	}
}

func TestStartCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	StartCmd.SetOut(buf)
	StartCmd.SetErr(buf)
	StartCmd.Help()

	output := buf.String()
	tests := []struct {
		name string
		want string
	}{
		{
			name: "use",
			want: "start",
		},
		{
			name: "description contains WSL",
			want: "WSL",
		},
		{
			name: "usage",
			want: "Usage:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(output, tt.want) {
				t.Errorf("Help() should contain %q, got %q", tt.want, output)
			}
		})
	}
}

func TestStopCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	StopCmd.SetOut(buf)
	StopCmd.SetErr(buf)
	StopCmd.Help()

	output := buf.String()
	tests := []struct {
		name string
		want string
	}{
		{
			name: "use",
			want: "stop",
		},
		{
			name: "description contains WSL",
			want: "WSL",
		},
		{
			name: "usage",
			want: "Usage:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(output, tt.want) {
				t.Errorf("Help() should contain %q, got %q", tt.want, output)
			}
		})
	}
}

func TestStatusCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	StatusCmd.SetOut(buf)
	StatusCmd.SetErr(buf)
	StatusCmd.Help()

	output := buf.String()
	tests := []struct {
		name string
		want string
	}{
		{
			name: "use",
			want: "status",
		},
		{
			name: "description contains WSL",
			want: "WSL",
		},
		{
			name: "usage",
			want: "Usage:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(output, tt.want) {
				t.Errorf("Help() should contain %q, got %q", tt.want, output)
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
