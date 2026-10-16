package gateway

import (
	"cmp"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/middlewares/requestdecorator"
	httpmuxer "github.com/traefik/traefik/v3/pkg/muxer/http"
	kubefake "k8s.io/client-go/kubernetes/fake"
	gatev1 "sigs.k8s.io/gateway-api/apis/v1"
)

func Test_buildHostRule(t *testing.T) {
	testCases := []struct {
		desc         string
		hostnames    []gatev1.Hostname
		expectedRule string
		expectErr    bool
	}{
		{
			desc:         "Empty (should not happen)",
			expectedRule: "",
		},
		{
			desc: "One Host",
			hostnames: []gatev1.Hostname{
				"Foo",
			},
			expectedRule: `Host("Foo")`,
		},
		{
			desc: "Multiple Hosts",
			hostnames: []gatev1.Hostname{
				"Foo",
				"Bar",
				"Bir",
			},
			expectedRule: `(Host("Foo") || Host("Bar") || Host("Bir"))`,
		},
		{
			desc: "Several Host and wildcard",
			hostnames: []gatev1.Hostname{
				"*.bar.foo",
				"bar.foo",
				"foo.foo",
			},
			expectedRule: `(Host("**.bar.foo") || Host("bar.foo") || Host("foo.foo"))`,
		},
		{
			desc: "Host with wildcard",
			hostnames: []gatev1.Hostname{
				"*.bar.foo",
			},
			expectedRule: `Host("**.bar.foo")`,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			rule := buildHostRule(test.hostnames)
			assert.Equal(t, test.expectedRule, rule)
		})
	}
}

func Test_groupHostnamesBySpecificity(t *testing.T) {
	testCases := []struct {
		desc      string
		hostnames []gatev1.Hostname
		expected  [][]gatev1.Hostname
	}{
		{
			desc:     "No hostname",
			expected: [][]gatev1.Hostname{nil},
		},
		{
			desc:      "Explicit hostnames of different lengths",
			hostnames: []gatev1.Hostname{"foo.com", "a-longer-one.com"},
			expected:  [][]gatev1.Hostname{{"foo.com", "a-longer-one.com"}},
		},
		{
			desc:      "Explicit and wildcard hostnames",
			hostnames: []gatev1.Hostname{"*.bar.foo", "bar.foo", "*.baz.foo", "*.a.bar.foo", "foo.foo"},
			expected: [][]gatev1.Hostname{
				{"bar.foo", "foo.foo"},
				{"*.bar.foo", "*.baz.foo"},
				{"*.a.bar.foo"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, groupHostnamesBySpecificity(test.hostnames))
		})
	}
}

func Test_buildMatchRule(t *testing.T) {
	testCases := []struct {
		desc          string
		match         gatev1.HTTPRouteMatch
		hostnames     []gatev1.Hostname
		expectedRule  string
		expectedError bool
	}{
		{
			desc:         "Empty rule and matches",
			expectedRule: `PathPrefix("/")`,
		},
		{
			desc:         "One Host rule without match",
			hostnames:    []gatev1.Hostname{"foo.com"},
			expectedRule: `Host("foo.com") && PathPrefix("/")`,
		},
		{
			desc: "One HTTPRouteMatch with nil HTTPHeaderMatch",
			match: gatev1.HTTPRouteMatch{
				Path: new(gatev1.HTTPPathMatch{
					Type:  new(gatev1.PathMatchPathPrefix),
					Value: new("/"),
				}),
				Headers: nil,
			},
			expectedRule: `PathPrefix("/")`,
		},
		{
			desc: "One HTTPRouteMatch with nil HTTPHeaderMatch Type",
			match: gatev1.HTTPRouteMatch{
				Path: new(gatev1.HTTPPathMatch{
					Type:  new(gatev1.PathMatchPathPrefix),
					Value: new("/"),
				}),
				Headers: []gatev1.HTTPHeaderMatch{
					{Name: "foo", Value: "bar"},
				},
			},
			expectedRule: `PathPrefix("/") && Header("foo","bar")`,
		},
		{
			desc:         "One HTTPRouteMatch with nil HTTPPathMatch",
			match:        gatev1.HTTPRouteMatch{Path: nil},
			expectedRule: `PathPrefix("/")`,
		},
		{
			desc: "One HTTPRouteMatch with nil HTTPPathMatch Type",
			match: gatev1.HTTPRouteMatch{
				Path: &gatev1.HTTPPathMatch{
					Type:  nil,
					Value: new("/foo/"),
				},
			},
			expectedRule: `(Path("/foo") || PathPrefix("/foo/"))`,
		},
		{
			desc: "One HTTPRouteMatch with nil HTTPPathMatch Values",
			match: gatev1.HTTPRouteMatch{
				Path: &gatev1.HTTPPathMatch{
					Type:  new(gatev1.PathMatchExact),
					Value: nil,
				},
			},
			expectedRule: `Path("/")`,
		},
		{
			desc: "One Path",
			match: gatev1.HTTPRouteMatch{
				Path: &gatev1.HTTPPathMatch{
					Type:  new(gatev1.PathMatchExact),
					Value: new("/foo/"),
				},
			},
			expectedRule: `Path("/foo/")`,
		},
		{
			desc: "Path && Header",
			match: gatev1.HTTPRouteMatch{
				Path: &gatev1.HTTPPathMatch{
					Type:  new(gatev1.PathMatchExact),
					Value: new("/foo/"),
				},
				Headers: []gatev1.HTTPHeaderMatch{
					{
						Type:  new(gatev1.HeaderMatchExact),
						Name:  "my-header",
						Value: "foo",
					},
				},
			},
			expectedRule: `Path("/foo/") && Header("my-header","foo")`,
		},
		{
			desc:      "Host && Path && Header",
			hostnames: []gatev1.Hostname{"foo.com"},
			match: gatev1.HTTPRouteMatch{
				Path: &gatev1.HTTPPathMatch{
					Type:  new(gatev1.PathMatchExact),
					Value: new("/foo/"),
				},
				Headers: []gatev1.HTTPHeaderMatch{
					{
						Type:  new(gatev1.HeaderMatchExact),
						Name:  "my-header",
						Value: "foo",
					},
				},
			},
			expectedRule: `Host("foo.com") && Path("/foo/") && Header("my-header","foo")`,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			rule := buildMatchRule(test.hostnames, test.match)
			assert.Equal(t, test.expectedRule, rule)
		})
	}
}

func Test_createCORS(t *testing.T) {
	testCases := []struct {
		desc     string
		filter   *gatev1.HTTPCORSFilter
		expected *dynamic.Middleware
	}{
		{
			desc:   "Empty filter",
			filter: &gatev1.HTTPCORSFilter{},
			expected: &dynamic.Middleware{
				Headers: &dynamic.Headers{
					AccessControlMaxAge: new(int64(0)),
				},
			},
		},
		{
			desc: "Plain origins",
			filter: &gatev1.HTTPCORSFilter{
				AllowOrigins: []gatev1.CORSOrigin{"https://foo.example.com", "https://bar.example.com"},
			},
			expected: &dynamic.Middleware{
				Headers: &dynamic.Headers{
					AccessControlAllowOriginList: []string{"https://foo.example.com", "https://bar.example.com"},
					AccessControlMaxAge:          new(int64(0)),
				},
			},
		},
		{
			desc: "Wildcard origin becomes catch-all regex",
			filter: &gatev1.HTTPCORSFilter{
				AllowOrigins: []gatev1.CORSOrigin{"*"},
			},
			expected: &dynamic.Middleware{
				Headers: &dynamic.Headers{
					AccessControlAllowOriginListRegex: []string{`.*`},
					AccessControlMaxAge:               new(int64(0)),
				},
			},
		},
		{
			desc: "Origin with bare wildcard host",
			filter: &gatev1.HTTPCORSFilter{
				AllowOrigins: []gatev1.CORSOrigin{"https://*"},
			},
			expected: &dynamic.Middleware{
				Headers: &dynamic.Headers{
					AccessControlAllowOriginListRegex: []string{`^https://.*$`},
					AccessControlMaxAge:               new(int64(0)),
				},
			},
		},
		{
			desc: "Origin with wildcard subdomain becomes regex",
			filter: &gatev1.HTTPCORSFilter{
				AllowOrigins: []gatev1.CORSOrigin{"https://*.example.com"},
			},
			expected: &dynamic.Middleware{
				Headers: &dynamic.Headers{
					AccessControlAllowOriginListRegex: []string{`^https://.*\.example\.com$`},
					AccessControlMaxAge:               new(int64(0)),
				},
			},
		},
		{
			desc: "Mixed plain and wildcard origins",
			filter: &gatev1.HTTPCORSFilter{
				AllowOrigins: []gatev1.CORSOrigin{"https://foo.example.com", "https://*.example.com"},
			},
			expected: &dynamic.Middleware{
				Headers: &dynamic.Headers{
					AccessControlAllowOriginList:      []string{"https://foo.example.com"},
					AccessControlAllowOriginListRegex: []string{`^https://.*\.example\.com$`},
					AccessControlMaxAge:               new(int64(0)),
				},
			},
		},
		{
			desc: "All fields set",
			filter: &gatev1.HTTPCORSFilter{
				AllowOrigins:     []gatev1.CORSOrigin{"https://foo.example.com"},
				AllowCredentials: new(true),
				AllowMethods:     []gatev1.HTTPMethodWithWildcard{"GET", "POST"},
				AllowHeaders:     []gatev1.HTTPHeaderName{"X-Foo", "X-Bar"},
				ExposeHeaders:    []gatev1.HTTPHeaderName{"X-Baz"},
				MaxAge:           600,
			},
			expected: &dynamic.Middleware{
				Headers: &dynamic.Headers{
					AccessControlAllowCredentials: true,
					AccessControlAllowOriginList:  []string{"https://foo.example.com"},
					AccessControlAllowMethods:     []string{"GET", "POST"},
					AccessControlAllowHeaders:     []string{"X-Foo", "X-Bar"},
					AccessControlExposeHeaders:    []string{"X-Baz"},
					AccessControlMaxAge:           new(int64(600)),
				},
			},
		},
		{
			desc: "AllowCredentials explicitly false",
			filter: &gatev1.HTTPCORSFilter{
				AllowCredentials: new(false),
			},
			expected: &dynamic.Middleware{
				Headers: &dynamic.Headers{
					AccessControlAllowCredentials: false,
					AccessControlMaxAge:           new(int64(0)),
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, createCORS(test.filter))
		})
	}
}

// TestHTTPRoutePrecedence routes requests through the HTTP muxer, as the routers of the HTTPRoutes attached to a
// listener are, to check the matching precedence defined by the HTTPRoute specification.
func TestHTTPRoutePrecedence(t *testing.T) {
	k8sObjects, gwObjects := readResources(t, []string{"services.yml", "httproute/precedence.yml"})

	client := newClientImpl(kubefake.NewClientset(k8sObjects...), newGatewaySimpleClientSet(t, gwObjects...))

	eventCh, err := client.WatchAll(nil, make(chan struct{}))
	require.NoError(t, err)
	<-eventCh

	p := Provider{
		EntryPoints: map[string]Entrypoint{"web": {Address: ":80"}},
		client:      client,
	}

	conf, _, err := p.loadConfigurationFromGateways(t.Context())
	require.NoError(t, err)

	parser, err := httpmuxer.NewSyntaxParser()
	require.NoError(t, err)
	muxer := httpmuxer.NewMuxer(parser, nil)

	for name, rt := range conf.HTTP.Routers {
		// The routers of the HTTPRoutes are the children of the listener router.
		if len(rt.ParentRefs) == 0 {
			continue
		}

		priority := rt.Priority
		if priority == 0 {
			priority, err = parser.RulePriority(rt.RuleSyntax, rt.Rule, rt.PriorityMode)
			require.NoError(t, err)
		}

		err = muxer.AddRouteWithOrder(rt.Rule, rt.RuleSyntax, priority, rt.Order, "kubernetesgateway", http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Set("X-Router", name)
		}))
		require.NoError(t, err)
	}

	testCases := []struct {
		desc          string
		host          string
		path          string
		method        string
		headers       []string
		expectedRoute string
		expectedRule  int
	}{
		{
			desc:          "Exact path match",
			host:          "exact.example.com",
			path:          "/x",
			expectedRoute: "exact-exact",
		},
		{
			desc:          "Prefix path match when the Exact one does not match",
			host:          "exact.example.com",
			path:          "/y",
			expectedRoute: "exact-prefix",
		},
		{
			desc:          "Prefix path match with the largest number of characters",
			host:          "prefix.example.com",
			path:          "/a/b/c",
			expectedRoute: "prefix-long",
		},
		{
			desc:          "Shorter Prefix path match when the longest one does not match",
			host:          "prefix.example.com",
			path:          "/a/c",
			expectedRoute: "prefix-short",
		},
		{
			desc:          "Method match before header matches",
			host:          "method.example.com",
			path:          "/m",
			headers:       []string{"h1", "h2"},
			expectedRoute: "method-method",
		},
		{
			desc:          "Largest number of header matches",
			host:          "headers.example.com",
			path:          "/h",
			headers:       []string{"h1", "h2"},
			expectedRoute: "headers-two",
		},
		{
			desc:          "Header matches before query param matches",
			host:          "query.example.com",
			path:          "/q?q1=v&q2=v",
			headers:       []string{"h1"},
			expectedRoute: "query-header",
		},
		{
			desc:          "Prefix path match with the largest number of characters before header matches",
			host:          "longest.example.com",
			path:          "/abc/d",
			headers:       []string{"h1"},
			expectedRoute: "longest-long",
		},
		{
			desc:          "First matching rule of an HTTPRoute",
			host:          "rules.example.com",
			path:          "/r",
			headers:       []string{"h1", "h2"},
			expectedRoute: "rules",
		},
		{
			desc:          "Oldest HTTPRoute",
			host:          "age.example.com",
			path:          "/t",
			headers:       []string{"h1", "h2"},
			expectedRoute: "age-z-old",
		},
		{
			desc:          "Matching non-wildcard hostname before any path match",
			host:          "foo.hostname.example.com",
			path:          "/x",
			expectedRoute: "hostname-explicit",
		},
		{
			desc:          "Wildcard hostname when the non-wildcard one does not match",
			host:          "bar.hostname.example.com",
			path:          "/x",
			expectedRoute: "hostname-wildcard",
		},
		{
			desc:          "Matching hostname with the largest number of characters before any path match",
			host:          "foo.a.wildcard.example.com",
			path:          "/p",
			expectedRoute: "wildcard-long",
		},
		{
			desc:          "Method match before any number of header matches",
			host:          "many.example.com",
			path:          "/o",
			headers:       []string{"h1", "h2", "h3", "h4", "h5", "h6", "h7", "h8", "h9", "h10", "h11"},
			expectedRoute: "many-method",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest(cmp.Or(test.method, http.MethodGet), "http://"+test.host+test.path, http.NoBody)
			for _, h := range test.headers {
				req.Header.Set(h, "v")
			}
			recorder := httptest.NewRecorder()

			requestdecorator.New(nil).ServeHTTP(recorder, req, muxer.ServeHTTP)

			expectedPrefix := fmt.Sprintf("httproute-default-%s-gw-default-my-gateway-ep-web-%d-", test.expectedRoute, test.expectedRule)
			assert.Regexp(t, "^"+regexp.QuoteMeta(expectedPrefix), recorder.Header().Get("X-Router"))
		})
	}
}
