package httpwasm

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"sort"
	"strings"

	handlerapi "github.com/http-wasm/http-wasm-host-go/api/handler"
)

// host implements handlerapi.Host on net/http requests, reading the request state from the call context.
type host struct{}

var _ handlerapi.Host = host{}

// EnableFeatures implements the same method as documented on handler.Host.
func (host) EnableFeatures(ctx context.Context, features handlerapi.Features) handlerapi.Features {
	if s, ok := ctx.Value(requestStateKey{}).(*requestState); ok {
		s.enableFeatures(features)
	}
	// Otherwise, this was called during init, but there's nothing to do because net/http supports all features.
	return features
}

// GetMethod implements the same method as documented on handler.Host.
func (host) GetMethod(ctx context.Context) string {
	return requestStateFromContext(ctx).r.Method
}

// SetMethod implements the same method as documented on handler.Host.
func (host) SetMethod(ctx context.Context, method string) {
	requestStateFromContext(ctx).r.Method = method
}

// GetURI implements the same method as documented on handler.Host.
func (host) GetURI(ctx context.Context) string {
	u := requestStateFromContext(ctx).r.URL
	result := u.EscapedPath()
	if result == "" {
		result = "/"
	}
	if u.ForceQuery || u.RawQuery != "" {
		result += "?" + u.RawQuery
	}

	return result
}

// SetURI implements the same method as documented on handler.Host.
func (host) SetURI(ctx context.Context, uri string) {
	r := requestStateFromContext(ctx).r
	if uri == "" { // url.ParseRequestURI fails on empty
		r.RequestURI = "/"
		r.URL.RawPath = "/"
		r.URL.Path = "/"
		r.URL.ForceQuery = false
		r.URL.RawQuery = ""

		return
	}
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		panic(err)
	}
	r.RequestURI = uri
	r.URL.RawPath = u.RawPath
	r.URL.Path = u.Path
	r.URL.ForceQuery = u.ForceQuery
	r.URL.RawQuery = u.RawQuery
}

// GetProtocolVersion implements the same method as documented on handler.Host.
func (host) GetProtocolVersion(ctx context.Context) string {
	return requestStateFromContext(ctx).r.Proto
}

// GetRequestHeaderNames implements the same method as documented on handler.Host.
func (host) GetRequestHeaderNames(ctx context.Context) []string {
	r := requestStateFromContext(ctx).r

	names := make([]string, 0, len(r.Header)+1)
	if r.Host != "" { // special-case the host header.
		names = append(names, "Host")
	}

	for n := range r.Header {
		if strings.HasPrefix(n, http.TrailerPrefix) {
			continue
		}
		names = append(names, n)
	}

	if len(names) == 0 { // E.g. only trailers
		return nil
	}

	// Keys in a Go map don't have consistent ordering.
	sort.Strings(names)

	return names
}

// GetRequestHeaderValues implements the same method as documented on handler.Host.
func (host) GetRequestHeaderValues(ctx context.Context, name string) []string {
	r := requestStateFromContext(ctx).r
	if textproto.CanonicalMIMEHeaderKey(name) == "Host" { // special-case the host header.
		return []string{r.Host}
	}

	return r.Header.Values(name)
}

// SetRequestHeaderValue implements the same method as documented on handler.Host.
func (host) SetRequestHeaderValue(ctx context.Context, name, value string) {
	requestStateFromContext(ctx).r.Header.Set(name, value)
}

// AddRequestHeaderValue implements the same method as documented on handler.Host.
func (host) AddRequestHeaderValue(ctx context.Context, name, value string) {
	requestStateFromContext(ctx).r.Header.Add(name, value)
}

// RemoveRequestHeader implements the same method as documented on handler.Host.
func (host) RemoveRequestHeader(ctx context.Context, name string) {
	requestStateFromContext(ctx).r.Header.Del(name)
}

// RequestBodyReader implements the same method as documented on handler.Host.
func (host) RequestBodyReader(ctx context.Context) io.ReadCloser {
	return requestStateFromContext(ctx).r.Body
}

// RequestBodyWriter implements the same method as documented on handler.Host.
func (host) RequestBodyWriter(ctx context.Context) io.Writer {
	s := requestStateFromContext(ctx)
	var b bytes.Buffer // reset
	s.r.Body = io.NopCloser(&b)

	return &b
}

// GetRequestTrailerNames implements the same method as documented on handler.Host.
func (host) GetRequestTrailerNames(ctx context.Context) []string {
	return trailerNames(requestStateFromContext(ctx).w.Header())
}

// GetRequestTrailerValues implements the same method as documented on handler.Host.
func (host) GetRequestTrailerValues(ctx context.Context, name string) []string {
	return getTrailers(requestStateFromContext(ctx).w.Header(), name)
}

// SetRequestTrailerValue implements the same method as documented on handler.Host.
func (host) SetRequestTrailerValue(ctx context.Context, name, value string) {
	setTrailer(requestStateFromContext(ctx).w.Header(), name, value)
}

// AddRequestTrailerValue implements the same method as documented on handler.Host.
func (host) AddRequestTrailerValue(ctx context.Context, name, value string) {
	addTrailer(requestStateFromContext(ctx).w.Header(), name, value)
}

// RemoveRequestTrailer implements the same method as documented on handler.Host.
func (host) RemoveRequestTrailer(ctx context.Context, name string) {
	removeTrailer(requestStateFromContext(ctx).w.Header(), name)
}

// GetStatusCode implements the same method as documented on handler.Host.
func (host) GetStatusCode(ctx context.Context) uint32 {
	s := requestStateFromContext(ctx)
	if statusCode := s.w.(*bufferingResponseWriter).statusCode; statusCode != 0 {
		return statusCode
	}

	return http.StatusOK
}

// SetStatusCode implements the same method as documented on handler.Host.
func (host) SetStatusCode(ctx context.Context, statusCode uint32) {
	s := requestStateFromContext(ctx)
	if w, ok := s.w.(*bufferingResponseWriter); ok {
		w.statusCode = statusCode
	} else {
		s.w.WriteHeader(int(statusCode))
	}
}

// GetResponseHeaderNames implements the same method as documented on handler.Host.
func (host) GetResponseHeaderNames(ctx context.Context) []string {
	w := requestStateFromContext(ctx).w

	names := make([]string, 0, len(w.Header()))
	for n := range w.Header() {
		if strings.HasPrefix(n, http.TrailerPrefix) {
			continue
		}
		names = append(names, n)
	}

	if len(names) == 0 { // E.g. only trailers
		return nil
	}
	// Keys in a Go map don't have consistent ordering.
	sort.Strings(names)

	return names
}

// GetResponseHeaderValues implements the same method as documented on handler.Host.
func (host) GetResponseHeaderValues(ctx context.Context, name string) []string {
	return requestStateFromContext(ctx).w.Header().Values(name)
}

// SetResponseHeaderValue implements the same method as documented on handler.Host.
func (host) SetResponseHeaderValue(ctx context.Context, name, value string) {
	requestStateFromContext(ctx).w.Header().Set(name, value)
}

// AddResponseHeaderValue implements the same method as documented on handler.Host.
func (host) AddResponseHeaderValue(ctx context.Context, name, value string) {
	requestStateFromContext(ctx).w.Header().Add(name, value)
}

// RemoveResponseHeader implements the same method as documented on handler.Host.
func (host) RemoveResponseHeader(ctx context.Context, name string) {
	requestStateFromContext(ctx).w.Header().Del(name)
}

// ResponseBodyReader implements the same method as documented on handler.Host.
func (host) ResponseBodyReader(ctx context.Context) io.ReadCloser {
	body := requestStateFromContext(ctx).w.(*bufferingResponseWriter).body

	return io.NopCloser(bytes.NewReader(body))
}

// ResponseBodyWriter implements the same method as documented on handler.Host.
func (host) ResponseBodyWriter(ctx context.Context) io.Writer {
	s := requestStateFromContext(ctx)
	if w, ok := s.w.(*bufferingResponseWriter); ok {
		w.body = nil // reset

		return w
	}

	return s.w
}

// GetResponseTrailerNames implements the same method as documented on handler.Host.
func (host) GetResponseTrailerNames(ctx context.Context) []string {
	return trailerNames(requestStateFromContext(ctx).w.Header())
}

// GetResponseTrailerValues implements the same method as documented on handler.Host.
func (host) GetResponseTrailerValues(ctx context.Context, name string) []string {
	return getTrailers(requestStateFromContext(ctx).w.Header(), name)
}

// SetResponseTrailerValue implements the same method as documented on handler.Host.
func (host) SetResponseTrailerValue(ctx context.Context, name, value string) {
	setTrailer(requestStateFromContext(ctx).w.Header(), name, value)
}

// AddResponseTrailerValue implements the same method as documented on handler.Host.
func (host) AddResponseTrailerValue(ctx context.Context, name, value string) {
	addTrailer(requestStateFromContext(ctx).w.Header(), name, value)
}

// RemoveResponseTrailer implements the same method as documented on handler.Host.
func (host) RemoveResponseTrailer(ctx context.Context, name string) {
	removeTrailer(requestStateFromContext(ctx).w.Header(), name)
}

// GetSourceAddr implements the same method as documented on handler.Host.
func (host) GetSourceAddr(ctx context.Context) string {
	return requestStateFromContext(ctx).r.RemoteAddr
}

func trailerNames(header http.Header) []string {
	var names []string
	for n := range header {
		if strings.HasPrefix(n, http.TrailerPrefix) {
			names = append(names, n[len(http.TrailerPrefix):])
		}
	}
	// Keys in a Go map don't have consistent ordering.
	sort.Strings(names)

	return names
}

func getTrailers(header http.Header, name string) []string {
	return header.Values(http.TrailerPrefix + name)
}

func setTrailer(header http.Header, name, value string) {
	header.Set(http.TrailerPrefix+name, value)
}

func addTrailer(header http.Header, name, value string) {
	header.Add(http.TrailerPrefix+name, value)
}

func removeTrailer(header http.Header, name string) {
	header.Del(http.TrailerPrefix + name)
}

// bufferingRequestBody keeps what the guest read from the request body, so that the next handler gets it back.
type bufferingRequestBody struct {
	delegate io.ReadCloser
	buffer   bytes.Buffer
}

// Read buffers anything read from the delegate.
func (b *bufferingRequestBody) Read(p []byte) (int, error) {
	n, err := b.delegate.Read(p)
	if n > 0 {
		b.buffer.Write(p[:n])
	}

	return n, err
}

// Close dispatches to the delegate.
func (b *bufferingRequestBody) Close() error {
	if b.delegate != nil {
		return b.delegate.Close()
	}

	return nil
}

// bufferingResponseWriter keeps the response in memory until release, so that the guest can read and rewrite it
// after the next handler.
type bufferingResponseWriter struct {
	delegate   http.ResponseWriter
	statusCode uint32
	body       []byte
}

// Header dispatches to the delegate.
func (w *bufferingResponseWriter) Header() http.Header {
	return w.delegate.Header()
}

// Write buffers the response body.
func (w *bufferingResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)

	return len(b), nil
}

// WriteHeader buffers the status code.
func (w *bufferingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = uint32(statusCode)
}

// release sends any response data collected.
func (w *bufferingResponseWriter) release() {
	if statusCode := w.statusCode; statusCode != 0 {
		w.delegate.WriteHeader(int(statusCode))
	}
	if body := w.body; len(body) != 0 {
		_, _ = w.delegate.Write(body)
	}
}
