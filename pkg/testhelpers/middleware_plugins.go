package testhelpers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/plugins"
)

// MiddlewarePlugins is a test double of the middleware plugins implementing the options of the dynamic configuration:
// the middlewares of Traefik are plugins, and the tests of the routing use a few of them.
// basicAuth refuses the users without password and the requests without credentials, headers sets its custom headers, addPrefix adds its prefix,
// and the other options do nothing.
type MiddlewarePlugins struct{}

// Build builds a plugin of the plugin option, which does nothing.
func (MiddlewarePlugins) Build(_ string, _ map[string]any, _ string) (plugins.Constructor, error) {
	return func(_ context.Context, next http.Handler) (http.Handler, error) {
		return next, nil
	}, nil
}

// BuildTyped builds the middleware implementing an option of the dynamic configuration.
func (MiddlewarePlugins) BuildTyped(_ string, config any, _ string) (plugins.Constructor, error) {
	return func(_ context.Context, next http.Handler) (http.Handler, error) {
		switch conf := config.(type) {
		case *dynamic.BasicAuth:
			for _, user := range conf.Users {
				if !strings.Contains(user, ":") {
					return nil, errors.New("error parsing Authenticator user: " + user)
				}
			}

			return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if _, _, ok := req.BasicAuth(); !ok {
					rw.WriteHeader(http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(rw, req)
			}), nil

		case *dynamic.Headers:
			return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				for name, value := range conf.CustomRequestHeaders {
					req.Header.Set(name, value)
				}
				for name, value := range conf.CustomResponseHeaders {
					rw.Header().Set(name, value)
				}
				next.ServeHTTP(rw, req)
			}), nil

		case *dynamic.AddPrefix:
			if conf.Prefix == "" {
				return nil, errors.New("prefix cannot be empty")
			}

			return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				req.URL.Path = conf.Prefix + "/" + strings.TrimPrefix(req.URL.Path, "/")
				next.ServeHTTP(rw, req)
			}), nil

		default:
			return next, nil
		}
	}, nil
}
