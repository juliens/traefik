// Package tap is the plugin of the tap middleware of Traefik.
package tap

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/plugins/host"
	tapmw "github.com/traefik/traefik/v3/plugins/middlewares/tap"
)

// CreateConfig creates the configuration of the tap middleware.
func CreateConfig() *dynamic.Tap {
	return &dynamic.Tap{}
}

// New creates the tap middleware: Traefik gives it the tap option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.Tap, name string) (http.Handler, error) {
	return tapmw.New(ctx, next, *config, services{}, name)
}

// services gives the services of the configuration to the middleware, from the context given to New.
type services struct{}

func (services) BuildHTTP(ctx context.Context, serviceName string) (http.Handler, error) {
	return host.BuildService(ctx, serviceName)
}
