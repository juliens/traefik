package wasihttp

import (
	"context"
	"errors"
	"sync"

	"github.com/traefik/traefik/v3/pkg/plugins/wasm/engine"
)

// WASI preview 1 errno values.
const (
	errnoSuccess uint32 = 0
	errnoAgain   uint32 = 6
	errnoBadf    uint32 = 8
	errnoInval   uint32 = 28
	errnoIO      uint32 = 29
	errnoNosys   uint32 = 52
	errnoPipe    uint32 = 64
)

// pipeSize bounds every pipe between the host and the guest: a slow reader on either side slows the writer down
// instead of growing the memory.
const pipeSize = 64 * 1024

// vfd is a virtual file descriptor of the guest, on top of a pipe. Its guest side never blocks.
type vfd struct {
	pipe *engine.Pipe

	// onOpen runs once, at the first read of the guest, to start a lazy producer.
	onOpen func()
	once   sync.Once
}

func newVFD(pipe *engine.Pipe) *vfd {
	return &vfd{pipe: pipe}
}

func (f *vfd) read(p []byte) (int, uint32) {
	if f.onOpen != nil {
		f.once.Do(f.onOpen)
	}

	n, err := f.pipe.GuestRead(p)
	switch {
	case err == nil:
		return n, errnoSuccess
	case errors.Is(err, engine.ErrAgain):
		return 0, errnoAgain
	default:
		return 0, errnoIO
	}
}

func (f *vfd) write(p []byte) (int, uint32) {
	n, err := f.pipe.GuestWrite(p)
	switch {
	case err == nil:
		return n, errnoSuccess
	case errors.Is(err, engine.ErrAgain):
		return 0, errnoAgain
	default:
		return 0, errnoPipe
	}
}

// readable does not start a lazy producer: a guest polls a descriptor only after a read returned EAGAIN, and that
// read started it.
func (f *vfd) readable() int {
	return f.pipe.Readable()
}

func (f *vfd) writable() bool {
	return f.pipe.Writable()
}

func (f *vfd) close() {
	_ = f.pipe.Close()
}

// pipeReader reads the host side of a pipe as an io.Reader.
type pipeReader struct {
	ctx  context.Context
	pipe *engine.Pipe
}

func (r pipeReader) Read(p []byte) (int, error) {
	return r.pipe.Read(r.ctx, p)
}
