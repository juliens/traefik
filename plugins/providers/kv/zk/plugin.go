package zk

import "context"

// CreateConfig creates the configuration of the zk provider plugin.
func CreateConfig() *Provider {
	p := &Provider{}
	p.SetDefaults()

	return p
}

// New creates the zk provider plugin: the provider itself, which implements the Traefik provider interface.
func New(_ context.Context, config *Provider, _ string) (*Provider, error) {
	return config, nil
}
