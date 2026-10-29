package govmplugin

import (
	"context"
	"net/http"
)

type Config struct {
	HeaderValue string
	Loop        bool
}

func CreateConfig() *Config {
	return &Config{}
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		for config.Loop {
		}

		rw.Header().Set("X-Govm-Plugin", config.HeaderValue)
		next.ServeHTTP(rw, req)
	}), nil
}
