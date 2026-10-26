package engine

import (
	"context"
	"errors"
	"io"
	"sync"
)

// Pipe is a byte stream between the host and a guest.
// The host end blocks (Write, Read), the guest end never does (GuestRead, GuestWrite report EAGAIN).
// A zero max size means unbounded.
type Pipe struct {
	mu      sync.Mutex
	buf     []byte
	max     int
	closed  bool
	err     error
	onEvent func()

	readable chan struct{}
	writable chan struct{}
}

// ErrAgain is returned by the guest end when it would block.
var ErrAgain = errors.New("would block")

// NewPipe creates a pipe; onEvent, if set, is called whenever the guest end may make progress.
func NewPipe(maxSize int, onEvent func()) *Pipe {
	return &Pipe{
		max:      maxSize,
		onEvent:  onEvent,
		readable: make(chan struct{}, 1),
		writable: make(chan struct{}, 1),
	}
}

func signal(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

// Write appends to the pipe, blocking while it is full.
func (p *Pipe) Write(b []byte) (int, error) {
	written := 0
	for written < len(b) {
		p.mu.Lock()
		if p.closed {
			err := p.err
			p.mu.Unlock()
			if err == nil {
				err = io.ErrClosedPipe
			}

			return written, err
		}

		room := len(b) - written
		if p.max > 0 {
			room = min(room, p.max-len(p.buf))
		}
		if room > 0 {
			p.buf = append(p.buf, b[written:written+room]...)
			written += room
			p.mu.Unlock()
			p.notify()

			continue
		}
		p.mu.Unlock()

		<-p.writable
	}

	return written, nil
}

// Read reads from the pipe, blocking until data, EOF or ctx is done.
func (p *Pipe) Read(ctx context.Context, b []byte) (int, error) {
	for {
		p.mu.Lock()
		if len(p.buf) > 0 {
			n := copy(b, p.buf)
			p.buf = p.buf[n:]
			p.mu.Unlock()
			p.notify()

			return n, nil
		}
		if p.closed {
			err := p.err
			p.mu.Unlock()
			if err == nil {
				err = io.EOF
			}

			return 0, err
		}
		p.mu.Unlock()

		select {
		case <-p.readable:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
}

// Close marks the end of the stream; readers get EOF after the buffered data.
func (p *Pipe) Close() error {
	return p.CloseWithError(nil)
}

// CloseWithError marks the end of the stream with an error that readers and writers get instead of EOF.
func (p *Pipe) CloseWithError(err error) error {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		p.err = err
	}
	p.mu.Unlock()
	p.notify()

	return nil
}

// GuestRead copies buffered data without blocking; it returns ErrAgain when nothing is available yet and (0, nil)
// at EOF.
func (p *Pipe) GuestRead(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.buf) == 0 {
		if p.closed {
			return 0, p.err
		}

		return 0, ErrAgain
	}

	n := copy(b, p.buf)
	p.buf = p.buf[n:]
	signal(p.writable)

	return n, nil
}

// GuestWrite appends without blocking; it returns ErrAgain when the pipe is full.
func (p *Pipe) GuestWrite(b []byte) (int, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()

		return 0, io.ErrClosedPipe
	}

	room := len(b)
	if p.max > 0 {
		room = min(room, p.max-len(p.buf))
	}
	if room == 0 && len(b) > 0 {
		p.mu.Unlock()

		return 0, ErrAgain
	}
	p.buf = append(p.buf, b[:room]...)
	p.mu.Unlock()
	signal(p.readable)

	return room, nil
}

// Readable reports the bytes the guest can read, or 1 at EOF, or 0 when it would block.
func (p *Pipe) Readable() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.buf) > 0 {
		return len(p.buf)
	}
	if p.closed {
		return 1
	}

	return 0
}

// Buffered reports the bytes written and not read yet.
func (p *Pipe) Buffered() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.buf)
}

// Writable reports whether the guest can write without blocking.
func (p *Pipe) Writable() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.closed || p.max == 0 || len(p.buf) < p.max
}

func (p *Pipe) notify() {
	signal(p.readable)
	signal(p.writable)
	if p.onEvent != nil {
		p.onEvent()
	}
}
