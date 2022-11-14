package httputil

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"golang.org/x/net/http2"
)

type h2cTransportWrapper struct {
	*http2.Transport
}

func (t *h2cTransportWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	return t.Transport.RoundTrip(req)
}

// ProxyBuilder handles the http.RoundTripper for httputil reverse proxies.
type ProxyBuilder struct {
	bufferPool *bufferPool
	// lock isn't needed because ProxyBuilder is not called concurrently.
	roundTrippers map[string]http.RoundTripper
}

// NewProxyBuilder creates a new ProxyBuilder.
func NewProxyBuilder() *ProxyBuilder {
	return &ProxyBuilder{
		bufferPool:    newBufferPool(),
		roundTrippers: make(map[string]http.RoundTripper),
	}
}

// Delete deletes the round-tripper corresponding to the given dynamic.HTTPClientConfig.
func (r *ProxyBuilder) Delete(cfgName string) {
	delete(r.roundTrippers, cfgName)
}

// Build builds a new httputil.ReverseProxy with the given configuration.
func (r *ProxyBuilder) Build(cfgName string, cfg *dynamic.HTTPClientConfig, tlsConfig *tls.Config, targetURL *url.URL) (http.Handler, error) {
	roundTripper, ok := r.roundTrippers[cfgName]
	if !ok {
		var err error
		roundTripper, err = createRoundTripper(cfg, tlsConfig)
		if err != nil {
			return nil, err
		}

		r.roundTrippers[cfgName] = roundTripper
	}

	return &httputil.ReverseProxy{
		Director:     DirectorBuilder(targetURL, cfg.PassHostHeader),
		Transport:    roundTripper,
		BufferPool:   r.bufferPool,
		ErrorHandler: ErrorHandler,
	}, nil
}

// createRoundTripper creates an http.RoundTripper configured with the Transport configuration settings.
// For the settings that can't be configured in Traefik it uses the default http.Transport settings.
// An exception to this is the MaxIdleConns setting as we only provide the option MaxIdleConnsPerHost in Traefik at this point in time.
// Setting this value to the default of 100 could lead to confusing behavior and backwards compatibility issues.
func createRoundTripper(cfg *dynamic.HTTPClientConfig, tlsConfig *tls.Config) (http.RoundTripper, error) {
	if cfg == nil {
		return nil, errors.New("no transport configuration given")
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if cfg.ForwardingTimeouts != nil {
		dialer.Timeout = time.Duration(cfg.ForwardingTimeouts.DialTimeout)
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ReadBufferSize:        64 * 1024,
		WriteBufferSize:       64 * 1024,
		TLSClientConfig:       tlsConfig,
	}

	if cfg.ForwardingTimeouts != nil {
		transport.ResponseHeaderTimeout = time.Duration(cfg.ForwardingTimeouts.ResponseHeaderTimeout)
		transport.IdleConnTimeout = time.Duration(cfg.ForwardingTimeouts.IdleConnTimeout)
	}

	return newSmartRoundTripper(transport, cfg.ForwardingTimeouts)
}
