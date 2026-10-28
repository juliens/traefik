package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/ratelimiter"
)

// NewRateLimit creates the rateLimit middleware: Traefik gives it the rateLimit option of the configuration of the middleware.
func NewRateLimit(ctx context.Context, next http.Handler, config *dynamic.RateLimit, name string) (http.Handler, error) {
	return ratelimiter.New(ctx, next, *config, name)
}
