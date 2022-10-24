package proxy

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/proxy/fasthttp"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
)

func buildFastHTTPProxy(u *url.URL) http.Handler {
	return fasthttp.NewReverseProxy(u, true, fasthttp.NewConnectionPool(func() (net.Conn, error) {
		return net.Dial("tcp", u.Host)
	}, 200))
}

func buildHTTPProxy(u *url.URL) http.Handler {
	builder := fasthttp.NewProxyBuilder()
	return builder.Build("default", &dynamic.HTTPClientConfig{PassHostHeader: true}, nil, u)
}

func TestPassHostHeader(t *testing.T) {
	passHostHeader(t, buildFastHTTPProxy)
	passHostHeader(t, buildHTTPProxy)
}

func TestEscapedPath(t *testing.T) {
	escapedPath(t, buildFastHTTPProxy)
	escapedPath(t, buildHTTPProxy)
}

func passHostHeader(t *testing.T, buildProxy func(u *url.URL) http.Handler) {
	t.Helper()

	var gotHostHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		gotHostHeader = req.Host
	}))

	u := testhelpers.MustParseURL(backend.URL)
	h := buildProxy(u)
	proxy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		h.ServeHTTP(rw, req)
	}))

	_, err := http.Get(proxy.URL)
	require.NoError(t, err)

	target := testhelpers.MustParseURL(proxy.URL)
	assert.Equal(t, target.Host, gotHostHeader)
}

func escapedPath(t *testing.T, builder func(u *url.URL) http.Handler) {
	t.Helper()

	var gotEscapedPath string
	backend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		gotEscapedPath = req.URL.EscapedPath()
	}))

	u := testhelpers.MustParseURL(backend.URL)
	h := builder(u)
	proxy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		h.ServeHTTP(rw, req)
	}))

	escapedPath := "/%3A%2F%2F"
	_, err := http.Get(proxy.URL + escapedPath)
	require.NoError(t, err)

	assert.Equal(t, escapedPath, gotEscapedPath)
}
