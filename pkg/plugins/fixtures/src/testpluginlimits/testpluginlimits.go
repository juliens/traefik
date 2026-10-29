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

		case "work":
			// About 10,000 steps: within the default budget.
			n := 0
			for i := range 10_000 {
				n += i
			}
			rw.Header().Set("X-Work", "done")

		case "loop-bounded":
			// About 200,000,000 steps: over the default budget.
			n := 0
			for i := range 200_000_000 {
				n += i
			}
			rw.Header().Set("X-Work", "done")

		case "some-goroutines":
			// 50 goroutines at the same time: within the default limit.
			block := make(chan struct{})
			for range 50 {
				go func() { <-block }()
			}
			close(block)
		}

		next.ServeHTTP(rw, req)
	}), nil
}
