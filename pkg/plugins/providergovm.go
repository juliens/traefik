package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"go/types"

	"github.com/traefik/go-vm/vm"
)

func newGoVMProvider(p *vm.Plugin, config map[string]any, providerName string) (*Provider, error) {
	cfg, err := createGoVMConfig(p, config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	res, err := p.Call(ctx, "New", ctx, cfg, providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to call New: %w", err)
	}

	if len(res) != 2 {
		return nil, fmt.Errorf("invalid number of return for the New function: %d", len(res))
	}

	if err := goVMError(res[1]); err != nil {
		return nil, err
	}

	// go-vm calls the methods of a value of a concrete type only.
	if types.IsInterface(res[0].Type()) {
		return nil, fmt.Errorf("the New function of the provider returns the interface %s, a concrete type is expected", res[0].Type())
	}

	return &Provider{name: providerName, pp: &goVMPP{obj: res[0]}}, nil
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
	res, err := p.obj.Call(context.Background(), method, args...)
	if err != nil {
		return fmt.Errorf("failed to call %s: %w", method, err)
	}

	if len(res) != 1 {
		return fmt.Errorf("invalid number of return for the %s method: %d", method, len(res))
	}

	return goVMError(res[0])
}
