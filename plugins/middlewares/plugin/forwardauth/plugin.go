// Package forwardauth is the plugin of the forwardAuth middleware of Traefik.
package forwardauth

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/auth"
)

// CreateConfig creates the configuration of the forwardAuth middleware.
func CreateConfig() *dynamic.ForwardAuth {
	return &dynamic.ForwardAuth{}
}

// New creates the forwardAuth middleware: Traefik gives it the forwardAuth option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.ForwardAuth, name string) (http.Handler, error) {
	return auth.NewForward(ctx, next, *config, name)
}
