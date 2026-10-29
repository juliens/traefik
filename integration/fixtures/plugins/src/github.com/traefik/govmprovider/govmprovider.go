package govmprovider

import (
	"context"
	"encoding/json"
	"errors"
)

type Config struct {
	ServerURL string
}

func CreateConfig() *Config {
	return &Config{}
}

type Provider struct {
	serverURL string
}

func New(ctx context.Context, config *Config, name string) (*Provider, error) {
	if config.ServerURL == "" {
		return nil, errors.New("serverURL is required")
	}

	return &Provider{serverURL: config.ServerURL}, nil
}

func (p *Provider) Init() error {
	return nil
}

func (p *Provider) Provide(cfgChan chan<- json.Marshaler) error {
	go func() {
		cfgChan <- &payload{
			HTTP: httpConfiguration{
				Routers: map[string]router{
					"govm": {EntryPoints: []string{"web"}, Rule: "PathPrefix(`/`)", Service: "govm"},
				},
				Services: map[string]service{
					"govm": {LoadBalancer: loadBalancer{Servers: []server{{URL: p.serverURL}}}},
				},
			},
		}
	}()

	return nil
}

func (p *Provider) Stop() error {
	return nil
}

type payload struct {
	HTTP httpConfiguration `json:"http"`
}

func (c *payload) MarshalJSON() ([]byte, error) {
	type alias payload

	return json.Marshal((*alias)(c))
}

type httpConfiguration struct {
	Routers  map[string]router  `json:"routers"`
	Services map[string]service `json:"services"`
}

type router struct {
	EntryPoints []string `json:"entryPoints"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
}

type service struct {
	LoadBalancer loadBalancer `json:"loadBalancer"`
}

type loadBalancer struct {
	Servers []server `json:"servers"`
}

type server struct {
	URL string `json:"url"`
}
