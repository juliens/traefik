// Package urlrewrite is the plugin of the URLRewrite middleware of Traefik.
package urlrewrite

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	urlrewritemw "github.com/traefik/traefik/v3/plugins/middlewares/gatewayapi/urlrewrite"
)

// CreateConfig creates the configuration of the URLRewrite middleware.
func CreateConfig() *dynamic.URLRewrite {
	return &dynamic.URLRewrite{}
}

// New creates the URLRewrite middleware: Traefik gives it the URLRewrite option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.URLRewrite, name string) (http.Handler, error) {
	return urlrewritemw.NewURLRewrite(ctx, next, *config, name), nil
}
