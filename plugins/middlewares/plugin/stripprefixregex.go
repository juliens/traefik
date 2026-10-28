package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/stripprefixregex"
)

// NewStripPrefixRegex creates the stripPrefixRegex middleware: Traefik gives it the stripPrefixRegex option of the configuration of the middleware.
func NewStripPrefixRegex(ctx context.Context, next http.Handler, config *dynamic.StripPrefixRegex, name string) (http.Handler, error) {
	return stripprefixregex.New(ctx, next, *config, name)
}
