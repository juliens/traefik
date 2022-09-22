package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"strings"
	"time"

	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
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

func newFastHTTPReverseProxy(client *fasthttp.Client) (http.Handler, error) {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

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
			req.Header.Set("Te", "trailers")
		}

		req.Header.Set("FastHTTP", "enabled")

		req.Header.SetHost(request.Host)
		req.SetHost(request.URL.Host)
		req.SetRequestURI(request.URL.RequestURI())

		// for k, v := range request.Header {
		// 	req.Header.Set(k, strings.Join(v, ", "))
		// }
		for k, v := range request.Header {
			for _, s := range v {
				req.Header.Add(k, s)
			}
		}

		req.SetBodyStream(request.Body, int(request.ContentLength))

		req.Header.SetMethod(request.Method)

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
				req.Header.Set("X-Forwarded-For", clientIP)
			}
		}

		res := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(res)

		err := client.Do(req, res)
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

		removeConnectionHeadersFastHTTP(res.Header)

		for _, header := range hopHeaders {
			res.Header.Del(header)
		}

		res.Header.VisitAll(func(key, value []byte) {
			// FIXME Add or Set
			writer.Header().Add(string(key), string(value))
		})

		// FIXME Trailer

		writer.WriteHeader(res.StatusCode())

		// FIXME test stream
		res.BodyWriteTo(writer)

		res.Header.VisitAllTrailer(func(key []byte) {
			writer.Header().Set(string(key), string(res.Header.Peek(string(key))))
		})

	}), nil
}

func buildProxy(responseForwarding *dynamic.ResponseForwarding, roundTripper http.RoundTripper, bufferPool httputil.BufferPool) (http.Handler, error) {
	var flushInterval ptypes.Duration
	if responseForwarding != nil {
		err := flushInterval.Set(responseForwarding.FlushInterval)
		if err != nil {
			return nil, fmt.Errorf("error creating flush interval: %w", err)
		}
	}
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

	return proxy, nil
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
