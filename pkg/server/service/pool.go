package service

import (
	"net"
	"sync"
)

type ConnectionPool struct {
	address  string
	idleConn chan net.Conn
	newConn  chan func() net.Conn
}

func NewConnectionPool(addr string, maxIdleConn int) *ConnectionPool {
	conns := make(chan func() net.Conn)
	go func() {
		for {
			select {
			case conns <- newConn(addr):
			}
		}
	}()
	return &ConnectionPool{
		address:  addr,
		idleConn: make(chan net.Conn, maxIdleConn),
		newConn:  conns,
	}
}

func newConn(addr string) func() net.Conn {
	return func() net.Conn {
		dial, _ := net.Dial("tcp", addr)
		return dial
	}
}

func (c *ConnectionPool) AcquireConn() net.Conn {
	var conn net.Conn
	select {
	case conn = <-c.idleConn:
	default:
		select {
		case conn = <-c.idleConn:
		case connFn := <-c.newConn:
			conn = connFn()
		}
	}
	return conn
}

func (c *ConnectionPool) ReleaseConn(conn net.Conn) {
	select {
	case c.idleConn <- conn:
	default:
		// DROP IDLE CONN
		conn.Close()
	}
}

func NewHostChooser() *HostChooser {
	return &HostChooser{
		pools: make(map[string]*ConnectionPool),
	}
}

type HostChooser struct {
	pools  map[string]*ConnectionPool
	poolMu sync.RWMutex
}

func (h *HostChooser) GetPool(scheme string, url string) *ConnectionPool {
	h.poolMu.RLock()
	key := scheme + "|" + url
	pool := h.pools[key]
	h.poolMu.RUnlock()
	if pool == nil {
		pool = NewConnectionPool(url, 200)
		h.poolMu.Lock()
		h.pools[key] = pool
		h.poolMu.Unlock()
	}
	return pool
}
