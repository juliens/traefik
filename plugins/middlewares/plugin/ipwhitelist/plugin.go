// Package ipwhitelist is the plugin of the ipWhiteList middleware of Traefik.
package ipwhitelist

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	ipwhitelistmw "github.com/traefik/traefik/v3/plugins/middlewares/ipwhitelist"
)

// CreateConfig creates the configuration of the ipWhiteList middleware.
func CreateConfig() *dynamic.IPWhiteList {
	return &dynamic.IPWhiteList{}
}

// New creates the ipWhiteList middleware: Traefik gives it the ipWhiteList option of the configuration of the middleware.
func New(ctx context.Context, next http.Handler, config *dynamic.IPWhiteList, name string) (http.Handler, error) {
	log.Warn().Msgf("Middleware %q of type IPWhiteList is deprecated, please use IPAllowList instead.", name)

	return ipwhitelistmw.New(ctx, next, *config, name)
}
