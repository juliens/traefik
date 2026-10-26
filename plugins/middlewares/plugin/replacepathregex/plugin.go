// Package replacepathregex is the plugin of the replacePathRegex middleware of Traefik.
package replacepathregex

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	replacepathregexmw "github.com/traefik/traefik/v3/plugins/middlewares/replacepathregex"
)

// CreateConfig creates the configuration of the replacePathRegex middleware.
func CreateConfig() *dynamic.ReplacePathRegex {
	return &dynamic.ReplacePathRegex{}
}

// New creates the replacePathRegex middleware: Traefik gives it the replacePathRegex option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.ReplacePathRegex, name string) (http.Handler, error) {
	return replacepathregexmw.New(ctx, next, *config, name)
}
