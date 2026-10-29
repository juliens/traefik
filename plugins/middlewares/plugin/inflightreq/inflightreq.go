// Package plugin is the inFlightReq middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/inflightreq"
)

// NewInFlightReq creates the inFlightReq middleware: Traefik gives it the inFlightReq option of the configuration of the middleware.
func NewInFlightReq(ctx context.Context, next http.Handler, config *dynamic.InFlightReq, name string) (http.Handler, error) {
	return inflightreq.New(ctx, next, *config, name)
}
