// Package plugin is the rewriteTarget middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/ingressnginx/rewritetarget"
)

// NewRewriteTarget creates the rewriteTarget middleware: Traefik gives it the rewriteTarget option of the configuration of the middleware.
func NewRewriteTarget(ctx context.Context, next http.Handler, config *dynamic.RewriteTarget, name string) (http.Handler, error) {
	return rewritetarget.New(ctx, next, *config, name)
}
