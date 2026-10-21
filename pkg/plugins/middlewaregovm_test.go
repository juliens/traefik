package plugins

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
