package wasihttp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

const fixture = "../../fixtures/wasihttp/plugin.wasm"

// syncBuffer collects the logs of an engine; the instances write to it from their own goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func newTestEngine(t *testing.T, cfg engine.Config) (engine.Engine, *syncBuffer) {
	t.Helper()

	code, err := os.ReadFile(fixture)
	require.NoError(t, err)

	logs := &syncBuffer{}
	logger := zerolog.New(logs).Level(zerolog.InfoLevel)

	cfg.Name = "test"
	cfg.Code = code
	cfg.Cache = wazero.NewCompilationCache()
	cfg.Logger = &logger
	if cfg.Instances == 0 {
		cfg.Instances = 1
	}
	if cfg.GuestConfig == nil {
		cfg.GuestConfig = []byte("{}")
	}

	eng, err := New(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = eng.Close(context.Background()) })

	return eng, logs
}

func serve(t *testing.T, eng engine.Engine, next http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody io.Reader = http.NoBody
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	rw := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), method, "http://example.com"+target, reqBody)
	eng.ServeHTTP(rw, req, next)

	return rw
}

// upstream is a next handler that tells what it received.
func upstream(delay time.Duration) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		time.Sleep(delay)

		body, _ := io.ReadAll(req.Body)
		rw.Header().Set("X-Upstream", "yes")
		rw.Header().Set("X-Seen-Plugin", req.Header.Get("X-Plugin"))
		rw.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(rw, "upstream saw %s %s %s %d:%s", req.Method, req.URL.RequestURI(), req.Host, req.ContentLength, body)
	})
}

func TestEngine_Echo(t *testing.T) {
	t.Parallel()

	eng, _ := newTestEngine(t, engine.Config{})

	rw := serve(t, eng, nil, http.MethodPost, "/echo?x=1", "ping")

	assert.Equal(t, http.StatusAccepted, rw.Code)
	assert.Equal(t, "POST /echo?x=1 4:ping", rw.Body.String())
	assert.Equal(t, "example.com", rw.Header().Get("X-Echo-Host"))
	assert.Equal(t, "192.0.2.1:1234", rw.Header().Get("X-Echo-Remote"))
	assert.Equal(t, "HTTP/1.1", rw.Header().Get("X-Echo-Proto"))
}

func TestEngine_Next(t *testing.T) {
	t.Parallel()

	eng, _ := newTestEngine(t, engine.Config{})

	rw := serve(t, eng, upstream(0), http.MethodPost, "/next", "ping")

	assert.Equal(t, http.StatusCreated, rw.Code)
	assert.Equal(t, "upstream saw POST /next example.com 4:ping", rw.Body.String())
	assert.Equal(t, "yes", rw.Header().Get("X-Upstream"))
	assert.Equal(t, "pass", rw.Header().Get("X-Seen-Plugin"))
	assert.Equal(t, "1", rw.Header().Get("X-Plugin-Response"))
}

func TestEngine_NextBodyFromGuest(t *testing.T) {
	t.Parallel()

	eng, _ := newTestEngine(t, engine.Config{})

	rw := serve(t, eng, upstream(0), http.MethodPost, "/rewrite?q=1", "ping")

	assert.Equal(t, http.StatusCreated, rw.Code)
	assert.Equal(t, "UPSTREAM SAW POST /REWRITTEN?Q=1 UPSTREAM 4:PING", rw.Body.String())
}

func TestEngine_NextPanics(t *testing.T) {
	t.Parallel()

	eng, logs := newTestEngine(t, engine.Config{})

	rw := serve(t, eng, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}), http.MethodGet, "/next", "")

	assert.Equal(t, http.StatusBadGateway, rw.Code)
	assert.Contains(t, logs.String(), "Next handler panicked")
}

func TestEngine_Config(t *testing.T) {
	t.Parallel()

	eng, _ := newTestEngine(t, engine.Config{GuestConfig: []byte(`{"a":1}`)})

	rw := serve(t, eng, nil, http.MethodGet, "/config", "")

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.JSONEq(t, `{"a":1}`, rw.Body.String())
}

func TestEngine_Log(t *testing.T) {
	t.Parallel()

	eng, logs := newTestEngine(t, engine.Config{})

	rw := serve(t, eng, nil, http.MethodGet, "/log", "")
	assert.Equal(t, "debug=0", rw.Body.String())
	assert.Contains(t, logs.String(), `"level":"info","message":"hello from guest"`)

	serve(t, eng, nil, http.MethodGet, "/print", "")
	assert.Contains(t, logs.String(), `"stream":"stdout","message":"printed on stdout"`)
}

func TestEngine_Envs(t *testing.T) {
	t.Setenv("PLUGIN_TEST", "MY-TEST")

	eng, _ := newTestEngine(t, engine.Config{Envs: []string{"PLUGIN_TEST"}})

	rw := serve(t, eng, nil, http.MethodGet, "/env", "")

	assert.Equal(t, "MY-TEST", rw.Body.String())
}

func TestEngine_Timers(t *testing.T) {
	t.Parallel()

	eng, _ := newTestEngine(t, engine.Config{})

	rw := serve(t, eng, nil, http.MethodGet, "/sleep", "")

	assert.Equal(t, "slept", rw.Body.String())
}

func TestEngine_LargeResponse(t *testing.T) {
	t.Parallel()

	eng, _ := newTestEngine(t, engine.Config{})

	// Larger than the pipes between the guest and the host.
	const size = 1 << 20
	rw := serve(t, eng, nil, http.MethodGet, fmt.Sprintf("/big?size=%d", size), "")

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, size, rw.Body.Len())
}

func TestEngine_LargeBodyThroughNext(t *testing.T) {
	t.Parallel()

	eng, _ := newTestEngine(t, engine.Config{})

	body := strings.Repeat("a", 1<<20)
	rw := serve(t, eng, upstream(0), http.MethodPost, "/rewrite", body)

	assert.Equal(t, http.StatusCreated, rw.Code)
	assert.Equal(t, "UPSTREAM SAW POST /REWRITTEN? UPSTREAM 1048576:"+strings.ToUpper(body), rw.Body.String())
}

func TestEngine_Concurrency(t *testing.T) {
	t.Parallel()

	eng, _ := newTestEngine(t, engine.Config{Instances: 1})

	const (
		n     = 200
		delay = 50 * time.Millisecond
	)
	next := upstream(delay)

	start := time.Now()

	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			body := strings.Repeat("x", i)
			rw := serve(t, eng, next, http.MethodPost, fmt.Sprintf("/next?i=%d", i), body)

			assert.Equal(t, http.StatusCreated, rw.Code)
			assert.Equal(t, fmt.Sprintf("upstream saw POST /next?i=%d example.com %d:%s", i, len(body), body), rw.Body.String())
		})
	}
	wg.Wait()

	// One instance served the requests while their next handlers were waiting: a serial run would take n*delay.
	assert.Less(t, time.Since(start), n*delay/10)
}

func TestEngine_Cancel(t *testing.T) {
	t.Parallel()

	eng, _ := newTestEngine(t, engine.Config{})

	ctx, cancel := context.WithCancel(t.Context())
	rw := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/wait", http.NoBody)

	done := make(chan struct{})
	go func() {
		defer close(done)
		eng.ServeHTTP(rw, req, nil)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the request did not end after the cancellation")
	}

	assert.Equal(t, 499, rw.Code)
}

func TestEngine_InstanceDies(t *testing.T) {
	t.Parallel()

	eng, logs := newTestEngine(t, engine.Config{Instances: 1})

	waiting := make(chan *httptest.ResponseRecorder)
	go func() {
		waiting <- serve(t, eng, nil, http.MethodGet, "/wait", "")
	}()
	time.Sleep(50 * time.Millisecond)

	rw := serve(t, eng, nil, http.MethodGet, "/exit", "")
	assert.Equal(t, http.StatusBadGateway, rw.Code)
	assert.Contains(t, logs.String(), "exit_code(3)")

	// The request in flight on the instance that died gets a 502 too.
	select {
	case rw := <-waiting:
		assert.Equal(t, http.StatusBadGateway, rw.Code)
	case <-time.After(5 * time.Second):
		t.Fatal("the request in flight did not end with the instance")
	}

	// The instance is replaced, and the requests are served again.
	require.Eventually(t, func() bool {
		return eng.Stats().LiveInstances == 1
	}, 5*time.Second, 10*time.Millisecond)

	assert.Equal(t, http.StatusAccepted, serve(t, eng, nil, http.MethodGet, "/echo", "").Code)
}

func TestEngine_Close(t *testing.T) {
	t.Parallel()

	eng, _ := newTestEngine(t, engine.Config{Instances: 2})

	assert.Equal(t, http.StatusAccepted, serve(t, eng, nil, http.MethodGet, "/echo", "").Code)

	require.NoError(t, eng.Close(t.Context()))

	assert.Equal(t, http.StatusServiceUnavailable, serve(t, eng, nil, http.MethodGet, "/echo", "").Code)
}

func TestNew_Errors(t *testing.T) {
	t.Parallel()

	code, err := os.ReadFile(fixture)
	require.NoError(t, err)

	testCases := []struct {
		desc string
		cfg  engine.Config
		err  string
	}{
		{
			desc: "mounts are refused",
			cfg:  engine.Config{Mounts: []string{"/tmp"}},
			err:  "settings.mounts is not supported by plugins built with github.com/traefik/wasm-go",
		},
		{
			desc: "exclusive mode is refused",
			cfg:  engine.Config{Exclusive: true},
			err:  "settings.exclusive is only supported by http-wasm plugins",
		},
		{
			desc: "memory limit too small for the guest",
			cfg:  engine.Config{MemoryLimit: 1 << 20},
			err:  "creating instance 0: compiling guest module: section memory: min 37 pages (2 Mi) over limit of 16 pages (1 Mi)",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			logger := zerolog.Nop()
			test.cfg.Name = "test"
			test.cfg.Code = code
			test.cfg.Cache = wazero.NewCompilationCache()
			test.cfg.Logger = &logger
			test.cfg.Instances = 1

			_, err := New(t.Context(), test.cfg)
			require.ErrorContains(t, err, test.err)
		})
	}
}
