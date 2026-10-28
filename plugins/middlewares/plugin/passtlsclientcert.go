package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/passtlsclientcert"
)

// NewPassTLSClientCert creates the passTLSClientCert middleware: Traefik gives it the passTLSClientCert option of the configuration of the middleware.
func NewPassTLSClientCert(ctx context.Context, next http.Handler, config *dynamic.PassTLSClientCert, name string) (http.Handler, error) {
	return passtlsclientcert.New(ctx, next, *config, name)
}
