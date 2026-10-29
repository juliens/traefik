// Package plugin is the appRoot middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/ingressnginx/approot"
)

// NewAppRoot creates the appRoot middleware: Traefik gives it the appRoot option of the configuration of the middleware.
func NewAppRoot(ctx context.Context, next http.Handler, config *dynamic.AppRoot, name string) (http.Handler, error) {
	return approot.New(ctx, next, *config, name)
}
