package wasihttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/plugins/wasm/engine"
)

// wasiHTTP serves one plugin binary and one middleware configuration on a pool of instances.
type wasiHTTP struct {
	cfg    engine.Config
	pool   *engine.Pool
	client *http.Client
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

	store, err := configStore(cfg.GuestConfig)
	if err != nil {
		return nil, err
	}

	e := &wasiHTTP{cfg: cfg}
	if cfg.AllowOutbound {
		e.client = newOutboundClient()
	}

	var pool *engine.Pool
	pool, err = engine.NewPool(ctx, cfg.InstanceCount(), func(ctx context.Context, slot int) (engine.Instance, error) {
		return newInstance(ctx, &e.cfg, store, e.client, slot)
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
	err := e.pool.Close(ctx)

	if e.client != nil {
		e.client.CloseIdleConnections()
	}

	return err
}

// configStore builds the wasi:config/store of the guest from the middleware configuration: each top-level key is a
// key, with a string value as is and any other value as JSON.
func configStore(config []byte) (map[string]string, error) {
	store := map[string]string{}
	if len(bytes.TrimSpace(config)) == 0 {
		return store, nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(config, &raw); err != nil {
		return nil, fmt.Errorf("decoding the plugin configuration: %w", err)
	}

	for k, v := range raw {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			store[k] = s

			continue
		}
		store[k] = string(v)
	}

	return store, nil
}
