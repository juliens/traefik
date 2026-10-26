// Package stripprefix is the plugin of the stripPrefix middleware of Traefik.
package stripprefix

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	stripprefixmw "github.com/traefik/traefik/v3/plugins/middlewares/stripprefix"
)

// CreateConfig creates the configuration of the stripPrefix middleware.
func CreateConfig() *dynamic.StripPrefix {
	return &dynamic.StripPrefix{}
}

// New creates the stripPrefix middleware: Traefik gives it the stripPrefix option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.StripPrefix, name string) (http.Handler, error) {
	return stripprefixmw.New(ctx, next, *config, name)
}
