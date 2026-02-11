package tunnel

import (
	"net"

	"github.com/hashicorp/yamux"
)

// TunnelListener wraps a yamux.Session for gRPC
type TunnelListener struct {
	session *yamux.Session
}

// NewTunnelListener creates a TunnelListener from a YamuxSession
func NewTunnelListener(session *YamuxSession) *TunnelListener {
	return &TunnelListener{session: session.session}
}

// Accept implements net.Listener
func (l *TunnelListener) Accept() (net.Conn, error) {
	return l.session.Accept()
}

// Addr implements net.Listener
func (l *TunnelListener) Addr() net.Addr {
	return &net.UnixAddr{Name: "", Net: "unix"}
}

// Close implements net.Listener
func (l *TunnelListener) Close() error {
	return l.session.Close()
}

// UnixListener wraps a Unix socket listener for gRPC
type UnixListener struct {
	listener net.Listener
}

// NewUnixListener creates a UnixListener from a net.Listener
func NewUnixListener(listener net.Listener) *UnixListener {
	return &UnixListener{listener: listener}
}

// Accept implements net.Listener
func (l *UnixListener) Accept() (net.Conn, error) {
	return l.listener.Accept()
}

// Addr implements net.Listener
func (l *UnixListener) Addr() net.Addr {
	return l.listener.Addr()
}

// Close implements net.Listener
func (l *UnixListener) Close() error {
	return l.listener.Close()
}
