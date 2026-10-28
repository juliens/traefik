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
	"maps"
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
	// The middleware plugins share the state of the request with Traefik through the context:
	// the access log data, the observability of the router, the tracer, and the retry attempts that the proxy reads.
	"./pkg/middlewares/accesslog",
	"./pkg/middlewares/observability",
	"./pkg/middlewares/retry",
	"./pkg/observability/tracing",
	"./pkg/observability/logs",
	"./pkg/plugins/host",
	"./pkg/provider",
	"./pkg/safe",
	"./pkg/tls",
	"./pkg/types",
	"./pkg/version",
	"github.com/rs/zerolog/log",
	// The global state of OpenTelemetry (the tracer provider, the propagators): the types of otel/trace are native,
	// and an interpreted copy cannot keep its default tracer provider, whose interface has unexported methods, in them.
	"go.opentelemetry.io/otel",
	// The carriers given to the native propagators of OpenTelemetry (otelhttp in the Docker client).
	"go.opentelemetry.io/otel/propagation",
	// The decoding of the configuration files: paerser/parser is native (pkg/config/label), and gets the reflect values of its callers.
	"github.com/traefik/paerser/file",
	// Its init registers handlers in the native http.DefaultServeMux: a second copy in a plugin panics.
	"golang.org/x/net/trace",
	// It is written in assembly, which go-vm cannot run (lego, through golang.org/x/crypto).
	"golang.org/x/sys/cpu",
	// It configures the transports of the native net/http, which only accepts its own HTTP/2 configuration.
	"golang.org/x/net/http2",
}

// platforms are the platforms Traefik is built for: the tables only hold the declarations that exist on all of them
// (golang.org/x/sys/cpu declares Errno on darwin only), so that the generated files build everywhere.
var platforms = []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "freebsd/amd64"}

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

	common, err := commonNames(paths)
	if err != nil {
		return err
	}

	for i, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return fmt.Errorf("%s: %v", pkg.PkgPath, pkg.Errors)
		}

		src, err := table(pkg.Types, i, common[pkg.PkgPath])
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
			for arg := range t.TypeArgs().Types() {
				walk(arg)
			}
			for method := range t.Methods() {
				if method.Exported() {
					walk(method.Type())
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
			for v := range t.Variables() {
				walk(v.Type())
			}
		case *types.Struct:
			// go-vm converts all the fields of a struct, the unexported ones too.
			for field := range t.Fields() {
				walk(field.Type())
			}
		case *types.Interface:
			for method := range t.Methods() {
				walk(method.Type())
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

// commonNames returns the exported declarations of the packages that exist on all the platforms, by package.
func commonNames(paths []string) (map[string]map[string]bool, error) {
	var common map[string]map[string]bool

	for _, platform := range platforms {
		goos, goarch, _ := strings.Cut(platform, "/")

		cfg := &packages.Config{
			Mode: packages.NeedName | packages.NeedTypes,
			Env:  append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch),
		}

		pkgs, err := packages.Load(cfg, paths...)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", platform, err)
		}

		names := map[string]map[string]bool{}
		for _, pkg := range pkgs {
			if len(pkg.Errors) > 0 {
				return nil, fmt.Errorf("%s: %s: %v", platform, pkg.PkgPath, pkg.Errors)
			}

			names[pkg.PkgPath] = map[string]bool{}
			for _, name := range pkg.Types.Scope().Names() {
				names[pkg.PkgPath][name] = true
			}
		}

		if common == nil {
			common = names
			continue
		}

		for path, set := range common {
			for name := range set {
				if !names[path][name] {
					delete(set, name)
				}
			}
		}
	}

	return common, nil
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

// table returns the table of the package, with the declarations of common, the declarations that exist on all the platforms.
func table(pkg *types.Package, index int, common map[string]bool) ([]byte, error) {
	alias := fmt.Sprintf("p%d", index)

	var funcs, typs, vars, wrappers, decls []string
	imports := map[string]string{pkg.Path(): alias}

	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() || !common[name] {
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

			if decl, ok := wrapper(o, index, imports); ok {
				decls = append(decls, decl)
				wrappers = append(wrappers, fmt.Sprintf("%q: reflect.TypeFor[_%s_%s](),", name, alias, name))
			}

		case *types.Var:
			vars = append(vars, fmt.Sprintf("%q: reflect.ValueOf(&%s.%s),", name, alias, name))
		}
	}

	var importLines []string
	for path, name := range imports {
		// A package that exports only constants and generic declarations has an empty table.
		if path == pkg.Path() && len(funcs)+len(typs)+len(vars) == 0 {
			name = "_"
		}
		importLines = append(importLines, fmt.Sprintf("%s %q", name, path))
	}
	slices.Sort(importLines)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `// Code generated by cmd/internal/hostsyms. DO NOT EDIT.

package hostsyms

import (
	"reflect"

	"github.com/traefik/go-vm/vm"
	%s
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
		Wrappers: map[string]reflect.Type{
%s
		},
	})
}

%s
`, strings.Join(importLines, "\n"), pkg.Path(), strings.Join(funcs, "\n"), strings.Join(typs, "\n"), strings.Join(vars, "\n"),
		strings.Join(wrappers, "\n"), strings.Join(decls, "\n"))

	return format.Source(buf.Bytes())
}

// wrapper returns the declaration of the wrapper of the interface o, in the format of the go-vm stdsyms wrappers:
// go-vm boxes the plugin values given to native code as this interface in it (klog gives its logr.LogSink to logr.New).
// It returns false for the interfaces that a plugin cannot implement or that the generated package cannot name.
func wrapper(o *types.TypeName, index int, imports map[string]string) (string, bool) {
	iface, ok := o.Type().Underlying().(*types.Interface)
	if !ok || o.IsAlias() || iface.NumMethods() == 0 {
		return "", false
	}

	// The qualifier names the packages of the signatures; their imports are only added when the wrapper is kept.
	added := map[string]string{}
	named := true
	qualifier := func(p *types.Package) string {
		if name, ok := imports[p.Path()]; ok {
			return name
		}
		if name, ok := added[p.Path()]; ok {
			return name
		}
		if !exposable(p.Path()) {
			named = false
		}
		name := fmt.Sprintf("p%d_%d", index, len(imports)+len(added))
		added[p.Path()] = name
		return name
	}

	name := fmt.Sprintf("_%s_%s", imports[o.Pkg().Path()], o.Name())

	var fields, methods []string
	for method := range iface.Methods() {
		sig := method.Signature()
		if !method.Exported() || !exportedOnly(sig) {
			return "", false
		}

		var params, args []string
		for i, p := range slices.Collect(sig.Params().Variables()) {
			typ := types.TypeString(p.Type(), qualifier)
			arg := fmt.Sprintf("a%d", i)
			if sig.Variadic() && i == sig.Params().Len()-1 {
				typ = "..." + types.TypeString(p.Type().(*types.Slice).Elem(), qualifier)
				arg += "..."
			}
			params = append(params, fmt.Sprintf("a%d %s", i, typ))
			args = append(args, arg)
		}

		var results string
		if sig.Results().Len() > 0 {
			var res []string
			for v := range sig.Results().Variables() {
				res = append(res, types.TypeString(v.Type(), qualifier))
			}
			results = "(" + strings.Join(res, ", ") + ")"
		}

		call := fmt.Sprintf("W.W%s(%s)", method.Name(), strings.Join(args, ", "))
		if results != "" {
			call = "return " + call
		}

		fields = append(fields, fmt.Sprintf("W%s func(%s) %s", method.Name(), strings.Join(params, ", "), results))
		methods = append(methods, fmt.Sprintf("func (W %s) %s(%s) %s {\n%s\n}", name, method.Name(), strings.Join(params, ", "), results, call))
	}

	if !named {
		return "", false
	}

	maps.Copy(imports, added)

	return fmt.Sprintf("// %s is the go-vm wrapper of the %s interface.\ntype %s struct {\nIValue any\n%s\n}\n\n%s\n",
		name, o.Name(), name, strings.Join(fields, "\n"), strings.Join(methods, "\n\n")), true
}

// exportedOnly reports whether the type only names exported types: the generated package cannot name the others.
func exportedOnly(t types.Type) bool {
	switch t := t.(type) {
	case *types.Named:
		if !t.Obj().Exported() && t.Obj().Pkg() != nil {
			return false
		}
		for arg := range t.TypeArgs().Types() {
			if !exportedOnly(arg) {
				return false
			}
		}
		return true
	case *types.Alias:
		return t.Obj().Exported() || t.Obj().Pkg() == nil
	case *types.Pointer:
		return exportedOnly(t.Elem())
	case *types.Slice:
		return exportedOnly(t.Elem())
	case *types.Array:
		return exportedOnly(t.Elem())
	case *types.Map:
		return exportedOnly(t.Key()) && exportedOnly(t.Elem())
	case *types.Chan:
		return exportedOnly(t.Elem())
	case *types.Signature:
		return exportedOnly(t.Params()) && exportedOnly(t.Results())
	case *types.Tuple:
		for v := range t.Variables() {
			if !exportedOnly(v.Type()) {
				return false
			}
		}
		return true
	case *types.Struct:
		for field := range t.Fields() {
			if !field.Exported() || !exportedOnly(field.Type()) {
				return false
			}
		}
		return true
	case *types.Interface:
		for method := range t.Methods() {
			if !method.Exported() || !exportedOnly(method.Type()) {
				return false
			}
		}
		return true
	case *types.TypeParam:
		return false
	default:
		return true
	}
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
