// Package snippet is the plugin of the snippet middleware of Traefik.
package snippet

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	snippetmw "github.com/traefik/traefik/v3/plugins/middlewares/ingressnginx/snippet"
)

// CreateConfig creates the configuration of the snippet middleware.
func CreateConfig() *dynamic.Snippet {
	return &dynamic.Snippet{}
}

// New creates the snippet middleware: Traefik gives it the snippet option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.Snippet, name string) (http.Handler, error) {
	return snippetmw.New(ctx, next, config, name)
}
