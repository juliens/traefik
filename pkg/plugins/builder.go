package plugins

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
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

// typedMiddlewareBuilder is the builder of a middleware plugin implementing options of the dynamic configuration (see Manifest.Implements).
type typedMiddlewareBuilder interface {
	newTypedMiddleware(option string, config any, middlewareName string) pluginMiddleware
	checkTypedMiddleware(option string) error
}

// Builder is a plugin builder.
type Builder struct {
	providerBuilders   map[string]providerBuilder
	middlewareBuilders map[string]middlewareBuilder

	// typedMiddlewareBuilders are the middleware plugins implementing an option of the dynamic configuration, by option.
	typedMiddlewareBuilders map[string]typedMiddlewareBuilder

	// typedProviderBuilders are the provider plugins implementing an option of the providers of the static configuration, by option.
	typedProviderBuilders map[string]providerBuilder
}

// pluginToLoad is a plugin of the static configuration, with its manifest.
type pluginToLoad struct {
	name       string
	manifest   *Manifest
	goPath     string
	local      bool
	moduleName string
	settings   Settings
	// native is the plugin when it runs natively (compiled into Traefik, or a shared object), already loaded to read its manifest.
	native *nativePlugin
	// mode is where the plugin comes from: builtin or go-vm.
	mode string

	middleware middlewareBuilder
	provider   providerBuilder
	err        error
}

// NewBuilder creates a new Builder.
func NewBuilder(manager *Manager, plugins map[string]Descriptor, localPlugins map[string]LocalDescriptor) (*Builder, error) {
	pb := &Builder{
		middlewareBuilders:      map[string]middlewareBuilder{},
		providerBuilders:        map[string]providerBuilder{},
		typedMiddlewareBuilders: map[string]typedMiddlewareBuilder{},
		typedProviderBuilders:   map[string]providerBuilder{},
	}

	var toLoad []*pluginToLoad

	for pName, desc := range plugins {
		manifest, err := manager.ReadManifest(desc.ModuleName)
		if err != nil {
			_ = manager.ResetAll()
			return nil, fmt.Errorf("%s: failed to read manifest: %w", desc.ModuleName, err)
		}

		toLoad = append(toLoad, &pluginToLoad{name: pName, manifest: manifest, goPath: manager.GoPath(), moduleName: desc.ModuleName, settings: desc.Settings, mode: modeGoVM})
	}

	for pName, desc := range localPlugins {
		plugin, err := resolveLocalPlugin(pName, desc)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", desc.ModuleName, err)
		}

		toLoad = append(toLoad, plugin)
	}

	slices.SortFunc(toLoad, func(a, b *pluginToLoad) int { return cmp.Compare(a.name, b.name) })

	// The middlewares and providers of Traefik are plugins: loading them one after the other takes minutes.
	loadPlugins(toLoad)

	for _, plugin := range toLoad {
		if plugin.err != nil {
			return nil, plugin.err
		}

		if err := pb.add(plugin); err != nil {
			return nil, fmt.Errorf("%s: %w", plugin.moduleName, err)
		}
	}

	// The plugins compiled into Traefik implement the options that no plugin of the static configuration implements,
	// so that Traefik works as before without plugin declarations.
	declared := map[string]struct{}{}
	for _, plugin := range toLoad {
		declared[plugin.moduleName] = struct{}{}
	}

	for _, moduleName := range slices.Sorted(maps.Keys(builtinPlugins)) {
		if _, ok := declared[moduleName]; ok {
			continue
		}

		native, manifest, _, err := loadBuiltinPlugin(moduleName)
		if err != nil {
			return nil, err
		}

		if err := pb.addBuiltin(&pluginToLoad{manifest: manifest, moduleName: moduleName, native: native}); err != nil {
			return nil, fmt.Errorf("%s: %w", moduleName, err)
		}
	}

	return pb, nil
}

const (
	modeBuiltin = "builtin"
	modeGoVM    = "go-vm"
)

// resolveLocalPlugin finds a local plugin by its module name:
// compiled into Traefik, or as sources in plugins-local, run by go-vm.
func resolveLocalPlugin(pName string, desc LocalDescriptor) (*pluginToLoad, error) {
	native, manifest, ok, err := loadBuiltinPlugin(desc.ModuleName)
	if err != nil {
		return nil, err
	}
	if ok {
		return &pluginToLoad{name: pName, manifest: manifest, moduleName: desc.ModuleName, native: native, mode: modeBuiltin}, nil
	}

	manifest, err = ReadManifest(localGoPath, desc.ModuleName)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	return &pluginToLoad{name: pName, manifest: manifest, goPath: localGoPath, local: true, moduleName: desc.ModuleName, settings: desc.Settings, mode: modeGoVM}, nil
}

// loadPlugins loads the plugins concurrently, each module in its own interpreter.
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
				Str("mode", plugin.mode).
				Logger()
			logCtx := logger.WithContext(ctx)

			logger.Debug().Msg("Loading the plugin")

			switch {
			case plugin.native != nil && plugin.manifest.Type == typeMiddleware:
				plugin.middleware = &nativeMiddlewareBuilder{plugin: plugin.native}

			case plugin.native != nil && plugin.manifest.Type == typeProvider:
				plugin.provider = providerBuilder{native: plugin.native}

			case plugin.manifest.Type == typeMiddleware:
				plugin.middleware, plugin.err = newMiddlewareBuilder(logCtx, plugin.goPath, plugin.local, plugin.manifest, plugin.moduleName, plugin.settings)

			case plugin.manifest.Type == typeProvider:
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

// BuildTyped builds the middleware plugin implementing the given option of the dynamic configuration (headers, stripPrefix, ...),
// with its configuration, a pointer to its type in pkg/config/dynamic.
func (b Builder) BuildTyped(option string, config any, middlewareName string) (Constructor, error) {
	builder, ok := b.typedMiddlewareBuilders[option]
	if !ok {
		return nil, fmt.Errorf("the %q middleware is a plugin: no plugin of experimental.plugins or experimental.localPlugins implements it", option)
	}

	return builder.newTypedMiddleware(option, config, middlewareName).NewHandler, nil
}

// addMiddlewareBuilder adds the builder of a middleware plugin,
// as the implementation of an option of the dynamic configuration when its manifest says so.
func (b Builder) addMiddlewareBuilder(pName string, manifest *Manifest, builder middlewareBuilder) error {
	if len(manifest.Implements) == 0 {
		b.middlewareBuilders[pName] = builder
		return nil
	}

	typedBuilder, ok := builder.(typedMiddlewareBuilder)
	if !ok {
		return fmt.Errorf("the plugin implements middleware options %v, which needs the Go runtime", manifest.Implements)
	}

	for _, option := range manifest.Implements {
		if err := typedBuilder.checkTypedMiddleware(option); err != nil {
			return err
		}

		if _, exists := b.typedMiddlewareBuilders[option]; exists {
			return fmt.Errorf("two plugins implement the %q middleware option", option)
		}

		b.typedMiddlewareBuilders[option] = typedBuilder
	}

	return nil
}

// add adds a plugin of the static configuration.
func (b Builder) add(plugin *pluginToLoad) error {
	switch plugin.manifest.Type {
	case typeMiddleware:
		return b.addMiddlewareBuilder(plugin.name, plugin.manifest, plugin.middleware)

	case typeProvider:
		b.providerBuilders[plugin.name] = plugin.provider

		for _, option := range plugin.manifest.Implements {
			if _, exists := b.typedProviderBuilders[option]; exists {
				return fmt.Errorf("two plugins implement the %q option", option)
			}

			b.typedProviderBuilders[option] = plugin.provider
		}
	}

	return nil
}

// addBuiltin adds a plugin compiled into Traefik for the options that no plugin of the static configuration implements.
func (b Builder) addBuiltin(plugin *pluginToLoad) error {
	switch plugin.manifest.Type {
	case typeMiddleware:
		builder := &nativeMiddlewareBuilder{plugin: plugin.native}
		for _, option := range plugin.manifest.Implements {
			if _, exists := b.typedMiddlewareBuilders[option]; exists {
				continue
			}

			if err := builder.checkTypedMiddleware(option); err != nil {
				return err
			}

			b.typedMiddlewareBuilders[option] = builder
		}

	case typeProvider:
		for _, option := range plugin.manifest.Implements {
			if _, exists := b.typedProviderBuilders[option]; !exists {
				b.typedProviderBuilders[option] = providerBuilder{native: plugin.native}
			}
		}
	}

	return nil
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
