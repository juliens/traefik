package service

import (
	"net"
	"sync"
)

type Pool[T any] struct {
	pool sync.Pool
}

func (p *Pool[T]) Get() T {
	var res T
	if temp := p.pool.Get(); temp != nil {
		return temp.(T)
	}
	return res
}

func (p *Pool[T]) Put(x T) {
	p.pool.Put(x)
}

type ConnectionPool struct {
	address  string
	idleConn chan net.Conn
	dialer   func() (net.Conn, error)
}

// TODO handle errors ( and timeout )
// TODO handle idleConnLifetime

func NewConnectionPool(dial func() (net.Conn, error), maxIdleConn int) *ConnectionPool {
	return &ConnectionPool{
		dialer:   dial,
		idleConn: make(chan net.Conn, maxIdleConn),
	}
}

func (c *ConnectionPool) askForNewConn() {
	conn, _ := c.dialer()

	c.ReleaseConn(conn)
}

func (c *ConnectionPool) AcquireConn() net.Conn {
	var conn net.Conn
	select {
	case conn = <-c.idleConn:
	default:
		go c.askForNewConn()
		select {
		case conn = <-c.idleConn:
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
		pool = NewConnectionPool(func() (net.Conn, error) {
			return net.Dial("tcp", url)
		}, 200)
		h.poolMu.Lock()
		h.pools[key] = pool
		h.poolMu.Unlock()
	}
	return pool
}
