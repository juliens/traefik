// Package redirectregex is the plugin of the redirectRegex middleware of Traefik.
package redirectregex

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/redirect"
)

// CreateConfig creates the configuration of the redirectRegex middleware.
func CreateConfig() *dynamic.RedirectRegex {
	return &dynamic.RedirectRegex{}
}

// New creates the redirectRegex middleware: Traefik gives it the redirectRegex option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.RedirectRegex, name string) (http.Handler, error) {
	return redirect.NewRedirectRegex(ctx, next, *config, name)
}
