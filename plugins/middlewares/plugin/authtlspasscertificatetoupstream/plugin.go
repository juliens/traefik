// Package authtlspasscertificatetoupstream is the plugin of the authTLSPassCertificateToUpstream middleware of Traefik.
package authtlspasscertificatetoupstream

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	authtlspasscertificatetoupstreammw "github.com/traefik/traefik/v3/plugins/middlewares/ingressnginx/authtlspasscertificatetoupstream"
)

// CreateConfig creates the configuration of the authTLSPassCertificateToUpstream middleware.
func CreateConfig() *dynamic.AuthTLSPassCertificateToUpstream {
	return &dynamic.AuthTLSPassCertificateToUpstream{}
}

// New creates the authTLSPassCertificateToUpstream middleware: Traefik gives it the authTLSPassCertificateToUpstream option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.AuthTLSPassCertificateToUpstream, name string) (http.Handler, error) {
	return authtlspasscertificatetoupstreammw.NewAuthTLSPassCertificateToUpstream(ctx, next, *config, name)
}
