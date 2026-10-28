package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/customerrors"
)

// NewErrors creates the errors middleware: Traefik gives it the errors option of the configuration of the middleware.
func NewErrors(ctx context.Context, next http.Handler, config *dynamic.ErrorPage, name string) (http.Handler, error) {
	return customerrors.New(ctx, next, *config, services{}, name)
}
