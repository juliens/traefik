package acme

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/plugins/host"
)

func TestHTTPChallengeRouters(t *testing.T) {
	testCases := []struct {
		desc        string
		challenge   *HTTPChallenge
		entryPoints map[string]host.EntryPoint
		expected    map[string]*dynamic.Router
	}{
		{
			desc: "no HTTP challenge",
		},
		{
			desc:        "HTTP challenge",
			challenge:   &HTTPChallenge{EntryPoint: "web"},
			entryPoints: map[string]host.EntryPoint{"web": {Address: ":80"}},
			expected: map[string]*dynamic.Router{
				"acme-http": {
					Rule:        "PathPrefix(`/.well-known/acme-challenge/`)",
					RuleSyntax:  "default",
					EntryPoints: []string{"web"},
					Service:     "le@internal",
					Priority:    maxUserPriority,
				},
			},
		},
		{
			desc:        "HTTP challenge on an entry point allowing the ACME bypass",
			challenge:   &HTTPChallenge{EntryPoint: "web"},
			entryPoints: map[string]host.EntryPoint{"web": {Address: ":80", AllowACMEByPass: true}},
			expected: map[string]*dynamic.Router{
				"acme-http-bypass": {
					Rule:        "PathPrefix(`/.well-known/acme-challenge/`)",
					EntryPoints: []string{"web"},
					Service:     "le@internal",
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, httpChallengeRouters("le", test.challenge, test.entryPoints))
		})
	}
}

func TestSetEffectiveConfiguration(t *testing.T) {
	testCases := []struct {
		desc     string
		config   *Configuration
		expected *Configuration
	}{
		{
			desc:   "default CA server",
			config: &Configuration{DNSChallenge: &DNSChallenge{Provider: "bar"}},
			expected: &Configuration{
				CAServer:     defaultCAServer,
				DNSChallenge: &DNSChallenge{Provider: "bar"},
			},
		},
		{
			desc:   "deprecated DelayBeforeCheck",
			config: &Configuration{DNSChallenge: &DNSChallenge{Provider: "bar", DelayBeforeCheck: 123}},
			expected: &Configuration{
				CAServer: defaultCAServer,
				DNSChallenge: &DNSChallenge{
					Provider:         "bar",
					DelayBeforeCheck: 123,
					Propagation:      Propagation{DelayBeforeChecks: 123},
				},
			},
		},
		{
			desc:   "deprecated DisablePropagationCheck",
			config: &Configuration{DNSChallenge: &DNSChallenge{Provider: "bar", DisablePropagationCheck: true}},
			expected: &Configuration{
				CAServer: defaultCAServer,
				DNSChallenge: &DNSChallenge{
					Provider:                "bar",
					DisablePropagationCheck: true,
					Propagation:             Propagation{DisableChecks: true},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			setEffectiveConfiguration(test.config)

			assert.Equal(t, test.expected, test.config)
		})
	}
}

func TestChallengeHTTP_sharedBetweenResolvers(t *testing.T) {
	ctx := t.Context()

	// A resolver presents the challenge, the router of another resolver answers it.
	presenter, server := NewChallengeHTTP(), NewChallengeHTTP()
	assert.NoError(t, presenter.Present(ctx, "example.com", "token", "keyAuth"))

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/.well-known/acme-challenge/token", nil)
	server.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "keyAuth", rw.Body.String())

	assert.NoError(t, presenter.CleanUp(ctx, "example.com", "token", "keyAuth"))

	rw = httptest.NewRecorder()
	server.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusNotFound, rw.Code)
}
