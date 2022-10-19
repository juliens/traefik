package fasthttp

import (
	"net"
	"sync"
	"time"
)

type Pool[T any] struct {
	pool sync.Pool
	New  func() T
}

func (p *Pool[T]) Get() T {
	if temp := p.pool.Get(); temp != nil {
		return temp.(T)
	}

	if p.New != nil {
		return p.New()
	}

	var res T
	return res
}

func (p *Pool[T]) Put(x T) {
	p.pool.Put(x)
}

type ConnectionPool interface {
	AcquireConn() (*conn, error)
	ReleaseConn(co *conn)
}

type ChannelConnectionPool struct {
	idleConn chan *conn
	dialer   func() (net.Conn, error)
}

// TODO handle errors ( and timeout )
// TODO handle idleConnLifetime

type conn struct {
	net.Conn
	lastUseTime time.Time
	timer       *time.Timer
}

func (c *conn) isExpired() bool {
	lu := c.lastUseTime
	after := lu.Add(60 * time.Second)
	return time.Now().After(after)
}

func NewConnectionPool(dial func() (net.Conn, error), maxIdleConn int) ConnectionPool {
	c := &ChannelConnectionPool{
		dialer:   dial,
		idleConn: make(chan *conn, maxIdleConn),
	}
	c.cleanIdleConns()

	return c
}

func (c *ChannelConnectionPool) cleanIdleConns() {
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

func (c *ChannelConnectionPool) askForNewConn(errChan chan<- error) {
	co, err := c.dialer()
	if err != nil {
		errChan <- err
		return
	}

	c.ReleaseConn(&conn{Conn: co})
}

func (c *ChannelConnectionPool) acquireConn() (*conn, error) {
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

func (c *ChannelConnectionPool) AcquireConn() (*conn, error) {
	for {
		conn, err := c.acquireConn()
		if err != nil {
			return nil, err
		}

		if !conn.isExpired() {
			return conn, nil
		}
	}
}

func (c *ChannelConnectionPool) ReleaseConn(co *conn) {
	select {
	case c.idleConn <- &conn{Conn: co.Conn, lastUseTime: time.Now()}:

	default:
		// DROP IDLE CONN
		co.Close()
	}
}
