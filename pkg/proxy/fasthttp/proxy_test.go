package fasthttp

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/armon/go-socks5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
	"github.com/traefik/traefik/v2/pkg/tls/generate"
)

type proxyHandler struct {
	call func()
}

func newProxy(call func()) http.Handler {
	return &proxyHandler{call: call}
}

func (p *proxyHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.call()
	if req.Method == http.MethodConnect {
		conn, err := net.Dial("tcp", req.Host)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		hj, ok := rw.(http.Hijacker)
		if !ok {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
		connHj, _, err := hj.Hijack()
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		go io.Copy(connHj, conn)
		io.Copy(conn, connHj)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(testhelpers.MustParseURL("http://" + req.Host))
	proxy.ServeHTTP(rw, req)
}

// func TestFastHTTPTrailer(t *testing.T) {
//	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
//		rw.Header().Add("Te", "trailers")
//		// rw.Header().Add("Trailer", "X-Test")
//		// rw.Header().Add("X-Test", "Toto")
//
//		rw.Write([]byte("test"))
//		rw.(http.Flusher).Flush()
//		time.Sleep(time.Second)
//		rw.Header().Add(http.TrailerPrefix+"X-Test", "Toto")
//		rw.Write([]byte("test"))
//
//	}))
//	fmt.Println(srv.URL)
//
//	client := &fasthttp.Client{}
//	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
//
//	req.SetRequestURI(srv.URL)
//	err := client.Do(req, resp)
//	require.NoError(t, err)
//
//	resp.BodyWriteTo(io.Discard)
//
//	// proxy := NewReverseProxy(&fasthttp.Client{})
//	// http.ListenAndServe(":8090", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
//	//
//	// 	fmt.Println("Host", req.Host)
//	// 	proxy.ServeHTTP(rw, req)
//	// }))
//	//
//	// ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
//	// <-ctx.Done()
//	// /*
//	// 	resp, err := http.Get(srv.URL)
//	// 	require.NoError(t, err)
//	// 	ioutil.ReadAll(resp.Body)
//	// 	fmt.Println(resp.Trailer)
//	//
//	// 	// */
//	//
//	// req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
//	// req.SetRequestURI(srv.URL)
//	// err := fasthttp.Do(req, resp)
//	// require.NoError(t, err)
//	//
//	// fmt.Println(resp.Body())
//	//
//	// resp.Header.VisitAllTrailer(func(key []byte) {
//	// 	fmt.Println("trailer", string(key))
//	// 	fmt.Println(string(resp.Header.PeekBytes(key)))
//	// })
// }

const (
	proxyHTTP   = "http"
	proxyHTTPS  = "https"
	proxySocks5 = "socks"
)

func TestProxyFromEnvironment(t *testing.T) {
	testCases := []struct {
		desc      string
		tls       bool
		auth      bool
		proxyType string
	}{
		{
			desc:      "Proxy HTTP with HTTP Backend",
			proxyType: proxyHTTP,
		},
		{
			desc:      "Proxy HTTP with HTTP Backend and proxy auth",
			proxyType: proxyHTTP,
			auth:      true,
			tls:       false,
		},
		{
			desc:      "Proxy HTTP with HTTPS Backend",
			proxyType: proxyHTTP,
			tls:       true,
		},
		{
			desc:      "Proxy HTTP with HTTPS Backend with proxy auth",
			proxyType: proxyHTTP,
			auth:      true,
			tls:       true,
		},
		{
			desc:      "Proxy HTTPS with HTTP Backend",
			proxyType: proxyHTTPS,
		},
		{
			desc:      "Proxy HTTPS with HTTP Backend and proxy auth",
			proxyType: proxyHTTPS,
			auth:      true,
			tls:       false,
		},
		{
			desc:      "Proxy HTTPS with HTTPS Backend",
			proxyType: proxyHTTPS,
			tls:       true,
		},
		{
			desc:      "Proxy HTTPS with HTTPS Backend with proxy auth",
			proxyType: proxyHTTPS,
			auth:      true,
			tls:       true,
		},
		{
			desc:      "Proxy Socks5 with HTTP Backend",
			proxyType: proxySocks5,
		},
		{
			desc:      "Proxy Socks5 with HTTP Backend with auth",
			proxyType: proxySocks5,
			auth:      true,
		},

		{
			desc:      "Proxy Socks5 with HTTPS Backend",
			proxyType: proxySocks5,
			tls:       true,
		},
		{
			desc:      "Proxy Socks5 with HTTPS Backend with auth",
			proxyType: proxySocks5,
			auth:      true,
			tls:       true,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			log.SetLevel(logrus.DebugLevel)
			backendURL, backendCert := backendServer(t, test.tls, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.Write([]byte("backend"))
			}))

			var proxyfied bool
			fwdProxyHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				proxyfied = true

				if test.auth {
					require.Equal(t, "Basic dXNlcjpwYXNzd29yZA==", req.Header.Get("Proxy-Authorization"))
				}

				if req.Method == http.MethodConnect {
					conn, err := net.Dial("tcp", req.Host)
					require.NoError(t, err)

					hj, ok := rw.(http.Hijacker)
					require.True(t, ok)

					rw.WriteHeader(http.StatusOK)
					connHj, _, err := hj.Hijack()
					require.NoError(t, err)

					go io.Copy(connHj, conn)
					io.Copy(conn, connHj)
					return
				}

				proxy := httputil.NewSingleHostReverseProxy(testhelpers.MustParseURL("http://" + req.Host))
				proxy.ServeHTTP(rw, req)
			})

			var proxyURL string
			var proxyCert *x509.Certificate
			switch test.proxyType {
			case proxySocks5:
				ln, err := net.Listen("tcp", ":0")
				require.NoError(t, err)

				proxyURL = fmt.Sprintf("socks5://%s", ln.Addr())

				go func() {
					conn, err := ln.Accept()
					require.NoError(t, err)

					proxyfied = true

					conf := &socks5.Config{}
					if test.auth {
						conf.Credentials = socks5.StaticCredentials{"user": "password"}
					}

					server, err := socks5.New(conf)
					require.NoError(t, err)

					err = server.ServeConn(conn)
					require.NoError(t, err)

					err = ln.Close()
					require.NoError(t, err)
				}()
			case proxyHTTP:
				proxy := httptest.NewServer(fwdProxyHandler)
				t.Cleanup(proxy.Close)

				proxyURL = proxy.URL
			case proxyHTTPS:
				proxy := httptest.NewServer(fwdProxyHandler)
				t.Cleanup(proxy.Close)

				proxyURL = proxy.URL
				proxyCert = proxy.Certificate()
			}

			builder := NewProxyBuilder()
			builder.proxy = func(req *http.Request) (*url.URL, error) {
				u, err := url.Parse(proxyURL)
				if test.auth {
					u.User = url.UserPassword("user", "password")
				}
				return u, err
			}

			certpool := x509.NewCertPool()
			if backendCert != nil {
				cert, err := x509.ParseCertificate(backendCert.Certificate[0])
				require.NoError(t, err)

				certpool.AddCert(cert)
			}
			if proxyCert != nil {
				certpool.AddCert(proxyCert)
			}
			tlsConfig := &tls.Config{
				RootCAs: certpool,
			}

			reverseProxy, err := builder.Build("toto", &dynamic.HTTPClientConfig{}, tlsConfig, testhelpers.MustParseURL(backendURL))
			require.NoError(t, err)

			reverseProxyServer := httptest.NewServer(reverseProxy)
			t.Cleanup(reverseProxyServer.Close)

			resp, err := http.Get(reverseProxyServer.URL)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, "backend", string(body))
			assert.True(t, proxyfied)
		})
	}
}

func newCertificate(domain string) (*tls.Certificate, error) {
	certPEM, keyPEM, err := generate.KeyPair(domain, time.Time{})
	if err != nil {
		return nil, err
	}

	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &certificate, nil
}

func backendServer(t *testing.T, isTLS bool, handler http.Handler) (string, *tls.Certificate) {
	var err error
	var ln net.Listener
	var cert *tls.Certificate

	domain := "backend.localhost"
	var scheme = "http"
	if isTLS {
		cert, err = newCertificate(domain)
		require.NoError(t, err)

		ln, err = tls.Listen("tcp", ":0", &tls.Config{Certificates: []tls.Certificate{*cert}})
		require.NoError(t, err)

		scheme = "https"
	} else {
		ln, err = net.Listen("tcp", ":0")
		require.NoError(t, err)
	}

	_, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)

	backendURL := fmt.Sprintf("%s://%s:%s", scheme, domain, port)

	srv := &http.Server{Handler: handler}
	go srv.Serve(ln)

	t.Cleanup(func() { _ = srv.Close() })

	return backendURL, cert
}
