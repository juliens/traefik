// Package retry is the plugin of the retry middleware of Traefik.
package retry

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	retrymw "github.com/traefik/traefik/v3/pkg/middlewares/retry"
)

// CreateConfig creates the configuration of the retry middleware.
func CreateConfig() *dynamic.Retry {
	return &dynamic.Retry{}
}

// New creates the retry middleware: Traefik gives it the retry option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.Retry, name string) (http.Handler, error) {
	return retrymw.New(ctx, next, *config, retrymw.Listeners{}, name)
}
