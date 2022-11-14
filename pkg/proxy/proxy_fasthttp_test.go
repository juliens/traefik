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
	"github.com/traefik/traefik/v2/pkg/proxy/httputil"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
)

// FIXME: refactor
func buildFastHTTPProxy(t *testing.T, u *url.URL) http.Handler {
	t.Helper()

	f, err := fasthttp.NewReverseProxy(u, nil, true, fasthttp.NewConnPool(200, 0, func() (net.Conn, error) {
		return net.Dial("tcp", u.Host)
	}))
	require.NoError(t, err)

	return f
}

func buildHTTPProxy(t *testing.T, u *url.URL) http.Handler {
	t.Helper()
	
	f, err := httputil.NewProxyBuilder().Build("default", &dynamic.HTTPClientConfig{PassHostHeader: true}, nil, u)
	require.NoError(t, err)

	return f
}

func TestPassHostHeader(t *testing.T) {
	passHostHeader(t, buildFastHTTPProxy)
	passHostHeader(t, buildHTTPProxy)
}

func TestEscapedPath(t *testing.T) {
	escapedPath(t, buildFastHTTPProxy)
	escapedPath(t, buildHTTPProxy)
}

func passHostHeader(t *testing.T, buildProxy func(*testing.T, *url.URL) http.Handler) {
	t.Helper()

	var gotHostHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		gotHostHeader = req.Host
	}))

	u := testhelpers.MustParseURL(backend.URL)
	h := buildProxy(t, u)
	proxy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		h.ServeHTTP(rw, req)
	}))

	_, err := http.Get(proxy.URL)
	require.NoError(t, err)

	target := testhelpers.MustParseURL(proxy.URL)
	assert.Equal(t, target.Host, gotHostHeader)
}

func escapedPath(t *testing.T, builder func(*testing.T, *url.URL) http.Handler) {
	t.Helper()

	var gotEscapedPath string
	backend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		gotEscapedPath = req.URL.EscapedPath()
	}))

	u := testhelpers.MustParseURL(backend.URL)
	h := builder(t, u)
	proxy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		h.ServeHTTP(rw, req)
	}))

	escapedPath := "/%3A%2F%2F"
	_, err := http.Get(proxy.URL + escapedPath)
	require.NoError(t, err)

	assert.Equal(t, escapedPath, gotEscapedPath)
}
