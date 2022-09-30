package service

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"

	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	traefiktls "github.com/traefik/traefik/v2/pkg/tls"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/http2"
)

type h2cTransportWrapper struct {
	*http2.Transport
}

func (t *h2cTransportWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	return t.Transport.RoundTrip(req)
}

// FIXME rename to proxyManager
// NewRoundTripperManager creates a new RoundTripperManager.
func NewRoundTripperManager() *RoundTripperManager {
	return &RoundTripperManager{
		roundTrippers: make(map[string]http.RoundTripper),
		configs:       make(map[string]*dynamic.ServersTransport),
		proxies:       make(map[string]http.Handler),
		bufferPool:    newBufferPool(),
	}
}

// RoundTripperManager handles roundtripper for the reverse proxy.
type RoundTripperManager struct {
	rtLock        sync.RWMutex
	roundTrippers map[string]http.RoundTripper
	configs       map[string]*dynamic.ServersTransport

	proxies    map[string]http.Handler
	bufferPool *bufferPool
}

// Update updates the roundtrippers configurations.
func (r *RoundTripperManager) Update(newConfigs map[string]*dynamic.ServersTransport) {
	r.rtLock.Lock()
	defer r.rtLock.Unlock()

	for configName := range r.configs {
		if _, ok := newConfigs[configName]; !ok {
			delete(r.configs, configName)
			delete(r.roundTrippers, configName)
			delete(r.proxies, configName)
		}
	}

	for newConfigName, newConfig := range newConfigs {
		if reflect.DeepEqual(newConfig, r.configs[newConfigName]) {
			continue
		}

		transport, err := createRoundTripper(newConfig)
		if err != nil {
			log.WithoutContext().Errorf("Could not configure HTTP Transport %s, fallback on default transport: %v", newConfigName, err)
			transport = http.DefaultTransport
		}
		r.roundTrippers[newConfigName] = transport

		if newConfig.FastHTTP == nil {
			// FIXME Handle flushInterval on serversTransport
			var flushInterval ptypes.Duration
			err := flushInterval.Set(newConfig.FlushInterval)
			if err != nil {
				log.WithoutContext().Errorf("Error creating flushInterval: %v", newConfigName, err)
			}

			r.proxies[newConfigName] = buildProxy(flushInterval, transport, r.bufferPool, newConfig.PassHostHeader)
		} else {
			// dialer := &net.Dialer{
			// 	Timeout:   30 * time.Second,
			// 	KeepAlive: 30 * time.Second,
			// }
			//
			// var maxIdleConnTimeout time.Duration
			// if newConfig.ForwardingTimeouts != nil {
			// 	dialer.Timeout = time.Duration(newConfig.ForwardingTimeouts.DialTimeout)
			// 	maxIdleConnTimeout = time.Duration(newConfig.ForwardingTimeouts.IdleConnTimeout)
			// }
			// //
			// tlsConfig, err := createTLSConfig(newConfig)
			// if err != nil {
			// 	log.WithoutContext().Errorf("Could not configure HTTP Transport %s, fallback on default transport: %v", newConfigName, err)
			//
			// }

			// &fasthttp.Client{
			// 	Dial: func(addr string) (net.Conn, error) {
			// 		return dialer.Dial("tcp", addr)
			// 	},
			//
			// 	TLSConfig:           tlsConfig,
			// 	MaxIdleConnDuration: maxIdleConnTimeout,
			// 	ReadBufferSize:      64 * 1024,
			// 	WriteBufferSize:     64 * 1024,
			//
			// 	DisableHeaderNamesNormalizing: true,
			// 	DisablePathNormalizing:        true,
			// 	NoDefaultUserAgentHeader:      false,
			// }

			r.proxies[newConfigName] = NewFastHTTPReverseProxy(newConfig.PassHostHeader)
		}
	}

	r.configs = newConfigs
}

func (r *RoundTripperManager) Get(name string) (http.RoundTripper, error) {
	if len(name) == 0 {
		name = "default@internal"
	}

	r.rtLock.RLock()
	defer r.rtLock.RUnlock()

	if rt, ok := r.roundTrippers[name]; ok {
		return rt, nil
	}

	return nil, fmt.Errorf("servers transport not found %s", name)
}

func (r *RoundTripperManager) GetProxy(name string) (http.Handler, error) {
	if len(name) == 0 {
		name = "default@internal"
	}

	r.rtLock.RLock()
	defer r.rtLock.RUnlock()

	if rt, ok := r.proxies[name]; ok {
		return rt, nil
	}

	return nil, fmt.Errorf("servers transport not found %s", name)
}

// createRoundTripper creates an http.RoundTripper configured with the Transport configuration settings.
// For the settings that can't be configured in Traefik it uses the default http.Transport settings.
// An exception to this is the MaxIdleConns setting as we only provide the option MaxIdleConnsPerHost in Traefik at this point in time.
// Setting this value to the default of 100 could lead to confusing behavior and backwards compatibility issues.
func createRoundTripper(cfg *dynamic.ServersTransport) (http.RoundTripper, error) {
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
	}

	if cfg.ForwardingTimeouts != nil {
		transport.ResponseHeaderTimeout = time.Duration(cfg.ForwardingTimeouts.ResponseHeaderTimeout)
		transport.IdleConnTimeout = time.Duration(cfg.ForwardingTimeouts.IdleConnTimeout)
	}

	var err error
	transport.TLSClientConfig, err = createTLSConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Return directly HTTP/1.1 transport when HTTP/2 is disabled
	if cfg.DisableHTTP2 {
		return transport, nil
	}

	return newSmartRoundTripper(transport, cfg.ForwardingTimeouts)
}

func createTLSConfig(cfg *dynamic.ServersTransport) (*tls.Config, error) {
	var tlsConfig *tls.Config
	if cfg.InsecureSkipVerify || len(cfg.RootCAs) > 0 || len(cfg.ServerName) > 0 || len(cfg.Certificates) > 0 || cfg.PeerCertURI != "" {
		tlsConfig = &tls.Config{
			ServerName:         cfg.ServerName,
			InsecureSkipVerify: cfg.InsecureSkipVerify,
			RootCAs:            createRootCACertPool(cfg.RootCAs),
			Certificates:       cfg.Certificates.GetCertificates(),
		}

		if cfg.PeerCertURI != "" {
			tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
				return traefiktls.VerifyPeerCertificate(cfg.PeerCertURI, tlsConfig, rawCerts)
			}
		}
	}
	return tlsConfig, nil
}

func createRootCACertPool(rootCAs []traefiktls.FileOrContent) *x509.CertPool {
	if len(rootCAs) == 0 {
		return nil
	}

	roots := x509.NewCertPool()

	for _, cert := range rootCAs {
		certContent, err := cert.Read()
		if err != nil {
			log.WithoutContext().Error("Error while read RootCAs", err)
			continue
		}
		roots.AppendCertsFromPEM(certContent)
	}

	return roots
}

type FastHTTPTransport struct {
	hc     *HostChooser
	brPool Pool[*bufio.Reader]
	wrPool Pool[*bufio.Writer]
	lrPool Pool[*io.LimitedReader]
}

func (f *FastHTTPTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	pool := f.hc.GetPool(request.URL.Scheme, request.URL.Host)
	conn := pool.AcquireConn()

	outReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(outReq)

	if request.Body != nil {
		defer request.Body.Close()
	}

	outReq.Header.Set("FastHTTP", "enabled")

	// SetRequestURI must be called before outReq.SetHost because it re-triggers uri parsing.
	outReq.SetRequestURI(request.URL.RequestURI())

	outReq.SetHost(request.URL.Host)
	outReq.Header.SetHost(request.Host)

	for k, v := range request.Header {
		for _, s := range v {
			outReq.Header.Add(k, s)
		}
	}

	outReq.SetBodyStream(request.Body, int(request.ContentLength))

	outReq.Header.SetMethod(request.Method)

	bw := f.wrPool.Get()
	if bw == nil {
		bw = bufio.NewWriterSize(conn, 4096)
	}

	defer f.wrPool.Put(bw)

	bw.Reset(conn)
	err := outReq.Write(bw)
	bw.Flush()
	if err != nil {
		return nil, err
	}

	br := f.brPool.Get()
	if br == nil {
		br = bufio.NewReaderSize(conn, 4096)
	}

	br.Reset(conn)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	res.Header.Read(br)

	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.StatusCode = res.StatusCode()
	res.Header.VisitAll(func(key, value []byte) {
		resp.Header.Add(string(key), string(value))
	})

	brl := f.lrPool.Get()
	if brl == nil {
		brl = &io.LimitedReader{}
	}

	brl.R = br
	brl.N = int64(res.Header.ContentLength())

	resp.Body = io.NopCloser(brl)

	return resp, nil
}
