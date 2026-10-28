package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/replacepath"
)

// NewReplacePath creates the replacePath middleware: Traefik gives it the replacePath option of the configuration of the middleware.
func NewReplacePath(ctx context.Context, next http.Handler, config *dynamic.ReplacePath, name string) (http.Handler, error) {
	return replacepath.New(ctx, next, *config, name)
}
