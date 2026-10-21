package plugins

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	// The AOT copies of the standard library and of common libraries, used by vm.TraefikDefaults.
	_ "github.com/traefik/go-vm/aot/all"
	"github.com/traefik/go-vm/vm"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
)

type goVMMiddlewareBuilder struct {
	plugin *vm.Plugin
}

// newGoVMMiddlewareBuilder loads a Go middleware plugin with the go-vm interpreter.
func newGoVMMiddlewareBuilder(ctx context.Context, goPath string, manifest *Manifest, moduleName string, settings Settings) (*goVMMiddlewareBuilder, error) {
	p, err := loadGoVMPlugin(ctx, goPath, manifest, moduleName, settings)
	if err != nil {
		return nil, err
	}

	return &goVMMiddlewareBuilder{plugin: p}, nil
}

// loadGoVMPlugin loads a Go plugin with the go-vm interpreter.
// It keeps at least the restrictions of the former Yaegi runtime: restricted standard library,
// unsafe and syscall only when both the manifest and the settings allow them,
// and a copy of the process environment.
// Unrestricted and RealProcess must never be set: they would give the plugin
// os/exec, syscall, and the ability to stop Traefik.
func loadGoVMPlugin(ctx context.Context, goPath string, manifest *Manifest, moduleName string, settings Settings) (*vm.Plugin, error) {
	if manifest.UseUnsafe && !settings.UseUnsafe {
		return nil, errors.New("this plugin uses restricted imports. If you want to use it, you need to allow useUnsafe in the settings")
	}

	logger := log.Ctx(ctx)

	// TraefikDefaults bounds the plugin calls (budget), the plugin goroutines, and the plugin load,
	// and runs the AOT copies registered by the aot/all import.
	opts := vm.TraefikDefaults()
	opts.Dir = filepath.Join(goPath, goPathSrc, filepath.FromSlash(moduleName))
	opts.UseUnsafe = manifest.UseUnsafe && settings.UseUnsafe
	// TraefikDefaults gives an empty environment, but the plugins have always read the environment of Traefik.
	opts.Env = os.Environ()
	opts.Stdout = logs.NoLevel(*logger, zerolog.DebugLevel)
	opts.Stderr = logs.NoLevel(*logger, zerolog.ErrorLevel)
	// A panic in a goroutine started by the plugin must not stop Traefik.
	opts.GoPanic = func(r any) {
		logger.Error().Msgf("Panic in a goroutine of the plugin: %v", r)
	}

	p, err := vm.Load(opts, manifest.Import)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin code %q: %w", manifest.Import, err)
	}

	if p.Pkg.Func("CreateConfig") == nil {
		return nil, errors.New("failed to find CreateConfig")
	}

	if p.Pkg.Func("New") == nil {
		return nil, errors.New("failed to find New")
	}

	return p, nil
}

func (b *goVMMiddlewareBuilder) newMiddleware(config map[string]any, middlewareName string) (pluginMiddleware, error) {
	cfg, err := b.plugin.CreateConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to call CreateConfig: %w", err)
	}

	if len(config) > 0 {
		err = b.plugin.DecodeConfig(cfg, func(result any) error {
			return decodePluginConfig(config, result)
		})
		if err != nil {
			return nil, err
		}
	}

	return &GoVMMiddleware{
		middlewareName: middlewareName,
		config:         cfg,
		plugin:         b.plugin,
	}, nil
}

// decodePluginConfig decodes the configuration with the same options as the Yaegi runtime.
func decodePluginConfig(config map[string]any, result any) error {
	cfg := &mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.StringToSliceHookFunc(","),
		WeaklyTypedInput: true,
		Result:           result,
	}

	decoder, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		return fmt.Errorf("failed to create configuration decoder: %w", err)
	}

	if err := decoder.Decode(config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return nil
}

// GoVMMiddleware is an HTTP handler plugin wrapper for the go-vm interpreter.
type GoVMMiddleware struct {
	middlewareName string
	config         vm.Value
	plugin         *vm.Plugin
}

// NewHandler creates a new HTTP handler.
func (m *GoVMMiddleware) NewHandler(ctx context.Context, next http.Handler) (http.Handler, error) {
	return m.plugin.New(ctx, next, m.config, m.middlewareName)
}
