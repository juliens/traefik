package httpwasm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/http-wasm/http-wasm-host-go/api"
	handlerapi "github.com/http-wasm/http-wasm-host-go/api/handler"
	"github.com/rs/zerolog"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/plugins/wasm/engine"
)

// bodyPrefetchSize is how much of the request body is read before taking the instance lock, so that a guest reading
// a small body never waits for the client while it holds the lock.
const bodyPrefetchSize = 64 * 1024

// httpWasm serves one plugin binary and one middleware configuration on a pool of instances.
type httpWasm struct {
	cfg    engine.Config
	host   host
	logger api.Logger
	zlog   *zerolog.Logger

	// features are the ones enabled by the guest at initialization; identical on every instance of the engine.
	features atomic.Uint64

	// pool serves the shared mode; exclusive serves the exclusive mode.
	pool      *engine.Pool
	exclusive *exclusivePool
}

// New starts the instances of an http-wasm plugin.
func New(ctx context.Context, cfg engine.Config) (engine.Engine, error) {
	e := &httpWasm{cfg: cfg, zlog: cfg.Logger, logger: logs.NewWasmLogger(cfg.Logger)}

	if cfg.Exclusive {
		e.exclusive = newExclusivePool(cfg.Instances, e.newInstance)
		// Fail fast, and let the first instance enable the guest features.
		inst, err := e.exclusive.get(ctx)
		if err != nil {
			return nil, err
		}
		e.exclusive.put(inst, false)

		return e, nil
	}

	pool, err := engine.NewPool(ctx, cfg.InstanceCount(), func(ctx context.Context, slot int) (engine.Instance, error) {
		return e.newInstance(ctx, slot)
	}, cfg.Logger)
	if err != nil {
		return nil, err
	}
	e.pool = pool

	return e, nil
}

// ServeHTTP implements engine.Engine.
func (e *httpWasm) ServeHTTP(rw http.ResponseWriter, req *http.Request, next http.Handler) {
	if e.exclusive != nil {
		e.serveExclusive(rw, req, next)

		return
	}

	if err := e.pool.Enter(); err != nil {
		http.Error(rw, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)

		return
	}
	defer e.pool.Leave()

	slot, err := e.pool.Pick()
	if err != nil {
		e.zlog.Error().Err(err).Msg("No wasm instance available")
		http.Error(rw, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)

		return
	}
	inst := slot.Instance().(*instance)

	e.serve(rw, req, next, inst, func(err error) {
		if inst.broken(err) {
			e.pool.Replace(slot, err)
		}
	})
}

// Stats implements engine.Engine.
func (e *httpWasm) Stats() engine.Stats {
	if e.exclusive != nil {
		return e.exclusive.stats()
	}

	return e.pool.Stats()
}

// Close implements engine.Engine.
func (e *httpWasm) Close(ctx context.Context) error {
	if e.exclusive != nil {
		return e.exclusive.close(ctx)
	}

	return e.pool.Close(ctx)
}

func (e *httpWasm) enableInstanceFeatures(features handlerapi.Features) handlerapi.Features {
	for {
		old := e.features.Load()
		updated := handlerapi.Features(old).WithEnabled(features)
		if e.features.CompareAndSwap(old, uint64(updated)) {
			return updated
		}
	}
}

func (e *httpWasm) serveExclusive(rw http.ResponseWriter, req *http.Request, next http.Handler) {
	inst, err := e.exclusive.get(req.Context())
	if err != nil {
		e.zlog.Error().Err(err).Msg("Creating wasm instance")
		http.Error(rw, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)

		return
	}

	broken := false
	e.serve(rw, req, next, inst, func(err error) {
		broken = inst.broken(err)
	})

	e.exclusive.put(inst, broken)
}

// serve runs one request on inst: handle_request, then next (with the instance free), then handle_response.
// onError is called with the error of a failed guest call.
func (e *httpWasm) serve(rw http.ResponseWriter, req *http.Request, next http.Handler, inst *instance, onError func(error)) {
	s := newRequestState(rw, req, next, handlerapi.Features(e.features.Load()))
	prefetchBody(req)

	ctx := context.WithValue(req.Context(), requestStateKey{}, s)

	ctxNext, err := inst.handleRequest(ctx)
	if err != nil {
		_ = s.close()
		e.fail(rw, "handle_request", err, onError)

		return
	}

	// If buffering was enabled, ensure it flushes.
	if bw, ok := s.w.(*bufferingResponseWriter); ok {
		defer bw.release()
	}

	// Returning zero means the guest wants to break the handler chain, and handle the response directly.
	if uint32(ctxNext) == 0 {
		_ = s.close()

		return
	}

	if err = s.closeRequest(); err != nil {
		e.zlog.Debug().Err(err).Msg("Closing request body")
	}

	// The instance is free while next runs.
	nextErr := s.handleNext()
	s.afterNext = true

	err = inst.handleResponse(ctx, uint32(ctxNext>>32), nextErr)
	_ = s.close()
	if err != nil {
		e.fail(rw, "handle_response", err, onError)
	}
}

// fail logs a guest call error and answers 500 when the response was not started.
func (e *httpWasm) fail(rw http.ResponseWriter, call string, err error, onError func(error)) {
	e.zlog.Error().Err(err).Str("call", call).Msg("Wasm guest call failed")
	onError(err)

	http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// prefetchBody reads the start of the request body before the guest is called, outside the instance lock.
func prefetchBody(req *http.Request) {
	if req.Body == nil || req.Body == http.NoBody {
		return
	}

	prefix, err := io.ReadAll(io.LimitReader(req.Body, bodyPrefetchSize))
	if err != nil {
		req.Body = &prefetchedBody{Reader: io.MultiReader(bytes.NewReader(prefix), errReader{err: err}), closer: req.Body}

		return
	}

	req.Body = &prefetchedBody{Reader: io.MultiReader(bytes.NewReader(prefix), req.Body), closer: req.Body}
}

type prefetchedBody struct {
	io.Reader

	closer io.Closer
}

func (b *prefetchedBody) Close() error {
	return b.closer.Close()
}

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}

// exclusivePool gives one instance to each in-flight request, creating instances on demand and keeping at most
// max idle ones (0 means unbounded).
type exclusivePool struct {
	factory func(ctx context.Context, slot int) (*instance, error)
	max     int

	mu       sync.Mutex
	idle     []*instance
	created  int
	inFlight int64
	closed   bool
}

func newExclusivePool(maxIdle int, factory func(ctx context.Context, slot int) (*instance, error)) *exclusivePool {
	return &exclusivePool{factory: factory, max: maxIdle}
}

func (p *exclusivePool) get(ctx context.Context) (*instance, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()

		return nil, engine.ErrClosing
	}

	p.inFlight++
	if n := len(p.idle); n > 0 {
		inst := p.idle[n-1]
		p.idle = p.idle[:n-1]
		p.mu.Unlock()

		return inst, nil
	}

	slot := p.created
	p.created++
	p.mu.Unlock()

	inst, err := p.factory(ctx, slot)
	if err != nil {
		p.mu.Lock()
		p.inFlight--
		p.mu.Unlock()

		return nil, fmt.Errorf("creating instance: %w", err)
	}

	return inst, nil
}

// put gives an instance back; a broken instance is closed instead of being kept.
func (p *exclusivePool) put(inst *instance, broken bool) {
	p.mu.Lock()
	p.inFlight--
	if broken || p.closed || (p.max > 0 && len(p.idle) >= p.max) {
		p.mu.Unlock()
		_ = inst.Close(context.Background())

		return
	}
	p.idle = append(p.idle, inst)
	p.mu.Unlock()
}

func (p *exclusivePool) stats() engine.Stats {
	p.mu.Lock()
	defer p.mu.Unlock()

	return engine.Stats{InFlight: p.inFlight, LiveInstances: len(p.idle) + int(p.inFlight)}
}

func (p *exclusivePool) close(ctx context.Context) error {
	p.mu.Lock()
	p.closed = true
	idle := p.idle
	p.idle = nil
	p.mu.Unlock()

	var errs []error
	for _, inst := range idle {
		if err := inst.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
