// Package errors is the plugin of the errors middleware of Traefik.
package errors

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/plugins/host"
	"github.com/traefik/traefik/v3/plugins/middlewares/customerrors"
)

// CreateConfig creates the configuration of the errors middleware.
func CreateConfig() *dynamic.ErrorPage {
	return &dynamic.ErrorPage{}
}

// New creates the errors middleware: Traefik gives it the errors option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.ErrorPage, name string) (http.Handler, error) {
	return customerrors.New(ctx, next, *config, services{}, name)
}

// services gives the services of the configuration to the middleware, from the context given to New.
type services struct{}

func (services) BuildHTTP(ctx context.Context, serviceName string) (http.Handler, error) {
	return host.BuildService(ctx, serviceName)
}
