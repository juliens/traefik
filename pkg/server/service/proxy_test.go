package service

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
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
	handler := buildProxy(0, &staticTransport{res}, pool, Bool(true))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}

func TestFastHTTPTrailer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Te", "trailer")
		rw.Header().Add("Trailer", "X-Test")

		rw.Write([]byte("test"))
		rw.(http.Flusher).Flush()

		rw.Header().Add("X-Test", "Toto")
		rw.Header().Add(http.TrailerPrefix+"X-Nontest", "Tata")

		rw.Write([]byte("test"))

	}))

	// req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	// req.SetRequestURI(srv.URL)
	// client := fasthttp.Client{}
	// client.Do(req, resp)
	//
	// resp.Body()
	// resp.Header.VisitAllTrailer(func(key []byte) {
	// 	fmt.Println(string(key))
	// })
	//
	// return
	// proxy := buildProxy(0, http.DefaultTransport, nil, Bool(true))

	proxy := NewFastHTTPReverseProxy(Bool(true))
	//
	go func() {
		log.Fatal(http.ListenAndServe(":8090", http.HandlerFunc(func(rw http.ResponseWriter, hreq *http.Request) {
			// req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
			// req.SetRequestURI(srv.URL)
			// client := fasthttp.Client{}
			// client.Do(req, resp)
			//
			// b := resp.Body()
			// resp.Header.VisitAllTrailer(func(key []byte) {
			// 	fmt.Println(string(key))
			// })
			// rw.Write(b)

			hreq.URL, _ = url.Parse(srv.URL)
			proxy.ServeHTTP(rw, hreq)
		})))
	}()
	time.Sleep(10 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:8090")
	require.NoError(t, err)
	fmt.Println(resp.Trailer)
	io.Copy(io.Discard, resp.Body)

	fmt.Println(resp.Trailer)
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
