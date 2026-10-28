package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/circuitbreaker"
)

// NewCircuitBreaker creates the circuitBreaker middleware: Traefik gives it the circuitBreaker option of the configuration of the middleware.
func NewCircuitBreaker(ctx context.Context, next http.Handler, config *dynamic.CircuitBreaker, name string) (http.Handler, error) {
	return circuitbreaker.New(ctx, next, *config, name)
}
