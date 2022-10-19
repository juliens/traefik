package fasthttp

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
)

// ProxyBuilder handles roundtripper for the reverse proxy.
type ProxyBuilder struct {
	pools map[string]map[string]ConnectionPool
}

// NewProxyBuilder creates a new ProxyBuilder.
func NewProxyBuilder() *ProxyBuilder {
	return &ProxyBuilder{
		pools: make(map[string]map[string]ConnectionPool),
	}
}

func (r *ProxyBuilder) Delete(configName string) {
	delete(r.pools, configName)
}

func (r *ProxyBuilder) Build(configName string, config *dynamic.FastHTTPConfig, tlsConfig *tls.Config, target *url.URL) http.Handler {
	pool := r.getPool(configName, config, tlsConfig, target)
	return NewFastHTTPReverseProxy(target, config.PassHostHeader, pool)
}

func (r *ProxyBuilder) getPool(configName string, config *dynamic.FastHTTPConfig, tlsConfig *tls.Config, target *url.URL) ConnectionPool {
	url := target.Host

	port := target.Port()
	if port == "" {
		if target.Scheme == "https" {
			url += ":443"
		} else {
			url += ":80"
		}
	}

	pool, ok := r.pools[configName]
	if !ok {
		pool = make(map[string]ConnectionPool)
		r.pools[configName] = pool
	}

	connectionPool, ok := pool[target.String()]
	if ok {
		return connectionPool
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// var maxIdleConnTimeout time.Duration
	if config.ForwardingTimeouts != nil {
		dialer.Timeout = time.Duration(config.ForwardingTimeouts.DialTimeout)
		// maxIdleConnTimeout = time.Duration(newConfig.ForwardingTimeouts.IdleConnTimeout)
	}

	connectionPool = NewConnectionPool(func() (net.Conn, error) {
		conn, err := dialer.Dial("tcp", url)
		if tlsConfig != nil {
			return tls.Client(conn, tlsConfig), nil
		}
		return conn, err
	}, config.MaxIdleConnsPerHost)

	r.pools[configName][target.String()] = connectionPool
	return connectionPool
}
