package fasthttp

import (
	"errors"
	"net"
	"sync"
	"time"
)

const DefaultMaxConnsPerHost = 2000

type SliceConnectionPool struct {
	conns               []*conn
	connsLock           sync.RWMutex
	MaxConnWaitTimeout  time.Duration
	connsWait           *wantConnQueue
	MaxConns            int
	connsCount          int
	connsCleanerRun     bool
	MaxIdleConnDuration time.Duration
	dialer              func() (net.Conn, error)
	connPool            Pool[*conn]
}

func NewSliceConnectionPool(dialer func() (net.Conn, error), max int) ConnectionPool {
	s := &SliceConnectionPool{
		conns: make([]*conn, 0, 1000),
		// connsLock:          sync.RWMutex{},
		MaxConnWaitTimeout: time.Minute,
		// connsWait:           nil,
		MaxConns: 5000,
		// connsCount:          0,
		// connsCleanerRun:     false,
		MaxIdleConnDuration: time.Minute,
		dialer:              dialer,
	}
	s.connPool.New = func() *conn {
		return &conn{}
	}
	return s
}

func (c *SliceConnectionPool) AcquireConn() (cc *conn, err error) {
	createConn := false
	startCleaner := false

	var n int
	c.connsLock.Lock()
	n = len(c.conns)
	if n == 0 {
		maxConns := c.MaxConns
		if maxConns <= 0 {

			maxConns = DefaultMaxConnsPerHost
		}
		if c.connsCount < maxConns {
			c.connsCount++
			createConn = true

			if !c.connsCleanerRun {
				startCleaner = true
				c.connsCleanerRun = true
			}
		}
	} else {
		n--
		cc = c.conns[n]
		c.conns[n] = nil
		c.conns = c.conns[:n]
	}
	c.connsLock.Unlock()

	if cc != nil {
		return cc, nil
	}
	if !createConn {
		if c.MaxConnWaitTimeout <= 0 {
			return nil, errors.New("ErrNoFreeConns")
		}

		// reqTimeout    c.MaxConnWaitTimeout   wait duration
		//     d1                 d2            min(d1, d2)
		//  0(not set)            d2            d2
		//     d1            0(don't wait)      0(don't wait)
		//  0(not set)            d2            d2
		timeout := c.MaxConnWaitTimeout
		// timeoutOverridden := false
		// reqTimeout == 0 means not set
		reqTimeout := 0 * time.Second
		if reqTimeout > 0 && reqTimeout < timeout {
			timeout = reqTimeout
			// timeoutOverridden = true
		}

		// wait for a free connection
		// tc := AcquireTimer(timeout)
		// defer ReleaseTimer(tc)

		w := &wantConn{
			ready: make(chan struct{}, 1),
		}
		defer func() {
			if err != nil {
				w.cancel(c, err)
			}
		}()

		c.queueForIdle(w)

		select {
		case <-w.ready:
			return w.conn, w.err
			// case <-tc.C:
			// 	if timeoutOverridden {
			// 		return nil, ErrTimeout
			// 	}
			// 	return nil, ErrNoFreeConns
		}
	}

	if startCleaner {
		go c.connsCleaner()
	}

	con, err := c.dialer()
	if err != nil {
		c.decConnsCount()
		return nil, err
	}
	cc = c.connPool.Get()
	cc.Conn = con

	return cc, nil
}

func (c *SliceConnectionPool) decConnsCount() {
	if c.MaxConnWaitTimeout <= 0 {
		c.connsLock.Lock()
		c.connsCount--
		c.connsLock.Unlock()
		return
	}

	c.connsLock.Lock()
	defer c.connsLock.Unlock()
	dialed := false
	if q := c.connsWait; q != nil && q.len() > 0 {
		for q.len() > 0 {
			w := q.popFront()
			if w.waiting() {
				go c.dialConnFor(w)
				dialed = true
				break
			}
		}
	}
	if !dialed {
		c.connsCount--
	}
}

func (c *SliceConnectionPool) dialConnFor(w *wantConn) {
	con, err := c.dialHostHard()
	if err != nil {
		w.tryDeliver(nil, err)
		c.decConnsCount()
		return
	}

	cc := c.connPool.Get()
	cc.Conn = con
	delivered := w.tryDeliver(cc, nil)
	if !delivered {
		// not delivered, return idle connection
		c.ReleaseConn(cc)
	}
}

func (c *SliceConnectionPool) dialHostHard() (net.Conn, error) {
	return c.dialer()
}

func (c *SliceConnectionPool) connsCleaner() {
	var (
		scratch             []*conn
		maxIdleConnDuration = c.MaxIdleConnDuration
	)
	if maxIdleConnDuration <= 0 {
		const DefaultMaxIdleConnDuration = 60 * time.Second
		maxIdleConnDuration = DefaultMaxIdleConnDuration
	}
	for {
		currentTime := time.Now()

		// Determine idle connections to be closed.
		c.connsLock.Lock()
		conns := c.conns
		n := len(conns)
		i := 0
		for i < n && currentTime.Sub(conns[i].lastUseTime) > maxIdleConnDuration {
			i++
		}
		sleepFor := maxIdleConnDuration
		if i < n {
			// + 1 so we actually sleep past the expiration time and not up to it.
			// Otherwise the > check above would still fail.
			sleepFor = maxIdleConnDuration - currentTime.Sub(conns[i].lastUseTime) + 1
		}
		scratch = append(scratch[:0], conns[:i]...)
		if i > 0 {
			m := copy(conns, conns[i:])
			for i = m; i < n; i++ {
				conns[i] = nil
			}
			c.conns = conns[:m]
		}
		c.connsLock.Unlock()

		// Close idle connections.
		for i, cc := range scratch {
			c.decConnsCount()
			cc.Close()
			c.connPool.Put(cc)
			scratch[i] = nil
		}

		// Determine whether to stop the connsCleaner.
		c.connsLock.Lock()
		mustStop := c.connsCount == 0
		if mustStop {
			c.connsCleanerRun = false
		}
		c.connsLock.Unlock()
		if mustStop {
			break
		}

		time.Sleep(sleepFor)
	}
}

func (c *SliceConnectionPool) queueForIdle(w *wantConn) {
	c.connsLock.Lock()
	defer c.connsLock.Unlock()
	if c.connsWait == nil {
		c.connsWait = &wantConnQueue{}
	}
	c.connsWait.clearFront()
	c.connsWait.pushBack(w)
}

func (c *SliceConnectionPool) ReleaseConn(cc *conn) {
	cc.lastUseTime = time.Now()
	if c.MaxConnWaitTimeout <= 0 {
		c.connsLock.Lock()
		c.conns = append(c.conns, cc)
		c.connsLock.Unlock()
		return
	}

	// try to deliver an idle connection to a *wantConn
	c.connsLock.Lock()
	defer c.connsLock.Unlock()
	delivered := false
	if q := c.connsWait; q != nil && q.len() > 0 {
		for q.len() > 0 {
			w := q.popFront()
			if w.waiting() {
				delivered = w.tryDeliver(cc, nil)
				break
			}
		}
	}
	if !delivered {
		c.conns = append(c.conns, cc)
	}
}

type wantConnQueue struct {
	// This is a queue, not a deque.
	// It is split into two stages - head[headPos:] and tail.
	// popFront is trivial (headPos++) on the first stage, and
	// pushBack is trivial (append) on the second stage.
	// If the first stage is empty, popFront can swap the
	// first and second stages to remedy the situation.
	//
	// This two-stage split is analogous to the use of two lists
	// in Okasaki's purely functional queue but without the
	// overhead of reversing the list when swapping stages.
	head    []*wantConn
	headPos int
	tail    []*wantConn
}

// len returns the number of items in the queue.
func (q *wantConnQueue) len() int {
	return len(q.head) - q.headPos + len(q.tail)
}

// pushBack adds w to the back of the queue.
func (q *wantConnQueue) pushBack(w *wantConn) {
	q.tail = append(q.tail, w)
}

// popFront removes and returns the wantConn at the front of the queue.
func (q *wantConnQueue) popFront() *wantConn {
	if q.headPos >= len(q.head) {
		if len(q.tail) == 0 {
			return nil
		}
		// Pick up tail as new head, clear tail.
		q.head, q.headPos, q.tail = q.tail, 0, q.head[:0]
	}

	w := q.head[q.headPos]
	q.head[q.headPos] = nil
	q.headPos++
	return w
}

// peekFront returns the wantConn at the front of the queue without removing it.
func (q *wantConnQueue) peekFront() *wantConn {
	if q.headPos < len(q.head) {
		return q.head[q.headPos]
	}
	if len(q.tail) > 0 {
		return q.tail[0]
	}
	return nil
}

// cleanFront pops any wantConns that are no longer waiting from the head of the
// queue, reporting whether any were popped.
func (q *wantConnQueue) clearFront() (cleaned bool) {
	for {
		w := q.peekFront()
		if w == nil || w.waiting() {
			return cleaned
		}
		q.popFront()
		cleaned = true
	}
}

type wantConn struct {
	ready chan struct{}
	mu    sync.Mutex // protects conn, err, close(ready)
	conn  *conn
	err   error
}

// waiting reports whether w is still waiting for an answer (connection or error).
func (w *wantConn) waiting() bool {
	select {
	case <-w.ready:
		return false
	default:
		return true
	}
}

// tryDeliver attempts to deliver conn, err to w and reports whether it succeeded.
func (w *wantConn) tryDeliver(conn *conn, err error) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil || w.err != nil {
		return false
	}
	w.conn = conn
	w.err = err
	if w.conn == nil && w.err == nil {
		panic("fasthttp: internal error: misuse of tryDeliver")
	}
	close(w.ready)
	return true
}

// cancel marks w as no longer wanting a result (for example, due to cancellation).
// If a connection has been delivered already, cancel returns it with c.releaseConn.
func (w *wantConn) cancel(c *SliceConnectionPool, err error) {
	w.mu.Lock()
	if w.conn == nil && w.err == nil {
		close(w.ready) // catch misbehavior in future delivery
	}

	conn := w.conn
	w.conn = nil
	w.err = err
	w.mu.Unlock()

	if conn != nil {
		c.ReleaseConn(conn)
	}
}
