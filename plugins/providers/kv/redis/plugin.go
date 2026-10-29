package redis

import "context"

// CreateConfig creates the configuration of the redis provider plugin.
func CreateConfig() *Provider {
	p := &Provider{}
	p.SetDefaults()

	return p
}

// New creates the redis provider plugin: the provider itself, which implements the Traefik provider interface.
func New(_ context.Context, config *Provider, _ string) (*Provider, error) {
	return config, nil
}
