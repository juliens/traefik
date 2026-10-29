// Package plugin is the requestRedirect middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	gatewayredirect "github.com/traefik/traefik/v3/plugins/middlewares/gatewayapi/redirect"
)

// NewRequestRedirect creates the requestRedirect middleware: Traefik gives it the requestRedirect option of the configuration of the middleware.
func NewRequestRedirect(ctx context.Context, next http.Handler, config *dynamic.RequestRedirect, name string) (http.Handler, error) {
	return gatewayredirect.NewRequestRedirect(ctx, next, *config, name)
}
