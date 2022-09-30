package service

import (
	"crypto/tls"
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

func NewHostChooser(dialer func(network, address string) (net.Conn, error), maxIdleConnPerHost int, tlsConfig *tls.Config) *HostChooser {
	return &HostChooser{
		dialer:             dialer,
		pools:              make(map[string]*ConnectionPool),
		maxIdleConnPerHost: maxIdleConnPerHost,
		TLSConfig:          tlsConfig,
	}
}

type HostChooser struct {
	dialer             func(network, address string) (net.Conn, error)
	maxIdleConnPerHost int
	pools              map[string]*ConnectionPool
	poolMu             sync.RWMutex
	TLSConfig          *tls.Config
}

func (h *HostChooser) GetPool(scheme string, url string) *ConnectionPool {
	h.poolMu.RLock()
	key := scheme + "|" + url
	pool := h.pools[key]
	h.poolMu.RUnlock()
	if pool == nil {
		if scheme == "https" {
			pool = NewConnectionPool(func() (net.Conn, error) {
				conn, err := h.dialer("tcp", url)
				if err != nil {
					return nil, err
				}
				return tls.Client(conn, h.TLSConfig), nil
			}, h.maxIdleConnPerHost)
		} else {
			pool = NewConnectionPool(func() (net.Conn, error) {
				return h.dialer("tcp", url)
			}, h.maxIdleConnPerHost)
		}
		h.poolMu.Lock()
		h.pools[key] = pool
		h.poolMu.Unlock()
	}
	return pool
}
