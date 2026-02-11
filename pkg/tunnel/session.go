package tunnel

import (
	"net"

	"github.com/hashicorp/yamux"
)

type YamuxSession struct {
	session *yamux.Session
}

func NewYamuxSession(conn net.Conn) *YamuxSession {
	session, _ := yamux.Server(conn, nil)
	return &YamuxSession{session: session}
}

func NewYamuxClient(conn net.Conn) *YamuxSession {
	session, _ := yamux.Client(conn, nil)
	return &YamuxSession{session: session}
}

func (s *YamuxSession) Accept() (net.Conn, error) {
	return s.session.Accept()
}

func (s *YamuxSession) Open() (net.Conn, error) {
	return s.session.Open()
}

func (s *YamuxSession) Close() error {
	return s.session.Close()
}

func (s *YamuxSession) IsClosed() bool {
	return s.session.IsClosed()
}
