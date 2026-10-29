package ingressnginx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/plugins/host"
)

func TestNonTLSEntryPoints(t *testing.T) {
	testCases := []struct {
		desc        string
		entryPoints map[string]host.EntryPoint
		expected    []string
	}{
		{
			desc: "no asDefault, all non-TLS non-internal entrypoints included",
			entryPoints: map[string]host.EntryPoint{
				"web":       {Address: ":80"},
				"admin":     {Address: ":8081"},
				"traefik":   {Address: ":8080"},
				"websecure": {Address: ":443", HasHTTPTLSConf: true},
			},
			expected: []string{"admin", "web"},
		},
		{
			desc: "asDefault set, only marked entrypoint included",
			entryPoints: map[string]host.EntryPoint{
				"web":       {Address: ":80", AsDefault: true},
				"admin":     {Address: ":8081"},
				"traefik":   {Address: ":8080"},
				"websecure": {Address: ":443", HasHTTPTLSConf: true},
			},
			expected: []string{"web"},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.ElementsMatch(t, test.expected, nonTLSEntryPoints(test.entryPoints))
		})
	}
}

func TestNew_DisableNonTLSRouters(t *testing.T) {
	host.SetStatic(map[string]host.EntryPoint{"web": {Address: ":80"}}, "")

	config := CreateConfig()
	config.DisableNonTLSRouters = true

	p, err := New(t.Context(), config, "kubernetesingressnginx")
	require.NoError(t, err)

	assert.Empty(t, p.NonTLSEntryPoints)
}
