package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/traefik/traefik/v3/pkg/middlewares"
	"github.com/traefik/traefik/v3/pkg/plugins/wasm/engine"
	"github.com/traefik/traefik/v3/pkg/plugins/wasm/httpwasm"
	"github.com/traefik/traefik/v3/pkg/plugins/wasm/wasihttp"
)

// wasmMiddlewareBuilder holds what is shared by every middleware of a wasm plugin: the compiled guest and its
// settings, and the engines, one per middleware configuration.
type wasmMiddlewareBuilder struct {
	code     []byte
	mode     engine.Mode
	cache    wazero.CompilationCache
	settings Settings

	mu      sync.Mutex
	engines map[string]engine.Engine
}

func newWasmMiddlewareBuilder(ctx context.Context, goPath, moduleName, wasmPath string, settings Settings) (*wasmMiddlewareBuilder, error) {
	return newWasmMiddlewareBuilderFromPath(ctx, filepath.Join(goPath, "src", moduleName, wasmPath), settings)
}

func newWasmMiddlewareBuilderFromPath(ctx context.Context, path string, settings Settings) (*wasmMiddlewareBuilder, error) {
	code, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading Wasm binary: %w", err)
	}

	cache := wazero.NewCompilationCache()

	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCompilationCache(cache))
	defer func() { _ = rt.Close(ctx) }()

	compiled, err := rt.CompileModule(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("compiling guest module: %w", err)
	}

	mode, err := engine.DetectMode(compiled)
	if err != nil {
		return nil, fmt.Errorf("detecting guest ABI: %w", err)
	}

	if mode == engine.ModeWasiHTTP {
		if err := wasihttp.ValidateSettings(settings.Mounts, settings.Exclusive); err != nil {
			return nil, err
		}
	}

	if settings.Instances < 0 {
		return nil, errors.New("settings.instances must be positive")
	}

	if settings.MemoryLimit < 0 {
		return nil, errors.New("settings.memoryLimit must be positive")
	}

	return &wasmMiddlewareBuilder{
		code:     code,
		mode:     mode,
		cache:    cache,
		settings: settings,
		engines:  map[string]engine.Engine{},
	}, nil
}

// Close closes every engine of the plugin.
func (b *wasmMiddlewareBuilder) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var errs []error
	for key, eng := range b.engines {
		if err := eng.Close(ctx); err != nil {
			errs = append(errs, err)
		}
		delete(b.engines, key)
	}

	return errors.Join(errs...)
}

func (b *wasmMiddlewareBuilder) newMiddleware(config map[string]any, middlewareName string) (pluginMiddleware, error) {
	// encoding/json sorts map keys, so the same configuration always gives the same bytes.
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	return &WasmMiddleware{
		middlewareName: middlewareName,
		config:         data,
		builder:        b,
	}, nil
}

// engineFor returns the engine of a middleware configuration, creating it on first use.
// One middleware configuration may be referenced by several router chains: they all share the engine.
func (b *wasmMiddlewareBuilder) engineFor(ctx context.Context, middlewareName string, config []byte) (engine.Engine, error) {
	key := middlewareName + "\x00" + string(config)

	b.mu.Lock()
	defer b.mu.Unlock()

	if eng, ok := b.engines[key]; ok {
		return eng, nil
	}

	logger := middlewares.GetLogger(ctx, middlewareName, "wasm")

	cfg := engine.Config{
		Name:        middlewareName,
		Code:        b.code,
		Cache:       b.cache,
		Instances:   b.settings.Instances,
		MemoryLimit: b.settings.MemoryLimit,
		Exclusive:   b.settings.Exclusive,
		Envs:        b.settings.Envs,
		Mounts:      b.settings.Mounts,
		GuestConfig: config,
		Logger:      logger,
	}

	var (
		eng engine.Engine
		err error
	)
	switch b.mode {
	case engine.ModeHTTPWasm:
		eng, err = httpwasm.New(ctx, cfg)
	case engine.ModeWasiHTTP:
		eng, err = wasihttp.New(ctx, cfg)
	default:
		err = fmt.Errorf("unsupported guest ABI %q", b.mode)
	}
	if err != nil {
		return nil, fmt.Errorf("creating wasm engine: %w", err)
	}

	logger.Debug().Str("mode", string(b.mode)).Int("instances", cfg.InstanceCount()).Msg("Wasm engine created")

	b.engines[key] = eng

	return eng, nil
}

// WasmMiddleware is an HTTP handler plugin wrapper.
type WasmMiddleware struct {
	middlewareName string
	config         []byte
	builder        *wasmMiddlewareBuilder
}

// NewHandler creates a new HTTP handler.
func (m *WasmMiddleware) NewHandler(ctx context.Context, next http.Handler) (http.Handler, error) {
	eng, err := m.builder.engineFor(ctx, m.middlewareName, m.config)
	if err != nil {
		return nil, fmt.Errorf("building Wasm middleware: %w", err)
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		eng.ServeHTTP(rw, req, next)
	}), nil
}
