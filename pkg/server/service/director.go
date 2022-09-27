package service

import (
	"net/http"
	"net/url"
	"strings"
)

func directorBuilder(passHostHeader *bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		u := req.URL
		if req.RequestURI != "" {
			parsedURL, err := url.ParseRequestURI(req.RequestURI)
			if err == nil {
				u = parsedURL
			}
		}

		req.URL.Path = u.Path
		req.URL.RawPath = u.RawPath
		req.URL.RawQuery = strings.ReplaceAll(u.RawQuery, ";", "&")
		req.RequestURI = "" // Outgoing request should not have RequestURI

		req.Proto = "HTTP/1.1"
		req.ProtoMajor = 1
		req.ProtoMinor = 1

		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}

		// Do not pass client Host header unless optsetter PassHostHeader is set.
		if passHostHeader != nil && !*passHostHeader {
			req.Host = req.URL.Host
		}

		// Even if the websocket RFC says that headers should be case-insensitive,
		// some servers need Sec-WebSocket-Key, Sec-WebSocket-Extensions, Sec-WebSocket-Accept,
		// Sec-WebSocket-Protocol and Sec-WebSocket-Version to be case-sensitive.
		// https://tools.ietf.org/html/rfc6455#page-20
		if isWebSocketUpgrade(req) {
			req.Header["Sec-WebSocket-Key"] = req.Header["Sec-Websocket-Key"]
			req.Header["Sec-WebSocket-Extensions"] = req.Header["Sec-Websocket-Extensions"]
			req.Header["Sec-WebSocket-Accept"] = req.Header["Sec-Websocket-Accept"]
			req.Header["Sec-WebSocket-Protocol"] = req.Header["Sec-Websocket-Protocol"]
			req.Header["Sec-WebSocket-Version"] = req.Header["Sec-Websocket-Version"]
			delete(req.Header, "Sec-Websocket-Key")
			delete(req.Header, "Sec-Websocket-Extensions")
			delete(req.Header, "Sec-Websocket-Accept")
			delete(req.Header, "Sec-Websocket-Protocol")
			delete(req.Header, "Sec-Websocket-Version")
		}

		next.ServeHTTP(rw, req)
	})
}
