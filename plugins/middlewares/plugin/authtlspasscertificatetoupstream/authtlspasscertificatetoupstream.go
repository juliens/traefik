// Package plugin is the authTLSPassCertificateToUpstream middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/ingressnginx/authtlspasscertificatetoupstream"
)

// NewAuthTLSPassCertificateToUpstream creates the authTLSPassCertificateToUpstream middleware: Traefik gives it the authTLSPassCertificateToUpstream option of the configuration of the middleware.
func NewAuthTLSPassCertificateToUpstream(ctx context.Context, next http.Handler, config *dynamic.AuthTLSPassCertificateToUpstream, name string) (http.Handler, error) {
	return authtlspasscertificatetoupstream.NewAuthTLSPassCertificateToUpstream(ctx, next, *config, name)
}
