// Package plugin is the upstreamVHost middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/ingressnginx/upstreamvhost"
)

// NewUpstreamVHost creates the upstreamVHost middleware: Traefik gives it the upstreamVHost option of the configuration of the middleware.
func NewUpstreamVHost(ctx context.Context, next http.Handler, config *dynamic.UpstreamVHost, name string) (http.Handler, error) {
	return upstreamvhost.New(ctx, next, *config, name)
}
