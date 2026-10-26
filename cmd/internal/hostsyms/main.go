// Command hostsyms generates the go-vm native symbol tables of the Traefik SDK: the packages that Traefik shares with its plugins.
//
// The packages of the SDK, and the packages of the types they give to the plugins, run natively in the Go plugins (vm.ModeNative):
// a plugin calls the code of Traefik instead of interpreting a second copy of it.
// The plugins and Traefik then share the state of these packages (the logger, the registered values)
// and their types: a provider plugin sends the dynamic.Message of Traefik.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/tools/go/packages"
)

// sdk are the packages of Traefik given to the plugins.
var sdk = []string{
	"./pkg/config/dynamic",
	"./pkg/observability/logs",
	"./pkg/plugins/host",
	"./pkg/provider",
	"./pkg/safe",
	"./pkg/tls",
	"./pkg/types",
	"./pkg/version",
	"github.com/rs/zerolog/log",
	// The decoding of the configuration files: paerser/parser is native (pkg/config/label), and gets the reflect values of its callers.
	"github.com/traefik/paerser/file",
	// Its init registers handlers in the native http.DefaultServeMux: a second copy in a plugin panics.
	"golang.org/x/net/trace",
	// It is written in assembly, which go-vm cannot run (lego, through golang.org/x/crypto).
	"golang.org/x/sys/cpu",
	// It configures the transports of the native net/http, which only accepts its own HTTP/2 configuration.
	"golang.org/x/net/http2",
}

func main() {
	output := flag.String("output", "pkg/plugins/hostsyms", "output directory of the generated package")
	flag.Parse()

	if err := run(*output); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func run(output string) error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes,
		Env:  append(os.Environ(), "CGO_ENABLED=0"),
	}

	roots, err := packages.Load(cfg, sdk...)
	if err != nil {
		return err
	}

	for _, pkg := range roots {
		if len(pkg.Errors) > 0 {
			return fmt.Errorf("%s: %v", pkg.PkgPath, pkg.Errors)
		}
	}

	var paths, std []string
	api := apiPackages(roots)
	for path := range api {
		switch {
		case !exposable(path), strings.HasPrefix(path, "vendor/"):
		case isStd(path):
			std = append(std, path)
		default:
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	slices.Sort(std)

	if err := os.RemoveAll(output); err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}

	// The packages imported by the SDK only hold the declarations the SDK uses: the tables need all of them.
	pkgs, err := packages.Load(cfg, paths...)
	if err != nil {
		return err
	}

	for i, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return fmt.Errorf("%s: %v", pkg.PkgPath, pkg.Errors)
		}

		src, err := table(pkg.Types, i)
		if err != nil {
			return fmt.Errorf("%s: %w", pkg.PkgPath, err)
		}

		name := fmt.Sprintf("zz_%04d.go", i)
		if err := os.WriteFile(filepath.Join(output, name), src, 0o644); err != nil {
			return err
		}
	}

	src, err := pathsFile(paths, std)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(output, "paths.go"), src, 0o644); err != nil {
		return err
	}

	log.Info().Msgf("Generated the native symbol tables of %d packages in %s", len(paths), output)

	return nil
}

// apiPackages returns the SDK packages, and the packages of the types that they give to the plugins:
// the types of their exported declarations, and the types of the fields and methods of these types.
// Only these packages run natively: the other dependencies of the SDK are interpreted (or AOT for the standard library)
// in the plugins, like encoding/json, which calls the methods of the plugin values, and grpc, whose values are set
// in the fields of its own types.
func apiPackages(roots []*packages.Package) map[string]*types.Package {
	seen := map[types.Type]bool{}
	res := map[string]*types.Package{}

	var walk func(t types.Type)
	walk = func(t types.Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true

		switch t := t.(type) {
		case *types.Named:
			// The interfaces are structural: a plugin can give its values to an interface of another copy of the package
			// (a fmt.Stringer to zerolog), and fmt stays in AOT mode, where it calls the String methods of the plugin.
			if _, iface := t.Underlying().(*types.Interface); !iface && t.Obj().Pkg() != nil {
				res[t.Obj().Pkg().Path()] = t.Obj().Pkg()
			}
			for i := range t.TypeArgs().Len() {
				walk(t.TypeArgs().At(i))
			}
			for i := range t.NumMethods() {
				if t.Method(i).Exported() {
					walk(t.Method(i).Type())
				}
			}
			walk(t.Underlying())
		case *types.Alias:
			walk(types.Unalias(t))
		case *types.Pointer:
			walk(t.Elem())
		case *types.Slice:
			walk(t.Elem())
		case *types.Array:
			walk(t.Elem())
		case *types.Map:
			walk(t.Key())
			walk(t.Elem())
		case *types.Chan:
			walk(t.Elem())
		case *types.Signature:
			walk(t.Params())
			walk(t.Results())
		case *types.Tuple:
			for i := range t.Len() {
				walk(t.At(i).Type())
			}
		case *types.Struct:
			// go-vm converts all the fields of a struct, the unexported ones too.
			for i := range t.NumFields() {
				walk(t.Field(i).Type())
			}
		case *types.Interface:
			for i := range t.NumMethods() {
				walk(t.Method(i).Type())
			}
		}
	}

	for _, pkg := range roots {
		res[pkg.PkgPath] = pkg.Types

		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			if obj := scope.Lookup(name); obj.Exported() {
				walk(obj.Type())
			}
		}
	}

	return res
}

func isStd(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

// exposable reports whether the generated package can import the package:
// the internal packages of other modules cannot be imported.
func exposable(path string) bool {
	if path == "C" || strings.HasPrefix(path, "github.com/traefik/go-vm/") {
		return false
	}

	for elem := range strings.SplitSeq(path, "/") {
		if elem == "internal" {
			return strings.HasPrefix(path, "github.com/traefik/traefik/v3/")
		}
	}

	return true
}

func table(pkg *types.Package, index int) ([]byte, error) {
	alias := fmt.Sprintf("p%d", index)

	var funcs, typs, vars []string

	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}

		switch o := obj.(type) {
		case *types.Func:
			if o.Signature().TypeParams().Len() > 0 {
				continue
			}
			funcs = append(funcs, fmt.Sprintf("%q: reflect.ValueOf(%s.%s),", name, alias, name))

		case *types.TypeName:
			if isGeneric(o) {
				continue
			}
			typs = append(typs, fmt.Sprintf("%q: reflect.TypeFor[%s.%s](),", name, alias, name))

		case *types.Var:
			vars = append(vars, fmt.Sprintf("%q: reflect.ValueOf(&%s.%s),", name, alias, name))
		}
	}

	// A package that exports only constants and generic declarations has an empty table.
	if len(funcs)+len(typs)+len(vars) == 0 {
		alias = "_"
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `// Code generated by cmd/internal/hostsyms. DO NOT EDIT.

package hostsyms

import (
	"reflect"

	"github.com/traefik/go-vm/vm"
	%s %q
)

func init() {
	vm.RegisterNativePackage(&vm.Package{
		Path: %q,
		Funcs: map[string]reflect.Value{
%s
		},
		Types: map[string]reflect.Type{
%s
		},
		Vars: map[string]reflect.Value{
%s
		},
	})
}
`, alias, pkg.Path(), pkg.Path(), strings.Join(funcs, "\n"), strings.Join(typs, "\n"), strings.Join(vars, "\n"))

	return format.Source(buf.Bytes())
}

func isGeneric(o *types.TypeName) bool {
	// The interfaces with type constraints can only be used as constraints.
	if iface, ok := o.Type().Underlying().(*types.Interface); ok && !iface.IsMethodSet() {
		return true
	}

	if o.IsAlias() {
		alias, ok := o.Type().(*types.Alias)
		return ok && (alias.TypeParams().Len() > 0 || alias.TypeArgs().Len() > 0)
	}

	named, ok := o.Type().(*types.Named)
	return ok && named.TypeParams().Len() > 0
}

func pathsFile(paths, std []string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`// Code generated by cmd/internal/hostsyms. DO NOT EDIT.

// Package hostsyms registers the go-vm native symbol tables of the Traefik SDK given to the plugins (see cmd/internal/hostsyms).
package hostsyms

// Paths are the import paths of the packages with a native symbol table.
var Paths = []string{
`)
	for _, p := range paths {
		fmt.Fprintf(&buf, "\t%q,\n", p)
	}
	buf.WriteString(`}

// StdPaths are the standard packages of the types given by the packages of Paths: the plugins must run them natively too.
var StdPaths = []string{
`)
	for _, p := range std {
		fmt.Fprintf(&buf, "\t%q,\n", p)
	}
	buf.WriteString("}\n")

	return format.Source(buf.Bytes())
}
