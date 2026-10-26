package httpwasm

import (
	"bytes"

	handlerapi "github.com/http-wasm/http-wasm-host-go/api/handler"
	wazeroapi "github.com/tetratelabs/wazero/api"
)

func writeNULTerminated(mem wazeroapi.Memory, buf uint32, bufLimit handlerapi.BufLimit, input []string) handlerapi.CountLen {
	count := uint32(len(input))
	if count == 0 {
		return 0
	}

	byteCount := count // NUL terminator count
	for _, s := range input {
		byteCount += uint32(len(s))
	}

	countLen := handlerapi.CountLen(count)<<32 | handlerapi.CountLen(byteCount)

	if byteCount > bufLimit {
		return countLen // the guest can retry with a larger limit
	}

	// Write the NUL-terminated string to memory directly.
	s, ok := mem.Read(buf, byteCount)
	if !ok {
		panic("out of memory") // the guest passed a region outside memory.
	}

	b := bytes.NewBuffer(s)
	b.Reset()
	for _, h := range input {
		b.WriteString(h)
		b.WriteByte(0)
	}

	return countLen
}
