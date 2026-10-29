package rest

import (
	"context"
	"net/http"
)

// CreateConfig creates the configuration of the REST provider plugin.
func CreateConfig() *Provider {
	return &Provider{}
}

// New creates the REST provider plugin.
func New(_ context.Context, config *Provider, _ string) (*PluginProvider, error) {
	return &PluginProvider{Provider: config, router: config.CreateRouter()}, nil
}

// PluginProvider is the REST provider of the plugin.
// Traefik serves it on the service <plugin name>@internal: a router of the configuration sends the requests of the API to it.
type PluginProvider struct {
	*Provider

	router http.Handler
}

// InternalRule returns the rule of the router from the entry point named traefik to the REST API, when it is insecure.
func (p *PluginProvider) InternalRule() string {
	if !p.Insecure {
		return ""
	}

	return "PathPrefix(`/api/providers`)"
}

// ServeHTTP serves the REST API.
func (p *PluginProvider) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.router.ServeHTTP(rw, req)
}
