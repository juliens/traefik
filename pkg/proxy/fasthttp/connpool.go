package fasthttp

import (
	"fmt"
	"net"
	"time"

	"github.com/traefik/traefik/v2/pkg/log"
)

type conn struct {
	net.Conn
	lastUseTime time.Time
}

func (c *conn) isExpired() bool {
	expTime := c.lastUseTime.Add(60 * time.Second)
	return time.Now().After(expTime)
}

// ConnPool is a net.Conn pool implementation using channels.
// FIXME handle errors/timeout
// FIXME handle idleConnLifetime
type ConnPool struct {
	idleConns chan *conn
	dialer    func() (net.Conn, error)
}

// NewConnPool creates a new ConnPool.
func NewConnPool(dialer func() (net.Conn, error), maxIdleConn int) *ConnPool {
	c := &ConnPool{
		dialer:    dialer,
		idleConns: make(chan *conn, maxIdleConn),
	}
	c.cleanIdleConns()

	return c
}

// AcquireConn returns an idle net.Conn from the pool.
func (c *ConnPool) AcquireConn() (net.Conn, error) {
	for {
		co, err := c.acquireConn()
		if err != nil {
			return nil, err
		}

		if !co.isExpired() {
			return co.Conn, nil
		}

		// As the acquired conn is expired we can close it
		// without putting it again into the pool.
		if err := co.Close(); err != nil {
			log.WithoutContext().
				WithError(err).
				Debug("Unexpected error while releasing the connection")
		}
	}
}

// ReleaseConn releases the given net.Conn to the pool.
func (c *ConnPool) ReleaseConn(co net.Conn) {
	c.releaseConn(&conn{Conn: co, lastUseTime: time.Now()})
}

// cleanIdleConns is a routine cleaning the expired connections at a regular basis.
// FIXME should we cancel the timer when the pool is deleted?
func (c *ConnPool) cleanIdleConns() {
	defer time.AfterFunc(time.Minute, c.cleanIdleConns)

	for {
		select {
		case co := <-c.idleConns:
			if !co.isExpired() {
				c.releaseConn(co)
				return
			}

			if err := co.Close(); err != nil {
				log.WithoutContext().
					WithError(err).
					Debug("Unexpected error while releasing the connection")
			}

		default:
			return
		}
	}
}

func (c *ConnPool) acquireConn() (*conn, error) {
	select {
	case co := <-c.idleConns:
		return co, nil

	default:
		errCh := make(chan error, 1)
		go c.askForNewConn(errCh)

		select {
		case co := <-c.idleConns:
			return co, nil

		case err := <-errCh:
			return nil, err
		}
	}
}

func (c *ConnPool) releaseConn(co *conn) {
	select {
	case c.idleConns <- co:

	// Hitting the default case means that we have reached the maximum number of idle
	// connections, so we can close it.
	default:
		if err := co.Close(); err != nil {
			log.WithoutContext().
				WithError(err).
				Debug("Unexpected error while releasing the connection")
		}
	}
}

func (c *ConnPool) askForNewConn(errCh chan<- error) {
	co, err := c.dialer()
	if err != nil {
		errCh <- fmt.Errorf("create conn: %w", err)
		return
	}

	c.ReleaseConn(co)
}
