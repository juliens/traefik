package tcp

import (
	"crypto/tls"
	"net/http"

	"github.com/traefik/traefik/v2/pkg/log"
	tcpmuxer "github.com/traefik/traefik/v2/pkg/muxer/tcp"
	"github.com/traefik/traefik/v2/pkg/tcp"
)

const defaultBufSize = 4096

// Router is a TCP router.
type Router struct {
	// Contains TCP routes.
	muxerTCP tcpmuxer.Muxer

	// Forwarder handlers.
	// Handles all HTTP requests.
	httpForwarder tcp.Handler
	// Handles (indirectly through muxerHTTPS, or directly) all HTTPS requests.
	httpsForwarder tcp.Handler

	// Neither is used directly, but they are held here, and recreated on config
	// reload, so that they can be passed to the Switcher at the end of the config
	// reload phase.
	httpHandler  http.Handler
	httpsHandler http.Handler

	// TLS configs.
	httpsTLSConfig    *tls.Config            // default TLS config
	hostHTTPTLSConfig map[string]*tls.Config // TLS configs keyed by SNI
}

// NewRouter returns a new TCP router.
func NewRouter() (*Router, error) {
	muxTCP, err := tcpmuxer.NewMuxer()
	if err != nil {
		return nil, err
	}

	return &Router{
		muxerTCP: *muxTCP,
	}, nil
}

// GetTLSGetClientInfo is called after a ClientHello is received from a client.
func (r *Router) GetTLSGetClientInfo() func(info *tls.ClientHelloInfo) (*tls.Config, error) {
	return func(info *tls.ClientHelloInfo) (*tls.Config, error) {
		if tlsConfig, ok := r.hostHTTPTLSConfig[info.ServerName]; ok {
			return tlsConfig, nil
		}

		return r.httpsTLSConfig, nil
	}
}

// ServeTCP forwards the connection to the right TCP/HTTP handler.
func (r *Router) ServeTCP(conn tcp.WriteCloser) {
	r.muxerTCP.NotFound = tcp.HandlerFunc(func(conn tcp.WriteCloser) {
		eConn := &tcpmuxer.EnrichConn{WriteCloser: conn}
		if eConn.IsTLS() {
			r.httpsForwarder.ServeTCP(eConn)
			return
		}
		r.httpForwarder.ServeTCP(eConn)
	})
	r.muxerTCP.ServeTCP(conn)
}

// AddRoute defines a handler for the given rule.
func (r *Router) AddRoute(rule string, priority int, target tcp.Handler) error {
	return r.muxerTCP.AddRoute(rule, priority, target, false)
}

// AddRouteTLS defines a handler for a given rule and sets the matching tlsConfig.
func (r *Router) AddRouteTLS(rule string, priority int, target tcp.Handler, config *tls.Config) error {
	// TLS PassThrough
	if config == nil {
		return r.muxerTCP.AddRoute(rule, priority, target, true)
	}

	return r.muxerTCP.AddRoute(rule, priority, &tcp.TLSHandler{
		Next:   target,
		Config: config,
	}, true)
}

// AddHTTPTLSConfig defines a handler for a given sniHost and sets the matching tlsConfig.
func (r *Router) AddHTTPTLSConfig(sniHost string, config *tls.Config) {
	if r.hostHTTPTLSConfig == nil {
		r.hostHTTPTLSConfig = map[string]*tls.Config{}
	}

	r.hostHTTPTLSConfig[sniHost] = config
}

// GetHTTPHandler gets the attached http handler.
func (r *Router) GetHTTPHandler() http.Handler {
	return r.httpHandler
}

// GetHTTPSHandler gets the attached https handler.
func (r *Router) GetHTTPSHandler() http.Handler {
	return r.httpsHandler
}

// SetHTTPForwarder sets the tcp handler that will forward the connections to an http handler.
func (r *Router) SetHTTPForwarder(handler tcp.Handler) {
	r.httpForwarder = handler
}

// SetHTTPSForwarder sets the tcp handler that will forward the TLS connections to an http handler.
func (r *Router) SetHTTPSForwarder(handler tcp.Handler) {
	for sniHost, tlsConf := range r.hostHTTPTLSConfig {
		// muxerHTTPS only contains single HostSNI rules (and no other kind of rules),
		// so there's no need for specifying a priority for them.
		err := r.muxerTCP.AddRoute("HostSNI(`"+sniHost+"`)", 0, &tcp.TLSHandler{
			Next:   handler,
			Config: tlsConf,
		}, true)
		if err != nil {
			log.WithoutContext().Errorf("Error while adding route for host: %w", err)
		}
	}

	r.httpsForwarder = &tcp.TLSHandler{
		Next:   handler,
		Config: r.httpsTLSConfig,
	}
}

// SetHTTPHandler attaches http handlers on the router.
func (r *Router) SetHTTPHandler(handler http.Handler) {
	r.httpHandler = handler
}

// SetHTTPSHandler attaches https handlers on the router.
func (r *Router) SetHTTPSHandler(handler http.Handler, config *tls.Config) {
	r.httpsHandler = handler
	r.httpsTLSConfig = config
}
