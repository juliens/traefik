package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/middlewares/contenttype"
)

// NewContentType creates the contentType middleware: Traefik gives it the contentType option of the configuration of the middleware.
func NewContentType(ctx context.Context, next http.Handler, config *dynamic.ContentType, name string) (http.Handler, error) {
	return contenttype.New(ctx, next, *config, name)
}
