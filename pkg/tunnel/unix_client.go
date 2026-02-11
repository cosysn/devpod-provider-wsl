package tunnel

import "net"

type UnixClient struct {
	socketPath string
}

func NewUnixClient(socketPath string) *UnixClient {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	return &UnixClient{socketPath: socketPath}
}

func (c *UnixClient) Dial() (net.Conn, error) {
	return net.Dial("unix", c.socketPath)
}

func (c *UnixClient) Addr() net.Addr {
	return &net.UnixAddr{Name: c.socketPath, Net: "unix"}
}
