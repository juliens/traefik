// Package buffering is the plugin of the buffering middleware of Traefik.
package buffering

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	bufferingmw "github.com/traefik/traefik/v3/plugins/middlewares/buffering"
)

// CreateConfig creates the configuration of the buffering middleware.
func CreateConfig() *dynamic.Buffering {
	return &dynamic.Buffering{}
}

// New creates the buffering middleware: Traefik gives it the buffering option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.Buffering, name string) (http.Handler, error) {
	return bufferingmw.New(ctx, next, *config, name)
}
