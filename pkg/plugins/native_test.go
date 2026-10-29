package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/safe"
)

type builtinProviderConfig struct {
	Names    []string
	Throttle string
}

type builtinProvider struct {
	config *builtinProviderConfig
	name   string
}

func (p *builtinProvider) Init() error { return nil }

func (p *builtinProvider) Provide(configurationChan chan<- dynamic.Message, _ *safe.Pool) error {
	conf := &dynamic.Configuration{HTTP: &dynamic.HTTPConfiguration{Routers: map[string]*dynamic.Router{}}}
	for _, name := range p.config.Names {
		conf.HTTP.Routers[name] = &dynamic.Router{Rule: "Host(`" + name + "`)"}
	}

	configurationChan <- dynamic.Message{ProviderName: p.name, Configuration: conf}

	return nil
}

func (p *builtinProvider) ThrottleDuration() time.Duration {
	d, _ := time.ParseDuration(p.config.Throttle)
	return d
}

func registerBuiltin(t *testing.T, moduleName, manifest string, functions map[string]any) {
	t.Helper()

	RegisterBuiltin(moduleName, manifest, functions)

	t.Cleanup(func() {
		delete(builtinPlugins, moduleName)
		delete(builtinManifests, moduleName)
	})
}

func TestBuiltinPlugins(t *testing.T) {
	registerBuiltin(t, "example.com/builtin/provider", "type: provider\nimport: example.com/builtin/provider\n", map[string]any{
		"CreateConfig": func() *builtinProviderConfig { return &builtinProviderConfig{Throttle: "1s"} },
		"New": func(_ context.Context, config *builtinProviderConfig, name string) (*builtinProvider, error) {
			return &builtinProvider{config: config, name: name}, nil
		},
	})

	registerBuiltin(t, "example.com/builtin/middlewares", "type: middleware\nimport: example.com/builtin/middlewares\nimplements:\n  - addPrefix\n", map[string]any{
		"NewAddPrefix": func(_ context.Context, next http.Handler, config *dynamic.AddPrefix, _ string) (http.Handler, error) {
			return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				req.URL.Path = config.Prefix + req.URL.Path
				next.ServeHTTP(rw, req)
			}), nil
		},
	})

	builder, err := NewBuilder(nil, nil, map[string]LocalDescriptor{
		"test": {ModuleName: "example.com/builtin/provider"},
	})
	require.NoError(t, err)

	prov, err := builder.BuildProvider("test", map[string]any{"names": "foo,bar", "throttle": "3s"})
	require.NoError(t, err)

	throttled, ok := prov.(interface{ ThrottleDuration() time.Duration })
	require.True(t, ok)
	assert.Equal(t, 3*time.Second, throttled.ThrottleDuration())

	require.NoError(t, prov.Init())

	configurationChan := make(chan dynamic.Message, 1)
	require.NoError(t, prov.Provide(configurationChan, safe.NewPool(t.Context())))

	msg := <-configurationChan
	assert.Equal(t, "plugin-test", msg.ProviderName)
	assert.Len(t, msg.Configuration.HTTP.Routers, 2)

	// A plugin compiled into Traefik implements its options without a declaration in the static configuration.
	assert.True(t, builder.ImplementsMiddleware("addPrefix"))
	assert.False(t, builder.ImplementsMiddleware("stripPrefix"))

	constructor, err := builder.BuildTyped("addPrefix", &dynamic.AddPrefix{Prefix: "/api"}, "prefix")
	require.NoError(t, err)

	var path string
	handler, err := constructor(t.Context(), http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		path = req.URL.Path
	}))
	require.NoError(t, err)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/foo", nil))
	assert.Equal(t, "/api/foo", path)
}

type typedProviderConfig struct {
	Watch    bool
	Names    []string
	Throttle string
}

func TestBuiltinPlugins_typedProvider(t *testing.T) {
	var got *typedProviderConfig
	registerBuiltin(t, "example.com/builtin/typed", "type: provider\nimport: example.com/builtin/typed\nimplements:\n  - docker\n", map[string]any{
		"CreateConfig": func() *typedProviderConfig { return &typedProviderConfig{Watch: true, Throttle: "1s"} },
		"New": func(_ context.Context, config *typedProviderConfig, name string) (*builtinProvider, error) {
			got = config
			return &builtinProvider{config: &builtinProviderConfig{Names: config.Names}, name: name}, nil
		},
	})

	builder, err := NewBuilder(nil, nil, nil)
	require.NoError(t, err)

	assert.True(t, builder.ImplementsProvider("docker"))
	assert.False(t, builder.ImplementsProvider("file"))

	// The option set to false in the static configuration must not get the default value of the plugin.
	config := &struct {
		Watch    bool     `json:"watch,omitempty"`
		Names    []string `json:"names,omitempty"`
		Throttle string   `json:"throttle,omitempty"`
		Ignored  string   `json:"-"`
	}{Names: []string{"foo"}, Throttle: "2s", Ignored: "bar"}

	prov, err := builder.BuildTypedProvider("docker", config)
	require.NoError(t, err)

	assert.Equal(t, &typedProviderConfig{Watch: false, Names: []string{"foo"}, Throttle: "2s"}, got)
	assert.Equal(t, "docker", prov.(*pluginProvider).Provider.(*builtinProvider).name)

	_, err = builder.BuildTypedProvider("file", config)
	require.ErrorContains(t, err, `no plugin implements the "file" option`)
}

func TestBuiltinPlugins_wrongTypedConstructor(t *testing.T) {
	registerBuiltin(t, "example.com/builtin/wrong", "type: middleware\nimport: example.com/builtin/wrong\nimplements:\n  - addPrefix\n", map[string]any{
		"NewAddPrefix": func(_ context.Context, next http.Handler, _ *dynamic.StripPrefix, _ string) (http.Handler, error) {
			return next, nil
		},
	})

	_, err := NewBuilder(nil, nil, map[string]LocalDescriptor{"wrong": {ModuleName: "example.com/builtin/wrong"}})
	require.ErrorContains(t, err, "must take a *dynamic.AddPrefix configuration")
}
