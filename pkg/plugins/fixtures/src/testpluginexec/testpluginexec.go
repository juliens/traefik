package testpluginexec

import (
	"context"
	"net/http"
	"os/exec"
)

type Config struct {
	Command string
}

func CreateConfig() *Config {
	return &Config{Command: "true"}
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_ = exec.Command(config.Command).Run()
		next.ServeHTTP(rw, req)
	}), nil
}
