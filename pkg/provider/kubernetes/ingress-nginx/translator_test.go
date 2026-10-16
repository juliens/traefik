package ingressnginx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/middlewares/redirect"
	"github.com/traefik/traefik/v3/pkg/middlewares/requestdecorator"
	httpmuxer "github.com/traefik/traefik/v3/pkg/muxer/http"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestApplyFromToWwwRedirect(t *testing.T) {
	conf := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:     make(map[string]*dynamic.Router),
			Middlewares: make(map[string]*dynamic.Middleware),
		},
	}

	loc := &location{
		FromToWwwRedirect: &middlewareFromToWwwRedirect{
			ExtraRouterRule: `Host("www.example.com")`,
			TargetHostname:  "example.com",
		},
	}
	rt := &dynamic.Router{
		EntryPoints: []string{"web"},
		Service:     "backend",
		Middlewares: []string{"router-basic-auth"},
	}

	applyFromToWwwRedirect(loc, "router", rt, nil, conf)

	router := conf.HTTP.Routers["router-from-to-www-redirect"]
	require.NotNil(t, router)

	assert.Equal(t, unavailableServiceName, router.Service)
	assert.Equal(t, []string{"router-from-to-www-redirect"}, router.Middlewares)

	middleware := conf.HTTP.Middlewares["router-from-to-www-redirect"]
	require.NotNil(t, middleware.RedirectRegex)

	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
	})

	handler, err := redirect.NewRedirectRegex(t.Context(), next, *middleware.RedirectRegex, "test")
	require.NoError(t, err)

	testCases := []struct {
		desc         string
		host         string
		target       string
		expectedCode int
		expectedLoc  string
	}{
		{
			desc:         "no port",
			host:         "www.example.com",
			target:       "/foo",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com/foo",
		},
		{
			desc:         "numeric port",
			host:         "www.example.com:8080",
			target:       "/foo",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com:8080/foo",
		},
		{
			desc:         "non-numeric port",
			host:         "www.example.com:x",
			target:       "/foo",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com/foo",
		},
		{
			desc:         "empty port",
			host:         "www.example.com:",
			target:       "/foo",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com/foo",
		},
		{
			desc:         "multiple ports",
			host:         "www.example.com:8080:90",
			target:       "/foo",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com:8080/foo",
		},
		{
			desc:         "port followed by a delimiter",
			host:         "www.example.com:8080;x",
			target:       "/foo",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com:8080/foo",
		},
		{
			desc:         "IPv6 literal",
			host:         "[::1]:8080",
			target:       "/foo",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com:8080/foo",
		},
		{
			desc:         "empty authority",
			host:         "",
			target:       "/foo",
			expectedCode: http.StatusTeapot,
		},
		{
			desc:         "root path",
			host:         "www.example.com",
			target:       "/",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com/",
		},
		{
			desc:         "trailing slash",
			host:         "www.example.com",
			target:       "/foo/",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com/foo",
		},
		{
			desc:         "trailing slash followed by a query string",
			host:         "www.example.com",
			target:       "/foo/?bar=baz",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com/foo/?bar=baz",
		},
		{
			desc:         "double trailing slash",
			host:         "www.example.com",
			target:       "/foo//",
			expectedCode: http.StatusPermanentRedirect,
			expectedLoc:  "http://example.com/foo/",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, test.target, nil)
			req.Host = test.host

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, test.expectedCode, recorder.Code)
			assert.Equal(t, test.expectedLoc, recorder.Header().Get("Location"))
		})
	}
}

func TestMakeTrailingGroupOptional(t *testing.T) {
	testCases := []struct {
		desc     string
		path     string
		expected string
	}{
		{
			desc:     "trailing capture group",
			path:     "/original/(.*)",
			expected: "/original(?:/(.*))?",
		},
		{
			desc:     "multiple groups wraps only the last one",
			path:     "/foo/(bar)/(.*)",
			expected: "/foo/(bar)(?:/(.*))?",
		},
		{
			desc:     "path starting with a capture group is left unchanged",
			path:     "/(.*)",
			expected: "/(.*)",
		},
		{
			desc:     "path starting with a capture group followed by another group is left unchanged",
			path:     "/(something)(/.+)",
			expected: "/(something)(/.+)",
		},
		{
			desc:     "documented ingress-nginx rewrite pattern is left unchanged",
			path:     "/something(/|$)(.*)",
			expected: "/something(/|$)(.*)",
		},
		{
			desc:     "literal after the group is left unchanged",
			path:     "/foo/(.*)/bar",
			expected: "/foo/(.*)/bar",
		},
		{
			desc:     "escaped literal after the group is left unchanged",
			path:     `/foo/(.*)\.json`,
			expected: `/foo/(.*)\.json`,
		},
		{
			desc:     "anchor after the group is left unchanged",
			path:     "/foo/(.*)$",
			expected: "/foo/(.*)$",
		},
		{
			desc:     "nested groups closing at the end",
			path:     "/foo/((a|b)/.*)",
			expected: "/foo(?:/((a|b)/.*))?",
		},
		{
			desc:     "unbalanced group is left unchanged",
			path:     "/foo/((.*)",
			expected: "/foo/((.*)",
		},
		{
			desc:     "path without capture group",
			path:     "/foo",
			expected: "/foo",
		},
		{
			desc:     "path with trailing slash",
			path:     "/foo/",
			expected: "/foo/",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			assert.Equal(t, test.expected, makeTrailingGroupOptional(test.path))
		})
	}
}

// TestLocationPrecedence routes requests through the HTTP muxer, as NGINX selects a location: an explicit host before
// a wildcard one, before the rules without host, and within them an Exact path before any Prefix path.
func TestLocationPrecedence(t *testing.T) {
	k8sObjects := readResources(t, []string{
		"services.yml",
		"ingressclasses.yml",
		"ingresses/ingress-with-exact-path-precedence.yml",
	})
	client := newClient(kubefake.NewClientset(k8sObjects...))

	eventCh, err := client.WatchAll(t.Context(), "", "")
	require.NoError(t, err)
	<-eventCh

	p := Provider{
		k8sClient:         client,
		NonTLSEntryPoints: []string{"http"},
		TLSEntryPoints:    []string{"https"},
	}
	p.SetDefaults()

	conf := p.loadConfiguration(t.Context())

	parser, err := httpmuxer.NewSyntaxParser()
	require.NoError(t, err)
	muxer := httpmuxer.NewMuxer(parser, nil)

	for name, rt := range conf.HTTP.Routers {
		if rt.TLS != nil {
			continue
		}

		priority := rt.Priority
		if priority == 0 {
			priority, err = parser.RulePriority(rt.RuleSyntax, rt.Rule, rt.PriorityMode)
			require.NoError(t, err)
		}

		err = muxer.AddRouteWithOrder(rt.Rule, rt.RuleSyntax, priority, rt.Order, "kubernetesingressnginx", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Set("X-Router", name)
		}))
		require.NoError(t, err)
	}

	testCases := []struct {
		desc           string
		host           string
		path           string
		expectedRouter string
	}{
		{
			desc:           "Exact path under a Prefix / path",
			host:           "exact.localhost",
			path:           "/exact",
			expectedRouter: "default-7-exact-5-rule-0-path-0",
		},
		{
			desc:           "Exact path under a nested Prefix path",
			host:           "exact.localhost",
			path:           "/api/exact",
			expectedRouter: "default-7-api-exact-9-rule-0-path-0",
		},
		{
			desc:           "Subpath of an Exact path",
			host:           "exact.localhost",
			path:           "/exact/sub",
			expectedRouter: "default-7-prefix-6-rule-0-path-0",
		},
		{
			desc:           "Subpath of an Exact path under a nested Prefix path",
			host:           "exact.localhost",
			path:           "/api/exact/sub",
			expectedRouter: "default-7-api-prefix-10-rule-0-path-0",
		},
		{
			desc:           "Exact path without host on a declared host",
			host:           "declared.localhost",
			path:           "/hostless",
			expectedRouter: "default-7-declared-8-rule-0-path-0",
		},
		{
			desc:           "Exact path without host on an undeclared host",
			host:           "other.localhost",
			path:           "/hostless",
			expectedRouter: "default-7-hostless-8-rule-0-path-0",
		},
		{
			desc:           "Exact path with a wildcard host on a declared host",
			host:           "foo.wildcard.localhost",
			path:           "/exact",
			expectedRouter: "default-7-declared-wildcard-17-rule-0-path-0",
		},
		{
			desc:           "Exact path with a wildcard host on an undeclared host",
			host:           "bar.wildcard.localhost",
			path:           "/exact",
			expectedRouter: "default-7-wildcard-8-rule-0-path-0",
		},
		{
			desc:           "Subpath of an Exact path with a wildcard host",
			host:           "bar.wildcard.localhost",
			path:           "/exact/sub",
			expectedRouter: "default-7-wildcard-prefix-15-rule-0-path-0",
		},
		{
			desc:           "Exact path without host under a Prefix path without host",
			host:           "other.localhost",
			path:           "/a-rather-long-path-declared-without-host/exact",
			expectedRouter: "default-7-hostless-exact-14-rule-0-path-0",
		},
		{
			desc:           "Prefix path without host on an undeclared host",
			host:           "other.localhost",
			path:           "/a-rather-long-path-declared-without-host",
			expectedRouter: "default-7-hostless-prefix-15-rule-0-path-0",
		},
		{
			desc:           "Long Prefix path without host on a declared host",
			host:           "declared.localhost",
			path:           "/a-rather-long-path-declared-without-host",
			expectedRouter: "default-7-declared-8-rule-0-path-0",
		},
		{
			desc:           "Long Prefix path without host on a wildcard host",
			host:           "bar.wildcard.localhost",
			path:           "/a-rather-long-path-declared-without-host",
			expectedRouter: "default-7-wildcard-prefix-15-rule-0-path-0",
		},
		{
			desc:           "Long Prefix path with a wildcard host on a declared host",
			host:           "foo.wildcard.localhost",
			path:           "/a-rather-long-path-declared-on-the-wildcard-host",
			expectedRouter: "default-7-declared-wildcard-17-rule-0-path-0",
		},
		{
			desc:           "Long Prefix path with a wildcard host on an undeclared host",
			host:           "bar.wildcard.localhost",
			path:           "/a-rather-long-path-declared-on-the-wildcard-host",
			expectedRouter: "default-7-wildcard-prefix-15-rule-0-path-1",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://"+test.host+test.path, http.NoBody)
			recorder := httptest.NewRecorder()

			requestdecorator.New(nil).ServeHTTP(recorder, req, muxer.ServeHTTP)

			assert.Equal(t, test.expectedRouter, recorder.Header().Get("X-Router"))
		})
	}
}
