// Package contenttype is the plugin of the contentType middleware of Traefik.
package contenttype

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	contenttypemw "github.com/traefik/traefik/v3/pkg/middlewares/contenttype"
)

// CreateConfig creates the configuration of the contentType middleware.
func CreateConfig() *dynamic.ContentType {
	return &dynamic.ContentType{}
}

// New creates the contentType middleware: Traefik gives it the contentType option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.ContentType, name string) (http.Handler, error) {
	return contenttypemw.New(ctx, next, *config, name)
}
