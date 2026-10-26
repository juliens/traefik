// Package approot is the plugin of the appRoot middleware of Traefik.
package approot

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	approotmw "github.com/traefik/traefik/v3/plugins/middlewares/ingressnginx/approot"
)

// CreateConfig creates the configuration of the appRoot middleware.
func CreateConfig() *dynamic.AppRoot {
	return &dynamic.AppRoot{}
}

// New creates the appRoot middleware: Traefik gives it the appRoot option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.AppRoot, name string) (http.Handler, error) {
	return approotmw.New(ctx, next, *config, name)
}
