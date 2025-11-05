package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("X-Test", "toto")

	tomConfig := `{"rules":[{"name":"X-Test", "Value":"X-New", "Header":"X-Test", "Type":"Rename"}]}`
	demoT, err := NewPlugin(context.Background(), "#github.com/tomMoulard/fail2ban", tomConfig, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	require.NoError(t, err)

	// demoConfig := `{"headers":{"X-Test":"TOTO"}}`
	// demoM, err := NewPlugin(context.Background(), "demo", demoConfig, demoT)

	require.NoError(t, err)

	demoT.ServeHTTP(recorder, request)
	println(request.Header)
}
