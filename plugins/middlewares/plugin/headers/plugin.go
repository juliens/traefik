// Package headers is the plugin of the headers middleware of Traefik.
package headers

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	headersmw "github.com/traefik/traefik/v3/plugins/middlewares/headers"
)

// CreateConfig creates the configuration of the headers middleware.
func CreateConfig() *dynamic.Headers {
	return &dynamic.Headers{}
}

// New creates the headers middleware: Traefik gives it the headers option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.Headers, name string) (http.Handler, error) {
	return headersmw.New(ctx, next, *config, name)
}
