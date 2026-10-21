package testpluginlimits

import (
	"context"
	"net/http"
)

type Config struct {
	Mode string
}

func CreateConfig() *Config {
	return &Config{}
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch config.Mode {
		case "loop":
			for {
			}

		case "goroutines":
			block := make(chan struct{})
			for range 100_000 {
				go func() { <-block }()
			}
		}

		next.ServeHTTP(rw, req)
	}), nil
}
