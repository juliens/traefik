package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	traefiktls "github.com/traefik/traefik/v3/pkg/tls"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/plugins"
	"github.com/traefik/traefik/v3/pkg/plugins/host"
)

const outputDir = "./plugins-storage/"

func createPluginBuilder(staticConfiguration *static.Configuration) (*plugins.Builder, error) {
	manager, plgs, localPlgs, err := initPlugins(staticConfiguration)
	if err != nil {
		return nil, err
	}

	return plugins.NewBuilder(manager, plgs, localPlgs)
}

func initPlugins(staticCfg *static.Configuration) (*plugins.Manager, map[string]plugins.Descriptor, map[string]plugins.LocalDescriptor, error) {
	err := checkUniquePluginNames(staticCfg.Experimental)
	if err != nil {
		return nil, nil, nil, err
	}

	var manager *plugins.Manager
	plgs := map[string]plugins.Descriptor{}

	if hasPlugins(staticCfg) {
		httpClient := retryablehttp.NewClient()
		httpClient.Logger = logs.NewRetryableHTTPLogger(log.Logger)
		httpClient.HTTPClient = &http.Client{Timeout: 10 * time.Second}
		httpClient.RetryMax = 3

		// Create separate downloader for HTTP operations
		archivesPath := filepath.Join(outputDir, "archives")
		downloader, err := plugins.NewRegistryDownloader(plugins.RegistryDownloaderOptions{
			HTTPClient:   httpClient.HTTPClient,
			ArchivesPath: archivesPath,
		})
		if err != nil {
			return nil, nil, nil, fmt.Errorf("unable to create plugin downloader: %w", err)
		}

		opts := plugins.ManagerOptions{
			Output: outputDir,
		}
		manager, err = plugins.NewManager(downloader, opts)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("unable to create plugins manager: %w", err)
		}

		err = plugins.SetupRemotePlugins(manager, staticCfg.Experimental.Plugins)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("unable to set up plugins environment: %w", err)
		}

		plgs = staticCfg.Experimental.Plugins
	}

	localPlgs := map[string]plugins.LocalDescriptor{}

	if hasLocalPlugins(staticCfg) {
		err := plugins.SetupLocalPlugins(staticCfg.Experimental.LocalPlugins)
		if err != nil {
			return nil, nil, nil, err
		}

		localPlgs = staticCfg.Experimental.LocalPlugins
	}

	return manager, plgs, localPlgs, nil
}

func checkUniquePluginNames(e *static.Experimental) error {
	if e == nil {
		return nil
	}

	for s := range e.LocalPlugins {
		if _, ok := e.Plugins[s]; ok {
			return fmt.Errorf("the plugin's name %q must be unique", s)
		}
	}

	return nil
}

func hasPlugins(staticCfg *static.Configuration) bool {
	return staticCfg.Experimental != nil && len(staticCfg.Experimental.Plugins) > 0
}

func hasLocalPlugins(staticCfg *static.Configuration) bool {
	return staticCfg.Experimental != nil && len(staticCfg.Experimental.LocalPlugins) > 0
}

// setPluginsHost gives the plugins the information of the static configuration they need (see pkg/plugins/host).
func setPluginsHost(staticCfg *static.Configuration, tlsManager *traefiktls.Manager) {
	eps := make(map[string]host.EntryPoint, len(staticCfg.EntryPoints))
	for name, ep := range staticCfg.EntryPoints {
		eps[name] = host.EntryPoint{
			Address:         ep.GetAddress(),
			HasHTTPTLSConf:  ep.HTTP.TLS != nil,
			AsDefault:       ep.AsDefault,
			AllowACMEByPass: ep.AllowACMEByPass,
		}
	}

	var ruleSyntax string
	if staticCfg.Core != nil {
		ruleSyntax = staticCfg.Core.DefaultRuleSyntax
	}

	host.SetStatic(eps, ruleSyntax)

	host.SetCertificateDomains(func(store string) []string {
		st := tlsManager.GetStore(store)
		if st == nil {
			return nil
		}

		return st.GetAllDomains()
	})
}
