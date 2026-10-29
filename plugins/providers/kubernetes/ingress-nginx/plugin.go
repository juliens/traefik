package ingressnginx

import (
	"context"
	"errors"

	"github.com/traefik/traefik/v3/pkg/plugins/host"
)

// internalEntryPointName is the name of the entry point of the internal services of Traefik.
const internalEntryPointName = "traefik"

// CreateConfig creates the configuration of the Kubernetes Ingress NGINX provider plugin.
func CreateConfig() *Provider {
	p := &Provider{}
	p.SetDefaults()

	return p
}

// New creates the Kubernetes Ingress NGINX provider plugin: the provider itself, which implements the Traefik provider interface.
func New(_ context.Context, config *Provider, _ string) (*Provider, error) {
	if config.WatchNamespace != "" && config.WatchNamespaceSelector != "" {
		return nil, errors.New("watchNamespace and watchNamespaceSelector options are mutually exclusive")
	}

	if !config.DisableNonTLSRouters {
		config.NonTLSEntryPoints = nonTLSEntryPoints(host.EntryPoints())
	}

	return config, nil
}

// nonTLSEntryPoints returns the entry points without TLS of the routers: the default entry points when there are some,
// all the entry points otherwise, except the internal one.
func nonTLSEntryPoints(entryPoints map[string]host.EntryPoint) []string {
	var hasDefaults bool
	for _, ep := range entryPoints {
		if ep.AsDefault {
			hasDefaults = true
			break
		}
	}

	var names []string
	for name, ep := range entryPoints {
		if (hasDefaults && !ep.AsDefault) || name == internalEntryPointName {
			continue
		}

		if !ep.HasHTTPTLSConf {
			names = append(names, name)
		}
	}

	return names
}
