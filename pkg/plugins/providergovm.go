package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"go/types"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/traefik/go-vm/vm"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/provider"
	"github.com/traefik/traefik/v3/pkg/safe"
)

const dynamicMessageType = "github.com/traefik/traefik/v3/pkg/config/dynamic.Message"

// newGoVMProvider creates the provider of a Go plugin.
//
// A provider plugin implements one of two interfaces:
//   - the Traefik provider interface (Init, and Provide with a channel of dynamic.Message and a safe.Pool):
//     the plugin sends the configuration types of Traefik, which run natively (see pkg/plugins/hostsyms),
//     and names its configuration itself;
//   - the JSON interface (Init, Provide with a channel of json.Marshaler, and Stop):
//     the plugin sends its configuration as JSON, and its provider is named plugin-<name>.
func newGoVMProvider(p *vm.Plugin, config map[string]any, pName string) (provider.Provider, error) {
	cfg, err := createGoVMConfig(p, config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	res, err := p.Call(ctx, "New", ctx, cfg, "plugin-"+pName)
	if err != nil {
		return nil, fmt.Errorf("failed to call New: %w", err)
	}

	if len(res) != 2 {
		return nil, fmt.Errorf("invalid number of return for the New function: %d", len(res))
	}

	if err := goVMError(res[1]); err != nil {
		return nil, err
	}

	obj := res[0]

	// go-vm calls the methods of a value of a concrete type only.
	if types.IsInterface(obj.Type()) {
		return nil, fmt.Errorf("the New function of the provider returns the interface %s, a concrete type is expected", obj.Type())
	}

	if !sendsDynamicMessages(obj.Type()) {
		return &Provider{name: "plugin-" + pName, pp: &goVMPP{obj: obj}}, nil
	}

	gp := &goVMProvider{name: pName, obj: obj, listens: hasMethod(obj.Type(), "ListenConfiguration"), throttle: -1}

	if hasMethod(obj.Type(), "ThrottleDuration") {
		res, err := obj.Call(ctx, "ThrottleDuration")
		if err != nil {
			return nil, fmt.Errorf("failed to call ThrottleDuration: %w", err)
		}

		if len(res) != 1 {
			return nil, fmt.Errorf("invalid number of return for the ThrottleDuration method: %d", len(res))
		}

		v, err := res[0].Interface()
		if err != nil {
			return nil, err
		}

		throttle, ok := v.(time.Duration)
		if !ok {
			return nil, fmt.Errorf("the ThrottleDuration method returns %T, a time.Duration is expected", v)
		}

		gp.throttle = throttle
	}

	if hasMethod(obj.Type(), "ServeHTTP") {
		// go-vm does not give the value of a type with promoted methods as an http.Handler: the method is called on the object.
		handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if _, err := obj.Call(req.Context(), "ServeHTTP", rw, req); err != nil {
				log.Ctx(req.Context()).Error().Err(err).Str(logs.ProviderName, pName).Msg("Failed to serve the request")
				http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		})

		hp := &goVMHandlerProvider{goVMProvider: gp, handler: handler}

		if hasMethod(obj.Type(), "InternalRule") {
			res, err := obj.Call(ctx, "InternalRule")
			if err != nil {
				return nil, fmt.Errorf("failed to call InternalRule: %w", err)
			}

			if len(res) != 1 {
				return nil, fmt.Errorf("invalid number of return for the InternalRule method: %d", len(res))
			}

			v, err := res[0].Interface()
			if err != nil {
				return nil, err
			}

			rule, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("the InternalRule method returns %T, a string is expected", v)
			}

			hp.rule = rule
		}

		return hp, nil
	}

	return gp, nil
}

// sendsDynamicMessages reports whether the Provide method of the type takes a channel of dynamic.Message.
func sendsDynamicMessages(t types.Type) bool {
	sel := types.NewMethodSet(t).Lookup(nil, "Provide")
	if sel == nil {
		return false
	}

	sig, ok := sel.Type().(*types.Signature)
	if !ok || sig.Params().Len() == 0 {
		return false
	}

	ch, ok := sig.Params().At(0).Type().Underlying().(*types.Chan)
	if !ok {
		return false
	}

	return types.TypeString(ch.Elem(), nil) == dynamicMessageType
}

func hasMethod(t types.Type, name string) bool {
	return types.NewMethodSet(t).Lookup(nil, name) != nil
}

// goVMProvider is a provider plugin that implements the Traefik provider interface.
type goVMProvider struct {
	name string
	obj  *vm.Object

	// listens is true when the plugin has a ListenConfiguration method (dynamic.Configuration):
	// Traefik gives it each new configuration, as to the ACME provider, which resolves the certificates of the routers.
	listens bool

	// throttle is the throttle duration of the plugin, negative when the plugin uses the providersThrottleDuration option.
	throttle time.Duration
}

// ThrottleDuration returns the throttle duration of the plugin, negative when the plugin uses the providersThrottleDuration option.
func (p *goVMProvider) ThrottleDuration() time.Duration {
	return p.throttle
}

// Name returns the name of the plugin.
func (p *goVMProvider) Name() string {
	return p.name
}

// Init initializes the provider.
func (p *goVMProvider) Init() error {
	return callGoVMError(p.obj, "Init")
}

// Provide starts the provider.
func (p *goVMProvider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	return callGoVMError(p.obj, "Provide", configurationChan, pool)
}

// ConfigurationListener returns the function giving the new configurations of Traefik to the provider plugin,
// when the plugin listens to them.
func ConfigurationListener(p provider.Provider) (func(dynamic.Configuration), bool) {
	var gp *goVMProvider
	switch v := p.(type) {
	case *goVMProvider:
		gp = v
	case *goVMHandlerProvider:
		gp = v.goVMProvider
	case *nativeProvider:
		return v.configurationListener()
	case *nativeHandlerProvider:
		return v.configurationListener()
	}

	if gp == nil || !gp.listens {
		return nil, false
	}

	return func(conf dynamic.Configuration) {
		if _, err := gp.obj.Call(context.Background(), "ListenConfiguration", conf); err != nil {
			log.Error().Err(err).Str(logs.ProviderName, gp.name).Msg("Failed to give the configuration to the provider plugin")
		}
	}, true
}

// HandlerProvider is a provider plugin that also serves HTTP requests,
// on the service <name>@internal (the REST provider gets its configuration this way).
type HandlerProvider interface {
	provider.Provider
	http.Handler

	// Name returns the name of the plugin.
	Name() string
	// InternalRule returns the rule of a router from the entry point named traefik to the service of the plugin,
	// or an empty string when the plugin does not ask for this router.
	InternalRule() string
}

type goVMHandlerProvider struct {
	*goVMProvider

	handler http.Handler
	rule    string
}

func (p *goVMHandlerProvider) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.handler.ServeHTTP(rw, req)
}

func (p *goVMHandlerProvider) InternalRule() string {
	return p.rule
}

// goVMPP is the PP of a provider plugin run by go-vm.
type goVMPP struct {
	obj *vm.Object
}

func (p *goVMPP) Init() error {
	return callGoVMError(p.obj, "Init")
}

func (p *goVMPP) Provide(cfgChan chan<- json.Marshaler) error {
	return callGoVMError(p.obj, "Provide", cfgChan)
}

func (p *goVMPP) Stop() error {
	return callGoVMError(p.obj, "Stop")
}

func callGoVMError(obj *vm.Object, method string, args ...any) error {
	res, err := obj.Call(context.Background(), method, args...)
	if err != nil {
		return fmt.Errorf("failed to call %s: %w", method, err)
	}

	if len(res) != 1 {
		return fmt.Errorf("invalid number of return for the %s method: %d", method, len(res))
	}

	return goVMError(res[0])
}
