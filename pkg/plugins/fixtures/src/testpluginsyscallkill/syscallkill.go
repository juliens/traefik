package testpluginsyscallkill

import (
	"context"
	"net/http"
	"syscall"
)

type Config struct{}

func CreateConfig() *Config {
	return &Config{}
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	// syscall.Kill changes another process: it must stay refused, even with useUnsafe.
	_ = syscall.Kill(syscall.Getpid(), 0)

	return next, nil
}
