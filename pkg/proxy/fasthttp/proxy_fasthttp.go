package fasthttp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"

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

func upgradeType(h http.Header) string {
	if !httpguts.HeaderValuesContainsToken(h["Connection"], "Upgrade") {
		return ""
	}
	return h.Get("Upgrade")
}

func upgradeTypeFastHTTPReq(header *fasthttp.RequestHeader) string {
	if !bytes.Contains(header.Peek("Connection"), []byte("Upgrade")) {
		return ""
	}
	return string(header.Peek("Upgrade"))
}

func upgradeTypeFastHTTP(header *fasthttp.ResponseHeader) string {
	if !bytes.Contains(header.Peek("Connection"), []byte("Upgrade")) {
		return ""
	}
	return string(header.Peek("Upgrade"))
}

func (r *FastHTTPReverseProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// FIXME adds auto gzip?
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

	reqUpType := upgradeType(request.Header)
	// FIXME needs ascii.IsPrint?
	// if !ascii.IsPrint(reqUpType) {
	// 	p.getErrorHandler()(rw, req, fmt.Errorf("client tried to switch to invalid protocol %q", reqUpType))
	// 	return
	// }

	removeConnectionHeaders(request.Header)
	//
	for _, header := range hopHeaders {
		delete(request.Header, header)
	}

	if announcedTrailer {
		outReq.Header.Set("Te", "trailers")
	}

	outReq.Header.Set("FastHTTP", "enabled")

	if reqUpType != "" {
		outReq.Header.Set("Connection", "Upgrade")
		outReq.Header.Set("Upgrade", reqUpType)
	}

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

	// Deal with 101 Switching Protocols responses: (WebSocket, h2c, etc)
	if res.StatusCode() == http.StatusSwitchingProtocols {
		handleUpgradeResponse(writer, request, reqUpType, res, conn)
		return
	}

	removeConnectionHeadersFastHTTP(res.Header)

	for _, header := range hopHeaders {
		res.Header.Del(header)
	}

	if len(announcedTrailers) > 0 {
		res.Header.Add("Trailer", string(announcedTrailers))
	}

	// h := writer.Header()
	res.Header.VisitAll(func(key, value []byte) {
		writer.Header().Add(string(key), string(value))
		// h[string(key)] = append(h[string(key)], string(value))
	})

	writer.WriteHeader(res.StatusCode())

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
		_, err := io.CopyBuffer(writer, brl, b.([]byte))
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
	r.connectionPool.ReleaseConn(conn)
}

func handleUpgradeResponse(rw http.ResponseWriter, req *http.Request, reqUpType string, res *fasthttp.Response, backConn net.Conn) {
	resUpType := upgradeTypeFastHTTP(&res.Header)

	if !strings.EqualFold(reqUpType, resUpType) {
		httputil.ErrorHandler(rw, req, fmt.Errorf("backend tried to switch protocol %q when %q was requested", resUpType, reqUpType))
		return
	}

	hj, ok := rw.(http.Hijacker)
	if !ok {
		httputil.ErrorHandler(rw, req, fmt.Errorf("can't switch protocols using non-Hijacker ResponseWriter type %T", rw))
		return
	}
	backConnCloseCh := make(chan bool)
	go func() {
		// Ensure that the cancellation of a request closes the backend.
		// See issue https://golang.org/issue/35559.
		select {
		case <-req.Context().Done():
		case <-backConnCloseCh:
		}
		_ = backConn.Close()
	}()

	defer close(backConnCloseCh)

	conn, brw, err := hj.Hijack()
	if err != nil {
		httputil.ErrorHandler(rw, req, fmt.Errorf("Hijack failed on protocol switch: %v", err))
		return
	}
	defer conn.Close()

	if err := res.Header.Write(brw.Writer); err != nil {
		httputil.ErrorHandler(rw, req, fmt.Errorf("response write: %v", err))
		return
	}
	if err := brw.Flush(); err != nil {
		httputil.ErrorHandler(rw, req, fmt.Errorf("response flush: %v", err))
		return
	}
	errc := make(chan error, 1)
	spc := switchProtocolCopier{user: conn, backend: backConn}
	go spc.copyToBackend(errc)
	go spc.copyFromBackend(errc)
	<-errc
}

// switchProtocolCopier exists so goroutines proxying data back and
// forth have nice names in stacks.
type switchProtocolCopier struct {
	user, backend io.ReadWriter
}

func (c switchProtocolCopier) copyFromBackend(errc chan<- error) {
	_, err := io.Copy(c.user, c.backend)
	errc <- err
}

func (c switchProtocolCopier) copyToBackend(errc chan<- error) {
	_, err := io.Copy(c.backend, c.user)
	errc <- err
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
