// Package requestheadermodifier is the plugin of the requestHeaderModifier middleware of Traefik.
package requestheadermodifier

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/gatewayapi/headermodifier"
)

// CreateConfig creates the configuration of the requestHeaderModifier middleware.
func CreateConfig() *dynamic.HeaderModifier {
	return &dynamic.HeaderModifier{}
}

// New creates the requestHeaderModifier middleware: Traefik gives it the requestHeaderModifier option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.HeaderModifier, name string) (http.Handler, error) {
	return headermodifier.NewRequestHeaderModifier(ctx, next, *config, name), nil
}
