package cmd

import (
	"testing"
)

func TestStartCmd_Use(t *testing.T) {
	if StartCmd.Use != "start" {
		t.Errorf("StartCmd.Use = %q, want %q", StartCmd.Use, "start")
	}
}

func TestStartCmd_Short(t *testing.T) {
	want := "Start WSL distribution"
	if StartCmd.Short != want {
		t.Errorf("StartCmd.Short = %q, want %q", StartCmd.Short, want)
	}
}

func TestStopCmd_Use(t *testing.T) {
	if StopCmd.Use != "stop" {
		t.Errorf("StopCmd.Use = %q, want %q", StopCmd.Use, "stop")
	}
}

func TestStopCmd_Short(t *testing.T) {
	want := "Stop WSL distribution"
	if StopCmd.Short != want {
		t.Errorf("StopCmd.Short = %q, want %q", StopCmd.Short, want)
	}
}

func TestStatusCmd_Use(t *testing.T) {
	if StatusCmd.Use != "status" {
		t.Errorf("StatusCmd.Use = %q, want %q", StatusCmd.Use, "status")
	}
}

func TestStatusCmd_Short(t *testing.T) {
	want := "Check WSL status"
	if StatusCmd.Short != want {
		t.Errorf("StatusCmd.Short = %q, want %q", StatusCmd.Short, want)
	}
}
