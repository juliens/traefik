package httpwasm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	handlerapi "github.com/http-wasm/http-wasm-host-go/api/handler"
	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/sys"
)

// instance is one guest instance: its own runtime, WASI module, memory and lock.
type instance struct {
	rt       wazero.Runtime
	mod      wazeroapi.Module
	applyCtx contextApplier

	handleRequestFn  wazeroapi.Function
	handleResponseFn wazeroapi.Function

	// callTimeout bounds each guest call; the runtime closes the module when it is exceeded.
	callTimeout time.Duration

	// mu serializes the guest calls: the guest runtimes are not thread-safe.
	mu sync.Mutex
}

func (e *httpWasm) newInstance(ctx context.Context, slot int) (*instance, error) {
	rt := wazero.NewRuntimeWithConfig(ctx, e.cfg.RuntimeConfig())

	inst, err := e.instantiate(ctx, rt, slot)
	if err != nil {
		_ = rt.Close(ctx)

		return nil, err
	}

	return inst, nil
}

func (e *httpWasm) instantiate(ctx context.Context, rt wazero.Runtime, slot int) (*instance, error) {
	compiled, err := rt.CompileModule(ctx, e.cfg.Code)
	if err != nil {
		return nil, fmt.Errorf("compiling guest module: %w", err)
	}

	applyCtx, err := instantiateWASI(ctx, rt, compiled, e.cfg.Envs, e.cfg.Mounts)
	if err != nil {
		return nil, err
	}

	if _, err = e.instantiateHost(ctx, rt); err != nil {
		return nil, fmt.Errorf("instantiating http_handler module: %w", err)
	}

	config, err := e.moduleConfig(fmt.Sprintf("%s-%d", e.cfg.Name, slot))
	if err != nil {
		return nil, err
	}

	mod, err := rt.InstantiateModule(applyCtx(ctx), compiled, config)
	if err != nil {
		return nil, fmt.Errorf("instantiating guest module: %w", err)
	}

	// A guest built as a command exits with code 0 when its main returns, which wazero reports as a success,
	// leaving a closed module that cannot handle requests.
	if mod.IsClosed() {
		return nil, errors.New("guest module exited during its start function: build the plugin as a reactor (TinyGo -buildmode=c-shared)")
	}

	return &instance{
		rt:               rt,
		mod:              mod,
		applyCtx:         applyCtx,
		callTimeout:      e.cfg.CallTimeout,
		handleRequestFn:  mod.ExportedFunction(handlerapi.FuncHandleRequest),
		handleResponseFn: mod.ExportedFunction(handlerapi.FuncHandleResponse),
	}, nil
}

// moduleConfig builds the guest module configuration: start functions, environment and mounts.
func (e *httpWasm) moduleConfig(name string) (wazero.ModuleConfig, error) {
	config := wazero.NewModuleConfig().
		WithName(name).
		WithSysWalltime().
		WithStartFunctions("_start", "_initialize")

	for _, env := range e.cfg.Envs {
		config = config.WithEnv(env, os.Getenv(env))
	}

	if len(e.cfg.Mounts) == 0 {
		return config, nil
	}

	fsConfig := wazero.NewFSConfig()
	for _, mount := range e.cfg.Mounts {
		withDir := fsConfig.WithDirMount
		prefix, readOnly := strings.CutSuffix(mount, ":ro")
		if readOnly {
			withDir = fsConfig.WithReadOnlyDirMount
		}

		parts := strings.Split(prefix, ":")
		switch len(parts) {
		case 1:
			fsConfig = withDir(parts[0], parts[0])
		case 2:
			fsConfig = withDir(parts[0], parts[1])
		default:
			return nil, fmt.Errorf("invalid directory %q", mount)
		}
	}

	return config.WithFSConfig(fsConfig), nil
}

// Close closes the runtime of the instance, which closes the guest and host modules.
func (inst *instance) Close(ctx context.Context) error {
	return inst.rt.Close(ctx)
}

// handleRequest calls the guest function handle_request under the instance lock.
func (inst *instance) handleRequest(ctx context.Context) (handlerapi.CtxNext, error) {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	ctx, cancel := inst.callContext(ctx)
	defer cancel()

	results, err := inst.handleRequestFn.Call(inst.applyCtx(ctx))
	if err != nil {
		return 0, err
	}

	return handlerapi.CtxNext(results[0]), nil
}

// handleResponse calls the guest function handle_response under the instance lock.
func (inst *instance) handleResponse(ctx context.Context, reqCtx uint32, nextErr error) error {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	wasError := uint64(0)
	if nextErr != nil {
		wasError = 1
	}

	ctx, cancel := inst.callContext(ctx)
	defer cancel()

	_, err := inst.handleResponseFn.Call(inst.applyCtx(ctx), uint64(reqCtx), wasError)

	return err
}

// callContext bounds a guest call by the call timeout, counted once the instance lock is held.
// The call does not follow the request cancellation: with a timeout, a done context closes the whole module, and a
// client going away must not kill an instance shared by other requests.
func (inst *instance) callContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if inst.callTimeout <= 0 {
		return ctx, func() {}
	}

	return context.WithTimeout(context.WithoutCancel(ctx), inst.callTimeout)
}

// broken reports whether the guest cannot be trusted anymore after a call error: the module exited, trapped or
// was closed. Other errors are host function panics, left by the guest at a call boundary.
func (inst *instance) broken(err error) bool {
	if _, ok := errors.AsType[*sys.ExitError](err); ok {
		return true
	}

	// wazero reports traps (unreachable, out of bounds...) as errors prefixed by "wasm error:".
	return strings.HasPrefix(err.Error(), "wasm error:") || inst.mod.IsClosed()
}
