package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/plugins"
	"github.com/traefik/traefik/v3/pkg/server/provider"
)

func TestBuilder_BuildChainNilConfig(t *testing.T) {
	testConfig := map[string]*runtime.MiddlewareInfo{
		"empty": {},
	}
	middlewaresBuilder := NewBuilder(testConfig, nil, nil)

	chain := middlewaresBuilder.BuildMiddlewareChain(t.Context(), []string{"empty"})
	_, err := chain.Then(nil)
	require.Error(t, err)
}

func TestBuilder_BuildChainNonExistentChain(t *testing.T) {
	testConfig := map[string]*runtime.MiddlewareInfo{
		"foobar": {},
	}
	middlewaresBuilder := NewBuilder(testConfig, nil, nil)

	chain := middlewaresBuilder.BuildMiddlewareChain(t.Context(), []string{"empty"})
	_, err := chain.Then(nil)
	require.Error(t, err)
}

func TestBuilder_BuildChainWithContext(t *testing.T) {
	testCases := []struct {
		desc            string
		buildChain      []string
		configuration   map[string]*dynamic.Middleware
		expected        map[string]string
		contextProvider string
		expectedError   error
	}{
		{
			desc:       "Simple middleware",
			buildChain: []string{"middleware-1"},
			configuration: map[string]*dynamic.Middleware{
				"middleware-1": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{"middleware-1": "value-middleware-1"},
					},
				},
			},
			expected: map[string]string{"middleware-1": "value-middleware-1"},
		},
		{
			desc:       "Middleware that references a chain",
			buildChain: []string{"middleware-chain-1"},
			configuration: map[string]*dynamic.Middleware{
				"middleware-1": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{"middleware-1": "value-middleware-1"},
					},
				},
				"middleware-chain-1": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"middleware-1"},
					},
				},
			},
			expected: map[string]string{"middleware-1": "value-middleware-1"},
		},
		{
			desc:       "Should suffix the middlewareName with the provider in the context",
			buildChain: []string{"middleware-1"},
			configuration: map[string]*dynamic.Middleware{
				"middleware-1@provider-1": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{"middleware-1@provider-1": "value-middleware-1"},
					},
				},
			},
			expected:        map[string]string{"middleware-1@provider-1": "value-middleware-1"},
			contextProvider: "provider-1",
		},
		{
			desc:       "Should not suffix a qualified middlewareName with the provider in the context",
			buildChain: []string{"middleware-1@provider-1"},
			configuration: map[string]*dynamic.Middleware{
				"middleware-1@provider-1": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{"middleware-1@provider-1": "value-middleware-1"},
					},
				},
			},
			expected:        map[string]string{"middleware-1@provider-1": "value-middleware-1"},
			contextProvider: "provider-1",
		},
		{
			desc:       "Should be context aware if a chain references another middleware",
			buildChain: []string{"middleware-chain-1@provider-1"},
			configuration: map[string]*dynamic.Middleware{
				"middleware-1@provider-1": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{"middleware-1": "value-middleware-1"},
					},
				},
				"middleware-chain-1@provider-1": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"middleware-1"},
					},
				},
			},
			expected: map[string]string{"middleware-1": "value-middleware-1"},
		},
		{
			desc:       "Should handle nested chains with different context",
			buildChain: []string{"middleware-chain-1@provider-1", "middleware-chain-1"},
			configuration: map[string]*dynamic.Middleware{
				"middleware-1@provider-1": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{"middleware-1": "value-middleware-1"},
					},
				},
				"middleware-2@provider-1": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{"middleware-2": "value-middleware-2"},
					},
				},
				"middleware-chain-1@provider-1": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"middleware-1"},
					},
				},
				"middleware-chain-2@provider-1": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"middleware-2"},
					},
				},
				"middleware-chain-1@provider-2": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"middleware-2@provider-1", "middleware-chain-2@provider-1"},
					},
				},
			},
			expected:        map[string]string{"middleware-1": "value-middleware-1", "middleware-2": "value-middleware-2"},
			contextProvider: "provider-2",
		},
		{
			desc:       "Detects recursion in Middleware chain",
			buildChain: []string{"m1"},
			configuration: map[string]*dynamic.Middleware{
				"ok": {
					Retry: &dynamic.Retry{},
				},
				"m1": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"m2"},
					},
				},
				"m2": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"ok", "m3"},
					},
				},
				"m3": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"m1"},
					},
				},
			},
			expectedError: errors.New("could not instantiate middleware m1: recursion detected in middleware:m1->middleware:m2->middleware:m3->middleware:m1"),
		},
		{
			desc:       "Detects recursion in Middleware chain",
			buildChain: []string{"m1@provider"},
			configuration: map[string]*dynamic.Middleware{
				"ok@provider2": {
					Retry: &dynamic.Retry{},
				},
				"m1@provider": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"m2@provider2"},
					},
				},
				"m2@provider2": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"ok", "m3@provider"},
					},
				},
				"m3@provider": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"m1"},
					},
				},
			},
			expectedError: errors.New("could not instantiate middleware m1@provider: recursion detected in middleware:m1@provider->middleware:m2@provider2->middleware:m3@provider->middleware:m1@provider"),
		},
		{
			desc:       "Detects recursion in Middleware chain",
			buildChain: []string{"ok", "m0"},
			configuration: map[string]*dynamic.Middleware{
				"ok": {
					Retry: &dynamic.Retry{},
				},
				"m0": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"m0"},
					},
				},
			},
			expectedError: errors.New("could not instantiate middleware m0: recursion detected in middleware:m0->middleware:m0"),
		},
		{
			desc:       "Detects MiddlewareChain that references a Chain that references a Chain with a missing middleware",
			buildChain: []string{"m0"},
			configuration: map[string]*dynamic.Middleware{
				"m0": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"m1"},
					},
				},
				"m1": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"m2"},
					},
				},
				"m2": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"m3"},
					},
				},
				"m3": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"m2"},
					},
				},
			},
			expectedError: errors.New("could not instantiate middleware m2: recursion detected in middleware:m0->middleware:m1->middleware:m2->middleware:m3->middleware:m2"),
		},
		{
			desc:       "--",
			buildChain: []string{"m0"},
			configuration: map[string]*dynamic.Middleware{
				"m0": {
					Chain: &dynamic.Chain{
						Middlewares: []string{"m0"},
					},
				},
			},
			expectedError: errors.New("could not instantiate middleware m0: recursion detected in middleware:m0->middleware:m0"),
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if len(test.contextProvider) > 0 {
				ctx = provider.AddInContext(ctx, "foobar@"+test.contextProvider)
			}

			rtConf := runtime.NewConfig(dynamic.Configuration{
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: test.configuration,
				},
			})
			builder := NewBuilder(rtConf.Middlewares, nil, &fakePluginsBuilder{})

			result := builder.BuildMiddlewareChain(ctx, test.buildChain)

			handlers, err := result.Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
				recorder := httptest.NewRecorder()
				request, _ := http.NewRequest(http.MethodGet, "http://foo/", nil)
				handlers.ServeHTTP(recorder, request)

				for key, value := range test.expected {
					assert.Equal(t, value, request.Header.Get(key))
				}
			}
		})
	}
}

func TestBuilder_buildConstructor(t *testing.T) {
	testConfig := map[string]*dynamic.Middleware{
		"ap-foo": {
			AddPrefix: &dynamic.AddPrefix{Prefix: "foo/"},
		},
		"multi": {
			AddPrefix:   &dynamic.AddPrefix{Prefix: "foo/"},
			StripPrefix: &dynamic.StripPrefix{Prefixes: []string{"/foo"}},
		},
	}

	rtConf := runtime.NewConfig(dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Middlewares: testConfig,
		},
	})

	t.Run("the option goes to the plugin implementing it", func(t *testing.T) {
		pluginsBuilder := &fakePluginsBuilder{}
		builder := NewBuilder(rtConf.Middlewares, nil, pluginsBuilder)

		constructor, err := builder.buildConstructor(t.Context(), "ap-foo")
		require.NoError(t, err)

		_, err = constructor(http.NotFoundHandler())
		require.NoError(t, err)

		assert.Equal(t, "addPrefix", pluginsBuilder.typedOption)
		assert.Equal(t, &dynamic.AddPrefix{Prefix: "foo/"}, pluginsBuilder.typedConfig)
	})

	t.Run("a middleware has one type", func(t *testing.T) {
		builder := NewBuilder(rtConf.Middlewares, nil, &fakePluginsBuilder{})

		_, err := builder.buildConstructor(t.Context(), "multi")
		require.Error(t, err)
	})

	t.Run("the plugins are disabled", func(t *testing.T) {
		builder := NewBuilder(rtConf.Middlewares, nil, nil)

		_, err := builder.buildConstructor(t.Context(), "ap-foo")
		require.Error(t, err)
	})
}

type fakePluginsBuilder struct {
	config         map[string]any
	buildErr       error
	constructorErr error

	typedOption string
	typedConfig any
}

// BuildTyped builds the typed middlewares used by the tests: headers sets its custom request headers, the others do nothing.
func (b *fakePluginsBuilder) BuildTyped(option string, config any, _ string) (plugins.Constructor, error) {
	b.typedOption = option
	b.typedConfig = config

	return func(_ context.Context, next http.Handler) (http.Handler, error) {
		headers, ok := config.(*dynamic.Headers)
		if !ok {
			return next, nil
		}

		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			for name, value := range headers.CustomRequestHeaders {
				req.Header.Set(name, value)
			}
			next.ServeHTTP(rw, req)
		}), nil
	}, nil
}

func (b *fakePluginsBuilder) Build(_ string, config map[string]any, _ string) (plugins.Constructor, error) {
	b.config = config
	if b.buildErr != nil {
		return nil, b.buildErr
	}

	return func(_ context.Context, next http.Handler) (http.Handler, error) {
		if b.constructorErr != nil {
			return nil, b.constructorErr
		}

		return next, nil
	}, nil
}

func TestBuilder_BuildMiddlewareChain_pluginURNRefs(t *testing.T) {
	rtConf := runtime.NewConfig(dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Middlewares: map[string]*dynamic.Middleware{
				"plugin@file": {
					Plugin:  map[string]dynamic.PluginConf{"my-plugin": {"token": "urn:k8s:secret:creds:token"}},
					URNRefs: map[string]string{"/plugin/my-plugin/token": "secret"},
				},
			},
		},
	})

	pluginsBuilder := &fakePluginsBuilder{}
	builder := NewBuilder(rtConf.Middlewares, nil, pluginsBuilder)

	_, err := builder.BuildMiddlewareChain(t.Context(), []string{"plugin@file"}).Then(http.NotFoundHandler())
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"token": "secret"}, pluginsBuilder.config)
	assert.Equal(t, dynamic.PluginConf{"token": "urn:k8s:secret:creds:token"}, rtConf.Middlewares["plugin@file"].Plugin["my-plugin"])
}

func TestBuilder_BuildMiddlewareChain_pluginErrorURNRefs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc           string
		pluginsBuilder *fakePluginsBuilder
		expectedErr    string
	}{
		{
			desc:           "build error",
			pluginsBuilder: &fakePluginsBuilder{buildErr: errors.New(`strconv.ParseInt: parsing "s3cr3t": invalid syntax`)},
			expectedErr:    `plugin: strconv.ParseInt: parsing "urn:k8s:secret:creds:token": invalid syntax`,
		},
		{
			desc:           "constructor error",
			pluginsBuilder: &fakePluginsBuilder{constructorErr: errors.New(`strconv.ParseInt: parsing "s3cr3t": invalid syntax`)},
			expectedErr:    `strconv.ParseInt: parsing "urn:k8s:secret:creds:token": invalid syntax`,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			rtConf := runtime.NewConfig(dynamic.Configuration{
				HTTP: &dynamic.HTTPConfiguration{
					Middlewares: map[string]*dynamic.Middleware{
						"plugin@file": {
							Plugin:  map[string]dynamic.PluginConf{"my-plugin": {"token": "urn:k8s:secret:creds:token"}},
							URNRefs: map[string]string{"/plugin/my-plugin/token": "s3cr3t"},
						},
					},
				},
			})

			builder := NewBuilder(rtConf.Middlewares, nil, test.pluginsBuilder)

			_, err := builder.BuildMiddlewareChain(t.Context(), []string{"plugin@file"}).Then(http.NotFoundHandler())
			require.EqualError(t, err, test.expectedErr)

			assert.Equal(t, []string{test.expectedErr}, rtConf.Middlewares["plugin@file"].Err)
		})
	}
}

func TestBuilder_BuildMiddlewareChain_basicAuthURNRefs(t *testing.T) {
	rtConf := runtime.NewConfig(dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Middlewares: map[string]*dynamic.Middleware{
				"auth@file": {
					BasicAuth: &dynamic.BasicAuth{Users: dynamic.Users{"urn:k8s:secret:auth:users"}},
					URNRefs:   map[string]string{"/basicAuth/users/0": "test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/"},
				},
			},
		},
	})

	pluginsBuilder := &fakePluginsBuilder{}
	builder := NewBuilder(rtConf.Middlewares, nil, pluginsBuilder)

	_, err := builder.BuildMiddlewareChain(t.Context(), []string{"auth@file"}).Then(http.NotFoundHandler())
	require.NoError(t, err)

	// The plugin gets the resolved users, the runtime configuration keeps the reference.
	assert.Equal(t, &dynamic.BasicAuth{Users: dynamic.Users{"test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/"}}, pluginsBuilder.typedConfig)
	assert.Equal(t, dynamic.Users{"urn:k8s:secret:auth:users"}, rtConf.Middlewares["auth@file"].BasicAuth.Users)
}

func TestBuilder_BuildMiddlewareChain_invalidURNRefs(t *testing.T) {
	rtConf := runtime.NewConfig(dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Middlewares: map[string]*dynamic.Middleware{
				"auth@file": {
					BasicAuth: &dynamic.BasicAuth{Users: dynamic.Users{"urn:k8s:secret:auth:users"}},
					URNRefs:   map[string]string{"/basicAuth/users/1": "test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/"},
				},
			},
		},
	})

	builder := NewBuilder(rtConf.Middlewares, nil, nil)

	_, err := builder.BuildMiddlewareChain(t.Context(), []string{"auth@file"}).Then(http.NotFoundHandler())
	require.Error(t, err)

	assert.Equal(t, runtime.StatusDisabled, rtConf.Middlewares["auth@file"].Status)
}
