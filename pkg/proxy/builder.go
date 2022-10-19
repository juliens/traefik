package proxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/proxy/fasthttp"
	"github.com/traefik/traefik/v2/pkg/proxy/httputil"
)

type TLSConfigGetter interface {
	GetTLSConfig(configName string) (*tls.Config, error)
}

type Builder struct {
	fasthttpBuilder *fasthttp.ProxyBuilder
	httputilBuilder *httputil.ProxyBuilder

	tlsConfigManager TLSConfigGetter

	configsLock sync.RWMutex
	configs     map[string]*dynamic.ServersTransport
}

func NewBuilder(tlsConfigManager TLSConfigGetter) *Builder {
	return &Builder{
		fasthttpBuilder:  fasthttp.NewProxyBuilder(),
		httputilBuilder:  httputil.NewProxyBuilder(),
		tlsConfigManager: tlsConfigManager,

		configs: make(map[string]*dynamic.ServersTransport),
	}
}

func (b *Builder) Build(configName string, target *url.URL) (http.Handler, error) {
	if len(configName) == 0 {
		configName = "default@internal"
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
	// FIXME Raise error if both configured
	if config.HttpUtil != nil {
		return b.httputilBuilder.Build(configName, config.HttpUtil, tlsConfig, target)
	}

	if config.FastHTTP != nil {
		return b.fasthttpBuilder.Build(configName, config.FastHTTP, tlsConfig, target), nil
	}

	// return b.httputilBuilder.Build(configName, &dynamic.HttpUtilConfig{}, tlsConfig, target)

	return nil, fmt.Errorf("invalid serversTransport %s", configName)
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
