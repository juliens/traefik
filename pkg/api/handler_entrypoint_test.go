package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/http/httpguts"
)

func TestHandler_EntryPoints(t *testing.T) {
	type expected struct {
		statusCode int
		nextPage   string
		jsonFile   string
	}

	testCases := []struct {
		desc     string
		path     string
		conf     static.Configuration
		expected expected
	}{
		{
			desc: "all entry points, but no config",
			path: "/api/entrypoints",
			conf: static.Configuration{API: &static.API{}, Global: &static.Global{}},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/entrypoints-empty.json",
			},
		},
		{
			desc: "all entry points",
			path: "/api/entrypoints",
			conf: static.Configuration{
				Global: &static.Global{},
				API:    &static.API{},
				EntryPoints: map[string]*static.EntryPoint{
					"web": {
						Address: ":80",
						Transport: &static.EntryPointsTransport{
							LifeCycle: &static.LifeCycle{
								RequestAcceptGraceTimeout: 1,
								GraceTimeOut:              2,
							},
							RespondingTimeouts: &static.RespondingTimeouts{
								ReadTimeout:  3,
								WriteTimeout: 4,
								IdleTimeout:  5,
							},
						},
						ProxyProtocol: &static.ProxyProtocol{
							Insecure:   true,
							TrustedIPs: []string{"192.168.1.1", "192.168.1.2"},
						},
						ForwardedHeaders: &static.ForwardedHeaders{
							Insecure:   true,
							TrustedIPs: []string{"192.168.1.3", "192.168.1.4"},
						},
					},
					"websecure": {
						Address: ":443",
						Transport: &static.EntryPointsTransport{
							LifeCycle: &static.LifeCycle{
								RequestAcceptGraceTimeout: 10,
								GraceTimeOut:              20,
							},
							RespondingTimeouts: &static.RespondingTimeouts{
								ReadTimeout:  30,
								WriteTimeout: 40,
								IdleTimeout:  50,
							},
						},
						ProxyProtocol: &static.ProxyProtocol{
							Insecure:   true,
							TrustedIPs: []string{"192.168.1.10", "192.168.1.20"},
						},
						ForwardedHeaders: &static.ForwardedHeaders{
							Insecure:   true,
							TrustedIPs: []string{"192.168.1.30", "192.168.1.40"},
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/entrypoints.json",
			},
		},
		{
			desc: "all entry points, pagination, 1 res per page, want page 2",
			path: "/api/entrypoints?page=2&per_page=1",
			conf: static.Configuration{
				Global: &static.Global{},
				API:    &static.API{},
				EntryPoints: map[string]*static.EntryPoint{
					"web1": {Address: ":81"},
					"web2": {Address: ":82"},
					"web3": {Address: ":83"},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "3",
				jsonFile:   "testdata/entrypoints-page2.json",
			},
		},
		{
			desc: "all entry points, pagination, 19 results overall, 7 res per page, want page 3",
			path: "/api/entrypoints?page=3&per_page=7",
			conf: static.Configuration{
				Global:      &static.Global{},
				API:         &static.API{},
				EntryPoints: generateEntryPoints(19),
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/entrypoints-many-lastpage.json",
			},
		},
		{
			desc: "all entry points, pagination, 5 results overall, 10 res per page, want page 2",
			path: "/api/entrypoints?page=2&per_page=10",
			conf: static.Configuration{
				Global:      &static.Global{},
				API:         &static.API{},
				EntryPoints: generateEntryPoints(5),
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			desc: "all entry points, pagination, 10 results overall, 10 res per page, want page 2",
			path: "/api/entrypoints?page=2&per_page=10",
			conf: static.Configuration{
				Global:      &static.Global{},
				API:         &static.API{},
				EntryPoints: generateEntryPoints(10),
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			desc: "one entry point by id",
			path: "/api/entrypoints/bar",
			conf: static.Configuration{
				Global: &static.Global{},
				API:    &static.API{},
				EntryPoints: map[string]*static.EntryPoint{
					"bar": {Address: ":81"},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				jsonFile:   "testdata/entrypoint-bar.json",
			},
		},
		{
			desc: "one entry point by id, that does not exist",
			path: "/api/entrypoints/foo",
			conf: static.Configuration{
				Global: &static.Global{},
				API:    &static.API{},
				EntryPoints: map[string]*static.EntryPoint{
					"bar": {Address: ":81"},
				},
			},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
		{
			desc: "one entry point by id, but no config",
			path: "/api/entrypoints/foo",
			conf: static.Configuration{API: &static.API{}, Global: &static.Global{}},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			handler := New(test.conf, &runtime.Configuration{})
			server := httptest.NewServer(handler.createRouter())

			resp, err := http.DefaultClient.Get(server.URL + test.path)
			require.NoError(t, err)

			require.Equal(t, test.expected.statusCode, resp.StatusCode)

			assert.Equal(t, test.expected.nextPage, resp.Header.Get(nextPageHeader))

			if test.expected.jsonFile == "" {
				return
			}

			assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")
			contents, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			err = resp.Body.Close()
			require.NoError(t, err)

			if *updateExpected {
				var results interface{}
				err := json.Unmarshal(contents, &results)
				require.NoError(t, err)

				newJSON, err := json.MarshalIndent(results, "", "\t")
				require.NoError(t, err)

				err = os.WriteFile(test.expected.jsonFile, newJSON, 0o644)
				require.NoError(t, err)
			}

			data, err := os.ReadFile(test.expected.jsonFile)
			require.NoError(t, err)
			assert.JSONEq(t, string(data), string(contents))
		})
	}
}

func generateEntryPoints(nb int) map[string]*static.EntryPoint {
	eps := make(map[string]*static.EntryPoint, nb)
	for i := 0; i < nb; i++ {
		eps[fmt.Sprintf("ep%2d", i)] = &static.EntryPoint{
			Address: ":" + strconv.Itoa(i),
		}
	}

	return eps
}

func NewFastHTTPReverseProxy(client *fasthttp.Client) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
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

		res := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(res)

		err := client.Do(outReq, res)
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
			writer.Header().Add(string(key), string(value))
		})

		// FIXME Trailer

		writer.WriteHeader(res.StatusCode())

		go func() {
			for range time.Tick(time.Millisecond) {
				writer.(http.Flusher).Flush()
			}
		}()

		// FIXME test stream
		res.BodyWriteTo(writer)

		res.Header.VisitAllTrailer(func(key []byte) {
			writer.Header().Set(http.TrailerPrefix+string(key), string(res.Header.Peek(string(key))))
		})
	})
}

func statusText(code int) string {
	return ""
}

func TestFastHTTPTrailer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Te", "trailers")
		// rw.Header().Add("Trailer", "X-Test")
		// rw.Header().Add("X-Test", "Toto")

		rw.Write([]byte("test"))
		rw.(http.Flusher).Flush()
		time.Sleep(time.Second)
		rw.Header().Add(http.TrailerPrefix+"X-Test", "Toto")
		rw.Write([]byte("test"))

	}))
	fmt.Println(srv.URL)

	client := &fasthttp.Client{}
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()

	req.SetRequestURI(srv.URL)
	err := client.Do(req, resp)
	require.NoError(t, err)

	resp.BodyWriteTo(io.Discard)

	// proxy := NewFastHTTPReverseProxy(&fasthttp.Client{})
	// http.ListenAndServe(":8090", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
	//
	// 	fmt.Println("Host", req.Host)
	// 	proxy.ServeHTTP(rw, req)
	// }))
	//
	// ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	// <-ctx.Done()
	// /*
	// 	resp, err := http.Get(srv.URL)
	// 	require.NoError(t, err)
	// 	ioutil.ReadAll(resp.Body)
	// 	fmt.Println(resp.Trailer)
	//
	// 	// */
	//
	// req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	// req.SetRequestURI(srv.URL)
	// err := fasthttp.Do(req, resp)
	// require.NoError(t, err)
	//
	// fmt.Println(resp.Body())
	//
	// resp.Header.VisitAllTrailer(func(key []byte) {
	// 	fmt.Println("trailer", string(key))
	// 	fmt.Println(string(resp.Header.PeekBytes(key)))
	// })
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
