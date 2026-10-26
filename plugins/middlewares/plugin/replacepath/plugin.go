// Package replacepath is the plugin of the replacePath middleware of Traefik.
package replacepath

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	replacepathmw "github.com/traefik/traefik/v3/plugins/middlewares/replacepath"
)

// CreateConfig creates the configuration of the replacePath middleware.
func CreateConfig() *dynamic.ReplacePath {
	return &dynamic.ReplacePath{}
}

// New creates the replacePath middleware: Traefik gives it the replacePath option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.ReplacePath, name string) (http.Handler, error) {
	return replacepathmw.New(ctx, next, *config, name)
}
