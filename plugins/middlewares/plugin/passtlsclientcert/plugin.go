// Package passtlsclientcert is the plugin of the passTLSClientCert middleware of Traefik.
package passtlsclientcert

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	passtlsclientcertmw "github.com/traefik/traefik/v3/plugins/middlewares/passtlsclientcert"
)

// CreateConfig creates the configuration of the passTLSClientCert middleware.
func CreateConfig() *dynamic.PassTLSClientCert {
	return &dynamic.PassTLSClientCert{}
}

// New creates the passTLSClientCert middleware: Traefik gives it the passTLSClientCert option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.PassTLSClientCert, name string) (http.Handler, error) {
	return passtlsclientcertmw.New(ctx, next, *config, name)
}
