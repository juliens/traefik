package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/stripprefix"
)

// NewStripPrefix creates the stripPrefix middleware: Traefik gives it the stripPrefix option of the configuration of the middleware.
func NewStripPrefix(ctx context.Context, next http.Handler, config *dynamic.StripPrefix, name string) (http.Handler, error) {
	return stripprefix.New(ctx, next, *config, name)
}
