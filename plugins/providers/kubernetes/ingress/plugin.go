package ingress

import (
	"context"

	"github.com/traefik/traefik/v3/pkg/plugins/host"
)

// CreateConfig creates the configuration of the Kubernetes Ingress provider plugin.
func CreateConfig() *Provider {
	return &Provider{}
}

// New creates the Kubernetes Ingress provider plugin: the provider itself, which implements the Traefik provider interface.
func New(_ context.Context, config *Provider, _ string) (*Provider, error) {
	// The provider adapts the matchers of the rules to the default rule syntax of Traefik.
	config.DefaultRuleSyntax = host.DefaultRuleSyntax()

	return config, nil
}
