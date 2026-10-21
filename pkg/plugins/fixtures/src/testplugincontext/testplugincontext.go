package testplugincontext

import (
	"context"
	"net/http"
	"time"
)

type Config struct{}

func CreateConfig() *Config {
	return &Config{}
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		select {
		case <-req.Context().Done():
			rw.Header().Set("X-Context", req.Context().Err().Error())
		case <-time.After(10 * time.Second):
			rw.Header().Set("X-Context", "not canceled")
		}
	}), nil
}
