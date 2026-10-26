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
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/traefik/traefik/v3/pkg/middlewares"
	"github.com/traefik/traefik/v3/pkg/plugins/wasm/engine"
	"github.com/traefik/traefik/v3/pkg/plugins/wasm/httpwasm"
	"github.com/traefik/traefik/v3/pkg/plugins/wasm/wasihttp"
)

// engineDrainTimeout bounds how long an engine that the routers no longer use waits for its in-flight requests
// before it is closed.
const engineDrainTimeout = time.Minute

// wasmMiddlewareBuilder holds what is shared by every middleware of a wasm plugin: the compiled guest and its
// settings, and the engines, one per middleware configuration.
type wasmMiddlewareBuilder struct {
	code     []byte
	mode     engine.Mode
	cache    wazero.CompilationCache
	settings Settings

	mu sync.Mutex
	// generation counts the router configurations; an engine is kept as long as the last one uses it.
	generation uint64
	engines    map[string]*engineEntry
}

type engineEntry struct {
	engine     engine.Engine
	name       string
	generation uint64
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
		if err := wasihttp.ValidateSettings(settingsConfig(settings)); err != nil {
			return nil, err
		}
	}

	switch {
	case settings.Instances < 0:
		return nil, errors.New("settings.instances must be positive")
	case settings.MemoryLimit < 0:
		return nil, errors.New("settings.memoryLimit must be positive")
	case settings.CallTimeout < 0:
		return nil, errors.New("settings.callTimeout must be positive")
	case settings.BodyPrefetchSize < 0:
		return nil, errors.New("settings.bodyPrefetchSize must be positive")
	}

	return &wasmMiddlewareBuilder{
		code:     code,
		mode:     mode,
		cache:    cache,
		settings: settings,
		engines:  map[string]*engineEntry{},
	}, nil
}

// Close closes every engine of the plugin.
func (b *wasmMiddlewareBuilder) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var errs []error
	for key, entry := range b.engines {
		if err := entry.engine.Close(ctx); err != nil {
			errs = append(errs, err)
		}
		delete(b.engines, key)
	}

	return errors.Join(errs...)
}

// beginGeneration starts a new router configuration: the engines it uses are marked as it builds its middlewares.
func (b *wasmMiddlewareBuilder) beginGeneration() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.generation++
}

// endGeneration closes, in the background, the engines that the current router configuration does not use.
// The routers of the previous configuration may still serve requests: each engine waits for its in-flight requests,
// up to engineDrainTimeout, before closing its instances.
func (b *wasmMiddlewareBuilder) endGeneration(ctx context.Context) {
	b.mu.Lock()
	var unused []*engineEntry
	for key, entry := range b.engines {
		if entry.generation < b.generation {
			unused = append(unused, entry)
			delete(b.engines, key)
		}
	}
	b.mu.Unlock()

	for _, entry := range unused {
		go func() {
			logger := middlewares.GetLogger(ctx, entry.name, "wasm")

			drainCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), engineDrainTimeout)
			defer cancel()

			if err := entry.engine.Close(drainCtx); err != nil {
				logger.Warn().Err(err).Msg("Closing unused wasm engine")

				return
			}

			logger.Debug().Msg("Unused wasm engine closed")
		}()
	}
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

	if entry, ok := b.engines[key]; ok {
		entry.generation = b.generation

		return entry.engine, nil
	}

	logger := middlewares.GetLogger(ctx, middlewareName, "wasm")

	cfg := settingsConfig(b.settings)
	cfg.Name = middlewareName
	cfg.Code = b.code
	cfg.Cache = b.cache
	cfg.GuestConfig = config
	cfg.Logger = logger

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

	b.engines[key] = &engineEntry{engine: eng, name: middlewareName, generation: b.generation}

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

// settingsConfig returns the part of an engine configuration that comes from the plugin settings.
func settingsConfig(settings Settings) engine.Config {
	return engine.Config{
		Instances:        settings.Instances,
		MemoryLimit:      settings.MemoryLimit,
		Exclusive:        settings.Exclusive,
		CallTimeout:      time.Duration(settings.CallTimeout),
		BodyPrefetchSize: settings.BodyPrefetchSize,
		Envs:             settings.Envs,
		Mounts:           settings.Mounts,
	}
}
