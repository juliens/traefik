// Package host is the API that Traefik gives to its plugins, beside the configuration types.
// It runs natively in the plugins (see pkg/plugins/hostsyms): a plugin reads the state that Traefik sets here,
// and two plugins share the values they register.
package host

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"sync"
)

// EntryPoint describes an entry point of the static configuration.
type EntryPoint struct {
	// Address is the address the entry point listens on.
	Address string
	// HasHTTPTLSConf is true when the entry point has a default TLS configuration for its routers.
	HasHTTPTLSConf bool
	// AsDefault is true when the entry point is a default entry point of the routers.
	AsDefault bool
	// AllowACMEByPass is true when the routers of the configuration can handle the ACME challenges of the entry point.
	AllowACMEByPass bool
}

var (
	mu                sync.RWMutex
	entryPoints       map[string]EntryPoint
	defaultRuleSyntax string
	values            = map[string]any{}
	certDomains       func(store string) []string
)

// SetStatic sets the information of the static configuration given to the plugins.
// Traefik calls it before it loads the plugins.
func SetStatic(eps map[string]EntryPoint, ruleSyntax string) {
	mu.Lock()
	defer mu.Unlock()

	entryPoints = maps.Clone(eps)
	defaultRuleSyntax = ruleSyntax
}

// EntryPoints returns the entry points of the static configuration.
func EntryPoints() map[string]EntryPoint {
	mu.RLock()
	defer mu.RUnlock()

	return maps.Clone(entryPoints)
}

// DefaultRuleSyntax returns the default rule syntax of the static configuration.
func DefaultRuleSyntax() string {
	mu.RLock()
	defer mu.RUnlock()

	return defaultRuleSyntax
}

// Register registers a value that other plugins can look up.
// The value should be plain data: the plugins run in different machines and do not share their types.
func Register(key string, value any) {
	mu.Lock()
	defer mu.Unlock()

	values[key] = value
}

// Unregister removes the value registered with the given key.
func Unregister(key string) {
	mu.Lock()
	defer mu.Unlock()

	delete(values, key)
}

// SetCertificateDomains sets the function giving the domains of the certificates of a TLS store.
// Traefik calls it before it loads the plugins.
func SetCertificateDomains(domains func(store string) []string) {
	mu.Lock()
	defer mu.Unlock()

	certDomains = domains
}

// CertificateDomains returns the domains of the certificates of the given TLS store,
// as comma-separated lists of the domains of each certificate.
func CertificateDomains(store string) []string {
	mu.RLock()
	domains := certDomains
	mu.RUnlock()

	if domains == nil {
		return nil
	}

	return domains(store)
}

// Lookup returns the value registered with the given key.
func Lookup(key string) (any, bool) {
	mu.RLock()
	defer mu.RUnlock()

	v, ok := values[key]
	return v, ok
}

// NotifySignals returns a channel that receives the given signals until the context is done.
// go-vm cannot give the channel of a plugin to signal.Notify, so Traefik makes the channel.
func NotifySignals(ctx context.Context, sig ...os.Signal) <-chan os.Signal {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, sig...)

	go func() {
		<-ctx.Done()
		signal.Stop(signals)
	}()

	return signals
}

type serviceBuilderKey struct{}

// WithServiceBuilder returns a context giving the services of the configuration to the middleware plugins (see BuildService).
// Traefik calls it before it creates a middleware.
func WithServiceBuilder(ctx context.Context, build func(ctx context.Context, serviceName string) (http.Handler, error)) context.Context {
	return context.WithValue(ctx, serviceBuilderKey{}, build)
}

// BuildService builds the HTTP handler of a service of the configuration, from the context given to the New function of a middleware plugin.
// A middleware sends the requests to another service with it, as the errors middleware does to its error pages.
func BuildService(ctx context.Context, serviceName string) (http.Handler, error) {
	build, ok := ctx.Value(serviceBuilderKey{}).(func(ctx context.Context, serviceName string) (http.Handler, error))
	if !ok {
		return nil, errors.New("the services are not available in this context")
	}

	return build(ctx, serviceName)
}
