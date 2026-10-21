package plugins

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/traefik/go-vm/vm"
)

func newGoVMProvider(p *vm.Plugin, config map[string]any, providerName string) (*Provider, error) {
	cfg, err := p.CreateConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to call CreateConfig: %w", err)
	}

	err = p.DecodeConfig(cfg, func(result any) error {
		return decodePluginConfig(config, result)
	})
	if err != nil {
		return nil, err
	}

	obj, err := p.NewProvider(context.Background(), cfg, providerName)
	if err != nil {
		return nil, err
	}

	return &Provider{name: providerName, pp: &goVMPP{obj: obj}}, nil
}

// goVMPP is the PP of a provider plugin run by go-vm.
type goVMPP struct {
	obj *vm.Object
}

func (p *goVMPP) Init() error {
	return p.call("Init")
}

func (p *goVMPP) Provide(cfgChan chan<- json.Marshaler) error {
	return p.call("Provide", cfgChan)
}

func (p *goVMPP) Stop() error {
	return p.call("Stop")
}

func (p *goVMPP) call(method string, args ...any) error {
	res, err := p.obj.Call(method, args...)
	if err != nil {
		return fmt.Errorf("failed to call %s: %w", method, err)
	}

	if len(res) != 1 {
		return fmt.Errorf("invalid number of return for the %s method: %d", method, len(res))
	}

	if res[0] == nil {
		return nil
	}

	resErr, ok := res[0].(error)
	if !ok {
		return fmt.Errorf("invalid error type: %T", res[0])
	}

	return resErr
}
