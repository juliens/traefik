// Package redirectscheme is the plugin of the redirectScheme middleware of Traefik.
package redirectscheme

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/redirect"
)

// CreateConfig creates the configuration of the redirectScheme middleware.
func CreateConfig() *dynamic.RedirectScheme {
	return &dynamic.RedirectScheme{}
}

// New creates the redirectScheme middleware: Traefik gives it the redirectScheme option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.RedirectScheme, name string) (http.Handler, error) {
	return redirect.NewRedirectScheme(ctx, next, *config, name)
}
