package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/encodedcharacters"
)

// NewEncodedCharacters creates the encodedCharacters middleware: Traefik gives it the encodedCharacters option of the configuration of the middleware.
func NewEncodedCharacters(ctx context.Context, next http.Handler, config *dynamic.EncodedCharacters, name string) (http.Handler, error) {
	return encodedcharacters.NewEncodedCharacters(ctx, next, *config, name), nil
}
