package testpluginprocess

import (
	"context"
	"net/http"
	"os"
)

type Config struct {
	Command string
}

func CreateConfig() *Config {
	return &Config{Command: "/bin/true"}
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		p, err := os.StartProcess(config.Command, nil, &os.ProcAttr{})
		if err == nil {
			_, _ = p.Wait()
		}

		next.ServeHTTP(rw, req)
	}), nil
}
