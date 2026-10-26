package plugins

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/go-vm/vm"
)

func TestGoVMMiddleware(t *testing.T) {
	manifest := &Manifest{Type: typeMiddleware, Import: "testpluginsafe"}

	builder, err := newMiddlewareBuilder(t.Context(), "fixtures", false, manifest, "testpluginsafe", Settings{})
	require.NoError(t, err)
	require.IsType(t, &goVMMiddlewareBuilder{}, builder)

	middleware, err := builder.newMiddleware(map[string]any{"message": "test"}, "test-middleware")
	require.NoError(t, err)

	var nextCalled bool
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		nextCalled = true
		rw.WriteHeader(http.StatusTeapot)
	})

	handler, err := middleware.NewHandler(t.Context(), next)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://localhost/", nil))

	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusTeapot, recorder.Code)
	assert.Equal(t, "safe", recorder.Header().Get("X-Test-Plugin"))
}

func TestGoVMMiddleware_concurrentRequests(t *testing.T) {
	manifest := &Manifest{Type: typeMiddleware, Import: "testpluginsafe"}

	builder, err := newMiddlewareBuilder(t.Context(), "fixtures", false, manifest, "testpluginsafe", Settings{})
	require.NoError(t, err)

	middleware, err := builder.newMiddleware(nil, "test-middleware")
	require.NoError(t, err)

	handler, err := middleware.NewHandler(t.Context(), http.NotFoundHandler())
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 50 {
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://localhost/", nil))

				assert.Equal(t, "safe", recorder.Header().Get("X-Test-Plugin"))
			}
		})
	}

	wg.Wait()
}

func TestGoVMMiddleware_flusher(t *testing.T) {
	manifest := &Manifest{Type: typeMiddleware, Import: "testpluginflusher"}

	builder, err := newMiddlewareBuilder(t.Context(), "fixtures", false, manifest, "testpluginflusher", Settings{})
	require.NoError(t, err)

	middleware, err := builder.newMiddleware(nil, "test-middleware")
	require.NoError(t, err)

	handler, err := middleware.NewHandler(t.Context(), http.NotFoundHandler())
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(&unwrapOnlyWriter{ResponseWriter: recorder}, httptest.NewRequest(http.MethodGet, "http://localhost/", nil))

	assert.Equal(t, "true", recorder.Header().Get("X-Flusher"))
}

func TestGoVMMiddleware_requestContextCanceled(t *testing.T) {
	manifest := &Manifest{Type: typeMiddleware, Import: "testplugincontext"}

	builder, err := newMiddlewareBuilder(t.Context(), "fixtures", false, manifest, "testplugincontext", Settings{})
	require.NoError(t, err)

	middleware, err := builder.newMiddleware(nil, "test-middleware")
	require.NoError(t, err)

	handler, err := middleware.NewHandler(t.Context(), http.NotFoundHandler())
	require.NoError(t, err)

	result := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		handler.ServeHTTP(rw, req)
		result <- rw.Header().Get("X-Context")
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	// The client goes away before the plugin answers.
	_, err = http.DefaultClient.Do(req)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	select {
	case got := <-result:
		assert.Equal(t, context.Canceled.Error(), got)
	case <-time.After(5 * time.Second):
		t.Fatal("the plugin did not see the cancellation of the request context")
	}
}

func TestGoVMMiddleware_limits(t *testing.T) {
	testCases := []struct {
		desc          string
		mode          string
		goVM          GoVMSettings
		expectedPanic error
	}{
		{
			desc:          "infinite loop is stopped by the budget",
			mode:          "loop",
			expectedPanic: vm.ErrBudget,
		},
		{
			desc:          "goroutines are limited",
			mode:          "goroutines",
			expectedPanic: vm.ErrTooManyGoroutines,
		},
		{
			desc: "work within the default budget",
			mode: "work",
		},
		{
			desc:          "settings.goVM.budget lowers the budget",
			mode:          "work",
			goVM:          GoVMSettings{Budget: 1_000},
			expectedPanic: vm.ErrBudget,
		},
		{
			desc:          "a long loop passes the default budget",
			mode:          "loop-bounded",
			expectedPanic: vm.ErrBudget,
		},
		{
			desc: "settings.goVM.budget raises the budget",
			mode: "loop-bounded",
			goVM: GoVMSettings{Budget: 1_000_000_000},
		},
		{
			desc: "goroutines within the default limit",
			mode: "some-goroutines",
		},
		{
			desc:          "settings.goVM.maxGoroutines lowers the limit",
			mode:          "some-goroutines",
			goVM:          GoVMSettings{MaxGoroutines: 10},
			expectedPanic: vm.ErrTooManyGoroutines,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			manifest := &Manifest{Type: typeMiddleware, Import: "testpluginlimits"}

			builder, err := newMiddlewareBuilder(t.Context(), "fixtures", false, manifest, "testpluginlimits", Settings{GoVM: test.goVM})
			require.NoError(t, err)

			middleware, err := builder.newMiddleware(map[string]any{"mode": test.mode}, "test-middleware")
			require.NoError(t, err)

			handler, err := middleware.NewHandler(t.Context(), http.NotFoundHandler())
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			recovered := func() (r any) {
				defer func() { r = recover() }()

				handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://localhost/", nil))

				return nil
			}()

			if test.expectedPanic == nil {
				require.Nil(t, recovered)
				assert.Equal(t, http.StatusNotFound, recorder.Code)
				return
			}

			err, ok := recovered.(error)
			require.True(t, ok, "%v", recovered)
			assert.ErrorIs(t, err, test.expectedPanic)
		})
	}
}

func TestGoVMLimits(t *testing.T) {
	testCases := []struct {
		desc                  string
		settings              GoVMSettings
		expectedBudget        int64
		expectedMaxGoroutines int
		expectedError         string
	}{
		{
			desc:                  "defaults",
			expectedBudget:        goVMBudget,
			expectedMaxGoroutines: goVMMaxGoroutines,
		},
		{
			desc:                  "values of the settings",
			settings:              GoVMSettings{Budget: 5_000, MaxGoroutines: 5_000},
			expectedBudget:        5_000,
			expectedMaxGoroutines: 5_000,
		},
		{
			desc:          "negative budget",
			settings:      GoVMSettings{Budget: -1},
			expectedError: "settings.goVM.budget",
		},
		{
			desc:          "negative maxGoroutines",
			settings:      GoVMSettings{MaxGoroutines: -1},
			expectedError: "settings.goVM.maxGoroutines",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			budget, maxGoroutines, err := goVMLimits(test.settings)
			if test.expectedError != "" {
				require.ErrorContains(t, err, test.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedBudget, budget)
			assert.Equal(t, test.expectedMaxGoroutines, maxGoroutines)
		})
	}
}

// TestGoVMAOTCopies fails when Traefik is built with a Go release that the AOT copies of go-vm were not made for:
// update go-vm with the Go toolchain, else the plugins that run the standard library as AOT copies are refused.
func TestGoVMAOTCopies(t *testing.T) {
	require.NoError(t, vm.CheckAOT(false))
}

func TestGoVMMiddleware_localPluginWithoutGoMod(t *testing.T) {
	// The fixtures/_gopath directory is a plugins GOPATH, as ./plugins-local/:
	// testpluginnogomod has no go.mod, as Yaegi allowed for local plugins.
	manifest := &Manifest{Type: typeMiddleware, Import: "testpluginnogomod"}

	builder, err := newMiddlewareBuilder(t.Context(), "fixtures/_gopath", true, manifest, "testpluginnogomod", Settings{})
	require.NoError(t, err)

	middleware, err := builder.newMiddleware(nil, "test-middleware")
	require.NoError(t, err)

	handler, err := middleware.NewHandler(t.Context(), http.NotFoundHandler())
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://localhost/", nil))

	assert.Equal(t, "nogomod", recorder.Header().Get("X-Test-Plugin"))

	// A downloaded plugin must have its go.mod.
	_, err = newMiddlewareBuilder(t.Context(), "fixtures/_gopath", false, manifest, "testpluginnogomod", Settings{})
	require.ErrorContains(t, err, "no go.mod")
}

func TestGoVMMiddleware_notesAreLogged(t *testing.T) {
	var logs bytes.Buffer
	ctx := zerolog.New(&logs).WithContext(t.Context())

	// platform_onlyplan9.go has only a "//go:build" line: Yaegi kept it, go-vm leaves it out.
	manifest := &Manifest{Type: typeMiddleware, Import: "testplugingobuild"}

	builder, err := newMiddlewareBuilder(ctx, "fixtures", false, manifest, "testplugingobuild", Settings{})
	require.NoError(t, err)

	assert.Contains(t, logs.String(), `"level":"warn"`)
	assert.Contains(t, logs.String(), "platform_onlyplan9.go")

	middleware, err := builder.newMiddleware(nil, "test-middleware")
	require.NoError(t, err)

	handler, err := middleware.NewHandler(t.Context(), http.NotFoundHandler())
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://localhost/", nil))

	assert.Equal(t, "other", recorder.Header().Get("X-Test-Plugin"))
}

func TestGoVMMiddleware_restrictions(t *testing.T) {
	testCases := []struct {
		desc           string
		moduleName     string
		manifestUnsafe bool
		settingsUnsafe bool
		goVM           GoVMSettings
		expectedError  string
	}{
		{
			desc:          "os/exec is refused",
			moduleName:    "testpluginexec",
			expectedError: "os/exec",
		},
		{
			desc:       "os/exec is allowed with settings.goVM.execMock",
			moduleName: "testpluginexec",
			goVM:       GoVMSettings{ExecMock: true},
		},
		{
			desc:          "a function written in assembly is refused",
			moduleName:    "testpluginasm",
			expectedError: "assembly",
		},
		{
			desc:       "a function written in assembly is allowed with settings.goVM.stubAssembly",
			moduleName: "testpluginasm",
			goVM:       GoVMSettings{StubAssembly: true},
		},
		{
			desc:       "a plugin runs without the AOT mode with settings.goVM.disableAOT",
			moduleName: "testpluginsafe",
			goVM:       GoVMSettings{DisableAOT: true},
		},
		{
			desc:          "os.StartProcess is refused",
			moduleName:    "testpluginprocess",
			expectedError: "StartProcess",
		},
		{
			desc:           "unsafe is refused when the settings do not allow it",
			moduleName:     "testpluginunsafe",
			manifestUnsafe: true,
			expectedError:  "restricted imports",
		},
		{
			desc:          "unsafe is refused when the manifest does not ask for it",
			moduleName:    "testpluginunsafe",
			expectedError: "unsafe",
		},
		{
			desc:          "syscall is refused when the manifest does not ask for it",
			moduleName:    "testpluginsyscall",
			expectedError: "syscall",
		},
		{
			desc:           "read-only syscall functions are allowed when the manifest and the settings allow useUnsafe",
			moduleName:     "testpluginsyscall",
			manifestUnsafe: true,
			settingsUnsafe: true,
		},
		{
			desc:           "syscall functions that change the process are refused even when the manifest and the settings allow useUnsafe",
			moduleName:     "testpluginsyscallkill",
			manifestUnsafe: true,
			settingsUnsafe: true,
			expectedError:  "syscall.Kill",
		},
		{
			desc:           "unsafe is allowed when the manifest and the settings allow it",
			moduleName:     "testpluginunsafe",
			manifestUnsafe: true,
			settingsUnsafe: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			manifest := &Manifest{Type: typeMiddleware, Import: test.moduleName, UseUnsafe: test.manifestUnsafe}
			settings := Settings{UseUnsafe: test.settingsUnsafe, GoVM: test.goVM}

			builder, err := newMiddlewareBuilder(t.Context(), "fixtures", false, manifest, test.moduleName, settings)
			if test.expectedError != "" {
				require.ErrorContains(t, err, test.expectedError)
				return
			}

			require.NoError(t, err)

			middleware, err := builder.newMiddleware(nil, "test-middleware")
			require.NoError(t, err)

			_, err = middleware.NewHandler(t.Context(), http.NotFoundHandler())
			require.NoError(t, err)
		})
	}
}

func TestDecodePluginConfig_emptyValue(t *testing.T) {
	type challenge struct {
		Delay string
	}

	var result struct {
		TLSChallenge *challenge
		Labels       map[string]string
		Name         string
	}

	err := decodePluginConfig(map[string]any{"tlsChallenge": "", "labels": "", "name": ""}, &result)
	require.NoError(t, err)

	assert.Equal(t, &challenge{}, result.TLSChallenge)
	assert.Empty(t, result.Labels)
	assert.Empty(t, result.Name)
}
