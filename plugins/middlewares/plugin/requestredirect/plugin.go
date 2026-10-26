// Package requestredirect is the plugin of the requestRedirect middleware of Traefik.
package requestredirect

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/gatewayapi/redirect"
)

// CreateConfig creates the configuration of the requestRedirect middleware.
func CreateConfig() *dynamic.RequestRedirect {
	return &dynamic.RequestRedirect{}
}

// New creates the requestRedirect middleware: Traefik gives it the requestRedirect option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.RequestRedirect, name string) (http.Handler, error) {
	return redirect.NewRequestRedirect(ctx, next, *config, name)
}
