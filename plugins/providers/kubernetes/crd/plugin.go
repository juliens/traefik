package crd

import (
	"context"
	"encoding/json"

	"github.com/traefik/traefik/v3/pkg/plugins/host"
)

// registryKey is the key of the host registry (see pkg/plugins/host) where the plugin registers its configuration,
// so the Gateway API provider plugin resolves the Traefik CRD objects the Gateway API routes refer to.
const registryKey = "kubernetescrd"

// registeredConfig is the configuration registered by the plugin.
// The plugins run in different machines: the configuration goes through the registry as JSON, a native string.
type registeredConfig struct {
	Namespaces []string `json:"namespaces,omitempty"`
	SafeNaming bool     `json:"safeNaming,omitempty"`
}

// CreateConfig creates the configuration of the Kubernetes CRD provider plugin.
func CreateConfig() *Provider {
	return &Provider{}
}

// New creates the Kubernetes CRD provider plugin: the provider itself, which implements the Traefik provider interface.
func New(_ context.Context, config *Provider, _ string) (*Provider, error) {
	data, err := json.Marshal(registeredConfig{
		Namespaces: config.Namespaces,
		SafeNaming: config.SafeNaming != nil && *config.SafeNaming,
	})
	if err != nil {
		return nil, err
	}

	host.Register(registryKey, string(data))

	return config, nil
}

// Registered returns the Kubernetes CRD provider registered by the CRD provider plugin, initialized.
func Registered() (*Provider, bool) {
	v, ok := host.Lookup(registryKey)
	if !ok {
		return nil, false
	}

	data, ok := v.(string)
	if !ok {
		return nil, false
	}

	var cfg registeredConfig
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, false
	}

	p := &Provider{Namespaces: cfg.Namespaces}
	if cfg.SafeNaming {
		p.SafeNaming = &cfg.SafeNaming
	}

	if err := p.Init(); err != nil {
		return nil, false
	}

	return p, true
}
