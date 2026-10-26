package wasihttp

import (
	"context"
	"errors"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/plugins/wasm/engine"
)

// wasiHTTP serves one plugin binary and one middleware configuration on a pool of instances.
type wasiHTTP struct {
	cfg  engine.Config
	pool *engine.Pool
}

// ValidateSettings rejects the plugin settings that this mode does not support.
func ValidateSettings(cfg engine.Config) error {
	if len(cfg.Mounts) > 0 {
		return errors.New("settings.mounts is not supported by plugins built with github.com/traefik/wasm-go")
	}

	// The guest runs in one call that never returns, and reads the request body as a stream: these http-wasm
	// settings have no meaning here.
	switch {
	case cfg.Exclusive:
		return errors.New("settings.exclusive is only supported by http-wasm plugins")
	case cfg.CallTimeout > 0:
		return errors.New("settings.callTimeout is only supported by http-wasm plugins")
	case cfg.BodyPrefetchSize > 0:
		return errors.New("settings.bodyPrefetchSize is only supported by http-wasm plugins")
	}

	return nil
}

// New starts the instances of a plugin built with github.com/traefik/wasm-go.
func New(ctx context.Context, cfg engine.Config) (engine.Engine, error) {
	if err := ValidateSettings(cfg); err != nil {
		return nil, err
	}

	e := &wasiHTTP{cfg: cfg}

	pool, err := engine.NewPool(ctx, cfg.InstanceCount(), func(ctx context.Context, slot int) (engine.Instance, error) {
		return newInstance(ctx, &e.cfg, slot)
	}, cfg.Logger)
	if err != nil {
		return nil, err
	}
	e.pool = pool

	return e, nil
}

// ServeHTTP implements engine.Engine.
func (e *wasiHTTP) ServeHTTP(rw http.ResponseWriter, req *http.Request, next http.Handler) {
	if err := e.pool.Enter(); err != nil {
		http.Error(rw, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)

		return
	}
	defer e.pool.Leave()

	// An instance found dead is replaced, and the request goes to another one.
	for range e.cfg.InstanceCount() {
		slot, err := e.pool.Pick()
		if err != nil {
			break
		}

		inst, ok := slot.Instance().(*instance)
		if !ok {
			continue
		}

		if dead, cause := inst.isDead(); dead {
			e.pool.Replace(slot, cause)

			continue
		}

		if err := inst.serve(rw, req, next); err != nil {
			continue
		}

		if dead, cause := inst.isDead(); dead {
			e.pool.Replace(slot, cause)
		}

		return
	}

	e.cfg.Logger.Error().Msg("No wasm instance available")
	http.Error(rw, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
}

// Stats implements engine.Engine.
func (e *wasiHTTP) Stats() engine.Stats {
	return e.pool.Stats()
}

// Close implements engine.Engine.
func (e *wasiHTTP) Close(ctx context.Context) error {
	return e.pool.Close(ctx)
}
