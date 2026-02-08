package wsl

import (
	"os/exec"
	"strconv"
	"strings"
)

type WSL struct {
	Distro string
}

// Version returns WSL version (1 or 2)
func (w *WSL) Version() (int, error) {
	cmd := exec.Command("wsl.exe", "--version")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	return getVersionFromOutput(string(output))
}

func getVersionFromOutput(output string) (int, error) {
	output = strings.ToLower(output)
	if strings.Contains(output, "wsl 2") || strings.Contains(output, "2.") {
		return 2, nil
	}
	if strings.Contains(output, "wsl 1") || strings.Contains(output, "1.") {
		return 1, nil
	}
	// Default to WSL 2 for modern systems
	return 2, nil
}

// Exists checks if the distribution exists
func (w *WSL) Exists() bool {
	cmd := exec.Command("wsl.exe", "-l", "-q")
	output, _ := cmd.Output()
	return checkDistroExists(string(output), w.Distro)
}

func checkDistroExists(output, distro string) bool {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == distro {
			return true
		}
	}
	return false
}

// Start starts the WSL distribution
func (w *WSL) Start() error {
	cmd := exec.Command("wsl.exe", "-d", w.Distro)
	return cmd.Start()
}

// Stop terminates the WSL distribution
func (w *WSL) Stop() error {
	cmd := exec.Command("wsl.exe", "--terminate", w.Distro)
	return cmd.Run()
}

// Status returns the status of the distribution
func (w *WSL) Status() string {
	cmd := exec.Command("wsl.exe", "-d", w.Distro, "-e", "echo", "running")
	err := cmd.Run()
	return getStatusFromError(err)
}

func getStatusFromError(runError error) string {
	if runError == nil {
		return "Running"
	}
	return "Stopped"
}

// CheckDiskSpace checks if there's at least minGB free space
func (w *WSL) CheckDiskSpace(minGB int) error {
	cmd := exec.Command("wsl.exe", "-d", w.Distro, "-e", "df", "-BG", "/")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	return parseDiskSpace(string(output), minGB)
}

func parseDiskSpace(dfOutput string, minGB int) error {
	lines := strings.Split(dfOutput, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "/dev/sd") || strings.HasPrefix(line, "/dev/vd") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				available := strings.TrimSuffix(parts[3], "G")
				size, err := strconv.Atoi(available)
				if err != nil {
					continue
				}
				if size < minGB {
					return &DiskSpaceError{Available: size, Required: minGB}
				}
				return nil
			}
		}
	}
	return nil
}

type DiskSpaceError struct {
	Available int
	Required  int
}

func (e *DiskSpaceError) Error() string {
	return "insufficient disk space: " + strconv.Itoa(e.Available) + "G < " + strconv.Itoa(e.Required) + "G required"
}

// CheckTools checks if required tools are installed
func (w *WSL) CheckTools(tools []string) error {
	for _, tool := range tools {
		cmd := exec.Command("wsl.exe", "-d", w.Distro, "-e", "which", tool)
		if err := cmd.Run(); err != nil {
			return &MissingToolError{Tool: tool}
		}
	}
	return nil
}

type MissingToolError struct {
	Tool string
}

func (e *MissingToolError) Error() string {
	return "missing tool: " + e.Tool
}
