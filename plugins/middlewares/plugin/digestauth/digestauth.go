// Package plugin is the digestAuth middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/auth"
)

// NewDigestAuth creates the digestAuth middleware: Traefik gives it the digestAuth option of the configuration of the middleware.
func NewDigestAuth(ctx context.Context, next http.Handler, config *dynamic.DigestAuth, name string) (http.Handler, error) {
	return auth.NewDigest(ctx, next, *config, name)
}
