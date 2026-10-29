// Package plugin is the tap middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/plugins/host"
	"github.com/traefik/traefik/v3/plugins/middlewares/tap"
)

// NewTap creates the tap middleware: Traefik gives it the tap option of the configuration of the middleware.
func NewTap(ctx context.Context, next http.Handler, config *dynamic.Tap, name string) (http.Handler, error) {
	return tap.New(ctx, next, *config, services{}, name)
}

// services gives the services of the configuration to the middleware, from the context given to its constructor.
type services struct{}

func (services) BuildHTTP(ctx context.Context, serviceName string) (http.Handler, error) {
	return host.BuildService(ctx, serviceName)
}
