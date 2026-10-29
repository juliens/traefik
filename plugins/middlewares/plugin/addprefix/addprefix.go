// Package plugin is the addPrefix middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/addprefix"
)

// NewAddPrefix creates the addPrefix middleware: Traefik gives it the addPrefix option of the configuration of the middleware.
func NewAddPrefix(ctx context.Context, next http.Handler, config *dynamic.AddPrefix, name string) (http.Handler, error) {
	return addprefix.New(ctx, next, *config, name)
}
