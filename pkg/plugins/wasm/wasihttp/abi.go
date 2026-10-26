package wasihttp

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// The contract with the guest mirrors github.com/traefik/wasm-go/abi, version 1: the wasi:http/middleware@0.3.0 world
// with wasi:config/store and wasi:logging, on a core wasm ABI. It is kept here until that module is published and can
// be imported.

const abiVersion uint32 = 1

const (
	hostModule = "traefik"

	exportInitialize = "_initialize"
	exportServe      = "traefik_serve"
	exportVersion    = "traefik_abi_version"
)

const (
	fdListener uint32 = 3

	requestFDBase  uint32 = 1000
	requestFDCount uint32 = 4

	fdBodyIn      uint32 = 0
	fdResponseOut uint32 = 1
	fdCancel      uint32 = 2

	callFDBase  uint32 = 1 << 30
	callFDCount uint32 = 2

	fdCallResponseIn uint32 = 0
	fdCallBodyOut    uint32 = 1

	// maxRequestIDs bounds the request ids so that the request descriptors stay below the call ones, and maxCallIDs
	// bounds the call ids so that their descriptors fit in 32 bits. Ids are reused after that many requests or calls
	// on one instance.
	maxRequestIDs = 1 << 26
	maxCallIDs    = 1 << 26
)

// Call targets: the imported wasi:http/handler (the next handler) and the imported wasi:http/client.
const (
	targetHandler uint32 = 0
	targetClient  uint32 = 1
)

// noRequest is the incoming request id of a call made outside of any incoming request.
const noRequest uint32 = 0xFFFFFFFF

// Body modes of a call.
const (
	bodyNone      uint32 = 0
	bodyFromGuest uint32 = 1
	bodyIncoming  uint32 = 2
)

// Log levels, with the values of wasi:logging.
const (
	levelTrace uint32 = iota
	levelDebug
	levelInfo
	levelWarn
	levelError
	levelCritical
)

func requestFD(id, k uint32) uint32 {
	return requestFDBase + id*requestFDCount + k
}

func callFD(cid, k uint32) uint32 {
	return callFDBase + cid*callFDCount + k
}

func parseCallFD(fd uint32) (cid, k uint32, ok bool) {
	if fd < callFDBase {
		return 0, 0, false
	}
	fd -= callFDBase

	return fd / callFDCount, fd % callFDCount, true
}

// requestBlock is the request line and the headers of a request, as exchanged with the guest.
type requestBlock struct {
	Method        string
	Target        string
	Host          string
	Scheme        string
	Proto         string
	RemoteAddr    string
	ContentLength int64
	Header        http.Header
}

// responseBlock is the status and the headers of a response; a zero status is an error, described by Error.
type responseBlock struct {
	Status int
	Error  string
	Header http.Header
}

const emptyField = "-"

// appendRequest encodes the space separated request line, an empty field being "-", then one "Key: value" line per
// header value, keys sorted.
func appendRequest(r requestBlock) []byte {
	var dst []byte
	for i, field := range []string{r.Method, r.Target, r.Host, r.Scheme, r.Proto, r.RemoteAddr} {
		if i > 0 {
			dst = append(dst, ' ')
		}
		if field == "" {
			field = emptyField
		}
		dst = append(dst, field...)
	}
	dst = append(dst, ' ')
	dst = strconv.AppendInt(dst, r.ContentLength, 10)
	dst = append(dst, '\n')

	return appendHeader(dst, r.Header)
}

func parseRequest(b []byte) (requestBlock, error) {
	line, rest, _ := strings.Cut(string(b), "\n")
	fields := strings.Split(line, " ")
	if len(fields) != 7 {
		return requestBlock{}, fmt.Errorf("request line has %d fields, want 7", len(fields))
	}

	for i, field := range fields {
		if field == emptyField {
			fields[i] = ""
		}
	}

	length, err := strconv.ParseInt(fields[6], 10, 64)
	if err != nil {
		return requestBlock{}, fmt.Errorf("parsing content length: %w", err)
	}

	return requestBlock{
		Method:        fields[0],
		Target:        fields[1],
		Host:          fields[2],
		Scheme:        fields[3],
		Proto:         fields[4],
		RemoteAddr:    fields[5],
		ContentLength: length,
		Header:        parseHeader(rest),
	}, nil
}

func appendResponse(r responseBlock) []byte {
	dst := strconv.AppendInt(nil, int64(r.Status), 10)
	if r.Status == 0 && r.Error != "" {
		dst = append(dst, ' ')
		dst = append(dst, strings.ReplaceAll(r.Error, "\n", " ")...)
	}
	dst = append(dst, '\n')

	return appendHeader(dst, r.Header)
}

func parseResponse(b []byte) (responseBlock, error) {
	line, rest, _ := strings.Cut(string(b), "\n")
	code, msg, _ := strings.Cut(line, " ")

	status, err := strconv.Atoi(code)
	if err != nil {
		return responseBlock{}, fmt.Errorf("parsing status: %w", err)
	}

	return responseBlock{Status: status, Error: msg, Header: parseHeader(rest)}, nil
}

// appendFrame returns block prefixed by its length as a little endian uint32.
func appendFrame(block []byte) []byte {
	dst := binary.LittleEndian.AppendUint32(make([]byte, 0, 4+len(block)), uint32(len(block)))

	return append(dst, block...)
}

func appendHeader(dst []byte, h http.Header) []byte {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		for _, v := range h[k] {
			dst = append(dst, k...)
			dst = append(dst, ": "...)
			dst = append(dst, v...)
			dst = append(dst, '\n')
		}
	}

	return dst
}

// appendFields encodes a fields block, as used for trailers: one "Key: value" line per value, keys sorted.
func appendFields(h http.Header) []byte {
	return appendHeader(nil, h)
}

func parseHeader(s string) http.Header {
	h := http.Header{}
	for line := range strings.SplitSeq(s, "\n") {
		if k, v, ok := strings.Cut(line, ": "); ok {
			h[k] = append(h[k], v)
		}
	}

	return h
}
