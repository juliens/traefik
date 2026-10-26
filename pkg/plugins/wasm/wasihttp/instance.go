package wasihttp

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"sync"

	"github.com/rs/zerolog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/traefik/traefik/v3/pkg/plugins/wasm/engine"
)

var errRequestDone = errors.New("request done")

// instance is one guest instance: its own runtime and memory, one goroutine blocked in traefik_serve, and any number
// of in-flight requests.
type instance struct {
	cfg    *engine.Config
	logger *zerolog.Logger
	rt     wazero.Runtime
	mod    api.Module

	// client sends the outbound requests of the guest; ctx cancels them when the instance stops.
	client *http.Client
	ctx    context.Context
	cancel context.CancelFunc

	listener *engine.Pipe
	// wake is signaled by every event that may let the guest make progress.
	wake chan struct{}
	// closed is closed when the instance is closing, to stop the guest where it waits.
	closed    chan struct{}
	closeOnce sync.Once
	// stopped is closed when traefik_serve returned.
	stopped chan struct{}

	mu           sync.Mutex
	nextID       uint32
	nextClientID uint32
	reqs         map[uint32]*request
	fds          map[uint32]*vfd
	dead         bool
	deadErr      error
	closing      bool
}

func newInstance(ctx context.Context, cfg *engine.Config, client *http.Client, slot int) (*instance, error) {
	instCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))

	inst := &instance{
		cfg:     cfg,
		logger:  cfg.Logger,
		client:  client,
		ctx:     instCtx,
		cancel:  cancel,
		wake:    make(chan struct{}, 1),
		closed:  make(chan struct{}),
		stopped: make(chan struct{}),
		reqs:    map[uint32]*request{},
		fds:     map[uint32]*vfd{},
	}
	inst.listener = engine.NewPipe(0, inst.notify)
	inst.fds[fdListener] = newVFD(inst.listener)

	inst.rt = wazero.NewRuntimeWithConfig(ctx, cfg.RuntimeConfig())

	if err := inst.instantiate(ctx, fmt.Sprintf("%s-%d", cfg.Name, slot)); err != nil {
		cancel()
		_ = inst.rt.Close(ctx)

		return nil, err
	}

	go inst.run()

	return inst, nil
}

// Close stops the guest and releases the runtime of the instance.
func (inst *instance) Close(ctx context.Context) error {
	inst.mu.Lock()
	inst.closing = true
	inst.mu.Unlock()

	inst.closeOnce.Do(func() { close(inst.closed) })
	_ = inst.listener.Close()

	select {
	case <-inst.stopped:
	case <-ctx.Done():
	}

	inst.releaseDescriptors(engine.ErrClosing)

	return inst.rt.Close(ctx)
}

func (inst *instance) instantiate(ctx context.Context, name string) error {
	compiled, err := inst.rt.CompileModule(ctx, inst.cfg.Code)
	if err != nil {
		return fmt.Errorf("compiling guest module: %w", err)
	}

	if err = inst.instantiateWASI(ctx, inst.rt, compiled); err != nil {
		return fmt.Errorf("instantiating WASI module: %w", err)
	}

	if err = inst.instantiateHost(ctx); err != nil {
		return fmt.Errorf("instantiating %s module: %w", hostModule, err)
	}

	// The reactor entry point is called explicitly, so that a failure is reported instead of leaving a closed module.
	mod, err := inst.rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(name).WithStartFunctions())
	if err != nil {
		return fmt.Errorf("instantiating guest module: %w", err)
	}
	inst.mod = mod

	if _, err = mod.ExportedFunction(exportInitialize).Call(ctx); err != nil {
		return fmt.Errorf("calling %s: %w", exportInitialize, err)
	}

	versionFn := mod.ExportedFunction(exportVersion)
	if versionFn == nil {
		return fmt.Errorf("guest does not export %s: build it with github.com/traefik/wasm-go", exportVersion)
	}

	version, err := versionFn.Call(ctx)
	if err != nil {
		return fmt.Errorf("calling %s: %w", exportVersion, err)
	}

	if uint32(version[0]) != abiVersion {
		return fmt.Errorf("guest ABI version %d is not supported, expected %d", uint32(version[0]), abiVersion)
	}

	return nil
}

// run calls traefik_serve, which only returns when the guest stops: the instance is then dead.
func (inst *instance) run() {
	defer close(inst.stopped)

	_, err := inst.mod.ExportedFunction(exportServe).Call(context.Background())
	if err == nil {
		err = fmt.Errorf("%s returned", exportServe)
	}

	inst.mu.Lock()
	closing := inst.closing
	inst.dead = true
	inst.deadErr = err
	reqs := inst.reqs
	inst.reqs = map[uint32]*request{}
	inst.mu.Unlock()

	if !closing {
		inst.logger.Error().Err(err).Msg("Wasm instance stopped")
	}

	for _, st := range reqs {
		st.end(engine.ErrInstanceDied)
	}

	inst.releaseDescriptors(engine.ErrInstanceDied)
}

// releaseDescriptors cancels the outbound requests and closes every descriptor of a stopped instance, so that no
// host goroutine stays blocked on a pipe that the guest will never read or write again.
func (inst *instance) releaseDescriptors(cause error) {
	inst.cancel()

	inst.mu.Lock()
	fds := make([]*vfd, 0, len(inst.fds))
	for _, f := range inst.fds {
		fds = append(fds, f)
	}
	inst.mu.Unlock()

	for _, f := range fds {
		_ = f.pipe.CloseWithError(cause)
	}
}

// isDead reports whether the instance stopped, and why.
func (inst *instance) isDead() (bool, error) {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	return inst.dead, inst.deadErr
}

func (inst *instance) notify() {
	select {
	case inst.wake <- struct{}{}:
	default:
	}
}

func (inst *instance) fd(n uint32) *vfd {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	return inst.fds[n]
}

func (inst *instance) closeFD(n uint32) {
	inst.mu.Lock()
	f := inst.fds[n]
	if n != fdListener {
		delete(inst.fds, n)
	}
	inst.mu.Unlock()

	if f != nil && n != fdListener {
		f.close()
	}
}

func (inst *instance) request(id uint32) *request {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	return inst.reqs[id]
}

// output logs what the guest writes on its standard output and error.
func (inst *instance) output(fd uint32, data []byte) {
	msg := string(bytes.TrimRight(data, "\n"))
	if msg == "" {
		return
	}

	if fd == 2 {
		inst.logger.Error().Str("stream", "stderr").Msg(msg)

		return
	}

	inst.logger.Info().Str("stream", "stdout").Msg(msg)
}

// register adds a request to the instance and gives it its descriptors.
func (inst *instance) register(st *request) error {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.dead {
		return engine.ErrInstanceDied
	}

	// Ids wrap so that the descriptors of a request stay below the client ones; an id still in use is skipped.
	for {
		st.id = inst.nextID
		inst.nextID = (inst.nextID + 1) % maxRequestIDs
		if _, used := inst.reqs[st.id]; !used {
			break
		}
	}

	inst.reqs[st.id] = st
	inst.fds[requestFD(st.id, fdBodyIn)] = st.bodyIn
	inst.fds[requestFD(st.id, fdResponseOut)] = newVFD(st.respOut)
	inst.fds[requestFD(st.id, fdNextResponseIn)] = newVFD(st.nextRespIn)
	inst.fds[requestFD(st.id, fdNextBodyOut)] = newVFD(st.nextBodyOut)
	inst.fds[requestFD(st.id, fdCancel)] = newVFD(st.cancel)

	return nil
}

func (inst *instance) unregister(st *request) {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.reqs[st.id] == st {
		delete(inst.reqs, st.id)
	}
	for k := range requestFDCount {
		delete(inst.fds, requestFD(st.id, k))
	}
}

// serve hands a request to the guest and streams its response to rw.
// It returns an error, without writing anything to rw, only when the instance cannot take the request.
func (inst *instance) serve(rw http.ResponseWriter, req *http.Request, next http.Handler) error {
	st := newRequest(inst, req, next)
	if err := inst.register(st); err != nil {
		return err
	}
	defer inst.unregister(st)

	go st.watchCancel()

	var id [4]byte
	binary.LittleEndian.PutUint32(id[:], st.id)
	if _, err := inst.listener.Write(id[:]); err != nil {
		st.end(engine.ErrInstanceDied)
	}

	st.writeResponse(rw)

	<-st.done
	st.nextWG.Wait()

	return nil
}

// request is the host side of one in-flight request.
type request struct {
	id   uint32
	inst *instance
	req  *http.Request
	next http.Handler

	bodyIn      *vfd
	respOut     *engine.Pipe
	nextRespIn  *engine.Pipe
	nextBodyOut *engine.Pipe
	cancel      *engine.Pipe

	mu          sync.Mutex
	set         *requestBlock
	resp        *responseBlock
	nextStarted bool
	ended       bool
	done        chan struct{}
	nextWG      sync.WaitGroup
}

func newRequest(inst *instance, req *http.Request, next http.Handler) *request {
	st := &request{
		inst:        inst,
		req:         req,
		next:        next,
		respOut:     engine.NewPipe(pipeSize, inst.notify),
		nextRespIn:  engine.NewPipe(pipeSize, inst.notify),
		nextBodyOut: engine.NewPipe(pipeSize, inst.notify),
		cancel:      engine.NewPipe(0, inst.notify),
		done:        make(chan struct{}),
	}

	bodyIn := engine.NewPipe(pipeSize, inst.notify)
	st.bodyIn = newVFD(bodyIn)
	if req.Body == nil || req.Body == http.NoBody {
		_ = bodyIn.Close()
	} else {
		// The body is streamed to the guest only when it reads it; until then, next can get it untouched.
		st.bodyIn.onOpen = func() {
			go func() {
				_, err := io.Copy(bodyIn, req.Body)
				_ = bodyIn.CloseWithError(err)
			}()
		}
	}

	return st
}

// end completes the request on the host side: either the guest called done (err is nil) or the instance died.
func (st *request) end(err error) {
	st.mu.Lock()
	if st.ended {
		st.mu.Unlock()

		return
	}
	st.ended = true
	st.mu.Unlock()

	cause := err
	if cause == nil {
		cause = errRequestDone
	}

	_ = st.respOut.CloseWithError(err)
	_ = st.cancel.Close()
	_ = st.bodyIn.pipe.CloseWithError(cause)
	_ = st.nextRespIn.CloseWithError(cause)
	_ = st.nextBodyOut.CloseWithError(cause)

	close(st.done)
}

// watchCancel tells the guest when the client goes away.
func (st *request) watchCancel() {
	select {
	case <-st.req.Context().Done():
		_, _ = st.cancel.Write([]byte{1})
	case <-st.done:
	}
}

// writeResponse copies the response of the guest to rw: the status and the headers at the first byte of the body or
// when the guest is done, then the body as it comes.
func (st *request) writeResponse(rw http.ResponseWriter) {
	ctx := context.WithoutCancel(st.req.Context())
	headerSent := false
	sendHeader := func() {
		if headerSent {
			return
		}
		headerSent = true

		st.mu.Lock()
		resp := st.resp
		st.mu.Unlock()

		if resp == nil {
			// The instance died before the guest answered.
			http.Error(rw, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)

			return
		}

		maps.Copy(rw.Header(), resp.Header)
		rw.WriteHeader(resp.Status)
	}

	buf := make([]byte, 32*1024)
	for {
		n, err := st.respOut.Read(ctx, buf)
		if n > 0 {
			sendHeader()
			if _, werr := rw.Write(buf[:n]); werr != nil {
				// The client is gone: make the writes of the guest fail instead of blocking on a full pipe.
				_ = st.respOut.CloseWithError(werr)
			}
		}
		if err != nil {
			break
		}
	}

	<-st.done
	sendHeader()
}

// startNext runs next on its own goroutine, with the request as the guest set it; the guest reads the response on
// its next response descriptor.
func (st *request) startNext(mode uint32) {
	st.mu.Lock()
	if st.nextStarted || st.ended {
		st.mu.Unlock()
		st.inst.logger.Error().Msg("Wasm guest called next twice for the same request")

		return
	}
	st.nextStarted = true
	set := st.set
	st.mu.Unlock()

	req := st.req.Clone(st.req.Context())
	if set != nil {
		req.Method = set.Method
		if u, err := req.URL.Parse(set.Target); err == nil {
			req.URL = u
			req.RequestURI = set.Target
		}
		req.Host = set.Host
		req.Header = set.Header
	}

	switch mode {
	case bodyFromGuest:
		req.Body = io.NopCloser(pipeReader{ctx: req.Context(), pipe: st.nextBodyOut})
		req.ContentLength = -1
		if set != nil {
			req.ContentLength = set.ContentLength
		}
	case bodyNone:
		req.Body = http.NoBody
		req.ContentLength = 0
	default:
		// The guest did not read the body: next gets it as the client sent it.
	}

	st.nextWG.Go(func() {
		w := &nextWriter{pipe: st.nextRespIn, header: http.Header{}}
		defer func() {
			if r := recover(); r != nil {
				if r == http.ErrAbortHandler { //nolint:errorlint // Sentinel compared by identity, as net/http does.
					w.fail("next aborted the response")
				} else {
					st.inst.logger.Error().Interface("panic", r).Msg("Next handler panicked")
					w.fail(fmt.Sprintf("next panicked: %v", r))
				}
			}
			w.WriteHeader(http.StatusOK)
			_ = st.nextRespIn.Close()
		}()

		st.next.ServeHTTP(w, req)
	})
}

// nextWriter is the ResponseWriter given to next: the status and the headers go to the guest as a frame, the body
// right after.
type nextWriter struct {
	pipe   *engine.Pipe
	header http.Header
	sent   bool
}

func (w *nextWriter) Header() http.Header {
	return w.header
}

func (w *nextWriter) WriteHeader(code int) {
	if w.sent {
		return
	}
	w.sent = true

	block := appendResponse(responseBlock{Status: code, Header: w.header.Clone()})
	_, _ = w.pipe.Write(appendFrame(block))
}

func (w *nextWriter) Write(b []byte) (int, error) {
	w.WriteHeader(http.StatusOK)

	return w.pipe.Write(b)
}

func (w *nextWriter) fail(msg string) {
	if w.sent {
		return
	}
	w.sent = true

	_, _ = w.pipe.Write(appendFrame(appendResponse(responseBlock{Error: msg})))
}

// instantiateHost exposes the imports of the traefik module, all bound to this instance.
func (inst *instance) instantiateHost(ctx context.Context) error {
	read := func(m api.Module, ptr, size uint32) []byte {
		b, _ := m.Memory().Read(ptr, size)

		return b
	}
	// put writes block at ptr when it fits in size bytes, and returns its size in any case.
	put := func(m api.Module, ptr, size uint32, block []byte) uint32 {
		if uint32(len(block)) <= size {
			m.Memory().Write(ptr, block)
		}

		return uint32(len(block))
	}

	b := inst.rt.NewHostModuleBuilder(hostModule)

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, id, ptr, size uint32) uint32 {
		st := inst.request(id)
		if st == nil {
			return 0
		}

		return put(m, ptr, size, appendRequest(requestInfo(st.req)))
	}).Export("req_info")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, id, ptr, size uint32) {
		st := inst.request(id)
		if st == nil {
			return
		}

		set, err := parseRequest(read(m, ptr, size))
		if err != nil {
			inst.logger.Error().Err(err).Msg("Invalid request from the wasm guest")

			return
		}

		st.mu.Lock()
		st.set = &set
		st.mu.Unlock()
	}).Export("req_set")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, id, ptr, size uint32) {
		st := inst.request(id)
		if st == nil {
			return
		}

		resp, err := parseResponse(read(m, ptr, size))
		if err == nil && (resp.Status < 100 || resp.Status > 999) {
			err = fmt.Errorf("invalid status %d", resp.Status)
		}
		if err != nil {
			inst.logger.Error().Err(err).Msg("Invalid response from the wasm guest")
			resp = responseBlock{Status: http.StatusInternalServerError, Header: http.Header{}}
		}

		st.mu.Lock()
		if st.resp == nil {
			st.resp = &resp
		}
		st.mu.Unlock()
	}).Export("resp_set")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, _ api.Module, id, mode uint32) {
		if st := inst.request(id); st != nil {
			st.startNext(mode)
		}
	}).Export("next_start")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, _ api.Module, id uint32) {
		if st := inst.request(id); st != nil {
			st.end(nil)
		}
	}).Export("done")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, ptr, size uint32) uint32 {
		return put(m, ptr, size, inst.cfg.GuestConfig)
	}).Export("config_get")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, level, ptr, size uint32) {
		inst.logger.WithLevel(zerologLevel(level)).Msg(string(read(m, ptr, size)))
	}).Export("log")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, _ api.Module, level uint32) uint32 {
		lvl := zerologLevel(level)
		if inst.logger.GetLevel() <= lvl && zerolog.GlobalLevel() <= lvl {
			return 1
		}

		return 0
	}).Export("log_enabled")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, ptr, size, mode uint32) uint32 {
		return inst.startClient(bytes.Clone(read(m, ptr, size)), mode)
	}).Export("client_start")

	_, err := b.Instantiate(ctx)

	return err
}

func requestInfo(req *http.Request) requestBlock {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	// The request target of the client may be in absolute form: the guest always gets the path and the query.
	return requestBlock{
		Method:        req.Method,
		Target:        req.URL.RequestURI(),
		Host:          req.Host,
		Scheme:        scheme,
		Proto:         req.Proto,
		RemoteAddr:    req.RemoteAddr,
		ContentLength: req.ContentLength,
		Header:        req.Header,
	}
}

func zerologLevel(level uint32) zerolog.Level {
	switch level {
	case levelTrace:
		return zerolog.TraceLevel
	case levelDebug:
		return zerolog.DebugLevel
	case levelInfo:
		return zerolog.InfoLevel
	case levelWarn:
		return zerolog.WarnLevel
	default:
		return zerolog.ErrorLevel
	}
}
