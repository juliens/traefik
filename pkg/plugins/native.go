package plugins

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"time"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/provider"
	"gopkg.in/yaml.v3"
)

// nativePlugin is a plugin of the plugins module of Traefik that runs natively:
//   - compiled into Traefik (go build -tags builtinplugins), it registers itself with RegisterBuiltin;
//   - built with -buildmode=plugin (make plugins-so), Traefik opens its shared object.
//
// Either way, it is built from the same sources and module versions as Traefik, so it shares the packages of Traefik:
// its providers send the dynamic.Message of Traefik, and its middlewares take the types of pkg/config/dynamic.
// It gives its manifest, and its functions (CreateConfig, New, New<Option>), see cmd/internal/nativeplugins.
type nativePlugin struct {
	// source is the module name of a compiled-in plugin, or the path of a shared object.
	source string
	lookup func(name string) (any, error)
}

// builtinPlugins are the plugins compiled into Traefik, by module name.
var builtinPlugins = map[string]nativePlugin{}

// builtinManifests are the manifests of the plugins compiled into Traefik, by module name.
var builtinManifests = map[string]string{}

// RegisterBuiltin registers a plugin compiled into Traefik, with its manifest and its functions by name.
// The generated file of cmd/internal/nativeplugins calls it, with the builtinplugins build tag.
func RegisterBuiltin(moduleName, manifest string, functions map[string]any) {
	builtinManifests[moduleName] = manifest
	builtinPlugins[moduleName] = nativePlugin{
		source: moduleName,
		lookup: func(name string) (any, error) {
			fn, ok := functions[name]
			if !ok {
				return nil, fmt.Errorf("symbol %s not found", name)
			}

			return fn, nil
		},
	}
}

// loadBuiltinPlugin returns the plugin compiled into Traefik with the given module name, if any.
func loadBuiltinPlugin(moduleName string) (*nativePlugin, *Manifest, bool, error) {
	p, ok := builtinPlugins[moduleName]
	if !ok {
		return nil, nil, false, nil
	}

	manifest, err := decodeManifest(moduleName, builtinManifests[moduleName])
	if err != nil {
		return nil, nil, true, err
	}

	return &p, manifest, true, nil
}

// sharedPluginPath returns the path of the shared object of a plugin: plugins/<module name>.so, next to the binary of Traefik.
func sharedPluginPath(moduleName string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(exe), "plugins", filepath.FromSlash(moduleName)+".so"), nil
}

// runsNatively reports whether the local plugin with the given module name is compiled into Traefik, or has a shared object.
func runsNatively(moduleName string) bool {
	if _, ok := builtinPlugins[moduleName]; ok {
		return true
	}

	path, err := sharedPluginPath(moduleName)
	if err != nil {
		return false
	}

	_, err = os.Stat(path)

	return err == nil
}

// loadSharedPlugin opens the shared object of a plugin, and reads its manifest.
func loadSharedPlugin(path string) (*nativePlugin, *Manifest, error) {
	lib, err := plugin.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open the shared object %s: %w", path, err)
	}

	sym, err := lib.Lookup("Manifest")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find the manifest of the shared object %s: %w", path, err)
	}

	content, ok := sym.(*string)
	if !ok {
		return nil, nil, fmt.Errorf("the manifest of the shared object %s is a %T, a string is expected", path, sym)
	}

	manifest, err := decodeManifest(path, *content)
	if err != nil {
		return nil, nil, err
	}

	lookup := func(name string) (any, error) {
		return lib.Lookup(name)
	}

	return &nativePlugin{source: path, lookup: lookup}, manifest, nil
}

func decodeManifest(source, content string) (*Manifest, error) {
	manifest := &Manifest{}
	if err := yaml.Unmarshal([]byte(content), manifest); err != nil {
		return nil, fmt.Errorf("failed to decode the manifest of the plugin %s: %w", source, err)
	}

	return manifest, nil
}

// fn returns a function of the plugin: a function, or a variable holding a function (the main package of a shared object).
func (p *nativePlugin) fn(name string) (reflect.Value, error) {
	sym, err := p.lookup(name)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("failed to find %s in the plugin %s: %w", name, p.source, err)
	}

	fn := reflect.ValueOf(sym)
	if fn.Kind() == reflect.Pointer && fn.Elem().Kind() == reflect.Func {
		fn = fn.Elem()
	}

	if fn.Kind() != reflect.Func {
		return reflect.Value{}, fmt.Errorf("%s of the plugin %s is a %T, a function is expected", name, p.source, sym)
	}

	return fn, nil
}

// call calls a function of the plugin, and returns its results.
func (p *nativePlugin) call(name string, args ...any) ([]reflect.Value, error) {
	fn, err := p.fn(name)
	if err != nil {
		return nil, err
	}

	ft := fn.Type()
	if ft.IsVariadic() || ft.NumIn() != len(args) {
		return nil, fmt.Errorf("%s of the plugin %s takes %d arguments, %d are given", name, p.source, ft.NumIn(), len(args))
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		if arg == nil {
			in[i] = reflect.Zero(ft.In(i))
			continue
		}

		in[i] = reflect.ValueOf(arg)
		if !in[i].Type().AssignableTo(ft.In(i)) {
			return nil, fmt.Errorf("the argument %d of %s of the plugin %s is a %s, a %s is expected", i, name, p.source, in[i].Type(), ft.In(i))
		}
	}

	return fn.Call(in), nil
}

// callNew calls a constructor of the plugin, which returns a value and an error.
func (p *nativePlugin) callNew(name string, args ...any) (any, error) {
	res, err := p.call(name, args...)
	if err != nil {
		return nil, err
	}

	if len(res) != 2 {
		return nil, fmt.Errorf("invalid number of return for the %s function: %d", name, len(res))
	}

	if !res[1].IsNil() {
		resErr, ok := reflect.TypeAssert[error](res[1])
		if !ok {
			return nil, fmt.Errorf("invalid error type: %s", res[1].Type())
		}

		return nil, resErr
	}

	return res[0].Interface(), nil
}

// createConfig creates the configuration of the plugin with its CreateConfig function, and decodes the given configuration into it.
func (p *nativePlugin) createConfig(config map[string]any) (any, error) {
	res, err := p.call("CreateConfig")
	if err != nil {
		return nil, err
	}

	if len(res) != 1 {
		return nil, fmt.Errorf("invalid number of return for the CreateConfig function: %d", len(res))
	}

	cfg := res[0].Interface()
	if len(config) == 0 {
		return cfg, nil
	}

	if err := decodePluginConfig(config, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// newNativeProvider creates the provider of a native plugin, which implements the provider interface of Traefik.
func newNativeProvider(p *nativePlugin, config map[string]any, pName string) (provider.Provider, error) {
	cfg, err := p.createConfig(config)
	if err != nil {
		return nil, err
	}

	v, err := p.callNew("New", context.Background(), cfg, "plugin-"+pName)
	if err != nil {
		return nil, err
	}

	prd, ok := v.(provider.Provider)
	if !ok {
		return nil, fmt.Errorf("the provider of the plugin %s, a %T, does not implement the provider interface of Traefik", p.source, v)
	}

	np := &nativeProvider{Provider: prd, name: pName, throttle: -1}

	if throttled, ok := prd.(interface{ ThrottleDuration() time.Duration }); ok {
		np.throttle = throttled.ThrottleDuration()
	}

	handler, ok := prd.(http.Handler)
	if !ok {
		return np, nil
	}

	hp := &nativeHandlerProvider{nativeProvider: np, handler: handler}
	if internal, ok := prd.(interface{ InternalRule() string }); ok {
		hp.rule = internal.InternalRule()
	}

	return hp, nil
}

// nativeProvider is the provider of a native plugin.
type nativeProvider struct {
	provider.Provider

	name string

	// throttle is the throttle duration of the plugin, negative when the plugin uses the providersThrottleDuration option.
	throttle time.Duration
}

// ThrottleDuration returns the throttle duration of the plugin, negative when the plugin uses the providersThrottleDuration option.
func (p *nativeProvider) ThrottleDuration() time.Duration {
	return p.throttle
}

// Name returns the name of the plugin.
func (p *nativeProvider) Name() string {
	return p.name
}

// configurationListener returns the ListenConfiguration method of the plugin, when it has one.
func (p *nativeProvider) configurationListener() (func(dynamic.Configuration), bool) {
	listener, ok := p.Provider.(interface {
		ListenConfiguration(conf dynamic.Configuration)
	})
	if !ok {
		return nil, false
	}

	return listener.ListenConfiguration, true
}

type nativeHandlerProvider struct {
	*nativeProvider

	handler http.Handler
	rule    string
}

func (p *nativeHandlerProvider) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.handler.ServeHTTP(rw, req)
}

func (p *nativeHandlerProvider) InternalRule() string {
	return p.rule
}

// nativeMiddlewareBuilder is the builder of the middlewares of a native plugin.
type nativeMiddlewareBuilder struct {
	plugin *nativePlugin
}

func (b *nativeMiddlewareBuilder) newMiddleware(config map[string]any, middlewareName string) (pluginMiddleware, error) {
	cfg, err := b.plugin.createConfig(config)
	if err != nil {
		return nil, err
	}

	return &nativeMiddleware{plugin: b.plugin, constructor: "New", config: cfg, middlewareName: middlewareName}, nil
}

func (b *nativeMiddlewareBuilder) newTypedMiddleware(option string, config any, middlewareName string) pluginMiddleware {
	return &nativeMiddleware{plugin: b.plugin, constructor: typedConstructor(option), config: config, middlewareName: middlewareName}
}

// checkTypedMiddleware checks that the plugin has the constructor of the middleware implementing the option:
// New<Option>(ctx context.Context, next http.Handler, config *dynamic.<Type>, name string) (http.Handler, error).
func (b *nativeMiddlewareBuilder) checkTypedMiddleware(option string) error {
	field, ok := middlewareOption(option)
	if !ok {
		return fmt.Errorf("the middleware option %q does not exist", option)
	}

	name := typedConstructor(option)

	fn, err := b.plugin.fn(name)
	if err != nil {
		return err
	}

	if fn.Type().NumIn() != 4 || fn.Type().In(2) != field.Type {
		return fmt.Errorf("the %s function of a plugin implementing the %q middleware option must take a %s configuration", name, option, field.Type)
	}

	return nil
}

// nativeMiddleware is a middleware of a native plugin.
// The native plugins are built from the sources of Traefik, they get the writer as the middlewares of Traefik do.
type nativeMiddleware struct {
	plugin *nativePlugin
	// constructor is the function creating the handler: New, or New<Option> for an option of the dynamic configuration.
	constructor    string
	config         any
	middlewareName string
}

// NewHandler creates a new HTTP handler.
func (m *nativeMiddleware) NewHandler(ctx context.Context, next http.Handler) (http.Handler, error) {
	v, err := m.plugin.callNew(m.constructor, ctx, next, m.config, m.middlewareName)
	if err != nil {
		return nil, err
	}

	handler, ok := v.(http.Handler)
	if !ok {
		return nil, fmt.Errorf("invalid handler type: %T", v)
	}

	return handler, nil
}
