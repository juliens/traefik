// Package responseheadermodifier is the plugin of the responseHeaderModifier middleware of Traefik.
package responseheadermodifier

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/gatewayapi/headermodifier"
)

// CreateConfig creates the configuration of the responseHeaderModifier middleware.
func CreateConfig() *dynamic.HeaderModifier {
	return &dynamic.HeaderModifier{}
}

// New creates the responseHeaderModifier middleware: Traefik gives it the responseHeaderModifier option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.HeaderModifier, name string) (http.Handler, error) {
	return headermodifier.NewResponseHeaderModifier(ctx, next, *config, name), nil
}
