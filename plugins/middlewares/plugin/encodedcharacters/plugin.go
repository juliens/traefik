// Package encodedcharacters is the plugin of the encodedCharacters middleware of Traefik.
package encodedcharacters

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	encodedcharactersmw "github.com/traefik/traefik/v3/plugins/middlewares/encodedcharacters"
)

// CreateConfig creates the configuration of the encodedCharacters middleware.
func CreateConfig() *dynamic.EncodedCharacters {
	return &dynamic.EncodedCharacters{}
}

// New creates the encodedCharacters middleware: Traefik gives it the encodedCharacters option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.EncodedCharacters, name string) (http.Handler, error) {
	return encodedcharactersmw.NewEncodedCharacters(ctx, next, *config, name), nil
}
