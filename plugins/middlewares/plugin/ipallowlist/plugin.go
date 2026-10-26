// Package ipallowlist is the plugin of the ipAllowList middleware of Traefik.
package ipallowlist

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	ipallowlistmw "github.com/traefik/traefik/v3/plugins/middlewares/ipallowlist"
)

// CreateConfig creates the configuration of the ipAllowList middleware.
func CreateConfig() *dynamic.IPAllowList {
	return &dynamic.IPAllowList{}
}

// New creates the ipAllowList middleware: Traefik gives it the ipAllowList option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.IPAllowList, name string) (http.Handler, error) {
	return ipallowlistmw.New(ctx, next, *config, name)
}
