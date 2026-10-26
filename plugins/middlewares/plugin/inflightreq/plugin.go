// Package inflightreq is the plugin of the inFlightReq middleware of Traefik.
package inflightreq

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	inflightreqmw "github.com/traefik/traefik/v3/plugins/middlewares/inflightreq"
)

// CreateConfig creates the configuration of the inFlightReq middleware.
func CreateConfig() *dynamic.InFlightReq {
	return &dynamic.InFlightReq{}
}

// New creates the inFlightReq middleware: Traefik gives it the inFlightReq option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.InFlightReq, name string) (http.Handler, error) {
	return inflightreqmw.New(ctx, next, *config, name)
}
