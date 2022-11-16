package fasthttp

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/traefik/traefik/v2/pkg/log"
	proxyhttputil "github.com/traefik/traefik/v2/pkg/proxy/httputil"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/http/httpguts"
)

const bufferSize = 32 * 1024

const bufioSize = 64 * 1024

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

type pool[T any] struct {
	pool sync.Pool
}

func (p *pool[T]) Get() T {
	if tmp := p.pool.Get(); tmp != nil {
		return tmp.(T)
	}

	var res T
	return res
}

func (p *pool[T]) Put(x T) {
	p.pool.Put(x)
}

type buffConn struct {
	*bufio.Reader
	net.Conn
}

func (b buffConn) Read(p []byte) (int, error) {
	return b.Reader.Read(p)
}

type writeDetector struct {
	net.Conn

	written bool
}

func (w *writeDetector) Write(p []byte) (int, error) {
	n, err := w.Conn.Write(p)
	if n > 0 {
		w.written = true
	}
	return n, err
}

type writeFlusher struct {
	io.Writer
}

func (w *writeFlusher) Write(b []byte) (int, error) {
	n, err := w.Writer.Write(b)
	if f, ok := w.Writer.(http.Flusher); ok {
		f.Flush()
	}
	return n, err
}

type timeoutError struct {
	error
}

func (t timeoutError) Timeout() bool {
	return true
}

func (t timeoutError) Temporary() bool {
	return false
}

// ReverseProxy is the FastHTTP reverse proxy implementation.
type ReverseProxy struct {
	connPool *ConnPool

	bufferPool      pool[[]byte]
	readerPool      pool[*bufio.Reader]
	writerPool      pool[*bufio.Writer]
	limitReaderPool pool[*io.LimitedReader]

	director  func(req *http.Request)
	proxyAuth string

	responseHeaderTimeout time.Duration
}

// NewReverseProxy creates a new ReverseProxy.
func NewReverseProxy(targetURL *url.URL, proxyURL *url.URL, passHostHeader bool, responseHeaderTimeout time.Duration, connPool *ConnPool) (*ReverseProxy, error) {
	var proxyAuth string
	if proxyURL != nil && proxyURL.User != nil && proxyURL.Scheme != "socks5" && targetURL.Scheme == "http" {
		username := proxyURL.User.Username()
		password, _ := proxyURL.User.Password()
		proxyAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	}

	return &ReverseProxy{
		director:              proxyhttputil.DirectorBuilder(targetURL, passHostHeader),
		proxyAuth:             proxyAuth,
		connPool:              connPool,
		responseHeaderTimeout: responseHeaderTimeout,
	}, nil
}

func (p *ReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Cloning the request also clones the Header and URL to avoid mutating
	// caller (i.e. middlewares) ones.
	req = req.Clone(req.Context())
	if req.Body != nil {
		defer req.Body.Close()
	}

	p.director(req)

	announcedTrailer := httpguts.HeaderValuesContainsToken(req.Header["Te"], "trailers")

	reqUpType := upgradeType(req.Header)
	if !isPrint(reqUpType) {
		proxyhttputil.ErrorHandler(rw, req, fmt.Errorf("client tried to switch to invalid protocol %q", reqUpType))
		return
	}

	removeConnectionHeaders(req.Header)

	for _, header := range hopHeaders {
		delete(req.Header, header)
	}

	outReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(outReq)

	// This is not required as the headers are already normalized by net/http.
	outReq.Header.DisableNormalizing()

	if p.proxyAuth != "" {
		outReq.Header.Set("Proxy-Authorization", p.proxyAuth)
	}

	if announcedTrailer {
		outReq.Header.Set("Te", "trailers")
	}

	// TODO: removes after the beta, this allows to identity that the stack used to forward requests is the FastHTTP one.
	outReq.Header.Set("FastHTTP", "enabled")

	if reqUpType != "" {
		outReq.Header.Set("Connection", "Upgrade")
		outReq.Header.Set("Upgrade", reqUpType)
	}

	outReq.SetHost(req.URL.Host)

	outReq.UseHostHeader = true
	outReq.Header.SetHost(req.Host)

	outReq.SetRequestURI(req.URL.RequestURI())

	for k, v := range req.Header {
		for _, s := range v {
			outReq.Header.Add(k, s)
		}
	}

	outReq.SetBodyStream(req.Body, int(req.ContentLength))

	outReq.Header.SetMethod(req.Method)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		prior, ok := req.Header["X-Forwarded-For"]
		omit := ok && prior == nil // Go Issue 38079: nil now means don't populate the header
		if len(prior) > 0 {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		if !omit {
			outReq.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	if err := p.roundTrip(rw, req, outReq, reqUpType); err != nil {
		proxyhttputil.ErrorHandler(rw, req, err)
	}
}

// Note that unlike the net/http RoundTrip:
//   - we are not supporting "100 Continue" response to forward them as-is to the client.
//   - we are not asking for compressed response automatically. That is because this will add an extra cost when the
//     client is asking for an uncompressed response, as we will have to un-compress it, and nowadays most clients are
//     already asking for compressed response (allowing "passthrough" compression).
func (p *ReverseProxy) roundTrip(rw http.ResponseWriter, req *http.Request, outReq *fasthttp.Request, reqUpType string) error {
	ctx := req.Context()
	trace := httptrace.ContextClientTrace(ctx)

	var co net.Conn
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
		}

		var err error
		co, err = p.connPool.AcquireConn()
		if err != nil {
			return fmt.Errorf("acquire conn: %w", err)
		}

		wd := &writeDetector{Conn: co}

		err = p.writeRequest(wd, outReq)
		if wd.written && trace != nil && trace.WroteRequest != nil {
			// WroteRequest hook is used by the tracing middleware to detect if the request has been written.
			trace.WroteRequest(httptrace.WroteRequestInfo{})
		}
		if err == nil {
			break
		}

		log.FromContext(ctx).Debugf("Error while writing request: %s", err)

		co.Close()

		if wd.written && !isReplayable(req) {
			return err
		}
	}

	br := p.readerPool.Get()
	if br == nil {
		br = bufio.NewReaderSize(co, bufioSize)
	}
	defer p.readerPool.Put(br)

	br.Reset(co)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	res.Header.SetNoDefaultContentType(true)

	var responseHeaderTimer <-chan time.Time
	if p.responseHeaderTimeout > 0 {
		timer := time.NewTimer(p.responseHeaderTimeout)
		defer timer.Stop()
		responseHeaderTimer = timer.C
	}

	errCh := make(chan error, 1)
	go func() {
		if err := res.Header.Read(br); err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		co.Close()
		return ctx.Err()
	case err := <-errCh:
		if err != nil {
			co.Close()
			return err
		}
	case <-responseHeaderTimer:
		co.Close()
		return timeoutError{errors.New("timeout awaiting response headers")}
	}

	fixPragmaCacheControl(&res.Header)

	announcedTrailers := res.Header.Peek("Trailer")
	announcedTrailersKey := strings.Split(string(announcedTrailers), ",")

	// Deal with 101 Switching Protocols responses: (WebSocket, h2c, etc)
	if res.StatusCode() == http.StatusSwitchingProtocols {
		// As the connection has been hijacked, it cannot be added back to the pool.
		handleUpgradeResponse(rw, req, reqUpType, res, buffConn{Conn: co, Reader: br})
		return nil
	}

	removeConnectionHeadersFastHTTP(&res.Header)

	for _, header := range hopHeaders {
		res.Header.Del(header)
	}

	if len(announcedTrailers) > 0 {
		res.Header.Add("Trailer", string(announcedTrailers))
	}

	res.Header.VisitAll(func(key, value []byte) {
		rw.Header().Add(string(key), string(value))
	})

	rw.WriteHeader(res.StatusCode())

	// Chunked response, Content-Length is set to -1 by FastHTTP when "Transfer-Encoding: chunked" header is received.
	if res.Header.ContentLength() == -1 {
		cbr := httputil.NewChunkedReader(br)

		b := p.bufferPool.Get()
		if b == nil {
			b = make([]byte, bufferSize)
		}
		defer p.bufferPool.Put(b)

		if _, err := io.CopyBuffer(&writeFlusher{rw}, cbr, b); err != nil {
			co.Close()
			return err
		}

		res.Header.Reset()
		res.Header.SetNoDefaultContentType(true)
		if err := res.Header.ReadTrailer(br); err != nil {
			co.Close()
			return err
		}

		res.Header.VisitAll(func(key, value []byte) {
			for _, s := range announcedTrailersKey {
				if strings.EqualFold(s, strings.TrimSpace(string(key))) {
					rw.Header().Add(string(key), string(value))
					return
				}
			}
			rw.Header().Add(http.TrailerPrefix+string(key), string(value))
		})
	} else {
		brl := p.limitReaderPool.Get()
		if brl == nil {
			brl = &io.LimitedReader{}
		}
		defer p.limitReaderPool.Put(brl)

		brl.R = br
		brl.N = int64(res.Header.ContentLength())

		b := p.bufferPool.Get()
		if b == nil {
			b = make([]byte, bufferSize)
		}
		defer p.bufferPool.Put(b)

		if _, err := io.CopyBuffer(rw, brl, b); err != nil {
			co.Close()
			return err
		}
	}

	p.connPool.ReleaseConn(co)
	return nil
}

func (p *ReverseProxy) writeRequest(co net.Conn, outReq *fasthttp.Request) error {
	bw := p.writerPool.Get()
	if bw == nil {
		bw = bufio.NewWriterSize(co, bufioSize)
	}
	defer p.writerPool.Put(bw)

	bw.Reset(co)

	if err := outReq.Write(bw); err != nil {
		return err
	}
	return bw.Flush()
}

// isReplayable returns whether the request is replayable.
func isReplayable(req *http.Request) bool {
	if req.Body == nil || req.Body == http.NoBody {
		switch req.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			return true
		}
		// The Idempotency-Key, while non-standard, is widely used to
		// mean a POST or other request is idempotent. See
		// https://golang.org/issue/19943#issuecomment-421092421
		if _, ok := req.Header["Idempotency-Key"]; ok {
			return true
		}
		if _, ok := req.Header["X-Idempotency-Key"]; ok {
			return true
		}
	}
	return false
}

// isPrint returns whether s is ASCII and printable according to
// https://tools.ietf.org/html/rfc20#section-4.2.
func isPrint(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > '~' {
			return false
		}
	}
	return true
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1.
func removeConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = textproto.TrimString(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1.
func removeConnectionHeadersFastHTTP(h *fasthttp.ResponseHeader) {
	f := h.Peek(fasthttp.HeaderConnection)
	for _, sf := range bytes.Split(f, []byte{','}) {
		if sf = bytes.TrimSpace(sf); len(sf) > 0 {
			h.DelBytes(sf)
		}
	}
}

// RFC 7234, section 5.4: Should treat Pragma: no-cache like Cache-Control: no-cache
func fixPragmaCacheControl(header *fasthttp.ResponseHeader) {
	if pragma := header.Peek("Pragma"); bytes.Equal(pragma, []byte("no-cache")) {
		if len(header.Peek("Cache-Control")) == 0 {
			header.Set("Cache-Control", "no-cache")
		}
	}
}
