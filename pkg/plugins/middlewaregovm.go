package plugins

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	// The AOT copies of the standard library and of common libraries, used with vm.Options.AOT.
	_ "github.com/traefik/go-vm/aot/all"
	"github.com/traefik/go-vm/vm"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
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
// Unrestricted and RealProcess must never be set: they would give the plugin
// os/exec, syscall, and the ability to stop Traefik.
// A local plugin (local is true) is read in GOPATH mode when it has no go.mod, as Yaegi read it.
func loadGoVMPlugin(ctx context.Context, goPath string, local bool, manifest *Manifest, moduleName string, settings Settings) (*vm.Plugin, error) {
	if manifest.UseUnsafe && !settings.UseUnsafe {
		return nil, errors.New("this plugin uses restricted imports. If you want to use it, you need to allow useUnsafe in the settings")
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
		// The options of settings.goVM, all off by default.
		AOT:           settings.GoVM.AOT,
		StubAssembly:  settings.GoVM.StubAssembly,
		Budget:        goVMBudget,
		MaxGoroutines: goVMMaxGoroutines,
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

	p, err := vm.Load(opts, manifest.Import)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin code %q: %w", manifest.Import, err)
	}

	// The notes say where the plugin now runs other code than under Yaegi
	// (for example, files selected by "//go:build" lines that Yaegi ignored).
	for _, note := range p.Notes {
		logger.Warn().Msg(note)
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

// GoVMMiddleware is an HTTP handler plugin wrapper for the go-vm interpreter.
type GoVMMiddleware struct {
	middlewareName string
	config         *vm.Object
	plugin         *vm.Plugin
}

// NewHandler creates a new HTTP handler.
func (m *GoVMMiddleware) NewHandler(ctx context.Context, next http.Handler) (http.Handler, error) {
	return wrapExternal(next, func(next http.Handler) (http.Handler, error) {
		return m.newPluginHandler(ctx, next)
	})
}

func (m *GoVMMiddleware) newPluginHandler(ctx context.Context, next http.Handler) (http.Handler, error) {
	res, err := m.plugin.Call(ctx, "New", ctx, next, m.config, m.middlewareName)
	if err != nil {
		return nil, fmt.Errorf("failed to call New: %w", err)
	}

	if len(res) != 2 {
		return nil, fmt.Errorf("invalid number of return for the New function: %d", len(res))
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
