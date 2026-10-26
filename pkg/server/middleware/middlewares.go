package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/plugins/host"

	"github.com/containous/alice"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/middlewares/chain"
	"github.com/traefik/traefik/v3/pkg/middlewares/observability"
	"github.com/traefik/traefik/v3/pkg/server/provider"
	"github.com/traefik/traefik/v3/pkg/server/recursion"
)

// Builder the middleware builder.
type Builder struct {
	configs        map[string]*runtime.MiddlewareInfo
	pluginBuilder  PluginsBuilder
	serviceBuilder serviceBuilder
}

type serviceBuilder interface {
	BuildHTTP(ctx context.Context, serviceName string) (http.Handler, error)
}

// NewBuilder creates a new Builder.
func NewBuilder(configs map[string]*runtime.MiddlewareInfo, serviceBuilder serviceBuilder, pluginBuilder PluginsBuilder) *Builder {
	return &Builder{configs: configs, serviceBuilder: serviceBuilder, pluginBuilder: pluginBuilder}
}

// BuildMiddlewareChain creates a middleware chain.
func (b *Builder) BuildMiddlewareChain(ctx context.Context, middlewares []string) *alice.Chain {
	chain := alice.New()
	for _, name := range middlewares {
		middlewareName := provider.GetQualifiedName(ctx, name)

		chain = chain.Append(func(next http.Handler) (http.Handler, error) {
			constructorContext := provider.AddInContext(ctx, middlewareName)
			if midInf, ok := b.configs[middlewareName]; !ok || midInf.Middleware == nil {
				return nil, fmt.Errorf("middleware %q does not exist", middlewareName)
			}

			var err error
			if constructorContext, err = recursion.CheckRecursion(constructorContext, "middleware", middlewareName); err != nil {
				b.configs[middlewareName].AddError(err, true)
				return nil, err
			}

			constructor, err := b.buildConstructor(constructorContext, middlewareName)
			if err != nil {
				b.configs[middlewareName].AddError(err, true)
				return nil, err
			}

			handler, err := constructor(next)
			if err != nil {
				b.configs[middlewareName].AddError(err, true)
				return nil, err
			}

			return handler, nil
		})
	}
	return &chain
}

// it is the responsibility of the caller to make sure that b.configs[middlewareName].Middleware exists.
func (b *Builder) buildConstructor(ctx context.Context, middlewareName string) (alice.Constructor, error) {
	info := b.configs[middlewareName]
	if info == nil || info.Middleware == nil {
		return nil, fmt.Errorf("invalid middleware %q configuration", middlewareName)
	}

	config, err := info.Middleware.ResolveURNRefs()
	if err != nil {
		return nil, fmt.Errorf("resolving middleware %q configuration: %w", middlewareName, err)
	}

	var middleware alice.Constructor
	badConf := errors.New("cannot create middleware: multi-types middleware not supported, consider declaring two different pieces of middleware instead")

	options := typedOptions(config)
	if len(options) > 1 || (len(options) == 1 && (config.Chain != nil || config.Plugin != nil)) || (config.Chain != nil && config.Plugin != nil) {
		return nil, badConf
	}

	// Chain
	if config.Chain != nil {
		var qualifiedNames []string
		for _, name := range config.Chain.Middlewares {
			qualifiedNames = append(qualifiedNames, provider.GetQualifiedName(ctx, name))
		}
		config.Chain.Middlewares = qualifiedNames
		middleware = func(next http.Handler) (http.Handler, error) {
			return chain.New(ctx, next, *config.Chain, b, middlewareName)
		}
	}

	pluginsEnabled := !isNilBuilder(b.pluginBuilder)

	// The other middlewares of Traefik are plugins implementing an option of the dynamic configuration (headers, stripPrefix, ...).
	if len(options) == 1 {
		option := options[0]
		if !pluginsEnabled {
			return nil, fmt.Errorf("the %q middleware is a plugin, and the plugins are disabled", option.name)
		}

		plug, err := b.pluginBuilder.BuildTyped(option.name, option.value, middlewareName)
		if err != nil {
			return nil, info.RedactURNRefs(err)
		}

		pluginCtx := ctx
		if b.serviceBuilder != nil {
			pluginCtx = host.WithServiceBuilder(ctx, b.serviceBuilder.BuildHTTP)
		}

		middleware = func(next http.Handler) (http.Handler, error) {
			handler, err := newTraceablePlugin(pluginCtx, middlewareName, option.typeName, plug, next)
			if err != nil {
				return nil, info.RedactURNRefs(err)
			}

			return handler, nil
		}
	}

	// Plugin
	if config.Plugin != nil && pluginsEnabled {
		pluginType, rawPluginConfig, err := findPluginConfig(config.Plugin)
		if err != nil {
			return nil, fmt.Errorf("plugin: %w", err)
		}

		// Unlike the other middlewares, the decoding of a plugin configuration puts the invalid values in its errors.
		plug, err := b.pluginBuilder.Build(pluginType, rawPluginConfig, middlewareName)
		if err != nil {
			return nil, info.RedactURNRefs(fmt.Errorf("plugin: %w", err))
		}

		middleware = func(next http.Handler) (http.Handler, error) {
			handler, err := newTraceablePlugin(ctx, middlewareName, typeName, plug, next)
			if err != nil {
				return nil, info.RedactURNRefs(err)
			}

			return handler, nil
		}
	}

	if middleware == nil {
		return nil, fmt.Errorf("invalid middleware %q configuration: invalid middleware type or middleware does not exist", middlewareName)
	}

	return observability.WrapMiddleware(ctx, middleware), nil
}

// typedOption is an option of the dynamic configuration of a middleware, implemented by a plugin.
type typedOption struct {
	// name is the name of the option in the configuration (headers, stripPrefix, ...).
	name string
	// typeName is the type of the middleware in the traces (Headers, StripPrefix, ...).
	typeName string
	value    any
}

// typedOptions returns the options of the configuration implemented by the middleware plugins: the options other than chain and plugin.
func typedOptions(config *dynamic.Middleware) []typedOption {
	var options []typedOption

	val := reflect.ValueOf(config).Elem()
	for i := range val.NumField() {
		field := val.Type().Field(i)
		if field.Type.Kind() != reflect.Pointer || field.Type.Elem().Kind() != reflect.Struct || val.Field(i).IsNil() {
			continue
		}

		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "chain" {
			continue
		}

		options = append(options, typedOption{name: name, typeName: field.Name, value: val.Field(i).Interface()})
	}

	return options
}

// isNilBuilder reports whether the plugins builder is nil, including a nil pointer in the interface.
func isNilBuilder(builder PluginsBuilder) bool {
	if builder == nil {
		return true
	}

	val := reflect.ValueOf(builder)
	return val.Kind() == reflect.Pointer && val.IsNil()
}
