package static

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	ptypes "github.com/traefik/paerser/types"
	otypes "github.com/traefik/traefik/v3/pkg/observability/types"
	"github.com/traefik/traefik/v3/pkg/ping"
	"github.com/traefik/traefik/v3/pkg/tls"
	"github.com/traefik/traefik/v3/pkg/types"
)

const (
	// DefaultInternalEntryPointName the name of the default internal entry point.
	DefaultInternalEntryPointName = "traefik"

	// DefaultGraceTimeout controls how long Traefik serves pending requests
	// prior to shutting down.
	DefaultGraceTimeout = 10 * time.Second

	// DefaultIdleTimeout before closing an idle connection.
	DefaultIdleTimeout = 180 * time.Second

	// DefaultReadTimeout defines the default maximum duration for reading the entire request, including the body.
	DefaultReadTimeout = 60 * time.Second

	// DefaultUDPTimeout defines how long to wait by default on an idle session,
	// before releasing all resources related to that session.
	DefaultUDPTimeout = 3 * time.Second
)

// providerNames is the default routing precedence between the providers: the names of the providers of Traefik,
// which are now plugins that keep their name.
var providerNames = []string{
	"kubernetesgateway",
	"kubernetescrd",
	"kubernetes",
	"kubernetesingressnginx",
	"swarm",
	"docker",
	"file",
	"redis",
	"knative",
	"consul",
	"consulcatalog",
	"nomad",
	"etcd",
	"ecs",
	"http",
	"zookeeper",
	"rest",
}

// Allowed characters in URL following RFC 3986 (https://www.rfc-editor.org/rfc/rfc3986#section-2)
var validBasePath = regexp.MustCompile(`^/[a-zA-Z0-9/_.:~-]*$`)

// Configuration is the static configuration.
type Configuration struct {
	Global *Global `description:"Global configuration options" json:"global,omitempty" toml:"global,omitempty" yaml:"global,omitempty" export:"true"`

	ServersTransport    *ServersTransport    `description:"Servers default transport." json:"serversTransport,omitempty" toml:"serversTransport,omitempty" yaml:"serversTransport,omitempty" export:"true"`
	TCPServersTransport *TCPServersTransport `description:"TCP servers default transport." json:"tcpServersTransport,omitempty" toml:"tcpServersTransport,omitempty" yaml:"tcpServersTransport,omitempty" export:"true"`
	EntryPoints         EntryPoints          `description:"Entry points definition." json:"entryPoints,omitempty" toml:"entryPoints,omitempty" yaml:"entryPoints,omitempty" export:"true"`
	Providers           *Providers           `description:"Providers configuration." json:"providers,omitempty" toml:"providers,omitempty" yaml:"providers,omitempty" export:"true"`

	API     *API            `description:"Enable api/dashboard." json:"api,omitempty" toml:"api,omitempty" yaml:"api,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	Metrics *otypes.Metrics `description:"Enable a metrics exporter." json:"metrics,omitempty" toml:"metrics,omitempty" yaml:"metrics,omitempty" export:"true"`
	Ping    *ping.Handler   `description:"Enable ping." json:"ping,omitempty" toml:"ping,omitempty" yaml:"ping,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`

	Log       *otypes.TraefikLog `description:"Traefik log settings." json:"log,omitempty" toml:"log,omitempty" yaml:"log,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	AccessLog *otypes.AccessLog  `description:"Access log settings." json:"accessLog,omitempty" toml:"accessLog,omitempty" yaml:"accessLog,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	Tracing   *Tracing           `description:"Tracing configuration." json:"tracing,omitempty" toml:"tracing,omitempty" yaml:"tracing,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`

	HostResolver *types.HostResolverConfig `description:"Enable CNAME Flattening." json:"hostResolver,omitempty" toml:"hostResolver,omitempty" yaml:"hostResolver,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`

	CertificatesResolvers map[string]CertificateResolver `description:"Certificates resolvers configuration." json:"certificatesResolvers,omitempty" toml:"certificatesResolvers,omitempty" yaml:"certificatesResolvers,omitempty" export:"true"`

	Experimental *Experimental `description:"Experimental features." json:"experimental,omitempty" toml:"experimental,omitempty" yaml:"experimental,omitempty" export:"true"`

	Core *Core `description:"Core controls." json:"core,omitempty" toml:"core,omitempty" yaml:"core,omitempty" export:"true"`

	Spiffe *SpiffeClientConfig `description:"SPIFFE integration configuration." json:"spiffe,omitempty" toml:"spiffe,omitempty" yaml:"spiffe,omitempty" export:"true"`

	OCSP *tls.OCSPConfig `description:"OCSP configuration." json:"ocsp,omitempty" toml:"ocsp,omitempty" yaml:"ocsp,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// Core configures Traefik core behavior.
type Core struct {
	// Deprecated: Please do not use this field and rewrite the router rules to use the v3 syntax.
	DefaultRuleSyntax string `description:"Defines the rule parser default syntax (v2 or v3)" json:"defaultRuleSyntax,omitempty" toml:"defaultRuleSyntax,omitempty" yaml:"defaultRuleSyntax,omitempty"`

	StrictTLSOptions bool `description:"Disables the unsafe fallback to the default TLS options for the routers with conflicting TLS options." json:"strictTLSOptions,omitempty" toml:"strictTLSOptions,omitempty" yaml:"strictTLSOptions,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (c *Core) SetDefaults() {
	c.DefaultRuleSyntax = "v3"
}

// SpiffeClientConfig defines the SPIFFE client configuration.
type SpiffeClientConfig struct {
	WorkloadAPIAddr string `description:"Defines the workload API address." json:"workloadAPIAddr,omitempty" toml:"workloadAPIAddr,omitempty" yaml:"workloadAPIAddr,omitempty"`
}

// CertificateResolver contains the configuration for the different types of certificates resolver.
// The ACME resolvers are provider plugins: a router uses the name of the plugin as certificate resolver.
type CertificateResolver struct {
	Tailscale *struct{} `description:"Enables Tailscale certificate resolution." json:"tailscale,omitempty" toml:"tailscale,omitempty" yaml:"tailscale,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// Global holds the global configuration.
type Global struct {
	CheckNewVersion        bool `description:"Periodically check if a new version has been released." json:"checkNewVersion,omitempty" toml:"checkNewVersion,omitempty" yaml:"checkNewVersion,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	SendAnonymousUsage     bool `description:"Periodically send anonymous usage statistics. If the option is not specified, it will be disabled by default." json:"sendAnonymousUsage,omitempty" toml:"sendAnonymousUsage,omitempty" yaml:"sendAnonymousUsage,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	NotAppendXForwardedFor bool `description:"Disable appending RemoteAddr to X-Forwarded-For header. Defaults to false (appending is enabled)." json:"notAppendXForwardedFor,omitempty" toml:"notAppendXForwardedFor,omitempty" yaml:"notAppendXForwardedFor,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// ServersTransport options to configure communication between Traefik and the servers.
type ServersTransport struct {
	InsecureSkipVerify  bool                  `description:"Disable SSL certificate verification." json:"insecureSkipVerify,omitempty" toml:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty" export:"true"`
	RootCAs             []types.FileOrContent `description:"Add cert file for self-signed certificate." json:"rootCAs,omitempty" toml:"rootCAs,omitempty" yaml:"rootCAs,omitempty"`
	MaxIdleConnsPerHost int                   `description:"If non-zero, controls the maximum idle (keep-alive) to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used. If negative, disables connection reuse." json:"maxIdleConnsPerHost,omitempty" toml:"maxIdleConnsPerHost,omitempty" yaml:"maxIdleConnsPerHost,omitempty" export:"true"`
	ForwardingTimeouts  *ForwardingTimeouts   `description:"Timeouts for requests forwarded to the backend servers." json:"forwardingTimeouts,omitempty" toml:"forwardingTimeouts,omitempty" yaml:"forwardingTimeouts,omitempty" export:"true"`
	Spiffe              *Spiffe               `description:"Defines the SPIFFE configuration." json:"spiffe,omitempty" toml:"spiffe,omitempty" yaml:"spiffe,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// Spiffe holds the SPIFFE configuration.
type Spiffe struct {
	IDs         []string `description:"Defines the allowed SPIFFE IDs (takes precedence over the SPIFFE TrustDomain)." json:"ids,omitempty" toml:"ids,omitempty" yaml:"ids,omitempty"`
	TrustDomain string   `description:"Defines the allowed SPIFFE trust domain." json:"trustDomain,omitempty" toml:"trustDomain,omitempty" yaml:"trustDomain,omitempty"`
}

// TCPServersTransport options to configure communication between Traefik and the servers.
type TCPServersTransport struct {
	DialKeepAlive ptypes.Duration `description:"Defines the interval between keep-alive probes for an active network connection. If zero, keep-alive probes are sent with a default value (currently 15 seconds), if supported by the protocol and operating system. Network protocols or operating systems that do not support keep-alives ignore this field. If negative, keep-alive probes are disabled" json:"dialKeepAlive,omitempty" toml:"dialKeepAlive,omitempty" yaml:"dialKeepAlive,omitempty" export:"true"`
	DialTimeout   ptypes.Duration `description:"Defines the amount of time to wait until a connection to a backend server can be established. If zero, no timeout exists." json:"dialTimeout,omitempty" toml:"dialTimeout,omitempty" yaml:"dialTimeout,omitempty" export:"true"`
	// TerminationDelay, corresponds to the deadline that the proxy sets, after one
	// of its connected peers indicates it has closed the writing capability of its
	// connection, to close the reading capability as well, hence fully terminating the
	// connection. It is a duration in milliseconds, defaulting to 100. A negative value
	// means an infinite deadline (i.e. the reading capability is never closed).
	TerminationDelay ptypes.Duration  `description:"Defines the delay to wait before fully terminating the connection, after one connected peer has closed its writing capability." json:"terminationDelay,omitempty" toml:"terminationDelay,omitempty" yaml:"terminationDelay,omitempty" export:"true"`
	TLS              *TLSClientConfig `description:"Defines the TLS configuration." json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" label:"allowEmpty" file:"allowEmpty" kv:"allowEmpty" export:"true"`
}

// TLSClientConfig options to configure TLS communication between Traefik and the servers.
type TLSClientConfig struct {
	InsecureSkipVerify bool                  `description:"Disables SSL certificate verification." json:"insecureSkipVerify,omitempty" toml:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty" export:"true"`
	RootCAs            []types.FileOrContent `description:"Defines a list of CA secret used to validate self-signed certificate" json:"rootCAs,omitempty" toml:"rootCAs,omitempty" yaml:"rootCAs,omitempty"`
	Spiffe             *Spiffe               `description:"Defines the SPIFFE TLS configuration." json:"spiffe,omitempty" toml:"spiffe,omitempty" yaml:"spiffe,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// API holds the API configuration.
type API struct {
	BasePath           string `description:"Defines the base path where the API and Dashboard will be exposed." json:"basePath,omitempty" toml:"basePath,omitempty" yaml:"basePath,omitempty" export:"true"`
	Insecure           bool   `description:"Activate API directly on the entryPoint named traefik." json:"insecure,omitempty" toml:"insecure,omitempty" yaml:"insecure,omitempty" export:"true"`
	Dashboard          bool   `description:"Activate dashboard." json:"dashboard,omitempty" toml:"dashboard,omitempty" yaml:"dashboard,omitempty" export:"true"`
	Debug              bool   `description:"Enable additional endpoints for debugging and profiling." json:"debug,omitempty" toml:"debug,omitempty" yaml:"debug,omitempty" export:"true"`
	DisableDashboardAd bool   `description:"Disable ad in the dashboard." json:"disableDashboardAd,omitempty" toml:"disableDashboardAd,omitempty" yaml:"disableDashboardAd,omitempty" export:"true"`
	DashboardName      string `description:"Custom name for the dashboard." json:"dashboardName,omitempty" toml:"dashboardName,omitempty" yaml:"dashboardName,omitempty" export:"true"`
	// TODO: Re-enable statistics
	// Statistics      *types.Statistics `description:"Enable more detailed statistics." json:"statistics,omitempty" toml:"statistics,omitempty" yaml:"statistics,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// SetDefaults sets the default values.
func (a *API) SetDefaults() {
	a.BasePath = "/"
	a.Dashboard = true
	a.DashboardName = ""
}

// RespondingTimeouts contains timeout configurations for incoming requests to the Traefik instance.
type RespondingTimeouts struct {
	ReadTimeout  ptypes.Duration `description:"ReadTimeout is the maximum duration for reading the entire request, including the body. If zero, no timeout is set." json:"readTimeout,omitempty" toml:"readTimeout,omitempty" yaml:"readTimeout,omitempty" export:"true"`
	WriteTimeout ptypes.Duration `description:"WriteTimeout is the maximum duration before timing out writes of the response. If zero, no timeout is set." json:"writeTimeout,omitempty" toml:"writeTimeout,omitempty" yaml:"writeTimeout,omitempty" export:"true"`
	IdleTimeout  ptypes.Duration `description:"IdleTimeout is the maximum amount duration an idle (keep-alive) connection will remain idle before closing itself. If zero, no timeout is set." json:"idleTimeout,omitempty" toml:"idleTimeout,omitempty" yaml:"idleTimeout,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (a *RespondingTimeouts) SetDefaults() {
	a.ReadTimeout = ptypes.Duration(DefaultReadTimeout)
	a.IdleTimeout = ptypes.Duration(DefaultIdleTimeout)
}

// ForwardingTimeouts contains timeout configurations for forwarding requests to the backend servers.
type ForwardingTimeouts struct {
	DialTimeout           ptypes.Duration `description:"The amount of time to wait until a connection to a backend server can be established. If zero, no timeout exists." json:"dialTimeout,omitempty" toml:"dialTimeout,omitempty" yaml:"dialTimeout,omitempty" export:"true"`
	ResponseHeaderTimeout ptypes.Duration `description:"The amount of time to wait for a server's response headers after fully writing the request (including its body, if any). If zero, no timeout exists." json:"responseHeaderTimeout,omitempty" toml:"responseHeaderTimeout,omitempty" yaml:"responseHeaderTimeout,omitempty" export:"true"`
	IdleConnTimeout       ptypes.Duration `description:"The maximum period for which an idle HTTP keep-alive connection will remain open before closing itself" json:"idleConnTimeout,omitempty" toml:"idleConnTimeout,omitempty" yaml:"idleConnTimeout,omitempty" export:"true"`
	ReadTimeout           ptypes.Duration `description:"Defines a timeout for reading a response from the proxied server. The timeout between two successive read operations. If zero, no timeout exists." json:"readTimeout,omitempty" toml:"readTimeout,omitempty" yaml:"readTimeout,omitempty" export:"true"`
	WriteTimeout          ptypes.Duration `description:"Defines a timeout for transmitting a request to the proxied server. The timeout between two successive write operations. If zero, no timeout exists." json:"writeTimeout,omitempty" toml:"writeTimeout,omitempty" yaml:"writeTimeout,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (f *ForwardingTimeouts) SetDefaults() {
	f.DialTimeout = ptypes.Duration(30 * time.Second)
	f.IdleConnTimeout = ptypes.Duration(90 * time.Second)
}

// LifeCycle contains configurations relevant to the lifecycle (such as the shutdown phase) of Traefik.
type LifeCycle struct {
	RequestAcceptGraceTimeout ptypes.Duration `description:"Duration to keep accepting requests before Traefik initiates the graceful shutdown procedure." json:"requestAcceptGraceTimeout,omitempty" toml:"requestAcceptGraceTimeout,omitempty" yaml:"requestAcceptGraceTimeout,omitempty" export:"true"`
	GraceTimeOut              ptypes.Duration `description:"Duration to give active requests a chance to finish before Traefik stops." json:"graceTimeOut,omitempty" toml:"graceTimeOut,omitempty" yaml:"graceTimeOut,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (a *LifeCycle) SetDefaults() {
	a.GraceTimeOut = ptypes.Duration(DefaultGraceTimeout)
}

// Tracing holds the tracing configuration.
// The type is in pkg/observability/types: the tracing helpers, which the middleware plugins use, do not need the static configuration.
type Tracing = otypes.Tracing

// Providers contains providers configuration.
type Providers struct {
	ProvidersThrottleDuration ptypes.Duration `description:"Backends throttle duration: minimum duration between 2 events from providers before applying a new configuration. It avoids unnecessary reloads if multiples events are sent in a short amount of time." json:"providersThrottleDuration,omitempty" toml:"providersThrottleDuration,omitempty" yaml:"providersThrottleDuration,omitempty" export:"true"`
	Precedence                []string        `description:"Defines the routing precedence between providers." json:"precedence,omitempty" toml:"precedence,omitempty" yaml:"precedence,omitempty" export:"true"`

	// Plugin holds the provider plugins: the providers of Traefik (file, docker, kubernetesCRD, ...) are plugins.
	Plugin map[string]PluginConf `description:"Plugins configuration." json:"plugin,omitempty" toml:"plugin,omitempty" yaml:"plugin,omitempty"`
}

// SetDefaults sets the default values.
func (p *Providers) SetDefaults() {
	p.Precedence = providerNames
}

// SetEffectiveConfiguration adds missing configuration parameters derived from existing ones.
// It also takes care of maintaining backwards compatibility.
func (c *Configuration) SetEffectiveConfiguration() {
	// Creates the default entry point if needed
	if !c.hasUserDefinedEntrypoint() {
		ep := &EntryPoint{Address: ":80"}
		ep.SetDefaults()
		// TODO: double check this tomorrow
		if c.EntryPoints == nil {
			c.EntryPoints = make(EntryPoints)
		}
		c.EntryPoints["http"] = ep
	}

	// Creates the internal traefik entry point if needed
	if (c.API != nil && c.API.Insecure) ||
		(c.Ping != nil && !c.Ping.ManualRouting && c.Ping.EntryPoint == DefaultInternalEntryPointName) ||
		(c.Metrics != nil && c.Metrics.Prometheus != nil && !c.Metrics.Prometheus.ManualRouting && c.Metrics.Prometheus.EntryPoint == DefaultInternalEntryPointName) {
		if _, ok := c.EntryPoints[DefaultInternalEntryPointName]; !ok {
			ep := &EntryPoint{Address: ":8080"}
			ep.SetDefaults()
			c.EntryPoints[DefaultInternalEntryPointName] = ep
		}
	}

	// If the entry point exists without an address, which happens when only its options are configured,
	// this ensures the address defaults to :8080.
	if ep, ok := c.EntryPoints[DefaultInternalEntryPointName]; ok && ep.Address == "" {
		ep.Address = ":8080"
	}

	if c.Tracing != nil && c.Tracing.GlobalAttributes != nil && c.Tracing.ResourceAttributes == nil {
		c.Tracing.ResourceAttributes = c.Tracing.GlobalAttributes
	}

	for i, providerName := range c.Providers.Precedence {
		c.Providers.Precedence[i] = strings.ToLower(providerName)
	}
}

// ValidateConfiguration validate that configuration is coherent.
func (c *Configuration) ValidateConfiguration() error {
	if c.Core != nil {
		switch c.Core.DefaultRuleSyntax {
		case "v3": // NOOP
		case "v2":
			// TODO: point to migration guide.
			log.Warn().Msgf("v2 rules syntax is now deprecated, please use v3 instead...")
		default:
			return fmt.Errorf("unsupported default rule syntax configuration: %q", c.Core.DefaultRuleSyntax)
		}
	}

	for epName, ep := range c.EntryPoints {
		if ep.HTTP.UnderscoreHeadersStrategy != "" && ep.HTTP.AliasHeadersStrategy != "" &&
			ep.HTTP.AliasHeadersStrategy != ep.HTTP.UnderscoreHeadersStrategy {
			return fmt.Errorf("entry point %q cannot have both underscoreHeadersStrategy and aliasHeadersStrategy options configured with different values", epName)
		}
	}

	if c.Providers != nil {
		for _, providerName := range c.Providers.Precedence {
			if providerName == "" {
				return fmt.Errorf("provider %q is not a valid provider name", providerName)
			}
		}
	}

	if c.AccessLog != nil && c.AccessLog.OTLP != nil {
		if c.Experimental == nil || !c.Experimental.OTLPLogs {
			return errors.New("the experimental OTLPLogs feature must be enabled to use OTLP access logging")
		}

		if c.AccessLog.OTLP.GRPC != nil && c.AccessLog.OTLP.GRPC.TLS != nil && c.AccessLog.OTLP.GRPC.Insecure {
			return errors.New("access logs OTLP GRPC: TLS and Insecure options are mutually exclusive")
		}
	}

	if c.Log != nil && c.Log.OTLP != nil {
		if c.Experimental == nil || !c.Experimental.OTLPLogs {
			return errors.New("the experimental OTLPLogs feature must be enabled to use OTLP logging")
		}

		if c.Log.OTLP.GRPC != nil && c.Log.OTLP.GRPC.TLS != nil && c.Log.OTLP.GRPC.Insecure {
			return errors.New("logs OTLP GRPC: TLS and Insecure options are mutually exclusive")
		}
	}

	if c.Tracing != nil && c.Tracing.OTLP != nil {
		if c.Tracing.OTLP.GRPC != nil && c.Tracing.OTLP.GRPC.TLS != nil && c.Tracing.OTLP.GRPC.Insecure {
			return errors.New("tracing OTLP GRPC: TLS and Insecure options are mutually exclusive")
		}
	}

	if c.Metrics != nil && c.Metrics.OTLP != nil {
		if c.Metrics.OTLP.GRPC != nil && c.Metrics.OTLP.GRPC.TLS != nil && c.Metrics.OTLP.GRPC.Insecure {
			return errors.New("metrics OTLP GRPC: TLS and Insecure options are mutually exclusive")
		}
	}

	if c.API != nil && !validBasePath.MatchString(c.API.BasePath) {
		return errors.New("API basePath must be a valid absolute URL path")
	}

	if c.OCSP != nil {
		for responderURL, url := range c.OCSP.ResponderOverrides {
			if url == "" {
				return fmt.Errorf("OCSP responder override value for %s cannot be empty", responderURL)
			}
		}
	}

	return nil
}

const protocolTCP = "tcp"

// HasTCPEntryPoint reports whether at least one entryPoint carries TCP traffic,
// and therefore can serve HTTP requests.
func (c *Configuration) HasTCPEntryPoint() bool {
	for _, ep := range c.EntryPoints {
		protocol, err := ep.GetProtocol()
		if err == nil && protocol == protocolTCP {
			return true
		}
	}

	return false
}

// HasDeniedEncodedCharacters reports whether an entryPoint serving HTTP requests
// disallows at least one encoded character in the request path.
// UDP entryPoints are left out because they never parse a request path.
func (c *Configuration) HasDeniedEncodedCharacters() bool {
	for _, ep := range c.EntryPoints {
		protocol, err := ep.GetProtocol()
		if err != nil || protocol != protocolTCP {
			continue
		}

		encodedCharacters := ep.HTTP.EncodedCharacters
		if encodedCharacters == nil {
			continue
		}

		if !encodedCharacters.AllowEncodedSlash ||
			!encodedCharacters.AllowEncodedBackSlash ||
			!encodedCharacters.AllowEncodedNullCharacter ||
			!encodedCharacters.AllowEncodedSemicolon ||
			!encodedCharacters.AllowEncodedPercent ||
			!encodedCharacters.AllowEncodedQuestionMark ||
			!encodedCharacters.AllowEncodedHash {
			return true
		}
	}

	return false
}

func (c *Configuration) hasUserDefinedEntrypoint() bool {
	return len(c.EntryPoints) != 0
}
