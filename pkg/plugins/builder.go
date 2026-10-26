package plugins

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"github.com/rs/zerolog/log"
)

// Constructor creates a plugin handler.
type Constructor func(context.Context, http.Handler) (http.Handler, error)

type pluginMiddleware interface {
	NewHandler(ctx context.Context, next http.Handler) (http.Handler, error)
}

type middlewareBuilder interface {
	newMiddleware(config map[string]any, middlewareName string) (pluginMiddleware, error)
}

// Builder is a plugin builder.
type Builder struct {
	providerBuilders   map[string]providerBuilder
	middlewareBuilders map[string]middlewareBuilder

	// typedMiddlewareBuilders are the middleware plugins implementing an option of the dynamic configuration, by option.
	typedMiddlewareBuilders map[string]middlewareBuilder
}

// pluginToLoad is a plugin of the static configuration, with its manifest.
type pluginToLoad struct {
	name       string
	manifest   *Manifest
	goPath     string
	local      bool
	moduleName string
	settings   Settings

	middleware middlewareBuilder
	provider   providerBuilder
	err        error
}

// NewBuilder creates a new Builder.
func NewBuilder(manager *Manager, plugins map[string]Descriptor, localPlugins map[string]LocalDescriptor) (*Builder, error) {
	pb := &Builder{
		middlewareBuilders:      map[string]middlewareBuilder{},
		providerBuilders:        map[string]providerBuilder{},
		typedMiddlewareBuilders: map[string]middlewareBuilder{},
	}

	var toLoad []*pluginToLoad

	for pName, desc := range plugins {
		manifest, err := manager.ReadManifest(desc.ModuleName)
		if err != nil {
			_ = manager.ResetAll()
			return nil, fmt.Errorf("%s: failed to read manifest: %w", desc.ModuleName, err)
		}

		toLoad = append(toLoad, &pluginToLoad{name: pName, manifest: manifest, goPath: manager.GoPath(), moduleName: desc.ModuleName, settings: desc.Settings})
	}

	for pName, desc := range localPlugins {
		manifest, err := ReadManifest(localGoPath, desc.ModuleName)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to read manifest: %w", desc.ModuleName, err)
		}

		toLoad = append(toLoad, &pluginToLoad{name: pName, manifest: manifest, goPath: localGoPath, local: true, moduleName: desc.ModuleName, settings: desc.Settings})
	}

	slices.SortFunc(toLoad, func(a, b *pluginToLoad) int { return cmp.Compare(a.name, b.name) })

	// The middlewares and providers of Traefik are plugins: loading them one after the other takes minutes.
	loadPlugins(toLoad)

	for _, plugin := range toLoad {
		if plugin.err != nil {
			return nil, plugin.err
		}

		switch plugin.manifest.Type {
		case typeMiddleware:
			if err := pb.addMiddlewareBuilder(plugin.name, plugin.manifest, plugin.middleware); err != nil {
				return nil, fmt.Errorf("%s: %w", plugin.moduleName, err)
			}

		case typeProvider:
			pb.providerBuilders[plugin.name] = plugin.provider
		}
	}

	return pb, nil
}

// loadPlugins loads the plugins concurrently, each one in its own interpreter.
func loadPlugins(toLoad []*pluginToLoad) {
	ctx := context.Background()

	sem := make(chan struct{}, runtime.GOMAXPROCS(0))

	var wg sync.WaitGroup
	for _, plugin := range toLoad {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			logger := log.With().
				Str("plugin", "plugin-"+plugin.name).
				Str("module", plugin.moduleName).
				Str("runtime", plugin.manifest.Runtime).
				Logger()
			logCtx := logger.WithContext(ctx)

			switch plugin.manifest.Type {
			case typeMiddleware:
				plugin.middleware, plugin.err = newMiddlewareBuilder(logCtx, plugin.goPath, plugin.local, plugin.manifest, plugin.moduleName, plugin.settings)

			case typeProvider:
				plugin.provider, plugin.err = newProviderBuilder(logCtx, plugin.manifest, plugin.goPath, plugin.local, plugin.moduleName, plugin.settings)
				if plugin.err != nil {
					plugin.err = fmt.Errorf("%s: %w", plugin.moduleName, plugin.err)
				}

			default:
				plugin.err = fmt.Errorf("unknow plugin type: %s", plugin.manifest.Type)
			}
		})
	}

	wg.Wait()
}

// Build builds a middleware plugin.
func (b Builder) Build(pName string, config map[string]any, middlewareName string) (Constructor, error) {
	if len(b.middlewareBuilders) == 0 {
		return nil, fmt.Errorf("no plugin definitions in the static configuration: %s", pName)
	}

	// plugin (pName) can be located in yaegi or wasm middleware builders.
	if descriptor, ok := b.middlewareBuilders[pName]; ok {
		m, err := descriptor.newMiddleware(config, middlewareName)
		if err != nil {
			return nil, err
		}

		return m.NewHandler, nil
	}

	return nil, fmt.Errorf("unknown plugin type: %s", pName)
}

// addMiddlewareBuilder adds the builder of a middleware plugin,
// as the implementation of an option of the dynamic configuration when its manifest says so.
func (b Builder) addMiddlewareBuilder(pName string, manifest *Manifest, builder middlewareBuilder) error {
	if manifest.Implements == "" {
		b.middlewareBuilders[pName] = builder
		return nil
	}

	goVMBuilder, ok := builder.(*goVMMiddlewareBuilder)
	if !ok {
		return fmt.Errorf("the plugin implements the %q middleware option, which needs the Go runtime", manifest.Implements)
	}

	if err := checkTypedMiddleware(goVMBuilder.plugin, manifest.Implements); err != nil {
		return err
	}

	if _, exists := b.typedMiddlewareBuilders[manifest.Implements]; exists {
		return fmt.Errorf("two plugins implement the %q middleware option", manifest.Implements)
	}

	b.typedMiddlewareBuilders[manifest.Implements] = goVMBuilder

	return nil
}

// BuildTyped builds the middleware plugin implementing the given option of the dynamic configuration (headers, stripPrefix, ...),
// with its configuration, a pointer to its type in pkg/config/dynamic.
func (b Builder) BuildTyped(option string, config any, middlewareName string) (Constructor, error) {
	builder, ok := b.typedMiddlewareBuilders[option]
	if !ok {
		return nil, fmt.Errorf("the %q middleware is a plugin: no plugin of experimental.plugins or experimental.localPlugins implements it", option)
	}

	goVMBuilder, ok := builder.(*goVMMiddlewareBuilder)
	if !ok {
		return nil, fmt.Errorf("invalid builder for the %q middleware", option)
	}

	return goVMBuilder.newTypedMiddleware(config, middlewareName).NewHandler, nil
}

// local is true for a local plugin (experimental.localPlugins).
func newMiddlewareBuilder(ctx context.Context, goPath string, local bool, manifest *Manifest, moduleName string, settings Settings) (middlewareBuilder, error) {
	switch manifest.Runtime {
	case runtimeWasm:
		wasmPath, err := getWasmPath(manifest)
		if err != nil {
			return nil, fmt.Errorf("wasm path: %w", err)
		}

		return newWasmMiddlewareBuilder(goPath, moduleName, wasmPath, settings)

	case runtimeYaegi, "":
		return newGoVMMiddlewareBuilder(ctx, goPath, local, manifest, moduleName, settings)

	default:
		return nil, fmt.Errorf("unknown plugin runtime: %s", manifest.Runtime)
	}
}

// local is true for a local plugin (experimental.localPlugins).
func newProviderBuilder(ctx context.Context, manifest *Manifest, goPath string, local bool, moduleName string, settings Settings) (providerBuilder, error) {
	switch manifest.Runtime {
	case runtimeYaegi, "":
		p, err := loadGoVMPlugin(ctx, goPath, local, manifest, moduleName, settings)
		if err != nil {
			return providerBuilder{}, err
		}

		return providerBuilder{plugin: p}, nil

	default:
		return providerBuilder{}, fmt.Errorf("unknown plugin runtime: %s", manifest.Runtime)
	}
}

func getWasmPath(manifest *Manifest) (string, error) {
	wasmPath := manifest.WasmPath
	if wasmPath == "" {
		wasmPath = "plugin.wasm"
	}

	if !filepath.IsLocal(wasmPath) {
		return "", errors.New("wasmPath must be a local path")
	}

	return wasmPath, nil
}
