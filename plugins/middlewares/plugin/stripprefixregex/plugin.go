// Package stripprefixregex is the plugin of the stripPrefixRegex middleware of Traefik.
package stripprefixregex

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	stripprefixregexmw "github.com/traefik/traefik/v3/plugins/middlewares/stripprefixregex"
)

// CreateConfig creates the configuration of the stripPrefixRegex middleware.
func CreateConfig() *dynamic.StripPrefixRegex {
	return &dynamic.StripPrefixRegex{}
}

// New creates the stripPrefixRegex middleware: Traefik gives it the stripPrefixRegex option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.StripPrefixRegex, name string) (http.Handler, error) {
	return stripprefixregexmw.New(ctx, next, *config, name)
}
