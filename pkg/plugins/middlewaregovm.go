package plugins

import (
	"context"
	"errors"
	"fmt"
	"go/types"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	// The AOT copies of the standard library and of common libraries, used unless settings.goVM.disableAOT is set.
	_ "github.com/traefik/go-vm/aot/all"
	// The symbols of expvar, net/http/pprof and testing/quick, which go-vm leaves out of its standard library
	// because their init changes the state of the process: a plugin that imports expvar (the Prometheus client does)
	// gets its own table of variables. Traefik already imports expvar and net/http/pprof (pkg/api/debug.go).
	_ "github.com/traefik/go-vm/stdsyms/sideeffect"
	"github.com/traefik/go-vm/vm"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/plugins/hostsyms"
)

// The limits of the Go plugins.
const (
	// goVMBudget is the number of steps (interpreted calls and loop iterations) of each call into a plugin.
	// Every plugin of the catalog runs the same with a budget 100 times smaller, and a `for {}` stops after about 0.2 s.
	goVMBudget = 100_000_000
	// goVMMaxGoroutines is the maximum number of goroutines that a plugin runs at the same time.
	goVMMaxGoroutines = 1000
	// goVMLoadTimeout bounds the load of a plugin, so that a crafted plugin cannot block the load of the configuration.
	goVMLoadTimeout = 30 * time.Second
)

// goVMLimits returns the budget and the maximum number of goroutines of a Go plugin:
// the values of settings.goVM, or the defaults when they are zero (go-vm would read zero as no limit).
// A negative value is refused.
func goVMLimits(settings GoVMSettings) (int64, int, error) {
	budget := int64(goVMBudget)
	if settings.Budget < 0 {
		return 0, 0, fmt.Errorf("invalid settings.goVM.budget %d: it must be positive", settings.Budget)
	}
	if settings.Budget > 0 {
		budget = settings.Budget
	}

	maxGoroutines := goVMMaxGoroutines
	if settings.MaxGoroutines < 0 {
		return 0, 0, fmt.Errorf("invalid settings.goVM.maxGoroutines %d: it must be positive", settings.MaxGoroutines)
	}
	if settings.MaxGoroutines > 0 {
		maxGoroutines = settings.MaxGoroutines
	}

	return budget, maxGoroutines, nil
}

type goVMMiddlewareBuilder struct {
	plugin *vm.Plugin
}

// newGoVMMiddlewareBuilder loads a Go middleware plugin with the go-vm interpreter.
func newGoVMMiddlewareBuilder(ctx context.Context, goPath string, local bool, manifest *Manifest, moduleName string, settings Settings) (*goVMMiddlewareBuilder, error) {
	p, err := loadGoVMPlugin(ctx, goPath, local, manifest, moduleName, settings)
	if err != nil {
		return nil, err
	}

	return &goVMMiddlewareBuilder{plugin: p}, nil
}

// loadGoVMPlugin loads a Go plugin with the go-vm interpreter.
// It keeps at least the restrictions of the former Yaegi runtime: restricted standard library,
// unsafe and syscall only when both the manifest and the settings allow them,
// and a copy of the process environment.
// Unrestricted and RealProcess are only set for a trusted plugin (settings.goVM.trusted):
// they give the plugin os/exec, syscall, and the ability to stop Traefik.
// A local plugin (local is true) is read in GOPATH mode when it has no go.mod, as Yaegi read it.
func loadGoVMPlugin(ctx context.Context, goPath string, local bool, manifest *Manifest, moduleName string, settings Settings) (*vm.Plugin, error) {
	if manifest.UseUnsafe && !settings.UseUnsafe {
		return nil, errors.New("this plugin uses restricted imports. If you want to use it, you need to allow useUnsafe in the settings")
	}

	budget, maxGoroutines, err := goVMLimits(settings.GoVM)
	if err != nil {
		return nil, err
	}

	logger := log.Ctx(ctx)

	dir := filepath.Join(goPath, goPathSrc, filepath.FromSlash(moduleName))

	// The go.mod of a downloaded plugin must be in its archive:
	// the search for it never leaves the directories Traefik owns.
	root := dir

	var pluginGoPath string
	if local {
		// Yaegi loaded the local plugins in GOPATH mode and did not read go.mod,
		// so a local plugin without go.mod is read from the plugins GOPATH.
		root = filepath.Join(goPath, goPathSrc)
		pluginGoPath = goPath
	}

	opts := vm.Options{
		Dir:       dir,
		Root:      root,
		GoPath:    pluginGoPath,
		UseUnsafe: manifest.UseUnsafe && settings.UseUnsafe,
		// The options of settings.goVM, all off by default: the AOT mode is on unless disabled.
		AOT: !settings.GoVM.DisableAOT,
		// The Traefik SDK runs natively: the plugins share its state and types with Traefik.
		Modes:         sdkModes(),
		StubAssembly:  settings.GoVM.StubAssembly,
		Budget:        budget,
		MaxGoroutines: maxGoroutines,
		LoadTimeout:   goVMLoadTimeout,
		// Yaegi did not report unused variables, imports, labels and expression values,
		// nor identical copies of a declaration, and skipped the test files, so some plugins have them.
		IgnoreSoftErrors:     true,
		IgnoreUnusedExprs:    true,
		IgnoreTestFiles:      true,
		IgnoreDuplicateDecls: true,
		// Yaegi read the vendor directory as it is, and did not read the go directive of go.mod,
		// so some plugins have a vendor/modules.txt that go build refuses, or ask for a newer Go.
		IgnoreInconsistentVendoring: true,
		AllowNewerGo:                true,
		// The plugins have always read the environment of Traefik.
		Env:    os.Environ(),
		Stdout: logs.NoLevel(*logger, zerolog.DebugLevel),
		Stderr: logs.NoLevel(*logger, zerolog.ErrorLevel),
		// A panic in a goroutine started by the plugin must not stop Traefik.
		GoPanic: func(r any) {
			logger.Error().Msgf("Panic in a goroutine of the plugin: %v", r)
		},
	}

	if settings.GoVM.ExecMock {
		// os/exec can be imported, but no program runs.
		opts.Exec = vm.ExecMock
	}

	if settings.GoVM.Trusted {
		// A trusted plugin runs as the code of Traefik, like the providers which watch files and run in-cluster clients.
		opts.Unrestricted = true
		opts.RealProcess = true
		opts.Budget = 0
		opts.MaxGoroutines = 0
		opts.LoadTimeout = 0
		opts.LoadLimits = &vm.LoadLimits{}
	}

	p, err := vm.Load(opts, manifest.Import)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin code %q: %w", manifest.Import, err)
	}

	// The notes say where the plugin now runs other code than under Yaegi
	// (for example, files selected by "//go:build" lines that Yaegi ignored).
	for _, note := range p.Notes {
		logger.Warn().Msg(note)
	}

	// A plugin implementing options of the dynamic configuration has a constructor per option (see checkTypedMiddleware).
	if manifest.Type == typeMiddleware && len(manifest.Implements) > 0 {
		return p, nil
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
	cfg, err := createGoVMConfig(b.plugin, config)
	if err != nil {
		return nil, err
	}

	return &GoVMMiddleware{
		middlewareName: middlewareName,
		config:         cfg,
		plugin:         b.plugin,
	}, nil
}

// createGoVMConfig calls the CreateConfig function of the plugin, and decodes the configuration into its result.
func createGoVMConfig(p *vm.Plugin, config map[string]any) (*vm.Object, error) {
	res, err := p.Call(context.Background(), "CreateConfig")
	if err != nil {
		return nil, fmt.Errorf("failed to call CreateConfig: %w", err)
	}

	if len(res) != 1 {
		return nil, fmt.Errorf("invalid number of return for the CreateConfig function: %d", len(res))
	}

	cfg := res[0]
	if len(config) == 0 {
		return cfg, nil
	}

	err = cfg.Decode(func(result any) error {
		return decodePluginConfig(config, result)
	})
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// decodePluginConfig decodes the configuration with the same options as the Yaegi runtime.
func decodePluginConfig(config map[string]any, result any) error {
	cfg := &mapstructure.DecoderConfig{
		// The static configuration gives the durations as text ("1s"): ptypes.Duration decodes it.
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToSliceHookFunc(","),
			emptyStringToStruct,
			mapstructure.TextUnmarshallerHookFunc(),
		),
		WeaklyTypedInput: true,
		// The options of an embedded struct are at the level of the struct that embeds it, as in the static configuration.
		Squash: true,
		Result: result,
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

// emptyStringToStruct decodes an empty value of the configuration as an empty struct or map:
// the static configuration gives "" for an option without value (tlsChallenge: {}), which enables a feature.
func emptyStringToStruct(_, to reflect.Type, data any) (any, error) {
	if str, ok := data.(string); !ok || str != "" {
		return data, nil
	}

	for to.Kind() == reflect.Pointer {
		to = to.Elem()
	}

	if to.Kind() == reflect.Struct || to.Kind() == reflect.Map {
		return map[string]any{}, nil
	}

	return data, nil
}

// sdkModes runs natively the packages of the Traefik SDK (see pkg/plugins/hostsyms).
func sdkModes() map[string]vm.Mode {
	modes := make(map[string]vm.Mode, len(hostsyms.Paths)+len(hostsyms.StdPaths))
	for _, path := range slices.Concat(hostsyms.Paths, hostsyms.StdPaths) {
		modes[path] = vm.ModeNative
	}

	return modes
}

// goVMError returns the error held by an error result of a plugin call.
func goVMError(res *vm.Object) error {
	v, err := res.Interface()
	if err != nil {
		return err
	}

	if v == nil {
		return nil
	}

	resErr, ok := v.(error)
	if !ok {
		return fmt.Errorf("invalid error type: %T", v)
	}

	return resErr
}

// newTypedMiddleware creates the middleware of a plugin implementing an option of the dynamic configuration:
// the plugin gets the configuration as it is, a value of the native types of pkg/config/dynamic.
func (b *goVMMiddlewareBuilder) newTypedMiddleware(option string, config any, middlewareName string) pluginMiddleware {
	return &GoVMMiddleware{
		middlewareName: middlewareName,
		config:         config,
		plugin:         b.plugin,
		constructor:    typedConstructor(option),
	}
}

func (b *goVMMiddlewareBuilder) checkTypedMiddleware(option string) error {
	return checkTypedMiddleware(b.plugin, option)
}

// typedConstructor returns the name of the constructor of the middleware implementing the option: NewHeaders for headers.
func typedConstructor(option string) string {
	return "New" + strings.ToUpper(option[:1]) + option[1:]
}

// checkTypedMiddleware checks that a plugin implementing an option of the dynamic configuration has its constructor:
// New<Option>(ctx context.Context, next http.Handler, config *dynamic.<Type>, name string) (http.Handler, error).
func checkTypedMiddleware(p *vm.Plugin, option string) error {
	field, ok := middlewareOption(option)
	if !ok {
		return fmt.Errorf("the middleware option %q does not exist", option)
	}

	name := typedConstructor(option)

	fn := p.Pkg.Func(name)
	if fn == nil {
		return fmt.Errorf("failed to find %s", name)
	}

	expected := "*" + field.Type.Elem().PkgPath() + "." + field.Type.Elem().Name()

	params := fn.Signature.Params()
	if params.Len() != 4 || types.TypeString(params.At(2).Type(), nil) != expected {
		return fmt.Errorf("the %s function of a plugin implementing the %q middleware option must take a %s configuration", name, option, expected)
	}

	return nil
}

// middlewareOption returns the field of the given option of the dynamic configuration of the middlewares.
func middlewareOption(option string) (reflect.StructField, bool) {
	typ := reflect.TypeFor[dynamic.Middleware]()
	for field := range typ.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == option && field.Type.Kind() == reflect.Pointer && field.Type.Elem().Kind() == reflect.Struct {
			return field, true
		}
	}

	return reflect.StructField{}, false
}

// GoVMMiddleware is an HTTP handler plugin wrapper for the go-vm interpreter.
type GoVMMiddleware struct {
	middlewareName string
	// config is the configuration given to the New function of the plugin:
	// the result of its CreateConfig function, or the value of an option of the dynamic configuration.
	config any
	plugin *vm.Plugin
	// constructor is the function creating the handler: New, or New<Option> for an option of the dynamic configuration.
	constructor string
}

// NewHandler creates a new HTTP handler.
func (m *GoVMMiddleware) NewHandler(ctx context.Context, next http.Handler) (http.Handler, error) {
	handler, traceable, err := wrapExternalWith(next, func(next http.Handler) (http.Handler, traceable, error) {
		handler, err := m.newPluginHandler(ctx, next)
		if err != nil {
			return nil, nil, err
		}

		return handler, asTraceable(handler), nil
	})
	if err != nil {
		return nil, err
	}

	// The middlewares of Traefik name their traces themselves (RateLimiter, CustomError, ...), the wrapper must keep it.
	if traceable != nil {
		return &traceableHandler{Handler: handler, traceable: traceable}, nil
	}

	return handler, nil
}

// traceable is a handler naming its traces itself (see observability.Traceable).
type traceable interface {
	GetTracingInformation() (name string, typeName string)
}

type traceableHandler struct {
	http.Handler
	traceable
}

// go-vm shows the methods of a plugin value to native code only through a wrapper per interface:
// the http.Handler returned by a plugin does not have GetTracingInformation without this wrapper.
func init() {
	vm.RegisterWrapper[traceable](func(o vm.Obj) traceable { return traceableObj{o} })
}

type traceableObj struct{ vm.Obj }

func (t traceableObj) GetTracingInformation() (string, string) {
	out := t.Call("GetTracingInformation")
	return vm.Out[string](out, 0), vm.Out[string](out, 1)
}

// asTraceable returns the handler of a plugin as a traceable, or nil when it does not name its traces.
func asTraceable(handler http.Handler) (t traceable) {
	// vm.AssertI needs the AOT mode: with settings.goVM.disableAOT, it panics and the handler is not traceable.
	defer func() {
		if recover() != nil {
			t = nil
		}
	}()

	t, _ = vm.AssertI[traceable](handler)

	return t
}

func (m *GoVMMiddleware) newPluginHandler(ctx context.Context, next http.Handler) (http.Handler, error) {
	constructor := m.constructor
	if constructor == "" {
		constructor = "New"
	}

	res, err := m.plugin.Call(ctx, constructor, ctx, next, m.config, m.middlewareName)
	if err != nil {
		return nil, fmt.Errorf("failed to call %s: %w", constructor, err)
	}

	if len(res) != 2 {
		return nil, fmt.Errorf("invalid number of return for the %s function: %d", constructor, len(res))
	}

	if err := goVMError(res[1]); err != nil {
		return nil, err
	}

	v, err := res[0].Interface()
	if err != nil {
		return nil, err
	}

	handler, ok := v.(http.Handler)
	if !ok {
		return nil, fmt.Errorf("invalid handler type: %T", v)
	}

	return handler, nil
}
