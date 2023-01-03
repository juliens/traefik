package tcp

import (
	"crypto/tls"
	"errors"
	"syscall"
)

// TLSHandler handles TLS connections.
type TLSHandler struct {
	Next   Handler
	Config *tls.Config
}

// ServeTCP terminates the TLS connection.
func (t *TLSHandler) ServeTCP(conn WriteCloser) {
	t.Next.ServeTCP(SyscallConnWrapper{tls.Server(conn, t.Config)})
}

type SyscallConnWrapper struct {
	*tls.Conn
}

func (s SyscallConnWrapper) SyscallConn() (syscall.RawConn, error) {
	return nil, errors.New("not implementeds")
}
