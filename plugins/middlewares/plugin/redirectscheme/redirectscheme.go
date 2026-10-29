// Package plugin is the redirectScheme middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/redirect"
)

// NewRedirectScheme creates the redirectScheme middleware: Traefik gives it the redirectScheme option of the configuration of the middleware.
func NewRedirectScheme(ctx context.Context, next http.Handler, config *dynamic.RedirectScheme, name string) (http.Handler, error) {
	return redirect.NewRedirectScheme(ctx, next, *config, name)
}
