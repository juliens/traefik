package plugins

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"path/filepath"
	"slices"

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

// NewBuilder creates a new Builder.
func NewBuilder(manager *Manager, plugins map[string]Descriptor, localPlugins map[string]LocalDescriptor) (*Builder, error) {
	ctx := context.Background()

	pb := &Builder{
		middlewareBuilders:      map[string]middlewareBuilder{},
		providerBuilders:        map[string]providerBuilder{},
		typedMiddlewareBuilders: map[string]typedMiddlewareBuilder{},
		typedProviderBuilders:   map[string]providerBuilder{},
	}

	for _, pName := range slices.Sorted(maps.Keys(plugins)) {
		desc := plugins[pName]

		manifest, err := manager.ReadManifest(desc.ModuleName)
		if err != nil {
			_ = manager.ResetAll()
			return nil, fmt.Errorf("%s: failed to read manifest: %w", desc.ModuleName, err)
		}

		logger := log.With().
			Str("plugin", "plugin-"+pName).
			Str("module", desc.ModuleName).
			Str("runtime", manifest.Runtime).
			Logger()
		logCtx := logger.WithContext(ctx)

		if err := pb.addPlugin(logCtx, pName, manager.GoPath(), false, manifest, desc.ModuleName, desc.Settings); err != nil {
			return nil, err
		}
	}

	declared := map[string]struct{}{}

	for _, pName := range slices.Sorted(maps.Keys(localPlugins)) {
		desc := localPlugins[pName]
		declared[desc.ModuleName] = struct{}{}

		native, manifest, ok, err := loadBuiltinPlugin(desc.ModuleName)
		if err != nil {
			return nil, err
		}

		if ok {
			if err := pb.addNative(pName, manifest, native); err != nil {
				return nil, fmt.Errorf("%s: %w", desc.ModuleName, err)
			}

			continue
		}

		manifest, err = ReadManifest(localGoPath, desc.ModuleName)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to read manifest: %w", desc.ModuleName, err)
		}

		logger := log.With().
			Str("plugin", "plugin-"+pName).
			Str("module", desc.ModuleName).
			Str("runtime", manifest.Runtime).
			Logger()
		logCtx := logger.WithContext(ctx)

		if err := pb.addPlugin(logCtx, pName, localGoPath, true, manifest, desc.ModuleName, desc.Settings); err != nil {
			return nil, err
		}
	}

	// The plugins compiled into Traefik implement the options that no plugin of the static configuration implements.
	for _, moduleName := range slices.Sorted(maps.Keys(builtinPlugins)) {
		if _, ok := declared[moduleName]; ok {
			continue
		}

		native, manifest, _, err := loadBuiltinPlugin(moduleName)
		if err != nil {
			return nil, err
		}

		if err := pb.addBuiltin(manifest, native); err != nil {
			return nil, fmt.Errorf("%s: %w", moduleName, err)
		}
	}

	return pb, nil
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

// ImplementsMiddleware reports whether a plugin implements the given option of the dynamic configuration of the middlewares.
func (b Builder) ImplementsMiddleware(option string) bool {
	_, ok := b.typedMiddlewareBuilders[option]
	return ok
}

// BuildTyped builds the middleware plugin implementing the given option of the dynamic configuration (headers, stripPrefix, ...),
// with its configuration, a pointer to its type in pkg/config/dynamic.
func (b Builder) BuildTyped(option string, config any, middlewareName string) (Constructor, error) {
	builder, ok := b.typedMiddlewareBuilders[option]
	if !ok {
		return nil, fmt.Errorf("no plugin implements the %q middleware option", option)
	}

	return builder.newTypedMiddleware(option, config, middlewareName).NewHandler, nil
}

// addPlugin adds a Go or Wasm plugin of the static configuration.
// local is true for a local plugin (experimental.localPlugins).
func (b Builder) addPlugin(ctx context.Context, pName, goPath string, local bool, manifest *Manifest, moduleName string, settings Settings) error {
	if len(manifest.Implements) > 0 {
		return fmt.Errorf("%s: only the plugins compiled into Traefik can implement options of the configuration", moduleName)
	}

	switch manifest.Type {
	case typeMiddleware:
		middleware, err := newMiddlewareBuilder(ctx, goPath, local, manifest, moduleName, settings)
		if err != nil {
			return err
		}

		b.middlewareBuilders[pName] = middleware

	case typeProvider:
		builder, err := newProviderBuilder(ctx, manifest, goPath, local, moduleName, settings)
		if err != nil {
			return fmt.Errorf("%s: %w", moduleName, err)
		}

		b.providerBuilders[pName] = builder

	default:
		return fmt.Errorf("unknow plugin type: %s", manifest.Type)
	}

	return nil
}

// addNative adds a plugin of the static configuration compiled into Traefik.
func (b Builder) addNative(pName string, manifest *Manifest, native *nativePlugin) error {
	switch manifest.Type {
	case typeMiddleware:
		builder := &nativeMiddlewareBuilder{plugin: native}
		if len(manifest.Implements) == 0 {
			b.middlewareBuilders[pName] = builder
			return nil
		}

		for _, option := range manifest.Implements {
			if err := builder.checkTypedMiddleware(option); err != nil {
				return err
			}

			if _, exists := b.typedMiddlewareBuilders[option]; exists {
				return fmt.Errorf("two plugins implement the %q middleware option", option)
			}

			b.typedMiddlewareBuilders[option] = builder
		}

	case typeProvider:
		builder := providerBuilder{native: native}
		b.providerBuilders[pName] = builder

		for _, option := range manifest.Implements {
			if _, exists := b.typedProviderBuilders[option]; exists {
				return fmt.Errorf("two plugins implement the %q option", option)
			}

			b.typedProviderBuilders[option] = builder
		}

	default:
		return fmt.Errorf("unknow plugin type: %s", manifest.Type)
	}

	return nil
}

// addBuiltin adds a plugin compiled into Traefik for the options that no plugin of the static configuration implements.
func (b Builder) addBuiltin(manifest *Manifest, native *nativePlugin) error {
	switch manifest.Type {
	case typeMiddleware:
		builder := &nativeMiddlewareBuilder{plugin: native}
		for _, option := range manifest.Implements {
			if _, exists := b.typedMiddlewareBuilders[option]; exists {
				continue
			}

			if err := builder.checkTypedMiddleware(option); err != nil {
				return err
			}

			b.typedMiddlewareBuilders[option] = builder
		}

	case typeProvider:
		for _, option := range manifest.Implements {
			if _, exists := b.typedProviderBuilders[option]; !exists {
				b.typedProviderBuilders[option] = providerBuilder{native: native}
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
