package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/gatewayapi/urlrewrite"
)

// NewURLRewrite creates the URLRewrite middleware: Traefik gives it the URLRewrite option of the configuration of the middleware.
func NewURLRewrite(ctx context.Context, next http.Handler, config *dynamic.URLRewrite, name string) (http.Handler, error) {
	return urlrewrite.NewURLRewrite(ctx, next, *config, name), nil
}
