package wasihttp

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/traefik/traefik/v3/pkg/plugins/wasm/engine"
)

const (
	outboundDialTimeout      = 30 * time.Second
	outboundHeaderTimeout    = 30 * time.Second
	outboundMaxIdleConns     = 100
	outboundIdleConnsTimeout = 90 * time.Second
)

var errCallDone = errors.New("call done")

// newOutboundClient returns the client that sends the outbound requests of the guests of one engine.
// It follows no redirect: the guest gets the redirect response, as with a net/http.RoundTripper.
func newOutboundClient() *http.Client {
	dialer := &net.Dialer{Timeout: outboundDialTimeout, KeepAlive: 30 * time.Second}

	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          outboundMaxIdleConns,
			IdleConnTimeout:       outboundIdleConnsTimeout,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: outboundHeaderTimeout,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// call is the host side of one call of the guest to the next handler (the imported wasi:http/handler) or to the
// outbound client (the imported wasi:http/client).
type call struct {
	// respIn carries the response to the guest: a frame with the status and the headers, then the body.
	respIn *engine.Pipe
	// bodyOut carries the request body from the guest, when the guest streams it.
	bodyOut *engine.Pipe

	mu           sync.Mutex
	reqTrailers  http.Header
	respTrailers http.Header
}

// startCall registers the descriptors of a call and runs it on its own goroutine.
func (inst *instance) startCall(target, id uint32, block []byte, mode uint32) uint32 {
	c := &call{
		respIn:  engine.NewPipe(pipeSize, inst.notify),
		bodyOut: engine.NewPipe(pipeSize, inst.notify),
	}

	inst.mu.Lock()
	// Ids wrap like the request ones; an id still in use is skipped.
	var cid uint32
	for {
		cid = inst.nextCallID
		inst.nextCallID = (inst.nextCallID + 1) % maxCallIDs
		if _, used := inst.calls[cid]; !used {
			break
		}
	}
	inst.calls[cid] = c
	inst.fds[callFD(cid, fdCallResponseIn)] = newVFD(c.respIn)
	if mode == bodyFromGuest {
		inst.fds[callFD(cid, fdCallBodyOut)] = newVFD(c.bodyOut)
	}
	var st *request
	if id != noRequest {
		st = inst.reqs[id]
	}
	dead := inst.dead
	inst.mu.Unlock()

	if dead {
		c.abort(engine.ErrInstanceDied)

		return cid
	}

	if st != nil && !st.track(c) {
		c.abort(errRequestDone)

		return cid
	}

	go func() {
		if st != nil {
			defer st.untrack(c)
		}
		defer func() { _ = c.respIn.Close() }()
		// A body the guest still writes after the call goes nowhere: its writes fail instead of blocking.
		defer func() { _ = c.bodyOut.CloseWithError(errCallDone) }()

		if target == targetHandler {
			inst.runNext(st, c, block, mode)

			return
		}

		inst.runClient(st, c, block, mode)
	}()

	return cid
}

// runNext runs the next handler of the incoming request st with the request of the call.
func (inst *instance) runNext(st *request, c *call, block []byte, mode uint32) {
	if st == nil || st.next == nil {
		c.fail("next needs an incoming request")

		return
	}

	info, err := parseRequest(block)
	if err != nil {
		c.fail(fmt.Sprintf("invalid request: %v", err))

		return
	}

	// The request keeps what Traefik knows of the incoming one: context, client address, TLS.
	req := st.req.Clone(st.req.Context())
	req.Method = info.Method
	u, err := req.URL.Parse(info.Target)
	if err != nil {
		c.fail(fmt.Sprintf("invalid request target: %v", err))

		return
	}
	req.URL = u
	req.RequestURI = info.Target
	req.Host = info.Host
	req.Header = info.Header

	if err = c.setBody(req, info, st, mode); err != nil {
		c.fail(err.Error())

		return
	}

	w := &nextWriter{c: c, header: http.Header{}}
	defer func() {
		if r := recover(); r != nil {
			if r == http.ErrAbortHandler { //nolint:errorlint // Sentinel compared by identity, as net/http does.
				w.fail("next aborted the response")
			} else {
				inst.logger.Error().Interface("panic", r).Msg("Next handler panicked")
				w.fail(fmt.Sprintf("next panicked: %v", r))
			}
		}
		w.WriteHeader(http.StatusOK)
		w.finish()
	}()

	st.next.ServeHTTP(w, req)
}

// runClient sends the outbound request of the call; it is canceled when the instance stops.
func (inst *instance) runClient(st *request, c *call, block []byte, mode uint32) {
	if !inst.cfg.AllowOutbound {
		c.fail("outbound requests are not allowed: set the allowOutbound plugin setting")

		return
	}

	info, err := parseRequest(block)
	if err != nil {
		c.fail(fmt.Sprintf("invalid request: %v", err))

		return
	}

	if info.Scheme != "http" && info.Scheme != "https" {
		c.fail(fmt.Sprintf("unsupported scheme %q", info.Scheme))

		return
	}

	req, err := http.NewRequestWithContext(inst.ctx, info.Method, info.Scheme+"://"+info.Host+info.Target, http.NoBody)
	if err != nil {
		c.fail(fmt.Sprintf("invalid request: %v", err))

		return
	}

	req.Header = info.Header
	if host := info.Header.Get("Host"); host != "" {
		req.Host = host
		req.Header.Del("Host")
	}

	if err = c.setBody(req, info, st, mode); err != nil {
		c.fail(err.Error())

		return
	}

	res, err := inst.client.Do(req)
	if err != nil {
		inst.logger.Debug().Err(err).Str("url", req.URL.Redacted()).Msg("Wasm outbound request failed")
		c.fail(err.Error())

		return
	}
	defer func() { _ = res.Body.Close() }()

	frame := appendFrame(appendResponse(responseBlock{Status: res.StatusCode, Header: res.Header}))
	if _, err = c.respIn.Write(frame); err != nil {
		return
	}

	// A write error means that the guest closed the response, or that the instance stopped: closing the body aborts
	// the request.
	if _, err = io.Copy(c.respIn, res.Body); err != nil {
		return
	}

	c.mu.Lock()
	c.respTrailers = res.Trailer
	c.mu.Unlock()
}

// setBody sets the body, the content length and the trailers of the request of a call, from the body mode.
func (c *call) setBody(req *http.Request, info requestBlock, st *request, mode uint32) error {
	// The trailers announced by the guest become Request.Trailer, which net/http announces again when it sends them.
	declared := declaredTrailers(req.Header)
	req.Header.Del("Trailer")

	switch mode {
	case bodyFromGuest:
		req.Body = &trailerReader{call: c, req: req}
		req.ContentLength = info.ContentLength
		req.Trailer = declared
	case bodyIncoming:
		if st == nil {
			return errors.New("no incoming request to forward the body of")
		}
		if st.bodyIn.opened() {
			return errors.New("the incoming request body was already read by the plugin")
		}
		// net/http fills the trailers of the incoming request when its body ends: they are shared.
		req.Body = st.req.Body
		req.ContentLength = st.req.ContentLength
		req.Trailer = st.req.Trailer
	default:
		req.Body = http.NoBody
		req.ContentLength = 0
		req.Trailer = nil
	}

	return nil
}

// fail answers the call with an error, as the error-code of handle and send.
func (c *call) fail(msg string) {
	_, _ = c.respIn.Write(appendFrame(appendResponse(responseBlock{Error: msg})))
}

// abort ends the call for the guest and for the host, when the request or the instance it belongs to ends.
func (c *call) abort(cause error) {
	_ = c.respIn.CloseWithError(cause)
	_ = c.bodyOut.CloseWithError(cause)
}

func declaredTrailers(h http.Header) http.Header {
	values := h.Values("Trailer")
	if len(values) == 0 {
		return nil
	}

	trailers := http.Header{}
	for _, v := range values {
		for k := range strings.SplitSeq(v, ",") {
			if k = strings.TrimSpace(k); k != "" {
				trailers[http.CanonicalHeaderKey(k)] = nil
			}
		}
	}

	return trailers
}

// trailerReader is the body of a call request streamed by the guest: at its end, it fills the trailers the guest set,
// as net/http does for a request body.
type trailerReader struct {
	call *call
	req  *http.Request
}

func (r *trailerReader) Read(p []byte) (int, error) {
	n, err := r.call.bodyOut.Read(r.req.Context(), p)
	if errors.Is(err, io.EOF) && r.req.Trailer != nil {
		r.call.mu.Lock()
		maps.Copy(r.req.Trailer, r.call.reqTrailers)
		r.call.mu.Unlock()
	}

	return n, err
}

func (r *trailerReader) Close() error {
	return nil
}

// nextWriter is the ResponseWriter given to next: the status and the headers go to the guest as a frame, the body
// streams right after, and the trailers are kept for the guest.
type nextWriter struct {
	c        *call
	header   http.Header
	sent     bool
	declared http.Header
}

var _ http.Flusher = (*nextWriter)(nil)

func (w *nextWriter) Header() http.Header {
	return w.header
}

func (w *nextWriter) WriteHeader(code int) {
	if w.sent {
		return
	}
	w.sent = true
	w.declared = declaredTrailers(w.header)

	block := appendResponse(responseBlock{Status: code, Header: w.header.Clone()})
	_, _ = w.c.respIn.Write(appendFrame(block))
}

func (w *nextWriter) Write(b []byte) (int, error) {
	w.WriteHeader(http.StatusOK)

	return w.c.respIn.Write(b)
}

// Flush implements http.Flusher: every write already reaches the guest.
func (w *nextWriter) Flush() {
	w.WriteHeader(http.StatusOK)
}

func (w *nextWriter) fail(msg string) {
	if w.sent {
		return
	}
	w.sent = true

	w.c.fail(msg)
}

// finish keeps the trailers set by next for the guest: the announced ones, and the ones with http.TrailerPrefix.
func (w *nextWriter) finish() {
	trailers := http.Header{}
	for k := range w.declared {
		if v := w.header.Values(k); len(v) > 0 {
			trailers[k] = v
		}
	}
	for k, v := range w.header {
		if name, ok := strings.CutPrefix(k, http.TrailerPrefix); ok {
			trailers[http.CanonicalHeaderKey(name)] = v
		}
	}

	w.c.mu.Lock()
	w.c.respTrailers = trailers
	w.c.mu.Unlock()
}
