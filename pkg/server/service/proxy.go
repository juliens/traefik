package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"strconv"
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

type Pool[T any] struct {
	pool sync.Pool
}

func (p *Pool[T]) Get() T {
	var res T
	if temp := p.pool.Get(); temp != nil {
		res = temp.(T)
	}
	return res
}

func (p *Pool[T]) Put(x T) {
	p.pool.Put(x)
}

func NewFastHTTPReverseProxy(passHostHeader *bool) http.Handler {
	hc := NewHostChooser()
	var readerPool Pool[*bufio.Reader]
	var writerPool Pool[*bufio.Writer]

	bufferPool := newBufferPool()
	var limitReaderPool Pool[*io.LimitedReader]

	return directorBuilder(passHostHeader, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		outReq := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(outReq)

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

		// FIXME compare performance
		// for k, v := range request.Header {
		// 	outReq.Header.Set(k, strings.Join(v, ", "))
		// }
		for k, v := range request.Header {
			for _, s := range v {
				outReq.Header.Add(k, s)
			}
		}

		// outReq.SetBodyStream(request.Body, int(request.ContentLength))

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

		pool := hc.GetPool(string(outReq.URI().Scheme()), addMissingPort(string(outReq.URI().Host()), bytes.EqualFold(outReq.URI().Scheme(), []byte("https"))))
		conn := pool.AcquireConn()
		defer pool.ReleaseConn(conn)

		bw := writerPool.Get()
		if bw == nil {
			bw = bufio.NewWriterSize(conn, 4096)
		}

		bw.Reset(conn)
		err := outReq.Write(bw)
		bw.Flush()
		writerPool.Put(bw)

		if err != nil {
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
			log.FromContext(request.Context()).Error(err)
			return
		}

		br := readerPool.Get()
		if br == nil {
			br = bufio.NewReaderSize(conn, 4096)
		}

		br.Reset(conn)

		res := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(res)

		res.Header.Read(br)
		removeConnectionHeadersFastHTTP(res.Header)

		for _, header := range hopHeaders {
			res.Header.Del(header)
		}

		res.Header.VisitAll(func(key, value []byte) {
			writer.Header().Add(string(key), string(value))
		})

		writer.WriteHeader(res.StatusCode())

		brl := limitReaderPool.Get()
		if brl == nil {
			brl = &io.LimitedReader{}
		}

		brl.R = br
		brl.N = int64(res.Header.ContentLength())

		b := bufferPool.Get()
		io.CopyBuffer(writer, brl, b)
		bufferPool.Put(b)

		limitReaderPool.Put(brl)
		readerPool.Put(br)

	}))
}
func addMissingPort(addr string, isTLS bool) string {
	n := strings.Index(addr, ":")
	if n >= 0 {
		return addr
	}
	port := 80
	if isTLS {
		port = 443
	}
	return net.JoinHostPort(addr, strconv.Itoa(port))
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
