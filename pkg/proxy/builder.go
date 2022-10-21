package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/proxy/fasthttp"
	"github.com/traefik/traefik/v2/pkg/proxy/httputil"
	"github.com/traefik/traefik/v2/pkg/tls/client"
)

type Builder struct {
	fasthttpBuilder *fasthttp.ProxyBuilder
	httputilBuilder *httputil.ProxyBuilder

	tlsConfigManager *client.TLSConfigManager

	configsLock sync.RWMutex
	configs     map[string]*dynamic.ServersTransport
}

func NewBuilder(tlsConfigManager *client.TLSConfigManager) *Builder {
	return &Builder{
		fasthttpBuilder:  fasthttp.NewProxyBuilder(),
		httputilBuilder:  httputil.NewProxyBuilder(),
		tlsConfigManager: tlsConfigManager,
		configs:          make(map[string]*dynamic.ServersTransport),
	}
}

func (b *Builder) Build(configName string, target *url.URL) (http.Handler, error) {
	if len(configName) == 0 {
		configName = "default"
	}

	b.configsLock.RLock()
	defer b.configsLock.RUnlock()

	config, ok := b.configs[configName]
	if !ok {
		return nil, fmt.Errorf("unknown serversTransport config %s", configName)
	}

	tlsConfig, err := b.tlsConfigManager.GetTLSConfig(configName)
	if err != nil {
		return nil, err
	}

	if config.HTTP != nil && config.HTTP.EnableHTTP2 && target.Scheme == "https" || target.Scheme == "h2c" {
		return b.httputilBuilder.Build(configName, config.HTTP, tlsConfig, target)
	}

	return b.fasthttpBuilder.Build(configName, config.HTTP, tlsConfig, target), nil
}

func (r *Builder) Update(newConfigs map[string]*dynamic.ServersTransport) {
	r.configsLock.Lock()
	defer r.configsLock.Unlock()

	for configName := range r.configs {
		if _, ok := newConfigs[configName]; !ok {
			r.httputilBuilder.Delete(configName)
			r.fasthttpBuilder.Delete(configName)
		}
	}

	for newConfigName, newConfig := range newConfigs {
		if !reflect.DeepEqual(newConfig, r.configs[newConfigName]) {
			// Delete previous cache on builders because configuration changed.
			r.httputilBuilder.Delete(newConfigName)
			r.fasthttpBuilder.Delete(newConfigName)
		}
	}

	r.configs = newConfigs
}
