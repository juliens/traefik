package plugin

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/plugins/middlewares/ipallowlist"
)

// NewIpAllowList creates the ipAllowList middleware: Traefik gives it the ipAllowList option of the configuration of the middleware.
func NewIpAllowList(ctx context.Context, next http.Handler, config *dynamic.IPAllowList, name string) (http.Handler, error) { //nolint:revive // Traefik builds the name of the constructor from the name of the option.
	return ipallowlist.New(ctx, next, *config, name)
}
