package plugins

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/safe"
)

func TestBuildProvider(t *testing.T) {
	manifest := &Manifest{Type: typeProvider, Import: "testpluginprovider"}

	pBuilder, err := newProviderBuilder(t.Context(), manifest, "fixtures", false, "testpluginprovider", Settings{})
	require.NoError(t, err)

	builder := Builder{providerBuilders: map[string]providerBuilder{"test": pBuilder}}

	_, err = builder.BuildProvider("test", nil)
	require.ErrorContains(t, err, "no names")

	prov, err := builder.BuildProvider("test", map[string]any{"names": "foo,bar"})
	require.NoError(t, err)

	require.NoError(t, prov.Init())

	pool := safe.NewPool(t.Context())
	configurationChan := make(chan dynamic.Message)

	t.Cleanup(func() {
		// The provider keeps sending configurations until it is stopped.
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-configurationChan:
				case <-done:
					return
				}
			}
		}()

		pool.Stop()
		close(done)
	})
	require.NoError(t, prov.Provide(configurationChan, pool))

	select {
	case msg := <-configurationChan:
		assert.Equal(t, "plugin-test", msg.ProviderName)
		require.NotNil(t, msg.Configuration.HTTP)
		require.Len(t, msg.Configuration.HTTP.Routers, 2)
		assert.Equal(t, "Host(`foo.localhost`)", msg.Configuration.HTTP.Routers["foo"].Rule)
		assert.Equal(t, "bar", msg.Configuration.HTTP.Routers["bar"].Service)
		require.NotNil(t, msg.Configuration.HTTP.Services["foo"].LoadBalancer)
		require.Len(t, msg.Configuration.HTTP.Services["foo"].LoadBalancer.Servers, 1)
		assert.Equal(t, "http://127.0.0.1:9000", msg.Configuration.HTTP.Services["foo"].LoadBalancer.Servers[0].URL)

	case <-time.After(5 * time.Second):
		t.Fatal("no configuration from the provider plugin")
	}
}
