package fasthttp

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
)

// ProxyBuilder handles the connection pools for the FastHTTP proxies.
type ProxyBuilder struct {
	pools map[string]map[string]*ConnPool // FIXME lock?
}

// NewProxyBuilder creates a new ProxyBuilder.
func NewProxyBuilder() *ProxyBuilder {
	return &ProxyBuilder{
		pools: make(map[string]map[string]*ConnPool),
	}
}

// Delete deletes the round-tripper corresponding to the given dynamic.HTTPClientConfig.
func (r *ProxyBuilder) Delete(cfgName string) {
	delete(r.pools, cfgName)
}

// Build builds a new ReverseProxy with the given configuration.
func (r *ProxyBuilder) Build(cfgName string, cfg *dynamic.HTTPClientConfig, tlsConfig *tls.Config, target *url.URL) http.Handler {
	pool := r.getPool(cfgName, cfg, tlsConfig, target)

	return NewReverseProxy(target, cfg.PassHostHeader, pool)
}

// FIXME Support IdleConnTimeout.
func (r *ProxyBuilder) getPool(cfgName string, config *dynamic.HTTPClientConfig, tlsConfig *tls.Config, target *url.URL) *ConnPool {
	addr := target.Host
	if target.Port() == "" {
		if target.Scheme == "https" {
			addr += ":443"
		} else {
			addr += ":80"
		}
	}

	pool, ok := r.pools[cfgName]
	if !ok {
		pool = make(map[string]*ConnPool)
		r.pools[cfgName] = pool
	}

	if connPool, ok := pool[target.String()]; ok {
		return connPool
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if config.ForwardingTimeouts != nil {
		dialer.Timeout = time.Duration(config.ForwardingTimeouts.DialTimeout)
	}

	dial := func() (net.Conn, error) {
		if tlsConfig != nil {
			return tls.Dial("tcp", addr, tlsConfig)
		}
		return dialer.Dial("tcp", addr)
	}
	connPool := NewConnPool(dial, config.MaxIdleConnsPerHost)

	r.pools[cfgName][target.String()] = connPool
	return connPool
}
