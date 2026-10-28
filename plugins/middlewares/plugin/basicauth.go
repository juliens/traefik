package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/auth"
)

// NewBasicAuth creates the basicAuth middleware: Traefik gives it the basicAuth option of the configuration of the middleware.
func NewBasicAuth(ctx context.Context, next http.Handler, config *dynamic.BasicAuth, name string) (http.Handler, error) {
	return auth.NewBasic(ctx, next, *config, name)
}
