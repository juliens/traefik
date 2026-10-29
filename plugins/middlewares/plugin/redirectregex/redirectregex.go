// Package plugin is the redirectRegex middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/redirect"
)

// NewRedirectRegex creates the redirectRegex middleware: Traefik gives it the redirectRegex option of the configuration of the middleware.
func NewRedirectRegex(ctx context.Context, next http.Handler, config *dynamic.RedirectRegex, name string) (http.Handler, error) {
	return redirect.NewRedirectRegex(ctx, next, *config, name)
}
