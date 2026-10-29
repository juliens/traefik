package plugins

import (
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

// typedConstructor returns the name of the constructor of the middleware implementing the option: NewHeaders for headers.
func typedConstructor(option string) string {
	return "New" + strings.ToUpper(option[:1]) + option[1:]
}

// middlewareOption returns the field of the given option of the dynamic configuration of the middlewares.
func middlewareOption(option string) (reflect.StructField, bool) {
	typ := reflect.TypeFor[dynamic.Middleware]()
	for field := range typ.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == option && field.Type.Kind() == reflect.Pointer && field.Type.Elem().Kind() == reflect.Struct {
			return field, true
		}
	}

	return reflect.StructField{}, false
}

// configValue returns a configuration of the static configuration as the configuration of a plugin, which the plugin decodes.
// It keeps all the values, the zero ones included: the plugin has its own default values,
// so an option set to false must stay false.
func configValue(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}

		return configValue(v.Elem())

	case reflect.Struct:
		m := map[string]any{}
		for field, value := range v.Fields() {
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if !field.IsExported() || name == "-" {
				continue
			}

			// The options of an embedded struct are at the level of the struct that embeds it, as in the static configuration.
			if field.Anonymous && name == "" {
				if embedded, ok := configValue(value).(map[string]any); ok {
					maps.Copy(m, embedded)
				}

				continue
			}

			if name == "" {
				name = field.Name
			}

			if cv := configValue(value); cv != nil {
				m[name] = cv
			}
		}

		return m

	case reflect.Slice:
		if v.IsNil() {
			return nil
		}

		s := make([]any, v.Len())
		for i := range v.Len() {
			s[i] = configValue(v.Index(i))
		}

		return s

	case reflect.Map:
		if v.IsNil() {
			return nil
		}

		m := map[string]any{}
		for key, value := range v.Seq2() {
			m[fmt.Sprint(key.Interface())] = configValue(value)
		}

		return m

	// The named types (ptypes.Duration, types.FileOrContent, ...) are given as their basic types:
	// the plugin decodes them into its own types.
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.String:
		return v.String()

	default:
		return v.Interface()
	}
}

// decodeBuiltinConfig decodes the configuration of a plugin compiled into Traefik.
func decodeBuiltinConfig(config map[string]any, result any) error {
	cfg := &mapstructure.DecoderConfig{
		// The static configuration gives the durations as text ("1s"): ptypes.Duration decodes it.
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToSliceHookFunc(","),
			emptyStringToStruct,
			mapstructure.TextUnmarshallerHookFunc(),
		),
		WeaklyTypedInput: true,
		// The options of an embedded struct are at the level of the struct that embeds it, as in the static configuration.
		Squash: true,
		Result: result,
	}

	decoder, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		return fmt.Errorf("failed to create configuration decoder: %w", err)
	}

	if err := decoder.Decode(config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return nil
}

// emptyStringToStruct decodes an empty value of the configuration as an empty struct or map:
// the static configuration gives "" for an option without value (tlsChallenge: {}), which enables a feature.
func emptyStringToStruct(_, to reflect.Type, data any) (any, error) {
	if str, ok := data.(string); !ok || str != "" {
		return data, nil
	}

	for to.Kind() == reflect.Pointer {
		to = to.Elem()
	}

	if to.Kind() == reflect.Struct || to.Kind() == reflect.Map {
		return map[string]any{}, nil
	}

	return data, nil
}
