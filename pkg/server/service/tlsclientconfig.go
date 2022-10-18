package service

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"sync"

	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	traefiktls "github.com/traefik/traefik/v2/pkg/tls"
)

func NewTLSConfigManager(spiffeX509Source SpiffeX509Source) *TLSConfigManager {
	return &TLSConfigManager{
		configs:          make(map[string]*dynamic.ServersTransport),
		spiffeX509Source: spiffeX509Source,
	}
}

type TLSConfigManager struct {
	configsLock      sync.RWMutex
	configs          map[string]*dynamic.ServersTransport
	spiffeX509Source SpiffeX509Source
}

func (r *TLSConfigManager) Update(configs map[string]*dynamic.ServersTransport {
	r.configsLock.Lock()
	defer r.configsLock.Unlock()

	r.configs = configs
}

func (r *TLSConfigManager) GetTLSConfig(name string) (*tls.Config, error) {
	if len(name) == 0 {
		name = "default@internal"
	}

	r.configsLock.RLock()
	defer r.configsLock.RUnlock()

	config, ok := r.configs[name]

	if ok {
		return r.createTLSConfig(config)
	}

	return nil, fmt.Errorf("unable to find tls config for serversTransport: %s", name)
}

func (r *TLSConfigManager) createTLSConfig(cfg *dynamic.ServersTransport) (*tls.Config, error) {
	var tlsConfig *tls.Config

	if cfg.Spiffe != nil {
		if r.spiffeX509Source == nil {
			return nil, errors.New("SPIFFE is enabled for this transport, but not configured")
		}

		spiffeAuthorizer, err := buildSpiffeAuthorizer(cfg.Spiffe)
		if err != nil {
			return nil, fmt.Errorf("unable to build SPIFFE authorizer: %w", err)
		}

		tlsConfig = tlsconfig.MTLSClientConfig(r.spiffeX509Source, r.spiffeX509Source, spiffeAuthorizer)
	}

	if cfg.InsecureSkipVerify || len(cfg.RootCAs) > 0 || len(cfg.ServerName) > 0 || len(cfg.Certificates) > 0 || cfg.PeerCertURI != "" {
		if tlsConfig != nil {
			return nil, errors.New("TLS and SPIFFE configuration cannot be defined at the same time")
		}

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
