// Package grpcweb is the plugin of the grpcWeb middleware of Traefik.
package grpcweb

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	grpcwebmw "github.com/traefik/traefik/v3/plugins/middlewares/grpcweb"
)

// CreateConfig creates the configuration of the grpcWeb middleware.
func CreateConfig() *dynamic.GrpcWeb {
	return &dynamic.GrpcWeb{}
}

// New creates the grpcWeb middleware: Traefik gives it the grpcWeb option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.GrpcWeb, name string) (http.Handler, error) {
	return grpcwebmw.New(ctx, next, *config, name), nil
}
