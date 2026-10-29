package plugins

const (
	runtimeYaegi = "yaegi"
	runtimeWasm  = "wasm"
)

const (
	typeMiddleware = "middleware"
	typeProvider   = "provider"
)

type Settings struct {
	Envs      []string `description:"Environment variables to forward to the wasm guest." json:"envs,omitempty" toml:"envs,omitempty" yaml:"envs,omitempty"`
	Mounts    []string `description:"Directory to mount to the wasm guest." json:"mounts,omitempty" toml:"mounts,omitempty" yaml:"mounts,omitempty"`
	UseUnsafe bool     `description:"Allow the plugin to use unsafe and syscall packages." json:"useUnsafe,omitempty" toml:"useUnsafe,omitempty" yaml:"useUnsafe,omitempty"`
	// GoVM (optional)
	GoVM GoVMSettings `description:"Options of the go-vm interpreter (Go plugins only)." json:"goVM,omitempty" toml:"goVM,omitempty" yaml:"goVM,omitempty" export:"true"`
}

// GoVMSettings are the options of the go-vm interpreter for one Go plugin.
// They are all off by default, and the limits have their default value when they are zero.
type GoVMSettings struct {
	DisableAOT    bool  `description:"Interpret the packages compiled ahead of time into Traefik instead of running them in AOT mode." json:"disableAOT,omitempty" toml:"disableAOT,omitempty" yaml:"disableAOT,omitempty" export:"true"`
	StubAssembly  bool  `description:"Load a plugin that reaches functions written in assembly: a call of one of them fails at run time." json:"stubAssembly,omitempty" toml:"stubAssembly,omitempty" yaml:"stubAssembly,omitempty" export:"true"`
	ExecMock      bool  `description:"Load a plugin that imports os/exec: the plugin cannot run programs, an attempt returns an error." json:"execMock,omitempty" toml:"execMock,omitempty" yaml:"execMock,omitempty" export:"true"`
	Budget        int64 `description:"Execution budget of each call into the plugin, in steps (function calls and loop iterations). Default: 100000000." json:"budget,omitempty" toml:"budget,omitempty" yaml:"budget,omitempty" export:"true"`
	MaxGoroutines int   `description:"Maximum number of goroutines that the plugin runs at the same time. Default: 1000." json:"maxGoroutines,omitempty" toml:"maxGoroutines,omitempty" yaml:"maxGoroutines,omitempty" export:"true"`
}

// Descriptor The static part of a plugin configuration.
type Descriptor struct {
	// ModuleName (required)
	ModuleName string `description:"plugin's module name." json:"moduleName,omitempty" toml:"moduleName,omitempty" yaml:"moduleName,omitempty" export:"true"`

	// Version (required)
	Version string `description:"plugin's version." json:"version,omitempty" toml:"version,omitempty" yaml:"version,omitempty" export:"true"`

	// Hash (optional)
	Hash string `description:"plugin's hash to validate'" json:"hash,omitempty" toml:"hash,omitempty" yaml:"hash,omitempty" export:"true"`

	// Settings (optional)
	Settings Settings `description:"Plugin's settings (works only for wasm plugins)." json:"settings,omitempty" toml:"settings,omitempty" yaml:"settings,omitempty" export:"true"`
}

// LocalDescriptor The static part of a local plugin configuration.
type LocalDescriptor struct {
	// ModuleName (required)
	ModuleName string `description:"Plugin's module name." json:"moduleName,omitempty" toml:"moduleName,omitempty" yaml:"moduleName,omitempty" export:"true"`

	// Settings (optional)
	Settings Settings `description:"Plugin's settings (works only for wasm plugins)." json:"settings,omitempty" toml:"settings,omitempty" yaml:"settings,omitempty" export:"true"`
}

// Manifest The plugin manifest.
type Manifest struct {
	DisplayName   string         `yaml:"displayName"`
	Type          string         `yaml:"type"`
	Runtime       string         `yaml:"runtime"`
	WasmPath      string         `yaml:"wasmPath"`
	Import        string         `yaml:"import"`
	BasePkg       string         `yaml:"basePkg"`
	Compatibility string         `yaml:"compatibility"`
	Summary       string         `yaml:"summary"`
	UseUnsafe     bool           `yaml:"useUnsafe"`
	TestData      map[string]any `yaml:"testData"`

	// Implements are the options of the configuration that the plugin implements:
	//   - for a middleware plugin, options of the dynamic configuration of the middlewares (headers, stripPrefix, ...).
	//     For each option, the plugin has a New<Option> function (NewHeaders, NewStripPrefix, ...)
	//     taking the configuration of the option, a pointer to its type in pkg/config/dynamic.
	//     Each middleware of Traefik other than chain is such a plugin;
	//   - for a provider plugin, options of the providers of the static configuration (docker, kubernetesCRD, ...).
	//     The plugin gets the configuration of the option with CreateConfig and New, as the other provider plugins:
	//     the types of these options in pkg/config/static are generated from its configuration (see cmd/internal/nativeplugins).
	Implements []string `yaml:"implements"`
}

// IsYaegiPlugin returns true if the plugin is a Yaegi plugin.
func (m *Manifest) IsYaegiPlugin() bool {
	// defaults always Yaegi to have backwards compatibility to plugins without runtime
	return m.Runtime == runtimeYaegi || m.Runtime == ""
}
