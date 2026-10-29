package testpluginprovider

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type Config struct {
	PollInterval string
	Names        []string
}

func CreateConfig() *Config {
	return &Config{PollInterval: "10ms"}
}

type Provider struct {
	name         string
	pollInterval time.Duration
	names        []string
	cancel       func()
}

func New(ctx context.Context, config *Config, name string) (*Provider, error) {
	pi, err := time.ParseDuration(config.PollInterval)
	if err != nil {
		return nil, err
	}

	if len(config.Names) == 0 {
		return nil, errors.New("no names")
	}

	return &Provider{name: name, pollInterval: pi, names: config.Names}, nil
}

func (p *Provider) Init() error {
	if p.pollInterval <= 0 {
		return errors.New("poll interval must be greater than 0")
	}

	return nil
}

func (p *Provider) Provide(cfgChan chan<- json.Marshaler) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go func() {
		ticker := time.NewTicker(p.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				select {
				case cfgChan <- &JSONPayload{Configuration: p.generate()}:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (p *Provider) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}

	return nil
}

func (p *Provider) generate() *Configuration {
	cfg := &Configuration{HTTP: &HTTPConfiguration{
		Routers:  map[string]*Router{},
		Services: map[string]*Service{},
	}}

	for _, n := range p.names {
		cfg.HTTP.Routers[n] = &Router{Service: n, Rule: "Host(`" + n + ".localhost`)"}
		cfg.HTTP.Services[n] = &Service{LoadBalancer: &LoadBalancer{Servers: []Server{{URL: "http://127.0.0.1:9000"}}}}
	}

	return cfg
}

type JSONPayload struct {
	*Configuration
}

func (c JSONPayload) MarshalJSON() ([]byte, error) {
	if c.Configuration == nil {
		return nil, nil
	}

	return json.Marshal(c.Configuration)
}

type Configuration struct {
	HTTP *HTTPConfiguration `json:"http,omitempty"`
}

type HTTPConfiguration struct {
	Routers  map[string]*Router  `json:"routers,omitempty"`
	Services map[string]*Service `json:"services,omitempty"`
}

type Router struct {
	Service string `json:"service,omitempty"`
	Rule    string `json:"rule,omitempty"`
}

type Service struct {
	LoadBalancer *LoadBalancer `json:"loadBalancer,omitempty"`
}

type LoadBalancer struct {
	Servers []Server `json:"servers,omitempty"`
}

type Server struct {
	URL string `json:"url,omitempty"`
}
