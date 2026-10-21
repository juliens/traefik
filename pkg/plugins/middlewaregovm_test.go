package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/go-vm/vm"
)

func TestGoVMMiddleware(t *testing.T) {
	manifest := &Manifest{Type: typeMiddleware, Import: "testpluginsafe"}

	builder, err := newMiddlewareBuilder(t.Context(), "fixtures", manifest, "testpluginsafe", Settings{})
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

	builder, err := newMiddlewareBuilder(t.Context(), "fixtures", manifest, "testpluginsafe", Settings{})
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

func TestGoVMMiddleware_requestContextCanceled(t *testing.T) {
	manifest := &Manifest{Type: typeMiddleware, Import: "testplugincontext"}

	builder, err := newMiddlewareBuilder(t.Context(), "fixtures", manifest, "testplugincontext", Settings{})
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
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			manifest := &Manifest{Type: typeMiddleware, Import: "testpluginlimits"}

			builder, err := newMiddlewareBuilder(t.Context(), "fixtures", manifest, "testpluginlimits", Settings{})
			require.NoError(t, err)

			middleware, err := builder.newMiddleware(map[string]any{"mode": test.mode}, "test-middleware")
			require.NoError(t, err)

			handler, err := middleware.NewHandler(t.Context(), http.NotFoundHandler())
			require.NoError(t, err)

			recovered := func() (r any) {
				defer func() { r = recover() }()

				handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://localhost/", nil))

				return nil
			}()

			err, ok := recovered.(error)
			require.True(t, ok, "%v", recovered)
			assert.ErrorIs(t, err, test.expectedPanic)
		})
	}
}

func TestGoVMMiddleware_restrictions(t *testing.T) {
	testCases := []struct {
		desc           string
		moduleName     string
		manifestUnsafe bool
		settingsUnsafe bool
		expectedError  string
	}{
		{
			desc:          "os/exec is refused",
			moduleName:    "testpluginexec",
			expectedError: "os/exec",
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
			desc:           "syscall functions are refused even when the manifest and the settings allow useUnsafe",
			moduleName:     "testpluginsyscall",
			manifestUnsafe: true,
			settingsUnsafe: true,
			expectedError:  "syscall.Getpid not in registry",
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
			settings := Settings{UseUnsafe: test.settingsUnsafe}

			builder, err := newMiddlewareBuilder(t.Context(), "fixtures", manifest, test.moduleName, settings)
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
