package service

import (
	"bufio"
	"bytes"
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
	handler := buildProxy(0, &staticTransport{res}, pool, Bool(true))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}

func TestFastHTTPTrailer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Transfer-Encoding", "chunked")
		rw.Header().Add("Trailer", "X-Test")
		rw.Header().Add("Te", "trailers")

		rw.Write([]byte("one"))
		rw.(http.Flusher).Flush()

		time.Sleep(time.Second)
		rw.Header().Add("X-Test", "Toto")
		rw.Header().Add(http.TrailerPrefix+"X-Nontest", "Tata")
		rw.(http.Flusher).Flush()

		time.Sleep(time.Second)

		rw.Write([]byte("two"))
		rw.(http.Flusher).Flush()
		time.Sleep(time.Second)

		rw.Write([]byte("three"))
		rw.(http.Flusher).Flush()
	}))

	fmt.Println(srv.URL)
	time.Sleep(time.Minute)
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
		log.Fatal(http.ListenAndServe(":8091", http.HandlerFunc(func(rw http.ResponseWriter, hreq *http.Request) {
			hreq.URL, _ = url.Parse(srv.URL)
			proxy.ServeHTTP(rw, hreq)
		})))
	}()
	time.Sleep(10 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:8091")
	require.NoError(t, err)
	b := make([]byte, 1024)
	n, err := resp.Body.Read(b)
	fmt.Println(n, err, string(b[:n]))
	b = make([]byte, 1024)
	n, err = resp.Body.Read(b)
	fmt.Println(n, err, string(b[:n]))
	n, err = resp.Body.Read(b)
	fmt.Println(n, err, string(b[:n]))
	n, err = resp.Body.Read(b)
	fmt.Println(n, err, string(b[:n]))

	for k, v := range resp.Trailer {
		fmt.Println(k, v)
	}
	for k, v := range resp.Header {
		fmt.Println(k, v)
	}
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

func BenchmarkRequest(b *testing.B) {
	// req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	// if err != nil {
	// 	b.Fatalf("ERR")
	// }
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(http.MethodGet)
	req.SetRequestURI("http://localhost")

	bw := bufio.NewWriter(io.Discard)

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req.Write(bw)
	}
}

func BenchmarkResponse(b *testing.B) {
	content := "HTTP/1.1 200 OK\nConnection: keep-alive\nContent-Type: text/plain\nDate: Thu, 29 Sep 2022 08:25:34 GMT\nContent-Length: 1\n\n1\n"
	// content := "HTTP/1.1 200 OK\nTe: trailers\nTrailer: X-Test\nDate: Fri, 30 Sep 2022 09:10:58 GMT\nTransfer-Encoding: chunked\n\n3\none\n3\ntwo\n5\nthree\n0\nX-Nontest: Tata\nX-Test: Toto\n\n"
	b.ReportAllocs()

	resp := fasthttp.AcquireResponse()
	// resp.SkipBody = true

	br := bufio.NewReader(bytes.NewReader([]byte(content)))
	// var resp *http.Response

	// toto := func() {
	// 	resp.Header.Read(br)
	// }

	for i := 0; i < b.N; i++ {
		br.Reset(bytes.NewReader([]byte(content)))

		// resp, _ = http.ReadResponse(br, nil)
		// io.ReadAll(resp.Body)

		resp.Header.Read(br)
		brl := NewChunkedReader(br)
		io.ReadAll(brl)
		// resp.Read(br)
		resp.Reset()
		// resp.ResetBody()
		// resp.ReadLimitBody(br, 1000)
	}

	// fmt.Println(resp)
}
