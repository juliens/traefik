package testpluginflusher

import (
	"context"
	"net/http"
	"strconv"
)

type Config struct{}

func CreateConfig() *Config {
	return &Config{}
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, ok := rw.(http.Flusher)
		rw.Header().Set("X-Flusher", strconv.FormatBool(ok))
		next.ServeHTTP(rw, req)
	}), nil
}
