// Package main is a test guest of the wasihttp mode, written directly on the ABI of github.com/traefik/wasm-go
// with the standard library only, so that it builds without the SDK.
//
// Build with:
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o plugin.wasm .
package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

//go:wasmimport traefik req_info
func reqInfo(id uint32, ptr unsafe.Pointer, size uint32) uint32

//go:wasmimport traefik req_trailers
func reqTrailers(id uint32, ptr unsafe.Pointer, size uint32) uint32

//go:wasmimport traefik resp_set
func respSet(id uint32, ptr unsafe.Pointer, size uint32)

//go:wasmimport traefik resp_trailers
func respTrailers(id uint32, ptr unsafe.Pointer, size uint32)

//go:wasmimport traefik done
func done(id uint32)

//go:wasmimport traefik call
func hostCall(target, id uint32, ptr unsafe.Pointer, size, mode uint32) uint32

//go:wasmimport traefik call_set_trailers
func callSetTrailers(cid uint32, ptr unsafe.Pointer, size uint32)

//go:wasmimport traefik call_trailers
func callTrailers(cid uint32, ptr unsafe.Pointer, size uint32) uint32

//go:wasmimport traefik config_get
func configGet(kptr unsafe.Pointer, klen uint32, ptr unsafe.Pointer, size uint32) int32

//go:wasmimport traefik config_get_all
func configGetAll(ptr unsafe.Pointer, size uint32) uint32

//go:wasmimport traefik log
func hostLog(level uint32, ptr unsafe.Pointer, size uint32)

//go:wasmimport traefik log_enabled
func logEnabled(level uint32) uint32

const (
	fdBodyIn      = 0
	fdResponseOut = 1
	fdCancel      = 2

	fdCallResponseIn = 0
	fdCallBodyOut    = 1

	targetHandler = 0
	targetClient  = 1

	noRequest = 0xFFFFFFFF

	bodyNone      = 0
	bodyFromGuest = 1
	bodyIncoming  = 2

	levelDebug = 1
	levelInfo  = 2
)

//go:wasmexport traefik_abi_version
func abiVersion() uint32 {
	return 1
}

//go:wasmexport traefik_serve
func serve() {
	listener := os.NewFile(3, "listener")

	var id [4]byte
	for {
		if _, err := io.ReadFull(listener, id[:]); err != nil {
			return
		}
		go handle(binary.LittleEndian.Uint32(id[:]))
	}
}

func main() {}

type request struct {
	method string
	target string
	host   string
	proto  string
	remote string
	length int64
	header []string
}

func (r *request) path() string {
	p, _, _ := strings.Cut(r.target, "?")

	return p
}

func (r *request) query() string {
	_, q, _ := strings.Cut(r.target, "?")

	return q
}

func handle(id uint32) {
	defer done(id)

	r := parseRequest(fetch(func(ptr unsafe.Pointer, size uint32) uint32 { return reqInfo(id, ptr, size) }))

	switch r.path() {
	case "/echo":
		body := readAll(requestFD(id, fdBodyIn))
		respond(id, 202, []string{"X-Echo-Host: " + r.host, "X-Echo-Remote: " + r.remote, "X-Echo-Proto: " + r.proto},
			[]byte(r.method+" "+r.target+" "+strconv.FormatInt(r.length, 10)+":"+string(body)))

	case "/next":
		// The request goes to next with the body of the client, which the plugin does not read.
		cid := startCall(targetHandler, id, requestBlock(r.method, r.target, r.host, "", r.length, append(r.header, "X-Plugin: pass")), bodyIncoming)
		status, header, body, trailers := readCall(cid)
		respond(id, status, append(header, "X-Plugin-Response: 1"), body)
		setResponseTrailers(id, trailers)

	case "/rewrite":
		body := bytes.ToUpper(readAll(requestFD(id, fdBodyIn)))
		cid := startCall(targetHandler, id, requestBlock(r.method, "/rewritten?"+r.query(), "upstream", "", int64(len(body)), r.header), bodyFromGuest)
		writeAll(callFD(cid, fdCallBodyOut), body)
		status, header, resp, _ := readCall(cid)
		respond(id, status, header, bytes.ToUpper(resp))

	case "/twice":
		// Next is called twice for the same request, without body.
		var out []byte
		for i := range 2 {
			cid := startCall(targetHandler, id, requestBlock("GET", "/twice?call="+strconv.Itoa(i), r.host, "", 0, nil), bodyNone)
			_, _, body, _ := readCall(cid)
			out = append(append(out, body...), '|')
		}
		respond(id, 200, nil, out)

	case "/no-request":
		// Next without an incoming request has nothing to run.
		cid := startCall(targetHandler, noRequest, requestBlock("GET", "/", r.host, "", 0, nil), bodyNone)
		status, header, body, _ := readCall(cid)
		respond(id, status, header, body)

	case "/trailers":
		// The trailers of the request come back as the trailers of the response, after the body.
		body := readAll(requestFD(id, fdBodyIn))
		trailers := fetch(func(ptr unsafe.Pointer, size uint32) uint32 { return reqTrailers(id, ptr, size) })
		respond(id, 200, nil, body)
		setResponseTrailers(id, trailers)

	case "/next-trailers":
		// The plugin streams a body with trailers to next.
		cid := startCall(targetHandler, id, requestBlock("POST", "/next-trailers", r.host, "", -1, []string{"Trailer: X-Checksum"}), bodyFromGuest)
		trailer := []byte("X-Checksum: abc\n")
		callSetTrailers(cid, unsafe.Pointer(&trailer[0]), uint32(len(trailer)))
		writeAll(callFD(cid, fdCallBodyOut), []byte("streamed"))
		status, header, body, _ := readCall(cid)
		respond(id, status, header, body)

	case "/stream":
		// The second part comes later: the first one must reach the client before.
		setResponse(id, 200, nil)
		out := os.NewFile(uintptr(requestFD(id, fdResponseOut)), "response")
		_, _ = out.Write([]byte("first;"))
		time.Sleep(300 * time.Millisecond)
		_, _ = out.Write([]byte("second"))
		_ = out.Close()

	case "/config":
		key := []byte(strings.TrimPrefix(r.query(), "key="))
		value, ok := config(key)
		if !ok {
			respond(id, 404, nil, []byte("missing"))

			return
		}
		respond(id, 200, nil, value)

	case "/config-all":
		respond(id, 200, nil, fetch(configGetAll))

	case "/log":
		msg := []byte("hello from guest")
		hostLog(levelInfo, unsafe.Pointer(&msg[0]), uint32(len(msg)))
		respond(id, 200, nil, []byte("debug="+strconv.Itoa(int(logEnabled(levelDebug)))))

	case "/env":
		respond(id, 200, nil, []byte(os.Getenv("PLUGIN_TEST")))

	case "/print":
		_, _ = os.Stdout.WriteString("printed on stdout\n")
		respond(id, 200, nil, nil)

	case "/wait":
		cancel := os.NewFile(uintptr(requestFD(id, fdCancel)), "cancel")
		var b [1]byte
		_, _ = cancel.Read(b[:])
		_ = cancel.Close()
		respond(id, 499, nil, nil)

	case "/sleep":
		time.Sleep(20 * time.Millisecond)
		respond(id, 200, nil, []byte("slept"))

	case "/big":
		size, _ := strconv.Atoi(strings.TrimPrefix(r.query(), "size="))
		respond(id, 200, nil, bytes.Repeat([]byte("x"), size))

	case "/outbound":
		// The destination is the url query parameter, as scheme://host/path.
		dest := strings.TrimPrefix(r.query(), "url=")
		scheme, rest, _ := strings.Cut(dest, "://")
		host, path, _ := strings.Cut(rest, "/")
		body := readAll(requestFD(id, fdBodyIn))

		cid := startCall(targetClient, id, requestBlock("POST", "/"+path, host, scheme, int64(len(body)), []string{"X-From-Plugin: yes"}), bodyFromGuest)
		writeAll(callFD(cid, fdCallBodyOut), body)
		status, header, resp, _ := readCall(cid)
		respond(id, status, header, resp)

	case "/exit":
		os.Exit(3)

	default:
		respond(id, 404, nil, nil)
	}
}

func requestFD(id, k uint32) uint32 {
	return 1000 + id*4 + k
}

func callFD(cid, k uint32) uint32 {
	return 1<<30 + cid*2 + k
}

func fetch(call func(ptr unsafe.Pointer, size uint32) uint32) []byte {
	buf := make([]byte, 1024)
	for {
		n := call(unsafe.Pointer(&buf[0]), uint32(len(buf)))
		if n <= uint32(len(buf)) {
			return buf[:n]
		}
		buf = make([]byte, n)
	}
}

func config(key []byte) ([]byte, bool) {
	kptr := unsafe.Pointer(unsafe.StringData("-"))
	if len(key) > 0 {
		kptr = unsafe.Pointer(&key[0])
	}

	buf := make([]byte, 256)
	for {
		n := configGet(kptr, uint32(len(key)), unsafe.Pointer(&buf[0]), uint32(len(buf)))
		if n < 0 {
			return nil, false
		}
		if int(n) <= len(buf) {
			return buf[:n], true
		}
		buf = make([]byte, n)
	}
}

func parseRequest(block []byte) *request {
	line, rest, _ := strings.Cut(string(block), "\n")
	f := strings.Split(line, " ")
	for i := range f {
		if f[i] == "-" {
			f[i] = ""
		}
	}
	length, _ := strconv.ParseInt(f[6], 10, 64)

	return &request{method: f[0], target: f[1], host: f[2], proto: f[4], remote: f[5], length: length, header: lines(rest)}
}

func requestBlock(method, target, host, scheme string, length int64, header []string) []byte {
	if scheme == "" {
		scheme = "-"
	}

	block := method + " " + target + " " + host + " " + scheme + " - - " + strconv.FormatInt(length, 10) + "\n"
	for _, h := range header {
		block += h + "\n"
	}

	return []byte(block)
}

func startCall(target, id uint32, block []byte, mode uint32) uint32 {
	return hostCall(target, id, unsafe.Pointer(&block[0]), uint32(len(block)), mode)
}

// readCall reads the response of a call: a frame with the status and the headers, then the body, then the trailers.
// A failed call becomes a 502 response with the error as body.
func readCall(cid uint32) (int, []string, []byte, []byte) {
	f := os.NewFile(uintptr(callFD(cid, fdCallResponseIn)), "call")
	b, _ := io.ReadAll(f)
	// The trailers are read before closing the response, which ends the call.
	trailers := fetch(func(ptr unsafe.Pointer, size uint32) uint32 { return callTrailers(cid, ptr, size) })
	_ = f.Close()

	if len(b) < 4 {
		return 502, nil, nil, nil
	}

	size := binary.LittleEndian.Uint32(b)
	block, body := string(b[4:4+size]), b[4+size:]
	line, rest, _ := strings.Cut(block, "\n")
	code, msg, _ := strings.Cut(line, " ")
	status, _ := strconv.Atoi(code)
	if status == 0 {
		return 502, nil, []byte(msg), nil
	}

	return status, lines(rest), body, trailers
}

func lines(s string) []string {
	var out []string
	for l := range strings.SplitSeq(s, "\n") {
		if l != "" {
			out = append(out, l)
		}
	}

	return out
}

func readAll(fd uint32) []byte {
	f := os.NewFile(uintptr(fd), "in")
	defer func() { _ = f.Close() }()

	b, _ := io.ReadAll(f)

	return b
}

func writeAll(fd uint32, b []byte) {
	f := os.NewFile(uintptr(fd), "out")
	_, _ = f.Write(b)
	_ = f.Close()
}

func setResponse(id uint32, status int, header []string) {
	block := strconv.Itoa(status) + "\n"
	for _, h := range header {
		block += h + "\n"
	}
	b := []byte(block)
	respSet(id, unsafe.Pointer(&b[0]), uint32(len(b)))
}

func setResponseTrailers(id uint32, trailers []byte) {
	if len(trailers) == 0 {
		return
	}
	respTrailers(id, unsafe.Pointer(&trailers[0]), uint32(len(trailers)))
}

func respond(id uint32, status int, header []string, body []byte) {
	setResponse(id, status, header)

	out := os.NewFile(uintptr(requestFD(id, fdResponseOut)), "response")
	defer func() { _ = out.Close() }()

	for len(body) > 0 {
		n := min(len(body), 4096)
		_, _ = out.Write(body[:n])
		body = body[n:]
	}
}
