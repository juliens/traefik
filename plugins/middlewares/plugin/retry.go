package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/middlewares/retry"
)

// NewRetry creates the retry middleware: Traefik gives it the retry option of the configuration of the middleware.
func NewRetry(ctx context.Context, next http.Handler, config *dynamic.Retry, name string) (http.Handler, error) {
	return retry.New(ctx, next, *config, retry.Listeners{}, name)
}
