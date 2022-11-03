package fasthttp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"strings"
	"sync"

	proxyhttputil "github.com/traefik/traefik/v2/pkg/proxy/httputil"
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

type pool[T any] struct {
	pool sync.Pool
	New  func() T
}

func (p *pool[T]) Get() T {
	if tmp := p.pool.Get(); tmp != nil {
		return tmp.(T)
	}

	if p.New != nil {
		return p.New()
	}

	var res T
	return res
}

func (p *pool[T]) Put(x T) {
	p.pool.Put(x)
}

// ReverseProxy is the FastHTTP reverse proxy implementation.
type ReverseProxy struct {
	connPool *ConnPool

	bufferPool      sync.Pool
	readerPool      pool[*bufio.Reader]
	writerPool      pool[*bufio.Writer]
	limitReaderPool pool[*io.LimitedReader]

	director func(req *http.Request)
}

// NewReverseProxy creates a new ReverseProxy.
func NewReverseProxy(target *url.URL, passHostHeader bool, connPool *ConnPool) *ReverseProxy {
	return &ReverseProxy{
		director: proxyhttputil.DirectorBuilder(target, passHostHeader),
		connPool: connPool,
		bufferPool: sync.Pool{
			New: func() any {
				return make([]byte, 32*1024)
			},
		},
	}
}

// FIXME auto gzip like in httputil reverse proxy?
func (p *ReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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

	outReq.Header.DisableNormalizing()

	if announcedTrailer {
		outReq.Header.Set("Te", "trailers")
	}

	// FIXME remove
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
		omit := ok && prior == nil // Issue 38079: nil now means don't populate the header
		if len(prior) > 0 {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		if !omit {
			outReq.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	co, err := p.connPool.AcquireConn()
	if err != nil {
		proxyhttputil.ErrorHandler(rw, req, fmt.Errorf("acquire conn: %w", err))
		return
	}

	bw := p.writerPool.Get()
	if bw == nil {
		bw = bufio.NewWriterSize(co, 64*1024)
	}

	// FIXME retry on broken idle connection
	bw.Reset(co)
	if err = outReq.Write(bw); err != nil {
		proxyhttputil.ErrorHandler(rw, req, err)
		return
	}
	bw.Flush()
	p.writerPool.Put(bw)

	br := p.readerPool.Get()
	if br == nil {
		br = bufio.NewReaderSize(co, 64*1024)
	}

	br.Reset(co)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	if err = res.Header.Read(br); err != nil {
		co.Close()
		return
	}

	announcedTrailers := res.Header.Peek("Trailer")
	announcedTrailersKey := strings.Split(string(announcedTrailers), ",")

	// Deal with 101 Switching Protocols responses: (WebSocket, h2c, etc)
	if res.StatusCode() == http.StatusSwitchingProtocols {
		handleUpgradeResponse(rw, req, reqUpType, res, bufferedConnection{Conn: co, Reader: br})
		return
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

	if res.Header.ContentLength() == -1 {
		// READ CHUNK BODY
		cbr := httputil.NewChunkedReader(br)

		b := p.bufferPool.Get()
		_, err := io.CopyBuffer(&WriteFlusher{rw}, cbr, b.([]byte))
		if err != nil {
			co.Close()
			return
		}

		res.Header.Reset()
		res.Header.SetNoDefaultContentType(true)
		err = res.Header.ReadTrailer(br)
		if err != nil {
			co.Close()
			return
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
		p.bufferPool.Put(b)
	} else {
		brl := p.limitReaderPool.Get()
		if brl == nil {
			brl = &io.LimitedReader{}
		}
		brl.R = br
		brl.N = int64(res.Header.ContentLength())

		b := p.bufferPool.Get()
		_, err := io.CopyBuffer(rw, brl, b.([]byte))
		if err != nil {
			if err != nil {
				co.Close()
				return
			}
		}

		p.bufferPool.Put(b)

		p.limitReaderPool.Put(brl)
	}

	p.readerPool.Put(br)
	p.connPool.ReleaseConn(co)
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

type bufferedConnection struct {
	*bufio.Reader
	net.Conn
}

func (b bufferedConnection) Read(p []byte) (int, error) {
	return b.Reader.Read(p)
}
