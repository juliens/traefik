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

	// store is the wasi:config/store of the guest.
	store map[string]string

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

	mu         sync.Mutex
	nextID     uint32
	nextCallID uint32
	reqs       map[uint32]*request
	calls      map[uint32]*call
	fds        map[uint32]*vfd
	dead       bool
	deadErr    error
	closing    bool
}

func newInstance(ctx context.Context, cfg *engine.Config, store map[string]string, client *http.Client, slot int) (*instance, error) {
	instCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))

	inst := &instance{
		cfg:     cfg,
		logger:  cfg.Logger,
		store:   store,
		client:  client,
		ctx:     instCtx,
		cancel:  cancel,
		wake:    make(chan struct{}, 1),
		closed:  make(chan struct{}),
		stopped: make(chan struct{}),
		reqs:    map[uint32]*request{},
		calls:   map[uint32]*call{},
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
	if n == fdListener {
		return
	}

	inst.mu.Lock()
	f := inst.fds[n]
	delete(inst.fds, n)
	// Closing the response of a call ends the call for the guest.
	if cid, k, ok := parseCallFD(n); ok && k == fdCallResponseIn {
		delete(inst.calls, cid)
	}
	inst.mu.Unlock()

	if f != nil {
		f.close()
	}
}

func (inst *instance) request(id uint32) *request {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	return inst.reqs[id]
}

func (inst *instance) call(cid uint32) *call {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	return inst.calls[cid]
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

	// Ids wrap so that the descriptors of a request stay below the call ones; an id still in use is skipped.
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
	st.calls.Wait()

	return nil
}

// request is the host side of one in-flight request.
type request struct {
	id   uint32
	inst *instance
	req  *http.Request
	next http.Handler

	bodyIn  *vfd
	respOut *engine.Pipe
	cancel  *engine.Pipe

	mu           sync.Mutex
	resp         *responseBlock
	respTrailers http.Header
	// pending are the calls made for this request and not finished yet; they are aborted when the request ends.
	pending map[*call]struct{}
	ended   bool
	done    chan struct{}
	calls   sync.WaitGroup
}

func newRequest(inst *instance, req *http.Request, next http.Handler) *request {
	st := &request{
		inst:    inst,
		req:     req,
		next:    next,
		respOut: engine.NewPipe(pipeSize, inst.notify),
		cancel:  engine.NewPipe(0, inst.notify),
		pending: map[*call]struct{}{},
		done:    make(chan struct{}),
	}

	bodyIn := engine.NewPipe(pipeSize, inst.notify)
	st.bodyIn = newVFD(bodyIn)
	if req.Body == nil || req.Body == http.NoBody {
		_ = bodyIn.Close()
	} else {
		// The body is streamed to the guest only when it reads it; until then, a call can forward it untouched.
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
	pending := make([]*call, 0, len(st.pending))
	for c := range st.pending {
		pending = append(pending, c)
	}
	st.mu.Unlock()

	cause := err
	if cause == nil {
		cause = errRequestDone
	}

	_ = st.respOut.CloseWithError(err)
	_ = st.cancel.Close()
	_ = st.bodyIn.pipe.CloseWithError(cause)

	for _, c := range pending {
		c.abort(cause)
	}

	close(st.done)
}

// track adds a call to the request; it reports false when the request already ended.
func (st *request) track(c *call) bool {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.ended {
		return false
	}
	st.pending[c] = struct{}{}
	st.calls.Add(1)

	return true
}

func (st *request) untrack(c *call) {
	st.mu.Lock()
	delete(st.pending, c)
	st.mu.Unlock()

	st.calls.Done()
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
// when the guest is done, then the body as it comes, flushed whenever the guest has nothing more for now, then the
// trailers.
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

	rc := http.NewResponseController(rw)
	buf := make([]byte, 32*1024)
	for {
		n, err := st.respOut.Read(ctx, buf)
		if n > 0 {
			sendHeader()
			if _, werr := rw.Write(buf[:n]); werr != nil {
				// The client is gone: make the writes of the guest fail instead of blocking on a full pipe.
				_ = st.respOut.CloseWithError(werr)
			} else if st.respOut.Buffered() == 0 {
				_ = rc.Flush()
			}
		}
		if err != nil {
			break
		}
	}

	<-st.done
	sendHeader()

	st.mu.Lock()
	for k, v := range st.respTrailers {
		rw.Header()[http.TrailerPrefix+k] = v
	}
	st.mu.Unlock()
}

// instantiateHost exposes the imports of the traefik module, all bound to this instance.
func (inst *instance) instantiateHost(ctx context.Context) error {
	read := func(m api.Module, ptr, size uint32) []byte {
		b, _ := m.Memory().Read(ptr, size)

		return bytes.Clone(b)
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

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, id, ptr, size uint32) uint32 {
		st := inst.request(id)
		if st == nil {
			return 0
		}

		// net/http fills the trailers when the body ends, which the guest waits for before asking.
		return put(m, ptr, size, appendFields(st.req.Trailer))
	}).Export("req_trailers")

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

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, id, ptr, size uint32) {
		st := inst.request(id)
		if st == nil {
			return
		}

		trailers := parseHeader(string(read(m, ptr, size)))

		st.mu.Lock()
		st.respTrailers = trailers
		st.mu.Unlock()
	}).Export("resp_trailers")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, _ api.Module, id uint32) {
		if st := inst.request(id); st != nil {
			st.end(nil)
		}
	}).Export("done")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, target, id, ptr, size, mode uint32) uint32 {
		return inst.startCall(target, id, read(m, ptr, size), mode)
	}).Export("call")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, cid, ptr, size uint32) {
		c := inst.call(cid)
		if c == nil {
			return
		}

		trailers := parseHeader(string(read(m, ptr, size)))

		c.mu.Lock()
		c.reqTrailers = trailers
		c.mu.Unlock()
	}).Export("call_set_trailers")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, cid, ptr, size uint32) uint32 {
		c := inst.call(cid)
		if c == nil {
			return 0
		}

		c.mu.Lock()
		defer c.mu.Unlock()

		return put(m, ptr, size, appendFields(c.respTrailers))
	}).Export("call_trailers")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, kptr, klen, ptr, size uint32) int32 {
		v, ok := inst.store[string(read(m, kptr, klen))]
		if !ok {
			return -1
		}

		return int32(put(m, ptr, size, []byte(v)))
	}).Export("config_get")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, ptr, size uint32) uint32 {
		return put(m, ptr, size, appendConfig(inst.store))
	}).Export("config_get_all")

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

	_, err := b.Instantiate(ctx)

	return err
}

func requestInfo(req *http.Request) requestBlock {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	// net/http moves the Trailer header to Request.Trailer: the guest gets it back, as it announces the trailers.
	header := req.Header
	if len(req.Trailer) > 0 {
		header = header.Clone()
		for k := range req.Trailer {
			header.Add("Trailer", k)
		}
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
		Header:        header,
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
