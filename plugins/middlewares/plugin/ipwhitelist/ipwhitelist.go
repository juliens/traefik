// Package plugin is the ipWhiteList middleware of Traefik, as a plugin.
package plugin

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/ipwhitelist"
)

// NewIpWhiteList creates the ipWhiteList middleware: Traefik gives it the ipWhiteList option of the configuration of the middleware.
func NewIpWhiteList(ctx context.Context, next http.Handler, config *dynamic.IPWhiteList, name string) (http.Handler, error) { //nolint:revive // Traefik builds the name of the constructor from the name of the option.
	log.Warn().Msgf("Middleware %q of type IPWhiteList is deprecated, please use IPAllowList instead.", name)

	return ipwhitelist.New(ctx, next, *config, name)
}
