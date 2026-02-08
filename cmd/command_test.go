package cmd

import (
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
