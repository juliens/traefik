package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/headers"
)

// NewHeaders creates the headers middleware: Traefik gives it the headers option of the configuration of the middleware.
func NewHeaders(ctx context.Context, next http.Handler, config *dynamic.Headers, name string) (http.Handler, error) {
	return headers.New(ctx, next, *config, name)
}
