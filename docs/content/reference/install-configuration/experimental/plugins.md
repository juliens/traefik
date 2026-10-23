---
title: "Traefik Plugins Experimental Configuration"
description: "This section of the Traefik Proxy documentation explains how to use the new Plugins install configuration option."
---

# Traefik Plugins Experimental Configuration

## Overview

This guide provides instructions on how to configure and use the new experimental `plugins` install configuration option in Traefik. The `plugins` option introduces a system to extend Traefik capabilities with custom middlewares and providers.

!!! warning "Experimental"
    
    The `plugins` option is currently experimental and subject to change in future releases. 
    Use with caution in production environments.

## Enabling Plugins

The plugins option is an install configuration parameter.
To enable a plugin, you need to define it in your Traefik install configuration

```yaml tab="File (YAML)"
experimental:
  plugins:
    plugin-name: # The name of the plugin in the routing configuration
      moduleName: "github.com/github-organization/github-repository" # The plugin module name
      version: "vX.XX.X" # The version to use
```

```toml tab="File (TOML)"
[experimental.plugins.plugin-name]
  moduleName = "github.com/github-organization/github-repository" # The plugin module name
  version = "vX.XX.X" # The version to use
```

```bash tab="CLI"
# The plugin module name
# With plugin-name the name of the plugin in the routing configuration
--experimental.plugins.plugin-name.modulename=github.com/github-organization/github-repository
--experimental.plugins.plugin-name.version=vX.XX.X # The version to use
```

To learn more about how to add a new plugin to a Traefik instance, please refer to the [developer documentation](https://plugins.traefik.io/install).

### Plugin Options

| Field | Description | Type | Required |
|-------|-------------|------|----------|
| <a id="opt-moduleName" href="#opt-moduleName" title="#opt-moduleName">`moduleName`</a> | Plugin's module name. | string | Yes |
| <a id="opt-version" href="#opt-version" title="#opt-version">`version`</a> | Plugin's version. | string | Yes |
| <a id="opt-hash" href="#opt-hash" title="#opt-hash">`hash`</a> | Plugin's hash to validate. | string | No |
| <a id="opt-settings" href="#opt-settings" title="#opt-settings">`settings`</a> | Plugin's settings (works only for wasm plugins). | object | No |
| <a id="opt-settings-envs" href="#opt-settings-envs" title="#opt-settings-envs">`settings.envs`</a> | Environment variables to forward to the wasm guest. | []string | No |
| <a id="opt-settings-mounts" href="#opt-settings-mounts" title="#opt-settings-mounts">`settings.mounts`</a> | Directory to mount to the wasm guest. | []string | No |
| <a id="opt-settings-useUnsafe" href="#opt-settings-useUnsafe" title="#opt-settings-useUnsafe">`settings.useUnsafe`</a> | Allow the plugin to use unsafe and syscall packages. | bool | No |
| <a id="opt-settings-goVM" href="#opt-settings-goVM" title="#opt-settings-goVM">`settings.goVM`</a> | Options of the go-vm interpreter (Go plugins only), see [Go Plugins](#go-plugins). | object | No |
| <a id="opt-settings-goVM-aot" href="#opt-settings-goVM-aot" title="#opt-settings-goVM-aot">`settings.goVM.aot`</a> | Run the packages compiled ahead of time into Traefik (AOT mode) instead of interpreting them. | bool | No |
| <a id="opt-settings-goVM-stubAssembly" href="#opt-settings-goVM-stubAssembly" title="#opt-settings-goVM-stubAssembly">`settings.goVM.stubAssembly`</a> | Load a plugin that reaches functions written in assembly: a call of one of them fails at run time. | bool | No |
| <a id="opt-settings-goVM-execMock" href="#opt-settings-goVM-execMock" title="#opt-settings-goVM-execMock">`settings.goVM.execMock`</a> | Load a plugin that imports os/exec: the plugin cannot run programs, an attempt returns an error. | bool | No |

## Local Plugins

Local plugins allow you to use plugins from a local directory, without publishing them to the Traefik plugin catalog.

```yaml tab="File (YAML)"
experimental:
  localPlugins:
    plugin-name: # The name of the plugin in the routing configuration
      moduleName: "github.com/github-organization/github-repository" # The plugin module name
```

```toml tab="File (TOML)"
[experimental.localPlugins.plugin-name]
  moduleName = "github.com/github-organization/github-repository" # The plugin module name
```

```bash tab="CLI"
# The plugin module name
# With plugin-name the name of the plugin in the routing configuration
--experimental.localplugins.plugin-name.modulename=github.com/github-organization/github-repository
```

### Local Plugin Options

| Field | Description | Type | Required |
|-------|-------------|------|----------|
| <a id="opt-moduleName-2" href="#opt-moduleName-2" title="#opt-moduleName-2">`moduleName`</a> | Plugin's module name. | string | Yes |
| <a id="opt-settings-2" href="#opt-settings-2" title="#opt-settings-2">`settings`</a> | Plugin's settings (works only for wasm plugins). | object | No |
| <a id="opt-settings-envs-2" href="#opt-settings-envs-2" title="#opt-settings-envs-2">`settings.envs`</a> | Environment variables to forward to the wasm guest. | []string | No |
| <a id="opt-settings-mounts-2" href="#opt-settings-mounts-2" title="#opt-settings-mounts-2">`settings.mounts`</a> | Directory to mount to the wasm guest. | []string | No |
| <a id="opt-settings-useUnsafe-2" href="#opt-settings-useUnsafe-2" title="#opt-settings-useUnsafe-2">`settings.useUnsafe`</a> | Allow the plugin to use unsafe and syscall packages. | bool | No |
| <a id="opt-settings-goVM-2" href="#opt-settings-goVM-2" title="#opt-settings-goVM-2">`settings.goVM`</a> | Options of the go-vm interpreter (Go plugins only), see [Go Plugins](#go-plugins). | object | No |
| <a id="opt-settings-goVM-aot-2" href="#opt-settings-goVM-aot-2" title="#opt-settings-goVM-aot-2">`settings.goVM.aot`</a> | Run the packages compiled ahead of time into Traefik (AOT mode) instead of interpreting them. | bool | No |
| <a id="opt-settings-goVM-stubAssembly-2" href="#opt-settings-goVM-stubAssembly-2" title="#opt-settings-goVM-stubAssembly-2">`settings.goVM.stubAssembly`</a> | Load a plugin that reaches functions written in assembly: a call of one of them fails at run time. | bool | No |
| <a id="opt-settings-goVM-execMock-2" href="#opt-settings-goVM-execMock-2" title="#opt-settings-goVM-execMock-2">`settings.goVM.execMock`</a> | Load a plugin that imports os/exec: the plugin cannot run programs, an attempt returns an error. | bool | No |

## Go Plugins

Traefik runs the Go plugins (the plugins with the `yaegi` runtime in their manifest) with the go-vm interpreter, from their source code.
WASM plugins are not affected.

Traefik applies these restrictions to the Go plugins when it loads them:

- The `unsafe` package can only be imported when both the plugin manifest and the `settings.useUnsafe` option allow it.
- The `os/exec`, `plugin`, and `net/http/cgi` packages are refused.
- The functions of the `syscall` package are refused, except, when `useUnsafe` is allowed, the functions that only read information about the process (`Getpid`, `Getuid`, `Getenv`, `Gettimeofday`, ...).
  The functions that change the process or reach raw memory (`Kill`, `Exec`, `Mmap`, `Syscall`, ...) are always refused.
- The functions that start or signal a process (`os.StartProcess`, `os.FindProcess`), change the current directory (`os.Chdir`), or change the runtime settings (`runtime/debug.Set*`) are refused.
- `reflect.NewAt` is refused, unless `useUnsafe` is allowed.

By default, go-vm interprets every package of a Go plugin, and refuses a plugin that imports `os/exec` or reaches a function written in assembly.
The `settings.goVM` options change this for one plugin, and are all off by default:

- `aot`: the packages of the plugin that Traefik has compiled ahead of time (the standard library and common libraries) run as compiled code instead of being interpreted, which is faster.
- `stubAssembly`: a plugin that reaches functions written in assembly is loaded; a call of one of them fails at run time.
- `execMock`: a plugin that imports `os/exec` is loaded (as client-go or OPA do); it cannot run programs, an attempt returns an error.

```yaml tab="File (YAML)"
experimental:
  localPlugins:
    plugin-name:
      moduleName: "github.com/github-organization/github-repository"
      settings:
        goVM:
          aot: true
          execMock: true
```

```bash tab="CLI"
--experimental.localplugins.plugin-name.settings.govm.aot=true
--experimental.localplugins.plugin-name.settings.govm.execmock=true
```

A Go plugin gets a copy of the environment of Traefik, so it cannot change the environment of Traefik.
As with Yaegi, unused variables, imports, and labels in the code of a Go plugin do not prevent Traefik from loading it.
A panic in a goroutine started by a Go plugin is logged and does not stop Traefik.

As with Yaegi, a local plugin (`experimental.localPlugins`) can have no `go.mod` file: Traefik then reads it and its dependencies from `./plugins-local/src`, in GOPATH mode.
A local plugin with a `go.mod` file is a Go module: its packages must be under the module path, and its dependencies in its `vendor` directory.
The files of a local plugin can be symbolic links that stay inside `./plugins-local/src`, as in a Kubernetes ConfigMap volume.
A downloaded plugin must have a `go.mod` file at the root of its archive.

Traefik selects the files of a Go plugin with the `//go:build` lines, as the Go compiler does.
Yaegi only read the `// +build` lines.
When a package of a plugin now uses other files than with Yaegi, Traefik logs a warning that names the files.

Traefik also limits the resources of the Go plugins:

- Each call from Traefik into a plugin (`New`, the handling of a request, a method of a provider) has an execution budget of 100,000,000 steps (function calls and loop iterations).
  A call that uses its budget stops, so an infinite loop in a plugin stops after a fraction of a second, and the request gets a `500 Internal Server Error` response.
- A plugin can run at most 1,000 goroutines at the same time.
  Past this limit, the code that starts a new goroutine fails.
- The load of a plugin must end within 30 seconds, or Traefik refuses the plugin.

A request whose client goes away is not stopped in the middle of a plugin: the plugin call runs to its end, within its budget.

The `New` function of a provider plugin must return a concrete type, such as `*Provider`, and not an interface.

!!! danger "Not a security boundary"

    The Go plugins run inside the Traefik process.
    These restrictions stop a careless plugin, not a plugin written to attack Traefik: only install plugins that you trust.
