package pipe

import (
	"net"
	"os"

	"golang.org/x/sys/windows"
)

// CreateNamedPipe creates a Windows named pipe server
func CreateNamedPipe(name string) (net.Listener, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}

	// Pipe mode flags
	openMode := uint32(windows.PIPE_ACCESS_DUPLEX) // read/write access
	pipeMode := uint32(windows.PIPE_TYPE_BYTE | windows.PIPE_READMODE_BYTE | windows.PIPE_WAIT)

	handle, err := windows.CreateNamedPipe(
		namePtr,
		openMode,
		pipeMode,
		uint32(windows.PIPE_UNLIMITED_INSTANCES),
		uint32(8192),
		uint32(8192),
		uint32(0),
		nil,
	)

	if err != nil {
		return nil, err
	}

	// Create os.File from handle
	f := os.NewFile(uintptr(handle), name)
	return net.FileListener(f)
}

// GeneratePipeName generates a unique pipe name for a distro
func GeneratePipeName(distro string) string {
	return `\\.\pipe\devpod-wsl-` + distro
}
