package httpwasm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"github.com/traefik/traefik/v3/pkg/plugins/wasm/engine"
)

const (
	fixtureWithoutSocket = "../../fixtures/withoutsocket/plugin.wasm"
	fixtureWithNext      = "../../fixtures/withnext/plugin.wasm"
)

func newTestEngine(t *testing.T, fixture string, config map[string]any, cfg engine.Config) engine.Engine {
	t.Helper()

	code, err := os.ReadFile(fixture)
	require.NoError(t, err)

	guestConfig, err := json.Marshal(config)
	require.NoError(t, err)

	logger := zerolog.Nop()
	if os.Getenv("WASM_TEST_LOG") != "" {
		logger = zerolog.New(zerolog.NewConsoleWriter()).Level(zerolog.DebugLevel)
	}
	cfg.Name = "test"
	cfg.Code = code
	cfg.Cache = wazero.NewCompilationCache()
	cfg.GuestConfig = guestConfig
	cfg.Logger = &logger

	eng, err := New(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = eng.Close(context.Background()) })

	return eng
}

// echoNext copies the request headers set by the guest into the response and writes a body.
var echoNext = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("X-Next-Saw-Plugin", req.Header.Get("X-Plugin"))
	rw.Header().Set("X-Next-Saw-Seq", req.Header.Get("X-Plugin-Seq"))
	rw.WriteHeader(http.StatusCreated)
	_, _ = rw.Write([]byte("hello from next"))
})

func TestEngine_Settings(t *testing.T) {
	t.Setenv("PLUGIN_TEST", "MY-TEST")
	t.Setenv("PLUGIN_TEST_B", "MY-TEST_B")

	testCases := []struct {
		desc        string
		getSettings func(t *testing.T) (engine.Config, map[string]any)
		expected    string
	}{
		{
			desc: "mounts path",
			getSettings: func(t *testing.T) (engine.Config, map[string]any) {
				t.Helper()

				tempDir := t.TempDir()
				filePath := path.Join(tempDir, "hello.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("content_test"), 0o644))

				return engine.Config{Mounts: []string{tempDir}}, map[string]any{"file": filePath}
			},
			expected: "content_test",
		},
		{
			desc: "mounts src to dest",
			getSettings: func(t *testing.T) (engine.Config, map[string]any) {
				t.Helper()

				tempDir := t.TempDir()
				filePath := path.Join(tempDir, "hello.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("content_test"), 0o644))

				return engine.Config{Mounts: []string{tempDir + ":/tmp"}}, map[string]any{"file": "/tmp/hello.txt"}
			},
			expected: "content_test",
		},
		{
			desc: "one env",
			getSettings: func(t *testing.T) (engine.Config, map[string]any) {
				t.Helper()

				envs := []string{"PLUGIN_TEST"}

				return engine.Config{Envs: envs}, map[string]any{"envs": envs}
			},
			expected: "MY-TEST\n",
		},
		{
			desc: "two env",
			getSettings: func(t *testing.T) (engine.Config, map[string]any) {
				t.Helper()

				envs := []string{"PLUGIN_TEST", "PLUGIN_TEST_B"}

				return engine.Config{Envs: envs}, map[string]any{"envs": envs}
			},
			expected: "MY-TEST\nMY-TEST_B\n",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			cfg, config := test.getSettings(t)
			cfg.Instances = 1
			eng := newTestEngine(t, fixtureWithoutSocket, config, cfg)

			rw := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)

			eng.ServeHTTP(rw, req, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusTeapot)
			}))

			assert.Equal(t, http.StatusOK, rw.Code)
			assert.Equal(t, test.expected, rw.Body.String())
		})
	}
}

func TestEngine_Next(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, fixtureWithNext, map[string]any{"header": "hello"}, engine.Config{Instances: 1})

	rw := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)

	eng.ServeHTTP(rw, req, echoNext)

	assert.Equal(t, http.StatusCreated, rw.Code)
	assert.Equal(t, "hello from next", rw.Body.String())
	assert.Equal(t, "hello", rw.Header().Get("X-Next-Saw-Plugin"))
	assert.Equal(t, "1", rw.Header().Get("X-Next-Saw-Seq"))
	assert.Equal(t, "1", rw.Header().Get("X-Plugin-Resp"))
}

func TestEngine_BufferResponse(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, fixtureWithNext, map[string]any{"header": "h", "upperResponse": true}, engine.Config{Instances: 1})

	rw := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)

	eng.ServeHTTP(rw, req, echoNext)

	assert.Equal(t, http.StatusCreated, rw.Code)
	assert.Equal(t, "HELLO FROM NEXT", rw.Body.String())
	assert.Equal(t, "1", rw.Header().Get("X-Plugin-Resp"))
}

func TestEngine_NextError(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, fixtureWithNext, map[string]any{"header": "h"}, engine.Config{Instances: 1})

	rw := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)

	eng.ServeHTTP(rw, req, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("next failed")
	}))

	assert.Equal(t, "true", rw.Header().Get("X-Plugin-Next-Error"))
	assert.Equal(t, "1", rw.Header().Get("X-Plugin-Resp"))
}

// TestEngine_ConcurrentNext checks that one instance serves many requests at once: the instance is free while next
// runs, so a slow next does not serialize the requests.
func TestEngine_ConcurrentNext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc string
		cfg  engine.Config
	}{
		{desc: "shared instance", cfg: engine.Config{Instances: 1}},
		{desc: "exclusive instances", cfg: engine.Config{Instances: 2, Exclusive: true}},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			eng := newTestEngine(t, fixtureWithNext, map[string]any{"header": "h"}, test.cfg)

			const requests = 50
			const delay = 50 * time.Millisecond

			slowNext := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				time.Sleep(delay)
				echoNext.ServeHTTP(rw, req)
			})

			start := time.Now()

			var wg sync.WaitGroup
			results := make([]*httptest.ResponseRecorder, requests)
			for i := range requests {
				wg.Go(func() {
					rw := httptest.NewRecorder()
					req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
					eng.ServeHTTP(rw, req, slowNext)
					results[i] = rw
				})
			}
			wg.Wait()

			elapsed := time.Since(start)
			assert.Less(t, elapsed, requests*delay/4, "requests were serialized")

			for _, rw := range results {
				assert.Equal(t, http.StatusCreated, rw.Code)
				assert.Equal(t, "hello from next", rw.Body.String())
				// The response of a request must be handled with its own context.
				assert.Equal(t, rw.Header().Get("X-Next-Saw-Seq"), rw.Header().Get("X-Plugin-Resp"))
			}

			assert.Equal(t, int64(0), eng.Stats().InFlight)
		})
	}
}

func TestEngine_Panic(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, fixtureWithNext, map[string]any{"header": "h", "panicOn": "/boom"}, engine.Config{Instances: 1})

	rw := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/boom", http.NoBody)
	eng.ServeHTTP(rw, req, echoNext)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
	assert.Equal(t, "Internal Server Error\n", rw.Body.String())

	// The instance is replaced in the background.
	require.Eventually(t, func() bool {
		return eng.Stats().LiveInstances == 1
	}, 5*time.Second, 10*time.Millisecond)

	rw = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	eng.ServeHTTP(rw, req, echoNext)

	assert.Equal(t, http.StatusCreated, rw.Code)
	assert.Equal(t, "1", rw.Header().Get("X-Plugin-Resp"), "a fresh instance starts its sequence at 1")
}

func TestEngine_EchoBody(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, fixtureWithNext, map[string]any{"echoBody": true}, engine.Config{Instances: 1})

	// Larger than the prefetch size: the guest reads past the prefix.
	body := strings.Repeat("abcdefgh", 3*bodyPrefetchSize/8)

	rw := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
	eng.ServeHTTP(rw, req, echoNext)

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "196608", rw.Header().Get("X-Body-Length"))
	assert.Equal(t, body, rw.Body.String())
}

func TestEngine_Close(t *testing.T) {
	t.Parallel()

	eng := newTestEngine(t, fixtureWithNext, map[string]any{"header": "h"}, engine.Config{Instances: 1})

	require.NoError(t, eng.Close(t.Context()))

	rw := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	eng.ServeHTTP(rw, req, echoNext)

	assert.Equal(t, http.StatusServiceUnavailable, rw.Code)
	_ = io.Discard
}
