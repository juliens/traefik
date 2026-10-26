package knative

import "context"

// CreateConfig creates the configuration of the knative provider plugin.
func CreateConfig() *Provider {
	p := &Provider{}

	return p
}

// New creates the knative provider plugin: the provider itself, which implements the Traefik provider interface.
func New(_ context.Context, config *Provider, _ string) (*Provider, error) {
	return config, nil
}
