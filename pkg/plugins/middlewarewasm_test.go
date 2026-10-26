package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWasmMiddleware(t *testing.T) {
	t.Parallel()

	ctx := log.Logger.WithContext(t.Context())

	builder, err := newWasmMiddlewareBuilderFromPath(ctx, "./fixtures/withnext/plugin.wasm", Settings{Instances: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = builder.Close(context.Background()) })

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("X-Next-Saw-Plugin", req.Header.Get("X-Plugin"))
		rw.WriteHeader(http.StatusTeapot)
	})

	serve := func(t *testing.T, config map[string]any) *httptest.ResponseRecorder {
		t.Helper()

		m, err := builder.newMiddleware(config, "test")
		require.NoError(t, err)

		h, err := m.NewHandler(ctx, next)
		require.NoError(t, err)

		rw := httptest.NewRecorder()
		h.ServeHTTP(rw, httptest.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody))

		return rw
	}

	rw := serve(t, map[string]any{"header": "a"})
	assert.Equal(t, http.StatusTeapot, rw.Code)
	assert.Equal(t, "a", rw.Header().Get("X-Next-Saw-Plugin"))
	assert.Equal(t, "1", rw.Header().Get("X-Plugin-Resp"))

	// The same middleware configuration shares the engine, whatever the chain.
	rw = serve(t, map[string]any{"header": "a"})
	assert.Equal(t, "a", rw.Header().Get("X-Next-Saw-Plugin"))
	assert.Equal(t, "2", rw.Header().Get("X-Plugin-Resp"))
	assert.Len(t, builder.engines, 1)

	// Another configuration gets its own engine.
	rw = serve(t, map[string]any{"header": "b"})
	assert.Equal(t, "b", rw.Header().Get("X-Next-Saw-Plugin"))
	assert.Equal(t, "1", rw.Header().Get("X-Plugin-Resp"))
	assert.Len(t, builder.engines, 2)
}

func TestWasmMiddleware_WasiHTTP(t *testing.T) {
	t.Parallel()

	ctx := log.Logger.WithContext(t.Context())

	builder, err := newWasmMiddlewareBuilderFromPath(ctx, "./fixtures/wasihttp/plugin.wasm", Settings{Instances: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = builder.Close(context.Background()) })

	m, err := builder.newMiddleware(map[string]any{"a": 1}, "test")
	require.NoError(t, err)

	h, err := m.NewHandler(ctx, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("X-Next-Saw-Plugin", req.Header.Get("X-Plugin"))
		rw.WriteHeader(http.StatusTeapot)
	}))
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, httptest.NewRequestWithContext(ctx, http.MethodGet, "/next", http.NoBody))
	assert.Equal(t, http.StatusTeapot, rw.Code)
	assert.Equal(t, "pass", rw.Header().Get("X-Next-Saw-Plugin"))

	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, httptest.NewRequestWithContext(ctx, http.MethodGet, "/config", http.NoBody))
	assert.JSONEq(t, `{"a":1}`, rw.Body.String())
}

func TestNewWasmMiddlewareBuilder_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc     string
		path     string
		settings Settings
		err      string
	}{
		{
			desc: "missing binary",
			path: "./fixtures/missing.wasm",
			err:  "loading Wasm binary: open ./fixtures/missing.wasm: no such file or directory",
		},
		{
			desc:     "negative instances",
			path:     "./fixtures/withnext/plugin.wasm",
			settings: Settings{Instances: -1},
			err:      "settings.instances must be positive",
		},
		{
			desc:     "negative memory limit",
			path:     "./fixtures/withnext/plugin.wasm",
			settings: Settings{MemoryLimit: -1},
			err:      "settings.memoryLimit must be positive",
		},
		{
			desc:     "mounts with a wasm-go plugin",
			path:     "./fixtures/wasihttp/plugin.wasm",
			settings: Settings{Mounts: []string{"/tmp"}},
			err:      "settings.mounts is not supported by plugins built with github.com/traefik/wasm-go",
		},
		{
			desc:     "exclusive with a wasm-go plugin",
			path:     "./fixtures/wasihttp/plugin.wasm",
			settings: Settings{Exclusive: true},
			err:      "settings.exclusive is only supported by http-wasm plugins",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			_, err := newWasmMiddlewareBuilderFromPath(t.Context(), test.path, test.settings)
			require.EqualError(t, err, test.err)
		})
	}
}
