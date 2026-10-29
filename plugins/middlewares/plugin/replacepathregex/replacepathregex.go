// Package plugin is the replacePathRegex middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/replacepathregex"
)

// NewReplacePathRegex creates the replacePathRegex middleware: Traefik gives it the replacePathRegex option of the configuration of the middleware.
func NewReplacePathRegex(ctx context.Context, next http.Handler, config *dynamic.ReplacePathRegex, name string) (http.Handler, error) {
	return replacepathregex.New(ctx, next, *config, name)
}
