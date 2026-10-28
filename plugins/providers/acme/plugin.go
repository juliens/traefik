package acme

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"

	legolog "github.com/go-acme/lego/v5/log"
	"github.com/rs/zerolog/log"
	slogzerolog "github.com/samber/slog-zerolog/v2"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/plugins/host"
	"github.com/traefik/traefik/v3/pkg/safe"
)

// defaultCAServer is the default ACME API endpoint.
const defaultCAServer = "https://acme-v02.api.letsencrypt.org/directory"

// maxUserPriority is the highest priority of the routers of the providers other than the internal one.
const maxUserPriority = math.MaxInt - 1000

// CreateConfig creates the configuration of the ACME provider plugin.
func CreateConfig() *Configuration {
	config := &Configuration{}
	config.SetDefaults()

	return config
}

// New creates the ACME provider plugin: the certificate resolver named after the plugin.
// A router uses it with the name of the plugin in its tls.certResolver option.
func New(ctx context.Context, config *Configuration, name string) (*PluginProvider, error) {
	// Traefik gives the name plugin-<name> to the plugins, the routers use <name>.
	resolverName := strings.TrimPrefix(name, "plugin-")

	if len(config.Storage) == 0 {
		return nil, fmt.Errorf("unable to initialize certificates resolver %q with no storage location for the certificates", resolverName)
	}

	setEffectiveConfiguration(config)

	legolog.SetDefault(
		slog.New(
			slogzerolog.Option{Logger: &log.Logger}.
				NewZerologHandler().
				WithAttrs([]slog.Attr{slog.String("lib", "lego")}),
		),
	)

	httpChallenge := NewChallengeHTTP()
	tlsChallenge := NewChallengeTLSALPN(resolverName)

	return &PluginProvider{
		Provider: &Provider{
			Configuration:          config,
			ResolverName:           resolverName,
			Store:                  sharedLocalStore(ctx, config.Storage),
			HTTPChallengeProvider:  httpChallenge,
			TLSChallengeProvider:   tlsChallenge,
			HTTPChallengeRouters:   httpChallengeRouters(resolverName, config.HTTPChallenge, host.EntryPoints()),
			configFromListenerChan: make(chan dynamic.Configuration),
		},
		httpChallenge: httpChallenge,
		tlsChallenge:  tlsChallenge,
	}, nil
}

var (
	localStoresMu sync.Mutex
	localStores   = map[string]*LocalStore{}
)

// sharedLocalStore returns the store of a storage file: the resolvers using the same file share its store,
// which keeps the data of each resolver, as two stores would overwrite the file of each other.
// Traefik loads the module of the plugin once for all its resolvers, so they share this package.
func sharedLocalStore(ctx context.Context, filename string) *LocalStore {
	localStoresMu.Lock()
	defer localStoresMu.Unlock()

	if store, ok := localStores[filename]; ok {
		return store
	}

	store := NewLocalStore(filename, safe.NewPool(ctx))
	localStores[filename] = store

	return store
}

// PluginProvider is the ACME provider of the plugin.
// Traefik serves it on the service <resolver>@internal, where the routers of the HTTP challenges send the requests,
// and gives it each new configuration, from which it gets the domains of the routers using the resolver.
type PluginProvider struct {
	*Provider

	httpChallenge *ChallengeHTTP
	tlsChallenge  *ChallengeTLSALPN
}

// Provide starts the provider, and the provider of the TLS challenge certificates, which sends them on the same channel.
func (p *PluginProvider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	if err := p.tlsChallenge.Provide(configurationChan, pool); err != nil {
		return err
	}

	return p.Provider.Provide(configurationChan, pool)
}

// ListenConfiguration gets the new configurations of Traefik.
func (p *PluginProvider) ListenConfiguration(config dynamic.Configuration) {
	p.tlsChallenge.ListenConfiguration(config)
	p.Provider.ListenConfiguration(config)
}

// ServeHTTP answers the HTTP challenges.
func (p *PluginProvider) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.httpChallenge.ServeHTTP(rw, req)
}

// httpChallengeRouters returns the routers sending the HTTP challenges of the entry point to the service of the resolver.
func httpChallengeRouters(resolverName string, challenge *HTTPChallenge, entryPoints map[string]host.EntryPoint) map[string]*dynamic.Router {
	if challenge == nil || challenge.EntryPoint == "" {
		return nil
	}

	service := resolverName + "@internal"

	// The routers of the configuration handle the challenges when the entry point allows it.
	if entryPoints[challenge.EntryPoint].AllowACMEByPass {
		return map[string]*dynamic.Router{
			"acme-http-bypass": {
				Rule:        "PathPrefix(`/.well-known/acme-challenge/`)",
				EntryPoints: []string{challenge.EntryPoint},
				Service:     service,
			},
		}
	}

	return map[string]*dynamic.Router{
		"acme-http": {
			Rule: "PathPrefix(`/.well-known/acme-challenge/`)",
			// "default" stands for the default rule syntax in Traefik v3, i.e. the v3 syntax.
			RuleSyntax:  "default",
			EntryPoints: []string{challenge.EntryPoint},
			Service:     service,
			Priority:    maxUserPriority,
		},
	}
}

// setEffectiveConfiguration applies the deprecated options and the default CA server.
func setEffectiveConfiguration(config *Configuration) {
	config.CAServer = getSafeACMECAServer(config.CAServer)

	if config.DNSChallenge == nil {
		return
	}

	switch config.DNSChallenge.Provider {
	case "googledomains", "cloudxns", "brandit":
		log.Warn().Msgf("%s DNS provider is deprecated.", config.DNSChallenge.Provider)
	case "dnspod":
		log.Warn().Msgf("%s provider is deprecated, please use 'tencentcloud' provider instead.", config.DNSChallenge.Provider)
	case "azure":
		log.Warn().Msgf("%s provider is deprecated, please use 'azuredns' provider instead.", config.DNSChallenge.Provider)
	}

	if config.DNSChallenge.DisablePropagationCheck {
		log.Warn().Msgf("disablePropagationCheck is now deprecated, please use propagation.disableChecks instead.")

		config.DNSChallenge.Propagation.DisableChecks = true
	}

	if config.DNSChallenge.DelayBeforeCheck > 0 {
		log.Warn().Msgf("delayBeforeCheck is now deprecated, please use propagation.delayBeforeChecks instead.")

		config.DNSChallenge.Propagation.DelayBeforeChecks = config.DNSChallenge.DelayBeforeCheck
	}
}

func getSafeACMECAServer(caServerSrc string) string {
	if len(caServerSrc) == 0 {
		return defaultCAServer
	}

	if strings.HasPrefix(caServerSrc, "https://acme-v01.api.letsencrypt.org") {
		caServer := strings.Replace(caServerSrc, "v01", "v02", 1)
		log.Warn().Msgf("The CA server %[1]q refers to a v01 endpoint of the ACME API, please change to %[2]q. Fallback to %[2]q.", caServerSrc, caServer)
		return caServer
	}

	if strings.HasPrefix(caServerSrc, "https://acme-staging.api.letsencrypt.org") {
		caServer := strings.Replace(caServerSrc, "https://acme-staging.api.letsencrypt.org", "https://acme-staging-v02.api.letsencrypt.org", 1)
		log.Warn().Msgf("The CA server %[1]q refers to a v01 endpoint of the ACME API, please change to %[2]q. Fallback to %[2]q.", caServerSrc, caServer)
		return caServer
	}

	return caServerSrc
}
