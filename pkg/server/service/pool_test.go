package service

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockConn struct {
	closeFn func() error
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *mockConn) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	// TODO implement me
	panic("implement me")
}

func (m *mockConn) RemoteAddr() net.Addr {
	// TODO implement me
	panic("implement me")
}

func (m *mockConn) SetDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}

func TestConnectionReuse(t *testing.T) {
	testCases := []struct {
		desc     string
		poolFn   func(pool *ConnectionPool)
		expected int
	}{
		{
			desc: "Simple case",
			poolFn: func(pool *ConnectionPool) {
				c1 := pool.AcquireConn()
				pool.ReleaseConn(c1)

			},
			expected: 1,
		},
		{
			desc: "Simple with reuse",
			poolFn: func(pool *ConnectionPool) {
				c1 := pool.AcquireConn()
				pool.ReleaseConn(c1)

				c2 := pool.AcquireConn()
				pool.ReleaseConn(c2)
			},
			expected: 1,
		},
		{
			desc: "Two connection at the same time",
			poolFn: func(pool *ConnectionPool) {
				c1 := pool.AcquireConn()
				c2 := pool.AcquireConn()

				pool.ReleaseConn(c1)
				pool.ReleaseConn(c2)
			},
			expected: 2,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			var connAlloc int
			dialer := func() (net.Conn, error) {
				connAlloc++
				return &net.TCPConn{}, nil
			}
			pool := NewConnectionPool(dialer, 2)
			test.poolFn(pool)

			assert.Equal(t, test.expected, connAlloc)
		})
	}
}

func TestMaxIdleConn(t *testing.T) {
	testCases := []struct {
		desc        string
		poolFn      func(pool *ConnectionPool)
		maxIdleConn int
		expected    int
	}{
		{
			desc: "Simple case",
			poolFn: func(pool *ConnectionPool) {
				c1 := pool.AcquireConn()
				pool.ReleaseConn(c1)
			},
			maxIdleConn: 1,
			expected:    1,
		},
		{
			desc: "Multiple conn with release",
			poolFn: func(pool *ConnectionPool) {
				for i := 0; i < 7; i++ {
					c := pool.AcquireConn()
					defer pool.ReleaseConn(c)
				}
			},
			maxIdleConn: 5,
			expected:    5,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			var keepOpenedConn int
			dialer := func() (net.Conn, error) {
				keepOpenedConn++
				return &mockConn{closeFn: func() error {
					keepOpenedConn--
					return nil
				}}, nil
			}
			pool := NewConnectionPool(dialer, test.maxIdleConn)
			test.poolFn(pool)

			assert.Equal(t, test.expected, keepOpenedConn)
		})
	}
}

func TestName(t *testing.T) {
	pool := Pool[bufio.Reader]{}
	b := pool.Get()
	fmt.Println(b)
	// if b == nil {
	// 	fmt.Println("NIL")
	// }
	// require.NotNil(t, b)
}
