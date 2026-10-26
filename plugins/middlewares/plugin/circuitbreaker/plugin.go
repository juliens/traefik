// Package circuitbreaker is the plugin of the circuitBreaker middleware of Traefik.
package circuitbreaker

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	circuitbreakermw "github.com/traefik/traefik/v3/plugins/middlewares/circuitbreaker"
)

// CreateConfig creates the configuration of the circuitBreaker middleware.
func CreateConfig() *dynamic.CircuitBreaker {
	return &dynamic.CircuitBreaker{}
}

// New creates the circuitBreaker middleware: Traefik gives it the circuitBreaker option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.CircuitBreaker, name string) (http.Handler, error) {
	return circuitbreakermw.New(ctx, next, *config, name)
}
