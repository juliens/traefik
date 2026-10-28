// Package plugin is the plugin of the middlewares of Traefik other than chain.
// It implements the options of the dynamic configuration of the middlewares listed in its manifest,
// with a New<Option> function per option: loaded once, it gives all of them.
package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/plugins/host"
)

// services gives the services of the configuration to the middlewares, from the context given to their constructor.
type services struct{}

func (services) BuildHTTP(ctx context.Context, serviceName string) (http.Handler, error) {
	return host.BuildService(ctx, serviceName)
}
