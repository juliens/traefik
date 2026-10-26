// Package ratelimit is the plugin of the rateLimit middleware of Traefik.
package ratelimit

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/ratelimiter"
)

// CreateConfig creates the configuration of the rateLimit middleware.
func CreateConfig() *dynamic.RateLimit {
	return &dynamic.RateLimit{}
}

// New creates the rateLimit middleware: Traefik gives it the rateLimit option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.RateLimit, name string) (http.Handler, error) {
	return ratelimiter.New(ctx, next, *config, name)
}
