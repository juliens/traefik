package service

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"strings"
	"sync"
	"time"

	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/valyala/fasthttp"

	"golang.org/x/net/http/httpguts"
)

// StatusClientClosedRequest non-standard HTTP status code for client disconnection.
const StatusClientClosedRequest = 499

// StatusClientClosedRequestText non-standard HTTP status for client disconnection.
const StatusClientClosedRequestText = "Client Closed Request"

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

type FastHTTPReverseProxy struct {
	tlsConfig          *tls.Config
	dialer             func(network, address string) (net.Conn, error)
	maxIdleConnPerHost int

	pools   map[string]*ConnectionPool
	poolsMu sync.RWMutex

	tlsPools   map[string]*ConnectionPool
	tlsPoolsMu sync.RWMutex

	readerPool      Pool[*bufio.Reader]
	writerPool      Pool[*bufio.Writer]
	bufferPool      *bufferPool
	limitReaderPool Pool[*io.LimitedReader]
}

func NewFastHTTPReverseProxy(dialer func(network, address string) (net.Conn, error), maxIdleConnPerHost int, tlsConfig *tls.Config, passHostHeader *bool) http.Handler {
	fproxy := &FastHTTPReverseProxy{
		tlsConfig:          tlsConfig,
		dialer:             dialer,
		maxIdleConnPerHost: maxIdleConnPerHost,

		bufferPool: newBufferPool(),
		pools:      map[string]*ConnectionPool{},
		tlsPools:   map[string]*ConnectionPool{},
	}

	return directorBuilder(passHostHeader, fproxy)
}

func (r *FastHTTPReverseProxy) GetPool(req *http.Request) *ConnectionPool {
	url := req.URL.Host

	port := req.URL.Port()
	if port == "" {
		url += ":80"
	}

	r.poolsMu.RLock()
	pool := r.pools[url]
	r.poolsMu.RUnlock()

	if pool != nil {
		return pool
	}

	pool = NewConnectionPool(func() (net.Conn, error) {
		return r.dialer("tcp", url)
	}, r.maxIdleConnPerHost)

	r.poolsMu.Lock()
	r.pools[url] = pool
	r.poolsMu.Unlock()
	return pool
}

func (r *FastHTTPReverseProxy) GetTLSPool(req *http.Request) *ConnectionPool {
	url := req.URL.Host

	port := req.URL.Port()
	if port == "" {
		url += ":443"
	}

	r.poolsMu.RLock()
	pool := r.pools[url]
	r.poolsMu.RUnlock()

	if pool != nil {
		return pool
	}

	pool = NewConnectionPool(func() (net.Conn, error) {
		conn, err := r.dialer("tcp", url)
		if err != nil {
			return nil, err
		}
		return tls.Client(conn, r.tlsConfig), nil
	}, r.maxIdleConnPerHost)

	r.poolsMu.Lock()
	r.pools[url] = pool
	r.poolsMu.Unlock()
	return pool
}

func (r *FastHTTPReverseProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	outReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(outReq)

	outReq.Header.DisableNormalizing()

	outReq.URI().DisablePathNormalizing = true

	if request.Body != nil {
		defer request.Body.Close()
	}

	// FIXME try to handle websocket

	announcedTrailer := httpguts.HeaderValuesContainsToken(request.Header["Te"], "trailers")

	removeConnectionHeaders(request.Header)

	for _, header := range hopHeaders {
		request.Header.Del(header)
	}

	if announcedTrailer {
		outReq.Header.Set("Te", "trailers")
	}

	outReq.Header.Set("FastHTTP", "enabled")

	// SetRequestURI must be called before outReq.SetHost because it re-triggers uri parsing.
	outReq.SetRequestURI(request.URL.RequestURI())

	outReq.SetHost(request.URL.Host)
	outReq.Header.SetHost(request.Host)

	for k, v := range request.Header {
		for _, s := range v {
			outReq.Header.Add(k, s)
		}
	}

	outReq.SetBodyStream(request.Body, int(request.ContentLength))

	outReq.Header.SetMethod(request.Method)

	if clientIP, _, err := net.SplitHostPort(request.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		prior, ok := request.Header["X-Forwarded-For"]
		omit := ok && prior == nil // Issue 38079: nil now means don't populate the header
		if len(prior) > 0 {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		if !omit {
			outReq.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	var pool *ConnectionPool
	if request.URL.Scheme == "https" {
		pool = r.GetTLSPool(request)
	} else {
		pool = r.GetPool(request)
	}

	conn, err := pool.AcquireConn()
	if err != nil {
		handleError(request.Context(), writer, err)
		return
	}

	bw := r.writerPool.Get()
	if bw == nil {
		bw = bufio.NewWriterSize(conn, 64*1024)
	}

	bw.Reset(conn)
	err = outReq.Write(bw)
	bw.Flush()
	r.writerPool.Put(bw)

	if err != nil {
		handleError(request.Context(), writer, err)
		return
	}

	br := r.readerPool.Get()
	if br == nil {
		br = bufio.NewReaderSize(conn, 64*1024)
	}

	br.Reset(conn)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	err = res.Header.Read(br)
	if err != nil {
		conn.Close()
		return
	}
	announcedTrailers := res.Header.Peek("Trailer")
	announcedTrailersKey := strings.Split(string(announcedTrailers), ",")

	removeConnectionHeadersFastHTTP(res.Header)

	for _, header := range hopHeaders {
		res.Header.Del(header)
	}

	if len(announcedTrailers) > 0 {
		res.Header.Add("Trailer", string(announcedTrailers))
	}

	res.Header.VisitAll(func(key, value []byte) {
		writer.Header().Add(string(key), string(value))
	})

	writer.WriteHeader(res.StatusCode())

	if res.Header.ContentLength() == -1 {
		// READ CHUNK BODY
		cbr := NewChunkedReader(br)

		b := r.bufferPool.Get()
		_, err := io.CopyBuffer(&WriteFlusher{writer}, cbr, b)
		if err != nil {
			conn.Close()
			return
		}
		res.Header.Reset()
		res.Header.SetNoDefaultContentType(true)
		err = res.Header.ReadTrailer(br)
		if err != nil {
			conn.Close()
			return
		}

		res.Header.VisitAll(func(key, value []byte) {
			for _, s := range announcedTrailersKey {
				if strings.EqualFold(s, strings.TrimSpace(string(key))) {
					writer.Header().Add(string(key), string(value))
					return
				}
			}
			writer.Header().Add(http.TrailerPrefix+string(key), string(value))
		})
		r.bufferPool.Put(b)

	} else {

		brl := r.limitReaderPool.Get()
		if brl == nil {
			brl = &io.LimitedReader{}
		}
		brl.R = br
		brl.N = int64(res.Header.ContentLength())

		b := r.bufferPool.Get()
		_, err := io.CopyBuffer(writer, brl, b)
		if err != nil {
			if err != nil {
				conn.Close()
				return
			}
		}

		r.bufferPool.Put(b)

		r.limitReaderPool.Put(brl)
	}

	r.readerPool.Put(br)
	pool.ReleaseConn(conn)
}

func handleError(ctx context.Context, writer http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError

	switch {
	case errors.Is(err, io.EOF):
		statusCode = http.StatusBadGateway
	case errors.Is(err, context.Canceled):
		statusCode = StatusClientClosedRequest
	default:
		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				statusCode = http.StatusGatewayTimeout
			} else {
				statusCode = http.StatusBadGateway
			}
		}
	}

	log.Debugf("'%d %s' caused by: %v", statusCode, statusText(statusCode), err)
	writer.WriteHeader(statusCode)
	_, werr := writer.Write([]byte(statusText(statusCode)))
	if werr != nil {
		log.Debugf("Error while writing status code", werr)
	}
	log.FromContext(ctx).Error(err)
}

type DebugReader struct {
	r io.Reader
}

func (d *DebugReader) Read(b []byte) (int, error) {
	n, err := d.r.Read(b)
	fmt.Println("DEBUG", n, err, string(b[:n]), "DEBUG END")
	return n, err
}

type WriteFlusher struct {
	w io.Writer
}

func (w *WriteFlusher) Write(b []byte) (int, error) {
	n, err := w.w.Write(b)
	w.w.(http.Flusher).Flush()
	return n, err
}

func buildProxy(flushInterval ptypes.Duration, roundTripper http.RoundTripper, bufferPool httputil.BufferPool, passHostHeader *bool) http.Handler {
	if flushInterval == 0 {
		flushInterval = ptypes.Duration(100 * time.Millisecond)
	}

	proxy := &httputil.ReverseProxy{
		Director:      func(outReq *http.Request) {},
		Transport:     roundTripper,
		FlushInterval: time.Duration(flushInterval),
		BufferPool:    bufferPool,
		ErrorHandler: func(w http.ResponseWriter, request *http.Request, err error) {
			statusCode := http.StatusInternalServerError

			switch {
			case errors.Is(err, io.EOF):
				statusCode = http.StatusBadGateway
			case errors.Is(err, context.Canceled):
				statusCode = StatusClientClosedRequest
			default:
				var netErr net.Error
				if errors.As(err, &netErr) {
					if netErr.Timeout() {
						statusCode = http.StatusGatewayTimeout
					} else {
						statusCode = http.StatusBadGateway
					}
				}
			}

			log.Debugf("'%d %s' caused by: %v", statusCode, statusText(statusCode), err)
			w.WriteHeader(statusCode)
			_, werr := w.Write([]byte(statusText(statusCode)))
			if werr != nil {
				log.Debugf("Error while writing status code", werr)
			}
		},
	}

	return directorBuilder(passHostHeader, proxy)
}

func statusText(statusCode int) string {
	if statusCode == StatusClientClosedRequest {
		return StatusClientClosedRequestText
	}
	return http.StatusText(statusCode)
}

func isWebSocketUpgrade(req *http.Request) bool {
	if !httpguts.HeaderValuesContainsToken(req.Header["Connection"], "Upgrade") {
		return false
	}

	return strings.EqualFold(req.Header.Get("Upgrade"), "websocket")
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
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
// See RFC 7230, section 6.1
func removeConnectionHeadersFastHTTP(h fasthttp.ResponseHeader) {
	f := h.Peek(fasthttp.HeaderConnection)
	for _, sf := range strings.Split(string(f), ",") {
		if sf = textproto.TrimString(sf); sf != "" {
			h.Del(sf)
		}
	}
}
