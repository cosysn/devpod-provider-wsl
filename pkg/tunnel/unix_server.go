package tunnel

import (
	"net"
	"os"
	"path/filepath"
)

const DefaultSocketPath = "/var/tmp/devpod.sock"

type UnixServer struct {
	socketPath string
	listener   net.Listener
}

func NewUnixServer(socketPath string) *UnixServer {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	return &UnixServer{socketPath: socketPath}
}

func (s *UnixServer) Listen() error {
	// 确保目录存在
	dir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 删除已存在的 socket 文件
	os.Remove(s.socketPath)

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	s.listener = listener
	return nil
}

func (s *UnixServer) Accept() (net.Conn, error) {
	return s.listener.Accept()
}

func (s *UnixServer) Close() error {
	return s.listener.Close()
}

func (s *UnixServer) Addr() net.Addr {
	return &net.UnixAddr{Name: s.socketPath, Net: "unix"}
}
