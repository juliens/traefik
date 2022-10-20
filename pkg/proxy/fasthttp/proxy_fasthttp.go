package fasthttp

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"

	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/proxy/httputil"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/http/httpguts"
)

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
	connectionPool ConnectionPool

	readerPool      Pool[*bufio.Reader]
	readerSyncPool  sync.Pool
	writerPool      Pool[*bufio.Writer]
	bufferPool      sync.Pool
	limitReaderPool Pool[*io.LimitedReader]

	director func(req *http.Request)
}

func NewFastHTTPReverseProxy(target *url.URL, passHostHeader bool, connectionPool ConnectionPool) http.Handler {

	return &FastHTTPReverseProxy{
		director: httputil.DirectorBuilder(target, passHostHeader),

		bufferPool: sync.Pool{
			New: func() any {
				return make([]byte, 32*1024)
			},
		},
		readerSyncPool: sync.Pool{
			New: func() any {
				return bufio.NewReaderSize(nil, 64*1024)
			},
		},

		connectionPool: connectionPool,
	}
}

func (r *FastHTTPReverseProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	r.director(request)

	outReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(outReq)

	outReq.Header.DisableNormalizing()
	//
	outReq.URI().DisablePathNormalizing = true
	//
	if request.Body != nil {
		defer request.Body.Close()
	}

	// FIXME try to handle websocket

	announcedTrailer := httpguts.HeaderValuesContainsToken(request.Header["Te"], "trailers")

	removeConnectionHeaders(request.Header)
	//
	for _, header := range hopHeaders {
		delete(request.Header, header)
	}

	if announcedTrailer {
		outReq.Header.Set("Te", "trailers")
	}

	outReq.Header.Set("FastHTTP", "enabled")

	// SetRequestURI must be called before outReq.SetHost because it re-triggers uri parsing.
	outReq.SetRequestURI(request.URL.RequestURI())
	//
	outReq.SetHost(request.URL.Host)
	outReq.Header.SetHost(request.Host)
	//
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

	conn, err := r.connectionPool.AcquireConn()
	if err != nil {
		httputil.ErrorHandler(writer, request, err)
		return
	}

	bw := r.writerPool.Get()
	if bw == nil {
		bw = bufio.NewWriterSize(conn, 64*1024)
	}

	// FIXME retry on broken idle connection
	bw.Reset(conn)
	err = outReq.Write(bw)
	bw.Flush()
	r.writerPool.Put(bw)

	if err != nil {
		httputil.ErrorHandler(writer, request, err)
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

	// h := writer.Header()
	hijackedConn, hbr, err := writer.(http.Hijacker).Hijack()
	if err != nil {
		log.Fatal(err)
	}

	type connRe interface {
		net.Conn
		Release()
	}

	connReal := hijackedConn.(connRe)

	defer connReal.Release()
	res.Header.Write(hbr.Writer)
	hbr.Flush()
	// res.Header.Write(hbr.Writer)
	// res.Header.VisitAll(func(key, value []byte) {
	// 	writer.Header().Add(string(key), string(value))
	// 	// h[string(key)] = append(h[string(key)], string(value))
	// })

	// writer.WriteHeader(res.StatusCode())

	if res.Header.ContentLength() == -1 {
		// READ CHUNK BODY
		cbr := NewChunkedReader(br)

		b := r.bufferPool.Get()
		_, err := io.CopyBuffer(&WriteFlusher{writer}, cbr, b.([]byte))
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
		_, err := io.CopyBuffer(hbr, brl, b.([]byte))
		if err != nil {
			if err != nil {
				conn.Close()
				return
			}
		}

		r.bufferPool.Put(b)

		r.limitReaderPool.Put(brl)
	}
	hbr.Flush()

	r.readerPool.Put(br)
	r.connectionPool.ReleaseConn(conn)
}

type WriteFlusher struct {
	w io.Writer
}

func (w *WriteFlusher) Write(b []byte) (int, error) {
	n, err := w.w.Write(b)
	w.w.(http.Flusher).Flush()
	return n, err
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
	for _, sf := range bytes.Split(f, []byte{','}) {
		if sf = bytes.TrimSpace(sf); len(sf) > 0 {
			h.DelBytes(sf)
		}
	}
}
