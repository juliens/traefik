package plugins

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"plugin"
	"reflect"
)

func NewPlugin(ctx context.Context, pluginFile, pluginName, name string, config string, next http.Handler) (http.Handler, error) {
	plugin, err := plugin.Open(pluginFile)
	if err != nil {
		fmt.Println("error while opening shared object file", err)
		os.Exit(1)
	}
	symNew, err := plugin.Lookup("NewPlugin")
	if err != nil {
		fmt.Println("error while lookup", err)
		os.Exit(1)
	}

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
