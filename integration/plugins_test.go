package integration

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/traefik/traefik/v3/integration/try"
)

// PluginsSuite tests suite.
type PluginsSuite struct{ BaseSuite }

func TestPluginsSuite(t *testing.T) {
	suite.Run(t, new(PluginsSuite))
}

func (s *PluginsSuite) SetupTest() {
	// Traefik reads the local plugins from ./plugins-local, relative to its working directory.
	require.NoError(s.T(), os.CopyFS("plugins-local/src", os.DirFS("fixtures/plugins/src")))
}

func (s *PluginsSuite) TearDownTest() {
	_ = os.RemoveAll("plugins-local")
}

func (s *PluginsSuite) TestLocalProviderPlugin() {
	backend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("X-Backend", "govm-provider")
		rw.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	file := s.adaptFile("fixtures/plugins/provider.toml", struct {
		Server string
	}{
		Server: backend.URL,
	})

	s.traefikCmd(withConfigFile(file))

	err := try.GetRequest("http://127.0.0.1:8000/", 10*time.Second, try.StatusCodeIs(http.StatusOK), try.HasHeaderValue("X-Backend", "govm-provider", true))
	require.NoError(s.T(), err)
}

func (s *PluginsSuite) TestLocalMiddlewarePlugin() {
	backend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	file := s.adaptFile("fixtures/plugins/middleware.toml", struct {
		Server string
	}{
		Server: backend.URL,
	})

	s.traefikCmd(withConfigFile(file))

	err := try.GetRequest("http://127.0.0.1:8000/", 10*time.Second, try.StatusCodeIs(http.StatusOK), try.HasHeaderValue("X-Govm-Plugin", "govm", true))
	require.NoError(s.T(), err)

	// The execution budget stops the infinite loop of the plugin.
	err = try.GetRequest("http://127.0.0.1:8000/loop", 5*time.Second, try.StatusCodeIs(http.StatusInternalServerError))
	require.NoError(s.T(), err)

	err = try.GetRequest("http://127.0.0.1:8000/", time.Second, try.StatusCodeIs(http.StatusOK))
	require.NoError(s.T(), err)
}
