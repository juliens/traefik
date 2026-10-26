// Package plugin is the Kubernetes Gateway API provider plugin.
package plugin

import (
	"context"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/plugins/host"
	"github.com/traefik/traefik/v3/pkg/safe"
	"github.com/traefik/traefik/v3/plugins/providers/kubernetes/crd"
	"github.com/traefik/traefik/v3/plugins/providers/kubernetes/gateway"
)

// CreateConfig creates the configuration of the Kubernetes Gateway API provider plugin.
func CreateConfig() *gateway.Provider {
	p := &gateway.Provider{}
	p.SetDefaults()

	return p
}

// New creates the Kubernetes Gateway API provider plugin.
func New(_ context.Context, config *gateway.Provider, _ string) (*Provider, error) {
	config.EntryPoints = make(map[string]gateway.Entrypoint)
	for name, ep := range host.EntryPoints() {
		config.EntryPoints[name] = gateway.Entrypoint{Address: ep.Address, HasHTTPTLSConf: ep.HasHTTPTLSConf}
	}

	return &Provider{Provider: config}, nil
}

// Provider is the Gateway API provider of the plugin.
type Provider struct {
	*gateway.Provider
}

// Provide starts the provider.
// The Gateway API routes can refer to the Traefik CRD objects when the Kubernetes CRD provider plugin runs:
// its configuration is read when the provider starts, once all the provider plugins are initialized.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	if crdProvider, ok := crd.Registered(); ok {
		crdProvider.FillExtensionBuilderRegistry(p.Provider)
	}

	return p.Provider.Provide(configurationChan, pool)
}
