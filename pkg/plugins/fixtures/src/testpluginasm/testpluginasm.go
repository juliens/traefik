package testpluginasm

import (
	"context"
	"net/http"
)

type Config struct{}

func CreateConfig() *Config {
	return &Config{}
}

// sum is written in assembly (sum.s): go-vm cannot run it.
func sum(a, b int) int

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/sum" {
			_ = sum(1, 2)
		}
		next.ServeHTTP(rw, req)
	}), nil
}
