package docker

import "context"

// CreateConfig creates the configuration of the docker provider plugin.
func CreateConfig() *Provider {
	p := &Provider{}
	p.SetDefaults()

	return p
}

// New creates the docker provider plugin: the provider itself, which implements the Traefik provider interface.
func New(_ context.Context, config *Provider, _ string) (*Provider, error) {
	if config.HTTPClientTimeout < 0 {
		config.HTTPClientTimeout = 0
	}

	return config, nil
}
