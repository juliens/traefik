package service

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
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
	idleConn chan *conn
	dialer   func() (net.Conn, error)
}

// TODO handle errors ( and timeout )
// TODO handle idleConnLifetime

type conn struct {
	net.Conn
	lastUsed time.Time
	timer    *time.Timer
}

func (c *conn) isExpired() bool {
	return time.Now().After(c.lastUsed.Add(60 * time.Second))
}

func NewConnectionPool(dial func() (net.Conn, error), maxIdleConn int) *ConnectionPool {
	c := &ConnectionPool{
		dialer:   dial,
		idleConn: make(chan *conn, maxIdleConn),
	}
	c.cleanIdleConns()

	return c
}

func (c *ConnectionPool) cleanIdleConns() {
	defer time.AfterFunc(time.Minute, c.cleanIdleConns)

	for {
		select {
		case conn := <-c.idleConn:
			if conn.isExpired() {
				c.ReleaseConn(conn)
				return
			}
			_ = conn.Close()
		default:
			// EMPTY
			return
		}
	}
}

func (c *ConnectionPool) askForNewConn(errChan chan<- error) {
	co, err := c.dialer()
	if err != nil {
		errChan <- err
	}
	close(errChan)

	c.ReleaseConn(&conn{Conn: co})
}

func (c *ConnectionPool) acquireConn() (*conn, error) {
	var co net.Conn
	select {
	case co = <-c.idleConn:
	default:
		errChan := make(chan error, 1)

		go func() {
			c.askForNewConn(errChan)
		}()
		select {
		case co = <-c.idleConn:
		case err := <-errChan:
			return nil, err
		}
	}
	return &conn{Conn: co}, nil
}

func (c *ConnectionPool) AcquireConn() (net.Conn, error) {
	for {
		conn, err := c.acquireConn()
		if err != nil {
			return nil, err
		}
		if conn.isExpired() {
			return conn.Conn, nil
		}
	}
}

func (c *ConnectionPool) ReleaseConn(co net.Conn) {

	select {
	case c.idleConn <- &conn{Conn: co, lastUsed: time.Now()}:

	default:
		// DROP IDLE CONN
		co.Close()
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

func (h *HostChooser) GetPool(req http.Request) *ConnectionPool {
	h.poolMu.RLock()
	req.URL.Port()

	key := req.URL.Scheme + "|" + req.URL.Host
	url := addMissingPort(req.URL.Host, strings.EqualFold(req.URL.Scheme, "https"))
	pool := h.pools[key]
	h.poolMu.RUnlock()
	if pool != nil {
		return pool
	}

	if req.URL.Scheme == "https" {
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
	return pool
}
