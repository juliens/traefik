package engine

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

func TestDetectMode(t *testing.T) {
	t.Parallel()

	// An empty module: magic number and version only.
	emptyModule := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

	httpWasm, err := os.ReadFile("../../fixtures/withoutsocket/plugin.wasm")
	require.NoError(t, err)

	testCases := []struct {
		desc     string
		code     []byte
		expected Mode
		err      string
	}{
		{
			desc:     "http-wasm guest",
			code:     httpWasm,
			expected: ModeHTTPWasm,
		},
		{
			desc: "no known export",
			code: emptyModule,
			err:  "unsupported guest: expected exports handle_request/handle_response (http-wasm) or _initialize/traefik_serve (reactor), found []",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			rt := wazero.NewRuntime(t.Context())
			t.Cleanup(func() { _ = rt.Close(t.Context()) })

			compiled, err := rt.CompileModule(t.Context(), test.code)
			require.NoError(t, err)

			mode, err := DetectMode(compiled)
			if test.err != "" {
				require.EqualError(t, err, test.err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expected, mode)
		})
	}
}

func TestConfig_InstanceCount(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3, Config{Instances: 3}.InstanceCount())
	assert.LessOrEqual(t, Config{}.InstanceCount(), defaultMaxInstances)
	assert.GreaterOrEqual(t, Config{}.InstanceCount(), 1)
}
