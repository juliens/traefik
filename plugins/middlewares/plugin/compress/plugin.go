// Package compress is the plugin of the compress middleware of Traefik.
package compress

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	compressmw "github.com/traefik/traefik/v3/plugins/middlewares/compress"
)

// CreateConfig creates the configuration of the compress middleware.
func CreateConfig() *dynamic.Compress {
	return &dynamic.Compress{}
}

// New creates the compress middleware: Traefik gives it the compress option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.Compress, name string) (http.Handler, error) {
	return compressmw.New(ctx, next, *config, name)
}
