package nomad

import (
	"context"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/safe"
)

// CreateConfig creates the configuration of the nomad provider plugin.
func CreateConfig() *ProviderBuilder {
	p := &ProviderBuilder{}
	p.SetDefaults()

	return p
}

// New creates the nomad provider plugin, which runs a provider per namespace.
func New(_ context.Context, config *ProviderBuilder, _ string) (*Providers, error) {
	return &Providers{providers: config.BuildProviders()}, nil
}

// Providers runs the providers of the namespaces.
type Providers struct {
	providers []*Provider
}

// Init initializes the providers.
func (p *Providers) Init() error {
	for _, prd := range p.providers {
		if err := prd.Init(); err != nil {
			return err
		}
	}

	return nil
}

// Provide starts the providers.
func (p *Providers) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	for _, prd := range p.providers {
		if err := prd.Provide(configurationChan, pool); err != nil {
			return err
		}
	}

	return nil
}
