package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// InternalHandlers is the internal HTTP handlers builder.
type InternalHandlers struct {
	api        http.Handler
	dashboard  http.Handler
	prometheus http.Handler
	ping       http.Handler
	plugins    map[string]http.Handler
}

// NewInternalHandlers creates a new InternalHandlers.
// The plugins are the handlers of the provider plugins, by plugin name: each one serves the service <name>@internal.
func NewInternalHandlers(apiHandler, metricsHandler, pingHandler, dashboard http.Handler, plugins map[string]http.Handler) *InternalHandlers {
	return &InternalHandlers{
		api:        apiHandler,
		dashboard:  dashboard,
		prometheus: metricsHandler,
		ping:       pingHandler,
		plugins:    plugins,
	}
}

// BuildHTTP builds an HTTP handler.
func (m *InternalHandlers) BuildHTTP(rootCtx context.Context, serviceName string) (http.Handler, error) {
	if !strings.HasSuffix(serviceName, "@internal") {
		return nil, nil
	}

	internalHandler, err := m.get(serviceName)
	if err != nil {
		return nil, err
	}

	return internalHandler, nil
}

func (m *InternalHandlers) get(serviceName string) (http.Handler, error) {
	switch serviceName {
	case "noop@internal":
		return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		}), nil

	case "api@internal":
		if m.api == nil {
			return nil, errors.New("api is not enabled")
		}
		return m.api, nil

	case "dashboard@internal":
		if m.dashboard == nil {
			return nil, errors.New("dashboard is not enabled")
		}
		return m.dashboard, nil

	case "ping@internal":
		if m.ping == nil {
			return nil, errors.New("ping is not enabled")
		}
		return m.ping, nil

	case "prometheus@internal":
		if m.prometheus == nil {
			return nil, errors.New("prometheus is not enabled")
		}
		return m.prometheus, nil

	default:
		if handler, ok := m.plugins[strings.TrimSuffix(serviceName, "@internal")]; ok {
			return handler, nil
		}

		return nil, fmt.Errorf("unknown internal service %s", serviceName)
	}
}
