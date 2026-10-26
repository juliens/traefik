// Package addprefix is the plugin of the addPrefix middleware of Traefik.
package addprefix

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	addprefixmw "github.com/traefik/traefik/v3/plugins/middlewares/addprefix"
)

// CreateConfig creates the configuration of the addPrefix middleware.
func CreateConfig() *dynamic.AddPrefix {
	return &dynamic.AddPrefix{}
}

// New creates the addPrefix middleware: Traefik gives it the addPrefix option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.AddPrefix, name string) (http.Handler, error) {
	return addprefixmw.New(ctx, next, *config, name)
}
