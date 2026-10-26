// Package basicauth is the plugin of the basicAuth middleware of Traefik.
package basicauth

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/auth"
)

// CreateConfig creates the configuration of the basicAuth middleware.
func CreateConfig() *dynamic.BasicAuth {
	return &dynamic.BasicAuth{}
}

// New creates the basicAuth middleware: Traefik gives it the basicAuth option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.BasicAuth, name string) (http.Handler, error) {
	return auth.NewBasic(ctx, next, *config, name)
}
