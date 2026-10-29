package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPodUID(t *testing.T) {
	testCases := []struct {
		desc        string
		status      int
		body        string
		expectedUID string
		expectedErr error
	}{
		{
			desc:        "pod found",
			status:      http.StatusOK,
			body:        `{"metadata":{"name":"traefik-abc","uid":"1234-5678"}}`,
			expectedUID: "1234-5678",
		},
		{
			desc:        "pod forbidden",
			status:      http.StatusForbidden,
			expectedErr: errPodForbiddenOrNotFound,
		},
		{
			desc:        "pod not found",
			status:      http.StatusNotFound,
			expectedErr: errPodForbiddenOrNotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				assert.Equal(t, "/api/v1/namespaces/traefik/pods/traefik-abc", req.URL.Path)
				assert.Equal(t, "Bearer token", req.Header.Get("Authorization"))

				rw.WriteHeader(test.status)
				_, _ = rw.Write([]byte(test.body))
			}))
			t.Cleanup(server.Close)

			uid, err := getPodUID(t.Context(), server.Client(), server.URL, "token", "traefik", "traefik-abc")
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedUID, uid)
		})
	}
}
