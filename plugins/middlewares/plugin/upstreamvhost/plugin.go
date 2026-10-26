// Package upstreamvhost is the plugin of the upstreamVHost middleware of Traefik.
package upstreamvhost

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	upstreamvhostmw "github.com/traefik/traefik/v3/plugins/middlewares/ingressnginx/upstreamvhost"
)

// CreateConfig creates the configuration of the upstreamVHost middleware.
func CreateConfig() *dynamic.UpstreamVHost {
	return &dynamic.UpstreamVHost{}
}

// New creates the upstreamVHost middleware: Traefik gives it the upstreamVHost option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.UpstreamVHost, name string) (http.Handler, error) {
	return upstreamvhostmw.New(ctx, next, *config, name)
}
