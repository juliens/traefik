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

//go:wasmimport traefik req_set
func reqSet(id uint32, ptr unsafe.Pointer, size uint32)

//go:wasmimport traefik resp_set
func respSet(id uint32, ptr unsafe.Pointer, size uint32)

//go:wasmimport traefik next_start
func nextStart(id, mode uint32)

//go:wasmimport traefik done
func done(id uint32)

//go:wasmimport traefik config_get
func configGet(ptr unsafe.Pointer, size uint32) uint32

//go:wasmimport traefik log
func hostLog(level uint32, ptr unsafe.Pointer, size uint32)

//go:wasmimport traefik log_enabled
func logEnabled(level uint32) uint32

const (
	fdBodyIn         = 0
	fdResponseOut    = 1
	fdNextResponseIn = 2
	fdNextBodyOut    = 3
	fdCancel         = 4

	bodyOriginal  = 0
	bodyFromGuest = 1

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
	id     uint32
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

func handle(id uint32) {
	defer done(id)

	r := parseRequest(id, fetch(func(ptr unsafe.Pointer, size uint32) uint32 { return reqInfo(id, ptr, size) }))

	switch r.path() {
	case "/echo":
		body := readAll(id, fdBodyIn)
		respond(id, 202, []string{"X-Echo-Host: " + r.host, "X-Echo-Remote: " + r.remote, "X-Echo-Proto: " + r.proto},
			[]byte(r.method+" "+r.target+" "+strconv.FormatInt(r.length, 10)+":"+string(body)))

	case "/next":
		setRequest(id, r.method, r.target, r.host, r.length, append(r.header, "X-Plugin: pass"))
		nextStart(id, bodyOriginal)
		status, header, body := readNext(id)
		respond(id, status, append(header, "X-Plugin-Response: 1"), body)

	case "/rewrite":
		body := bytes.ToUpper(readAll(id, fdBodyIn))
		_, query, _ := strings.Cut(r.target, "?")
		setRequest(id, r.method, "/rewritten?"+query, "upstream", int64(len(body)), r.header)
		nextStart(id, bodyFromGuest)
		out := os.NewFile(uintptr(fd(id, fdNextBodyOut)), "next-body")
		_, _ = out.Write(body)
		_ = out.Close()
		status, header, resp := readNext(id)
		respond(id, status, header, bytes.ToUpper(resp))

	case "/config":
		respond(id, 200, nil, fetch(configGet))

	case "/log":
		msg := []byte("hello from guest")
		hostLog(levelInfo, unsafe.Pointer(&msg[0]), uint32(len(msg)))
		respond(id, 200, nil, []byte("debug="+strconv.Itoa(int(logEnabled(levelDebug)))))

	case "/env":
		respond(id, 200, nil, []byte(os.Getenv("PLUGIN_TEST")))

	case "/print":
		os.Stdout.WriteString("printed on stdout\n")
		respond(id, 200, nil, nil)

	case "/wait":
		cancel := os.NewFile(uintptr(fd(id, fdCancel)), "cancel")
		var b [1]byte
		_, _ = cancel.Read(b[:])
		_ = cancel.Close()
		respond(id, 499, nil, nil)

	case "/sleep":
		time.Sleep(20 * time.Millisecond)
		respond(id, 200, nil, []byte("slept"))

	case "/big":
		size, _ := strconv.Atoi(strings.TrimPrefix(r.target, "/big?size="))
		respond(id, 200, nil, bytes.Repeat([]byte("x"), size))

	case "/exit":
		os.Exit(3)

	default:
		respond(id, 404, nil, nil)
	}
}

func fd(id, k uint32) uint32 {
	return 1000 + id*8 + k
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

func parseRequest(id uint32, block []byte) *request {
	line, rest, _ := strings.Cut(string(block), "\n")
	f := strings.Split(line, " ")
	for i := range f {
		if f[i] == "-" {
			f[i] = ""
		}
	}
	length, _ := strconv.ParseInt(f[6], 10, 64)

	var header []string
	for _, h := range strings.Split(rest, "\n") {
		if h != "" {
			header = append(header, h)
		}
	}

	return &request{id: id, method: f[0], target: f[1], host: f[2], proto: f[4], remote: f[5], length: length, header: header}
}

func setRequest(id uint32, method, target, host string, length int64, header []string) {
	block := method + " " + target + " " + host + " - - - " + strconv.FormatInt(length, 10) + "\n"
	for _, h := range header {
		block += h + "\n"
	}
	b := []byte(block)
	reqSet(id, unsafe.Pointer(&b[0]), uint32(len(b)))
}

func readAll(id, k uint32) []byte {
	f := os.NewFile(uintptr(fd(id, k)), "in")
	defer f.Close()

	b, _ := io.ReadAll(f)

	return b
}

// readNext reads the response of next: a frame with the status and the headers, then the body.
func readNext(id uint32) (int, []string, []byte) {
	b := readAll(id, fdNextResponseIn)
	if len(b) < 4 {
		return 502, nil, nil
	}

	size := binary.LittleEndian.Uint32(b)
	block, body := string(b[4:4+size]), b[4+size:]
	line, rest, _ := strings.Cut(block, "\n")
	status, _ := strconv.Atoi(line)
	if status == 0 {
		return 502, nil, nil
	}

	var header []string
	for _, h := range strings.Split(rest, "\n") {
		if h != "" {
			header = append(header, h)
		}
	}

	return status, header, body
}

func respond(id uint32, status int, header []string, body []byte) {
	block := strconv.Itoa(status) + "\n"
	for _, h := range header {
		block += h + "\n"
	}
	b := []byte(block)
	respSet(id, unsafe.Pointer(&b[0]), uint32(len(b)))

	out := os.NewFile(uintptr(fd(id, fdResponseOut)), "response")
	defer out.Close()

	for len(body) > 0 {
		n := min(len(body), 4096)
		_, _ = out.Write(body[:n])
		body = body[n:]
	}
}
