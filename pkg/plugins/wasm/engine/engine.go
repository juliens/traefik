// Package engine holds what the wasm execution modes share: mode detection, configuration, the instance pool and
// the virtual pipes used to exchange bodies with a guest.
package engine

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"slices"
	"sort"
	"time"

	"github.com/rs/zerolog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Mode is the ABI a guest module speaks.
type Mode string

const (
	// ModeHTTPWasm is the http-wasm ABI: the host calls handle_request and handle_response for each request.
	ModeHTTPWasm Mode = "httpwasm"
	// ModeWasiHTTP is the Traefik reactor ABI: the host calls traefik_serve once and delivers requests as fd events.
	ModeWasiHTTP Mode = "wasihttp"
)

const (
	funcHandleRequest  = "handle_request"
	funcHandleResponse = "handle_response"
	funcInitialize     = "_initialize"
	funcServe          = "traefik_serve"

	defaultMaxInstances = 8
	wasmPageSize        = 65536
)

// DetectMode finds the ABI of a compiled guest from its exports.
func DetectMode(m wazero.CompiledModule) (Mode, error) {
	exports := m.ExportedFunctions()

	hasExport := func(name string, params, results []api.ValueType) bool {
		f, ok := exports[name]
		if !ok {
			return false
		}

		return slices.Equal(f.ParamTypes(), params) && slices.Equal(f.ResultTypes(), results)
	}

	httpWasm := hasExport(funcHandleRequest, nil, []api.ValueType{api.ValueTypeI64}) &&
		hasExport(funcHandleResponse, []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, nil)
	wasiHTTP := hasExport(funcInitialize, nil, nil) && hasExport(funcServe, nil, nil)

	switch {
	case httpWasm && wasiHTTP:
		return "", fmt.Errorf("ambiguous guest: exports both %s and %s", funcHandleRequest, funcServe)
	case httpWasm:
		return ModeHTTPWasm, nil
	case wasiHTTP:
		return ModeWasiHTTP, nil
	}

	names := make([]string, 0, len(exports))
	for name := range exports {
		names = append(names, name)
	}
	sort.Strings(names)

	return "", fmt.Errorf("unsupported guest: expected exports %s/%s (http-wasm) or %s/%s (reactor), found %v",
		funcHandleRequest, funcHandleResponse, funcInitialize, funcServe, names)
}

// Config describes one engine: one plugin binary served for one middleware configuration.
type Config struct {
	// Name is the middleware name, used in logs and instance names.
	Name string
	// Code is the guest binary.
	Code []byte
	// Cache is the compilation cache of the plugin, shared by every engine of the plugin.
	Cache wazero.CompilationCache
	// Instances is the number of guest instances; 0 means min(GOMAXPROCS, 8).
	Instances int
	// MemoryLimit is the maximum linear memory of one instance in bytes; 0 keeps the wazero default.
	MemoryLimit int64
	// Exclusive dedicates one instance to each in-flight request (http-wasm only).
	Exclusive bool
	// CallTimeout bounds each call to the guest; 0 means no limit (http-wasm only).
	CallTimeout time.Duration
	// BodyPrefetchSize is how much of the request body is read before the guest is called; 0 means the default
	// (http-wasm only).
	BodyPrefetchSize int
	// AllowOutbound lets the guest send outbound HTTP requests (wasm-go plugins only).
	AllowOutbound bool
	// Envs lists the environment variables forwarded to the guest.
	Envs []string
	// Mounts lists the directories mounted in the guest (http-wasm only).
	Mounts []string
	// GuestConfig is the middleware configuration as JSON.
	GuestConfig []byte
	// Logger is the middleware logger.
	Logger *zerolog.Logger
}

// InstanceCount returns the number of instances to start.
func (c Config) InstanceCount() int {
	if c.Instances > 0 {
		return c.Instances
	}

	return min(runtime.GOMAXPROCS(0), defaultMaxInstances)
}

// RuntimeConfig returns the wazero runtime configuration of an instance.
func (c Config) RuntimeConfig() wazero.RuntimeConfig {
	cfg := wazero.NewRuntimeConfig().WithCompilationCache(c.Cache)
	if c.MemoryLimit > 0 {
		pages := (c.MemoryLimit + wasmPageSize - 1) / wasmPageSize
		cfg = cfg.WithMemoryLimitPages(uint32(min(pages, 1<<16)))
	}

	// Interrupting a call on its deadline makes the compiled code check the context in loops, which has a cost:
	// it is only enabled when a timeout is set.
	if c.CallTimeout > 0 {
		cfg = cfg.WithCloseOnContextDone(true)
	}

	return cfg
}

// Engine serves requests for one plugin binary and one middleware configuration.
// The next handler is given per call because one middleware configuration is shared by every router chain that
// references it.
type Engine interface {
	ServeHTTP(rw http.ResponseWriter, req *http.Request, next http.Handler)
	Stats() Stats
	Close(ctx context.Context) error
}

// Stats is a snapshot of an engine, mostly for tests.
type Stats struct {
	// InFlight is the number of requests being served.
	InFlight int64
	// LiveInstances is the number of instances able to serve requests.
	LiveInstances int
}

var (
	// ErrNoInstance is returned when no instance can serve a request.
	ErrNoInstance = errors.New("no wasm instance available")
	// ErrInstanceDied is returned to requests in flight on an instance that trapped.
	ErrInstanceDied = errors.New("wasm instance stopped")
	// ErrClosing is returned when the engine is closing.
	ErrClosing = errors.New("wasm engine is closing")
)
