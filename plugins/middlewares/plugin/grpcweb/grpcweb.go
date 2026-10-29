// Package plugin is the grpcWeb middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/grpcweb"
)

// NewGrpcWeb creates the grpcWeb middleware: Traefik gives it the grpcWeb option of the configuration of the middleware.
func NewGrpcWeb(ctx context.Context, next http.Handler, config *dynamic.GrpcWeb, name string) (http.Handler, error) {
	return grpcweb.New(ctx, next, *config, name), nil
}
