// Package rewritetarget is the plugin of the rewriteTarget middleware of Traefik.
package rewritetarget

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	rewritetargetmw "github.com/traefik/traefik/v3/plugins/middlewares/ingressnginx/rewritetarget"
)

// CreateConfig creates the configuration of the rewriteTarget middleware.
func CreateConfig() *dynamic.RewriteTarget {
	return &dynamic.RewriteTarget{}
}

// New creates the rewriteTarget middleware: Traefik gives it the rewriteTarget option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.RewriteTarget, name string) (http.Handler, error) {
	return rewritetargetmw.New(ctx, next, *config, name)
}
