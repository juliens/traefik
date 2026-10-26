package dashboard

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ContentSecurityPolicy(t *testing.T) {
	testCases := []struct {
		desc     string
		handler  Handler
		expected int
	}{
		{
			desc: "OK",
			handler: Handler{
				assets: fstest.MapFS{"foobar.html": &fstest.MapFile{
					Mode:    0o755,
					ModTime: time.Now(),
				}},
			},
			expected: http.StatusOK,
		},
		{
			desc: "Not found",
			handler: Handler{
				assets: fstest.MapFS{},
			},
			expected: http.StatusNotFound,
		},
		{
			desc: "Internal server error",
			handler: Handler{
				assets: errorFS{},
			},
			expected: http.StatusInternalServerError,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/foobar.html", nil)

			rw := httptest.NewRecorder()

			test.handler.ServeHTTP(rw, req)

			assert.Equal(t, test.expected, rw.Code)
			assert.Equal(t, "frame-src 'self' https://traefik.io https://*.traefik.io;", rw.Result().Header.Get("Content-Security-Policy"))
		})
	}
}

func Test_XForwardedPrefix(t *testing.T) {
	testCases := []struct {
		desc     string
		prefix   string
		expected string
	}{
		{
			desc:     "location in X-Forwarded-Prefix",
			prefix:   "//foobar/test",
			expected: "/dashboard/",
		},
		{
			desc:     "scheme in X-Forwarded-Prefix",
			prefix:   "http://foobar",
			expected: "/dashboard/",
		},
		{
			desc:     "path in X-Forwarded-Prefix",
			prefix:   "foobar",
			expected: "/foobar/dashboard/",
		},
	}

	router := mux.NewRouter()
	err := Append(router, "/", fstest.MapFS{"index.html": &fstest.MapFile{
		Mode:    0o755,
		ModTime: time.Now(),
	}})
	require.NoError(t, err)

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			req.Header.Set("X-Forwarded-Prefix", test.prefix)
			rw := httptest.NewRecorder()

			router.ServeHTTP(rw, req)

			assert.Equal(t, http.StatusFound, rw.Code)
			assert.Equal(t, test.expected, rw.Result().Header.Get("Location"))
		})
	}
}

type errorFS struct{}

func (e errorFS) Open(name string) (fs.File, error) {
	return nil, errors.New("oops")
}

func Test_DashboardRouting(t *testing.T) {
	handler := Handler{
		assets: fstest.MapFS{
			"index.html":  &fstest.MapFile{Data: []byte("{{ .APIUrl }}")},
			"foobar.html": &fstest.MapFile{Data: []byte("foobar")},
		},
	}

	testCases := []struct {
		desc             string
		path             string
		expectedCode     int
		expectedLocation string
		expectedBody     string
	}{
		{
			desc:             "root redirects to the dashboard",
			path:             "/",
			expectedCode:     http.StatusMovedPermanently,
			expectedLocation: "/dashboard/",
		},
		{
			desc:         "index of the dashboard",
			path:         "/dashboard/",
			expectedCode: http.StatusOK,
			expectedBody: "/api/",
		},
		{
			desc:         "asset of the dashboard",
			path:         "/dashboard/foobar.html",
			expectedCode: http.StatusOK,
			expectedBody: "foobar",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			rw := httptest.NewRecorder()
			handler.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, test.path, nil))

			assert.Equal(t, test.expectedCode, rw.Code)
			assert.Equal(t, test.expectedLocation, rw.Header().Get("Location"))
			if test.expectedBody != "" {
				assert.Equal(t, test.expectedBody, rw.Body.String())
			}
		})
	}
}
