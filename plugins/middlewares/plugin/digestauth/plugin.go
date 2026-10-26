// Package digestauth is the plugin of the digestAuth middleware of Traefik.
package digestauth

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/auth"
)

// CreateConfig creates the configuration of the digestAuth middleware.
func CreateConfig() *dynamic.DigestAuth {
	return &dynamic.DigestAuth{}
}

// New creates the digestAuth middleware: Traefik gives it the digestAuth option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.DigestAuth, name string) (http.Handler, error) {
	return auth.NewDigest(ctx, next, *config, name)
}
