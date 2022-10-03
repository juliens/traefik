package service

import (
	"net"
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
	lu := c.lastUsed
	after := lu.Add(60 * time.Second)
	return time.Now().After(after)
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
				c.ReleaseConn(conn.Conn)
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
		return
	}

	c.ReleaseConn(co)
}

func (c *ConnectionPool) acquireConn() (*conn, error) {
	var co *conn
	select {
	case co = <-c.idleConn:
	default:
		errChan := make(chan error, 1)
		go c.askForNewConn(errChan)
		select {
		case co = <-c.idleConn:
		case err := <-errChan:
			return nil, err
		}
	}
	return co, nil
}

func (c *ConnectionPool) AcquireConn() (net.Conn, error) {
	for {
		conn, err := c.acquireConn()
		if err != nil {
			return nil, err
		}

		if !conn.isExpired() {
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
