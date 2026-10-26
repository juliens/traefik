package plugins

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/traefik/go-vm/vm"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/provider"
	"github.com/traefik/traefik/v3/pkg/safe"
)

// PP the interface of a plugin's provider.
type PP interface {
	Init() error
	Provide(cfgChan chan<- json.Marshaler) error
	Stop() error
}

// BuildProvider builds a plugin's provider.
func (b Builder) BuildProvider(pName string, config map[string]any) (provider.Provider, error) {
	if b.providerBuilders == nil {
		return nil, fmt.Errorf("no plugin definition in the static configuration: %s", pName)
	}

	builder, ok := b.providerBuilders[pName]
	if !ok {
		return nil, fmt.Errorf("unknown plugin type: %s", pName)
	}

	return newGoVMProvider(builder.plugin, config, pName)
}

type providerBuilder struct {
	plugin *vm.Plugin
}

// Provider is a plugin's provider wrapper.
type Provider struct {
	name string
	pp   PP
}

// Init wraps the Init method of a plugin.
func (p *Provider) Init() error {
	return p.pp.Init()
}

// Provide wraps the Provide method of a plugin.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error().Str(logs.ProviderName, p.name).Msgf("Panic inside the plugin %v", err)
		}
	}()

	cfgChan := make(chan json.Marshaler)

	pool.GoCtx(func(ctx context.Context) {
		logger := log.Ctx(ctx).With().Str(logs.ProviderName, p.name).Logger()

		for {
			select {
			case <-ctx.Done():
				err := p.pp.Stop()
				if err != nil {
					logger.Error().Err(err).Msg("Failed to stop the provider")
				}

				return

			case cfgPg := <-cfgChan:
				marshalJSON, err := cfgPg.MarshalJSON()
				if err != nil {
					logger.Error().Err(err).Msg("Failed to marshal configuration")
					continue
				}

				cfg := &dynamic.Configuration{}
				err = json.Unmarshal(marshalJSON, cfg)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to unmarshal configuration")
					continue
				}

				configurationChan <- dynamic.Message{
					ProviderName:  p.name,
					Configuration: cfg,
				}
			}
		}
	})

	err := p.pp.Provide(cfgChan)
	if err != nil {
		return fmt.Errorf("error from %s: %w", p.name, err)
	}

	return nil
}
