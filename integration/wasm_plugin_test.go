package integration

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/traefik/traefik/v3/integration/try"
)

// WasmPluginSuite runs the wasm plugin fixtures of pkg/plugins as local plugins.
type WasmPluginSuite struct {
	BaseSuite

	dynamicFile string
}

func TestWasmPluginSuite(t *testing.T) {
	suite.Run(t, new(WasmPluginSuite))
}

// SetupSuite does not call the BaseSuite one: the suite runs Traefik and a local backend, without Docker.
func (s *WasmPluginSuite) SetupSuite() {
	// Local plugins are read from ./plugins-local, which is ignored by git.
	installLocalPlugin(s.T(), "github.com/traefik/gowasmdemo", "../pkg/plugins/fixtures/wasihttp/plugin.wasm")
	installLocalPlugin(s.T(), "github.com/traefik/httpwasmdemo", "../pkg/plugins/fixtures/withnext/plugin.wasm")
}

func (s *WasmPluginSuite) TearDownSuite() {
	require.NoError(s.T(), os.RemoveAll("./plugins-local"))
}

func (s *WasmPluginSuite) SetupTest() {
	s.dynamicFile = filepath.Join(s.T().TempDir(), "dynamic.toml")
}

func installLocalPlugin(t *testing.T, moduleName, wasmPath string) {
	t.Helper()

	dir := filepath.Join("plugins-local", "src", moduleName)
	require.NoError(t, os.MkdirAll(dir, 0o755))

	code, err := os.ReadFile(wasmPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.wasm"), code, 0o644))

	manifest := fmt.Sprintf("displayName: %s\ntype: middleware\nruntime: wasm\nsummary: Integration test plugin.\ntestData: {}\n", moduleName)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".traefik.yml"), []byte(manifest), 0o644))
}

func (s *WasmPluginSuite) TestPlugins() {
	s.startBackend()
	s.writeDynamic("a", "1")

	file := s.adaptFile("fixtures/wasm/plugins.toml", struct{ DynamicFile string }{s.dynamicFile})
	s.traefikCmd(withConfigFile(file))

	// A plugin built with the Go SDK ABI, through to the backend.
	err := try.GetRequest("http://127.0.0.1:8000/next", 30*time.Second,
		try.StatusCodeIs(http.StatusOK), try.BodyContains("backend saw X-Plugin=pass"))
	require.NoError(s.T(), err)

	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:8000/next", strings.NewReader("ping"))
	require.NoError(s.T(), err)
	err = try.Request(req, 5*time.Second, try.StatusCodeIs(http.StatusOK), try.BodyContains("body=ping"))
	require.NoError(s.T(), err)

	err = try.GetRequest("http://127.0.0.1:8000/config", 5*time.Second, try.BodyContains(`{"value":"1"}`))
	require.NoError(s.T(), err)

	// An http-wasm plugin built with TinyGo.
	err = try.GetRequest("http://127.0.0.1:8000/tinygo", 5*time.Second,
		try.StatusCodeIs(http.StatusOK), try.BodyContains("backend saw X-Plugin=a"))
	require.NoError(s.T(), err)

	// A new routing configuration replaces the middleware configurations.
	s.writeDynamic("b", "2")

	err = try.GetRequest("http://127.0.0.1:8000/tinygo", 30*time.Second, try.BodyContains("backend saw X-Plugin=b"))
	require.NoError(s.T(), err)

	err = try.GetRequest("http://127.0.0.1:8000/config", 5*time.Second, try.BodyContains(`{"value":"2"}`))
	require.NoError(s.T(), err)
}

func (s *WasmPluginSuite) TestConcurrentRequests() {
	release := make(chan struct{})
	listener, err := net.Listen("tcp", "127.0.0.1:9000")
	require.NoError(s.T(), err)

	backend := &httptest.Server{
		Listener: listener,
		Config: &http.Server{Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Query().Get("wait") != "" {
				<-release
			}
			_, _ = fmt.Fprintf(rw, "backend saw X-Plugin=%s", req.Header.Get("X-Plugin"))
		})},
	}
	backend.Start()
	s.T().Cleanup(backend.Close)
	s.T().Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})

	s.writeDynamic("a", "1")

	file := s.adaptFile("fixtures/wasm/plugins.toml", struct{ DynamicFile string }{s.dynamicFile})
	s.traefikCmd(withConfigFile(file))

	err = try.GetRequest("http://127.0.0.1:8000/next", 30*time.Second, try.BodyContains("backend saw X-Plugin=pass"))
	require.NoError(s.T(), err)

	// With one instance per plugin, requests waiting on the backend do not hold it.
	const waiting = 20
	results := make(chan error, waiting)
	for range waiting {
		go func() {
			resp, err := http.Get("http://127.0.0.1:8000/next?wait=1")
			if err == nil {
				_ = resp.Body.Close()
			}
			results <- err
		}()
	}

	for _, path := range []string{"/next", "/tinygo"} {
		err = try.GetRequest("http://127.0.0.1:8000"+path, 5*time.Second, try.StatusCodeIs(http.StatusOK))
		require.NoError(s.T(), err)
	}

	close(release)
	for range waiting {
		require.NoError(s.T(), <-results)
	}
}

// writeDynamic writes the routing configuration: one router per plugin, both to the backend.
func (s *WasmPluginSuite) writeDynamic(header, config string) {
	s.T().Helper()

	dynamic := fmt.Sprintf(`
[http.routers.gowasm]
  rule = "Path(`+"`/next`"+`) || Path(`+"`/config`"+`)"
  middlewares = ["gowasm"]
  service = "backend"

[http.routers.httpwasm]
  rule = "Path(`+"`/tinygo`"+`)"
  middlewares = ["httpwasm"]
  service = "backend"

[http.middlewares.gowasm.plugin.gowasm]
  value = %q

[http.middlewares.httpwasm.plugin.httpwasm]
  header = %q

[http.services.backend.loadBalancer]
  [[http.services.backend.loadBalancer.servers]]
    url = "http://127.0.0.1:9000"
`, config, header)

	// Write then rename, so that the file provider never reads a partial file.
	tmp := s.dynamicFile + ".tmp"
	require.NoError(s.T(), os.WriteFile(tmp, []byte(dynamic), 0o644))
	require.NoError(s.T(), os.Rename(tmp, s.dynamicFile))
}

func (s *WasmPluginSuite) startBackend() {
	s.T().Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:9000")
	require.NoError(s.T(), err)

	backend := &httptest.Server{
		Listener: listener,
		Config: &http.Server{Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			body, _ := io.ReadAll(req.Body)
			_, _ = fmt.Fprintf(rw, "backend saw X-Plugin=%s body=%s", req.Header.Get("X-Plugin"), body)
		})},
	}
	backend.Start()
	s.T().Cleanup(backend.Close)
}
