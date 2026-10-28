package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/tap"
)

// NewTap creates the tap middleware: Traefik gives it the tap option of the configuration of the middleware.
func NewTap(ctx context.Context, next http.Handler, config *dynamic.Tap, name string) (http.Handler, error) {
	return tap.New(ctx, next, *config, services{}, name)
}
