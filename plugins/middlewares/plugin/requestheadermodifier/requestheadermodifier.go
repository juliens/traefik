// Package plugin is the requestHeaderModifier middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/gatewayapi/headermodifier"
)

// NewRequestHeaderModifier creates the requestHeaderModifier middleware: Traefik gives it the requestHeaderModifier option of the configuration of the middleware.
func NewRequestHeaderModifier(ctx context.Context, next http.Handler, config *dynamic.HeaderModifier, name string) (http.Handler, error) {
	return headermodifier.NewRequestHeaderModifier(ctx, next, *config, name), nil
}
