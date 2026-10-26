package httpwasm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/http-wasm/http-wasm-host-go/api"
	handlerapi "github.com/http-wasm/http-wasm-host-go/api/handler"
	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
)

const i32, i64 = wazeroapi.ValueTypeI32, wazeroapi.ValueTypeI64

// instantiateHost instantiates the http_handler host module in rt, bound to the engine.
func (e *httpWasm) instantiateHost(ctx context.Context, rt wazero.Runtime) (wazeroapi.Module, error) {
	return rt.NewHostModuleBuilder(handlerapi.HostModule).
		NewFunctionBuilder().
		WithGoFunction(wazeroapi.GoFunc(e.enableFeatures), []wazeroapi.ValueType{i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("features").Export(handlerapi.FuncEnableFeatures).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.getConfig), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(handlerapi.FuncGetConfig).
		NewFunctionBuilder().
		WithGoFunction(wazeroapi.GoFunc(e.logEnabled), []wazeroapi.ValueType{i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("level").Export(handlerapi.FuncLogEnabled).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.log), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("level", "message", "message_len").Export(handlerapi.FuncLog).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.getMethod), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(handlerapi.FuncGetMethod).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.setMethod), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("method", "method_len").Export(handlerapi.FuncSetMethod).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.getURI), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(handlerapi.FuncGetURI).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.setURI), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("uri", "uri_len").Export(handlerapi.FuncSetURI).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.getProtocolVersion), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(handlerapi.FuncGetProtocolVersion).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.getHeaderNames), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{i64}).
		WithParameterNames("kind", "buf", "buf_limit").Export(handlerapi.FuncGetHeaderNames).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.getHeaderValues), []wazeroapi.ValueType{i32, i32, i32, i32, i32}, []wazeroapi.ValueType{i64}).
		WithParameterNames("kind", "name", "name_len", "buf", "buf_limit").Export(handlerapi.FuncGetHeaderValues).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.setHeaderValue), []wazeroapi.ValueType{i32, i32, i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("kind", "name", "name_len", "value", "value_len").Export(handlerapi.FuncSetHeaderValue).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.addHeaderValue), []wazeroapi.ValueType{i32, i32, i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("kind", "name", "name_len", "value", "value_len").Export(handlerapi.FuncAddHeaderValue).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.removeHeader), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("kind", "name", "name_len").Export(handlerapi.FuncRemoveHeader).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.readBody), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{i64}).
		WithParameterNames("kind", "buf", "buf_limit").Export(handlerapi.FuncReadBody).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.writeBody), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("kind", "body", "body_len").Export(handlerapi.FuncWriteBody).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(e.getSourceAddr), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(handlerapi.FuncGetSourceAddr).
		NewFunctionBuilder().
		WithGoFunction(wazeroapi.GoFunc(e.getStatusCode), []wazeroapi.ValueType{}, []wazeroapi.ValueType{i32}).
		WithParameterNames().Export(handlerapi.FuncGetStatusCode).
		NewFunctionBuilder().
		WithGoFunction(wazeroapi.GoFunc(e.setStatusCode), []wazeroapi.ValueType{i32}, []wazeroapi.ValueType{}).
		WithParameterNames("status_code").Export(handlerapi.FuncSetStatusCode).
		Instantiate(ctx)
}

// enableFeatures implements the WebAssembly host function handler.FuncEnableFeatures.
// Outside a request (guest initialization), the features apply to the engine.
func (e *httpWasm) enableFeatures(ctx context.Context, stack []uint64) {
	features := handlerapi.Features(stack[0])

	var enabled handlerapi.Features
	if s, ok := ctx.Value(requestStateKey{}).(*requestState); ok {
		s.enableFeatures(features)
		enabled = s.features
	} else {
		enabled = e.enableInstanceFeatures(features)
	}

	stack[0] = uint64(enabled)
}

// getConfig implements the WebAssembly host function handler.FuncGetConfig.
func (e *httpWasm) getConfig(_ context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handlerapi.BufLimit(stack[1])

	configLen := writeIfUnderLimit(mod.Memory(), buf, bufLimit, e.cfg.GuestConfig)

	stack[0] = uint64(configLen)
}

// logEnabled implements the WebAssembly host function handler.FuncLogEnabled.
func (e *httpWasm) logEnabled(_ context.Context, stack []uint64) {
	level := api.LogLevel(stack[0])
	if e.logger.IsEnabled(level) {
		stack[0] = 1 // true
	} else {
		stack[0] = 0 // false
	}
}

// log implements the WebAssembly host function handler.FuncLog.
func (e *httpWasm) log(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	level := api.LogLevel(params[0])
	message := uint32(params[1])
	messageLen := uint32(params[2])

	if !e.logger.IsEnabled(level) {
		return
	}
	var msg string
	if messageLen > 0 {
		msg = mustReadString(mod.Memory(), "message", message, messageLen)
	}
	e.logger.Log(ctx, level, msg)
}

// getMethod implements the WebAssembly host function handler.FuncGetMethod.
func (e *httpWasm) getMethod(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handlerapi.BufLimit(stack[1])

	method := e.host.GetMethod(ctx)
	methodLen := writeStringIfUnderLimit(mod.Memory(), buf, bufLimit, method)

	stack[0] = uint64(methodLen)
}

// setMethod implements the WebAssembly host function handler.FuncSetMethod.
func (e *httpWasm) setMethod(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	method := uint32(params[0])
	methodLen := uint32(params[1])

	_ = mustBeforeNext(ctx, "set", "method")

	if methodLen == 0 {
		panic("HTTP method cannot be empty")
	}
	p := mustReadString(mod.Memory(), "method", method, methodLen)
	e.host.SetMethod(ctx, p)
}

// getURI implements the WebAssembly host function handler.FuncGetURI.
func (e *httpWasm) getURI(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handlerapi.BufLimit(stack[1])

	uri := e.host.GetURI(ctx)
	uriLen := writeStringIfUnderLimit(mod.Memory(), buf, bufLimit, uri)

	stack[0] = uint64(uriLen)
}

// setURI implements the WebAssembly host function handler.FuncSetURI.
func (e *httpWasm) setURI(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	uri := uint32(params[0])
	uriLen := uint32(params[1])

	_ = mustBeforeNext(ctx, "set", "uri")

	var p string
	if uriLen > 0 { // overwrite with empty is supported
		p = mustReadString(mod.Memory(), "uri", uri, uriLen)
	}
	e.host.SetURI(ctx, p)
}

// getProtocolVersion implements the WebAssembly host function handler.FuncGetProtocolVersion.
func (e *httpWasm) getProtocolVersion(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handlerapi.BufLimit(stack[1])

	protocolVersion := e.host.GetProtocolVersion(ctx)
	if len(protocolVersion) == 0 {
		panic("HTTP protocol version cannot be empty")
	}
	protocolVersionLen := writeStringIfUnderLimit(mod.Memory(), buf, bufLimit, protocolVersion)

	stack[0] = uint64(protocolVersionLen)
}

// getHeaderNames implements the WebAssembly host function handler.FuncGetHeaderNames.
func (e *httpWasm) getHeaderNames(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	kind := handlerapi.HeaderKind(stack[0])
	buf := uint32(stack[1])
	bufLimit := handlerapi.BufLimit(stack[2])

	var names []string
	switch kind {
	case handlerapi.HeaderKindRequest:
		names = e.host.GetRequestHeaderNames(ctx)
	case handlerapi.HeaderKindRequestTrailers:
		names = e.host.GetRequestTrailerNames(ctx)
	case handlerapi.HeaderKindResponse:
		names = e.host.GetResponseHeaderNames(ctx)
	case handlerapi.HeaderKindResponseTrailers:
		names = e.host.GetResponseTrailerNames(ctx)
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}

	for i := range names {
		names[i] = strings.ToLower(names[i])
	}

	stack[0] = writeNULTerminated(mod.Memory(), buf, bufLimit, names)
}

// getHeaderValues implements the WebAssembly host function handler.FuncGetHeaderValues.
func (e *httpWasm) getHeaderValues(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	kind := handlerapi.HeaderKind(stack[0])
	name := uint32(stack[1])
	nameLen := uint32(stack[2])
	buf := uint32(stack[3])
	bufLimit := handlerapi.BufLimit(stack[4])

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	n := mustReadString(mod.Memory(), "name", name, nameLen)

	var values []string
	switch kind {
	case handlerapi.HeaderKindRequest:
		values = e.host.GetRequestHeaderValues(ctx, n)
	case handlerapi.HeaderKindRequestTrailers:
		values = e.host.GetRequestTrailerValues(ctx, n)
	case handlerapi.HeaderKindResponse:
		values = e.host.GetResponseHeaderValues(ctx, n)
	case handlerapi.HeaderKindResponseTrailers:
		values = e.host.GetResponseTrailerValues(ctx, n)
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}

	stack[0] = writeNULTerminated(mod.Memory(), buf, bufLimit, values)
}

// setHeaderValue implements the WebAssembly host function handler.FuncSetHeaderValue.
func (e *httpWasm) setHeaderValue(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	kind := handlerapi.HeaderKind(params[0])
	name := uint32(params[1])
	nameLen := uint32(params[2])
	value := uint32(params[3])
	valueLen := uint32(params[4])

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	mustHeaderMutable(ctx, "set", kind)
	n := mustReadString(mod.Memory(), "name", name, nameLen)
	v := mustReadString(mod.Memory(), "value", value, valueLen)

	switch kind {
	case handlerapi.HeaderKindRequest:
		e.host.SetRequestHeaderValue(ctx, n, v)
	case handlerapi.HeaderKindRequestTrailers:
		e.host.SetRequestTrailerValue(ctx, n, v)
	case handlerapi.HeaderKindResponse:
		e.host.SetResponseHeaderValue(ctx, n, v)
	case handlerapi.HeaderKindResponseTrailers:
		e.host.SetResponseTrailerValue(ctx, n, v)
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}
}

// addHeaderValue implements the WebAssembly host function handler.FuncAddHeaderValue.
func (e *httpWasm) addHeaderValue(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	kind := handlerapi.HeaderKind(params[0])
	name := uint32(params[1])
	nameLen := uint32(params[2])
	value := uint32(params[3])
	valueLen := uint32(params[4])

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	mustHeaderMutable(ctx, "add", kind)
	n := mustReadString(mod.Memory(), "name", name, nameLen)
	v := mustReadString(mod.Memory(), "value", value, valueLen)

	switch kind {
	case handlerapi.HeaderKindRequest:
		e.host.AddRequestHeaderValue(ctx, n, v)
	case handlerapi.HeaderKindRequestTrailers:
		e.host.AddRequestTrailerValue(ctx, n, v)
	case handlerapi.HeaderKindResponse:
		e.host.AddResponseHeaderValue(ctx, n, v)
	case handlerapi.HeaderKindResponseTrailers:
		e.host.AddResponseTrailerValue(ctx, n, v)
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}
}

// removeHeader implements the WebAssembly host function handler.FuncRemoveHeader.
func (e *httpWasm) removeHeader(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	kind := handlerapi.HeaderKind(params[0])
	name := uint32(params[1])
	nameLen := uint32(params[2])

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	mustHeaderMutable(ctx, "remove", kind)
	n := mustReadString(mod.Memory(), "name", name, nameLen)

	switch kind {
	case handlerapi.HeaderKindRequest:
		e.host.RemoveRequestHeader(ctx, n)
	case handlerapi.HeaderKindRequestTrailers:
		e.host.RemoveRequestTrailer(ctx, n)
	case handlerapi.HeaderKindResponse:
		e.host.RemoveResponseHeader(ctx, n)
	case handlerapi.HeaderKindResponseTrailers:
		e.host.RemoveResponseTrailer(ctx, n)
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}
}

// readBody implements the WebAssembly host function handler.FuncReadBody.
func (e *httpWasm) readBody(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	kind := handlerapi.BodyKind(stack[0])
	buf := uint32(stack[1])
	bufLimit := handlerapi.BufLimit(stack[2])

	var r io.ReadCloser
	switch kind {
	case handlerapi.BodyKindRequest:
		s := mustBeforeNextOrFeature(ctx, handlerapi.FeatureBufferRequest, "read", "request body")
		r = s.requestBodyReader
		if r == nil {
			r = e.host.RequestBodyReader(ctx)
			s.requestBodyReader = r
		}
	case handlerapi.BodyKindResponse:
		s := mustBeforeNextOrFeature(ctx, handlerapi.FeatureBufferResponse, "read", "response body")
		r = s.responseBodyReader
		if r == nil {
			r = e.host.ResponseBodyReader(ctx)
			s.responseBodyReader = r
		}
	default:
		panic("unsupported body kind: " + strconv.Itoa(int(kind)))
	}

	stack[0] = readBody(mod, buf, bufLimit, r)
}

// writeBody implements the WebAssembly host function handler.FuncWriteBody.
func (e *httpWasm) writeBody(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	kind := handlerapi.BodyKind(params[0])
	buf := uint32(params[1])
	bufLen := uint32(params[2])

	var w io.Writer
	switch kind {
	case handlerapi.BodyKindRequest:
		s := mustBeforeNext(ctx, "write", "request body")
		w = s.requestBodyWriter
		if w == nil {
			w = e.host.RequestBodyWriter(ctx)
			s.requestBodyWriter = w
		}
	case handlerapi.BodyKindResponse:
		s := mustBeforeNextOrFeature(ctx, handlerapi.FeatureBufferResponse, "write", "response body")
		w = s.responseBodyWriter
		if w == nil {
			w = e.host.ResponseBodyWriter(ctx)
			s.responseBodyWriter = w
		}
	default:
		panic("unsupported body kind: " + strconv.Itoa(int(kind)))
	}

	writeBody(mod, buf, bufLen, w)
}

// getSourceAddr implements the WebAssembly host function handler.FuncGetSourceAddr.
func (e *httpWasm) getSourceAddr(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handlerapi.BufLimit(stack[1])

	addr := e.host.GetSourceAddr(ctx)
	addrLen := writeStringIfUnderLimit(mod.Memory(), buf, bufLimit, addr)

	stack[0] = uint64(addrLen)
}

// getStatusCode implements the WebAssembly host function handler.FuncGetStatusCode.
func (e *httpWasm) getStatusCode(ctx context.Context, results []uint64) {
	results[0] = uint64(e.host.GetStatusCode(ctx))
}

// setStatusCode implements the WebAssembly host function handler.FuncSetStatusCode.
func (e *httpWasm) setStatusCode(ctx context.Context, params []uint64) {
	statusCode := uint32(params[0])

	_ = mustBeforeNextOrFeature(ctx, handlerapi.FeatureBufferResponse, "set", "status code")

	e.host.SetStatusCode(ctx, statusCode)
}

func writeBody(mod wazeroapi.Module, buf, bufLen uint32, w io.Writer) {
	// buf_len 0 means to overwrite with nothing
	var b []byte
	if bufLen > 0 {
		b = mustRead(mod.Memory(), "body", buf, bufLen)
	}
	if _, err := w.Write(b); err != nil { // Write errs if it can't write n bytes
		panic(fmt.Errorf("error writing body: %w", err))
	}
}

func readBody(mod wazeroapi.Module, buf uint32, bufLimit handlerapi.BufLimit, r io.Reader) (eofLen uint64) {
	// buf_limit 0 serves no purpose as implementations won't return EOF on it.
	if bufLimit == 0 {
		panic(errors.New("buf_limit==0 reading body"))
	}

	// Allocate a buf to write into directly
	b := mustRead(mod.Memory(), "body", buf, bufLimit)

	// Attempt to fill the buffer until an error occurs.
	var err error
	n := uint32(0)
	for n < bufLimit && err == nil {
		var nn int
		nn, err = r.Read(b[n:])
		n += uint32(nn)
	}

	switch {
	case err == nil:
		return uint64(n) // Not EOF
	case errors.Is(err, io.EOF):
		return uint64(1<<32) | uint64(n)
	default:
		panic(fmt.Errorf("error reading body: %w", err))
	}
}

func mustBeforeNext(ctx context.Context, op, kind string) *requestState {
	s := requestStateFromContext(ctx)
	if s.afterNext {
		panic(fmt.Errorf("can't %s %s after next handler", op, kind))
	}

	return s
}

func mustBeforeNextOrFeature(ctx context.Context, feature handlerapi.Features, op, kind string) *requestState {
	s := requestStateFromContext(ctx)
	switch {
	case !s.afterNext:
		// Assume this is serving a response from the guest.
	case s.features.IsEnabled(feature):
		// Assume the guest is overwriting the response from next.
	default:
		panic(fmt.Errorf("can't %s %s after next handler unless %s is enabled", op, kind, feature))
	}

	return s
}

func mustHeaderMutable(ctx context.Context, op string, kind handlerapi.HeaderKind) {
	switch kind {
	case handlerapi.HeaderKindRequest:
		_ = mustBeforeNext(ctx, op, "request header")
	case handlerapi.HeaderKindRequestTrailers:
		_ = mustBeforeNext(ctx, op, "request trailer")
	case handlerapi.HeaderKindResponse:
		_ = mustBeforeNextOrFeature(ctx, handlerapi.FeatureBufferResponse, op, "response header")
	case handlerapi.HeaderKindResponseTrailers:
		_ = mustBeforeNextOrFeature(ctx, handlerapi.FeatureBufferResponse, op, "response trailer")
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}
}

// mustReadString is a convenience function that casts mustRead.
func mustReadString(mem wazeroapi.Memory, fieldName string, offset, byteCount uint32) string {
	if byteCount == 0 {
		return ""
	}

	return string(mustRead(mem, fieldName, offset, byteCount))
}

var emptyBody = make([]byte, 0)

// mustRead is like api.Memory except that it panics if the offset and byteCount are out of range.
func mustRead(mem wazeroapi.Memory, fieldName string, offset, byteCount uint32) []byte {
	if byteCount == 0 {
		return emptyBody
	}
	buf, ok := mem.Read(offset, byteCount)
	if !ok {
		panic(fmt.Errorf("out of memory reading %s", fieldName))
	}

	return buf
}

func writeIfUnderLimit(mem wazeroapi.Memory, offset, limit handlerapi.BufLimit, v []byte) uint32 {
	vLen := uint32(len(v))
	if vLen > limit || vLen == 0 {
		return vLen // the caller can retry with a larger limit, or there is nothing to write
	}
	mem.Write(offset, v)

	return vLen
}

func writeStringIfUnderLimit(mem wazeroapi.Memory, offset, limit handlerapi.BufLimit, v string) uint32 {
	vLen := uint32(len(v))
	if vLen > limit || vLen == 0 {
		return vLen
	}
	mem.WriteString(offset, v)

	return vLen
}
