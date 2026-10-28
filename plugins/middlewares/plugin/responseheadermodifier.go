package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/gatewayapi/headermodifier"
)

// NewResponseHeaderModifier creates the responseHeaderModifier middleware: Traefik gives it the responseHeaderModifier option of the configuration of the middleware.
func NewResponseHeaderModifier(ctx context.Context, next http.Handler, config *dynamic.HeaderModifier, name string) (http.Handler, error) {
	return headermodifier.NewResponseHeaderModifier(ctx, next, *config, name), nil
}
