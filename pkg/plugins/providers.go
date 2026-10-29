package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

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

	return builder.build(config, "plugin-"+pName)
}

// ImplementsProvider reports whether a plugin implements the given option of the providers of the static configuration.
func (b Builder) ImplementsProvider(option string) bool {
	_, ok := b.typedProviderBuilders[option]
	return ok
}

// BuildTypedProvider builds the provider plugin implementing the given option of the providers of the static configuration
// (docker, kubernetesCRD, ...), with its configuration, a pointer to its type in pkg/config/static.
// The provider is named after the option.
func (b Builder) BuildTypedProvider(option string, config any) (provider.Provider, error) {
	builder, ok := b.typedProviderBuilders[option]
	if !ok {
		return nil, fmt.Errorf("no plugin implements the %q option", option)
	}

	cfg, ok := configValue(reflect.ValueOf(config)).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("the configuration of the %q option is a %T, a pointer to a struct is expected", option, config)
	}

	return builder.build(cfg, option)
}

type providerBuilder struct {
	plugin *vm.Plugin

	// native is the plugin when it is compiled into Traefik.
	native *nativePlugin
}

// build builds the provider, whose New function gets the given name.
func (b providerBuilder) build(config map[string]any, name string) (provider.Provider, error) {
	if b.native != nil {
		return newNativeProvider(b.native, config, name)
	}

	return newGoVMProvider(b.plugin, config, name)
}

// pluginProvider is a provider plugin that implements the Traefik provider interface.
type pluginProvider struct {
	provider.Provider

	// throttle is the throttle duration of the plugin, negative when the plugin uses the providersThrottleDuration option.
	throttle time.Duration

	// listen gives each new configuration of Traefik to the plugin, as the ACME provider resolves the certificates of the routers.
	listen func(dynamic.Configuration)

	// handler serves the requests of the plugin, as the REST provider gets its configuration this way.
	handler http.Handler
}

// ThrottleDuration returns the throttle duration of the plugin, negative when the plugin uses the providersThrottleDuration option.
func (p *pluginProvider) ThrottleDuration() time.Duration {
	return p.throttle
}

// ConfigurationListener returns the function giving the new configurations of Traefik to a provider plugin,
// when the plugin listens to them.
func ConfigurationListener(p provider.Provider) (func(dynamic.Configuration), bool) {
	pp, ok := p.(*pluginProvider)
	if !ok || pp.listen == nil {
		return nil, false
	}

	return pp.listen, true
}

// Handler returns the handler of a provider plugin that serves HTTP requests.
func Handler(p provider.Provider) (http.Handler, bool) {
	pp, ok := p.(*pluginProvider)
	if !ok || pp.handler == nil {
		return nil, false
	}

	return pp.handler, true
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
