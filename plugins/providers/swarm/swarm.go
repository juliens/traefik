// Package swarm is the Docker Swarm provider plugin.
package swarm

import (
	"context"
	"time"

	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/plugins/providers/docker"
)

// CreateConfig creates the configuration of the Docker Swarm provider plugin.
func CreateConfig() *docker.SwarmProvider {
	p := &docker.SwarmProvider{}
	p.SetDefaults()

	return p
}

// New creates the Docker Swarm provider plugin: the provider itself, which implements the Traefik provider interface.
func New(_ context.Context, config *docker.SwarmProvider, _ string) (*docker.SwarmProvider, error) {
	if config.RefreshSeconds <= 0 {
		config.RefreshSeconds = ptypes.Duration(15 * time.Second)
	}

	if config.HTTPClientTimeout < 0 {
		config.HTTPClientTimeout = 0
	}

	return config, nil
}
