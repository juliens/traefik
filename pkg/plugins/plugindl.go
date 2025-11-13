package plugins

import (
	"context"
	"fmt"
	"net/http"
	"plugin"
	"reflect"
)

var NewPluginFn func(ctx context.Context, pluginName, name, config string, next http.Handler) (http.Handler, error)

var NewPluginProviderFn func(ctx context.Context, pluginName, config string) (*Provider, error)

func loadPlugins(pluginFile string) error {
	plugin, err := plugin.Open(pluginFile)
	if err != nil {
		fmt.Println("error while opening shared object file", err)
		return err
	}

	symList, err := plugin.Lookup("Plugins")
	if err != nil {
		fmt.Println("error while lookup", err)
		return err
	}

	fn, ok := symList.(func() []string)
	if !ok {
		fmt.Println("error while lookup symbol list")
	} else {
		fmt.Println(fn())
	}

	symNew, err := plugin.Lookup("NewPlugin")
	if err != nil {
		fmt.Println("error while lookup", err)
		return err
	}

	NewPluginFn = func(ctx context.Context, pluginName, name, config string, next http.Handler) (http.Handler, error) {
		results := reflect.ValueOf(symNew).Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(pluginName),
			reflect.ValueOf(config),
			reflect.ValueOf(next),
		})

		var h http.Handler
		var errNew error
		if !results[0].IsNil() {
			h = results[0].Interface().(http.Handler)
		}
		if !results[1].IsNil() {
			errNew = results[1].Interface().(error)
		}

		return h, errNew
	}

	symNewProvider, err := plugin.Lookup("NewProviderPlugin")
	if err != nil {
		fmt.Println("error while lookup", err)
		return err
	}

	NewPluginProviderFn = func(ctx context.Context, pluginName, config string) (*Provider, error) {
		results := reflect.ValueOf(symNewProvider).Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(pluginName),
			reflect.ValueOf(config),
		})

		var h PP
		var errNew error
		if !results[0].IsNil() {
			h = results[0].Interface().(PP)
		}
		if !results[1].IsNil() {
			errNew = results[1].Interface().(error)
		}

		return &Provider{name: pluginName, pp: h}, errNew
	}

	return nil
}
