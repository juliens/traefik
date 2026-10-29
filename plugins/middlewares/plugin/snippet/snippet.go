// Package plugin is the snippet middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/ingressnginx/snippet"
)

// NewSnippet creates the snippet middleware: Traefik gives it the snippet option of the configuration of the middleware.
func NewSnippet(ctx context.Context, next http.Handler, config *dynamic.Snippet, name string) (http.Handler, error) {
	return snippet.New(ctx, next, config, name)
}
