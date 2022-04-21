package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
	"github.com/valyala/fasthttp"
)

type staticTransport struct {
	res *http.Response
}

func (t *staticTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return t.res, nil
}

func BenchmarkProxy(b *testing.B) {
	res := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	w := httptest.NewRecorder()
	req := testhelpers.MustNewRequest(http.MethodGet, "http://foo.bar/", nil)

	pool := newBufferPool()
	handler, _ := buildProxy(Bool(false), nil, &staticTransport{res}, pool)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}

func TestFastHTTPTrailer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Te", "trailers")
		rw.Header().Add("Trailer", "X-Test")
		rw.Write([]byte("test"))

		rw.Header().Add("X-Test", "Toto")
		rw.Write([]byte("test"))

	}))

	/*
		resp, err := http.Get(srv.URL)
		require.NoError(t, err)
		ioutil.ReadAll(resp.Body)
		fmt.Println(resp.Trailer)

		// */

	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	req.SetRequestURI(srv.URL)
	err := fasthttp.Do(req, resp)
	require.NoError(t, err)
	fmt.Println(resp.Body())
	resp.Header.VisitAllTrailer(func(value []byte) {
		fmt.Println("trailer", string(value))
		fmt.Println(string(resp.Header.PeekBytes(value)))
	})
}
