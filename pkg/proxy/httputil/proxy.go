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

// ProxyBuilder handles roundtripper for the reverse proxy.
type ProxyBuilder struct {
	roundTrippers map[string]http.RoundTripper
	bufferPool    *bufferPool
}

// NewProxyBuilder creates a new ProxyBuilder.
func NewProxyBuilder() *ProxyBuilder {
	return &ProxyBuilder{
		roundTrippers: make(map[string]http.RoundTripper),
		bufferPool:    newBufferPool(),
	}
}

func (r *ProxyBuilder) Delete(configName string) {
	delete(r.roundTrippers, configName)
}

func (r *ProxyBuilder) Build(configName string, config *dynamic.HttpUtilConfig, tlsConfig *tls.Config, target *url.URL) (http.Handler, error) {
	roundTripper, ok := r.roundTrippers[configName]
	if !ok {
		var err error
		roundTripper, err = createRoundTripper(config, tlsConfig)
		if err != nil {
			return nil, err
		}

		r.roundTrippers[configName] = roundTripper
	}
	return buildSingleHostProxy(target, config.PassHostHeader, time.Duration(config.FlushInterval), roundTripper, r.bufferPool), nil
}

// createRoundTripper creates an http.RoundTripper configured with the Transport configuration settings.
// For the settings that can't be configured in Traefik it uses the default http.Transport settings.
// An exception to this is the MaxIdleConns setting as we only provide the option MaxIdleConnsPerHost in Traefik at this point in time.
// Setting this value to the default of 100 could lead to confusing behavior and backwards compatibility issues.
func createRoundTripper(cfg *dynamic.HttpUtilConfig, tlsConfig *tls.Config) (http.RoundTripper, error) {
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

	// Return directly HTTP/1.1 transport when HTTP/2 is disabled
	if cfg.DisableHTTP2 {
		return transport, nil
	}

	return newSmartRoundTripper(transport, cfg.ForwardingTimeouts)
}

func buildSingleHostProxy(target *url.URL, passHostHeader bool, flushInterval time.Duration, roundTripper http.RoundTripper, bufferPool httputil.BufferPool) http.Handler {
	return &httputil.ReverseProxy{
		Director:      DirectorBuilder(target, passHostHeader),
		Transport:     roundTripper,
		FlushInterval: flushInterval,
		BufferPool:    bufferPool,
		ErrorHandler:  ErrorHandler,
	}
}
