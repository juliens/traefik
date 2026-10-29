// Package plugin is the buffering middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/buffering"
)

// NewBuffering creates the buffering middleware: Traefik gives it the buffering option of the configuration of the middleware.
func NewBuffering(ctx context.Context, next http.Handler, config *dynamic.Buffering, name string) (http.Handler, error) {
	return buffering.New(ctx, next, *config, name)
}
