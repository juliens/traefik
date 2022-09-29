package service

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	pool := NewConnectionPool(ln.Addr().String(), 2)
	_ = pool

	// for i := 0; i < 11; i++ {
	// c1 := pool.AcquireConn()
	// c2 := pool.AcquireConn()
	// c3 := pool.AcquireConn()

	// pool.ReleaseConn(c1)
	// pool.ReleaseConn(c2)
	// pool.ReleaseConn(c3)
	// }
}
