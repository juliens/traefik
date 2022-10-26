package fasthttp

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
//}
