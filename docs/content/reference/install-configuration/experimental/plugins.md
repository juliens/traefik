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
| <a id="opt-settings-instances" href="#opt-settings-instances" title="#opt-settings-instances">`settings.instances`</a> | Number of wasm instances per middleware. Defaults to the number of CPUs, with a maximum of 8. | int | No |
| <a id="opt-settings-memoryLimit" href="#opt-settings-memoryLimit" title="#opt-settings-memoryLimit">`settings.memoryLimit`</a> | Maximum linear memory of a wasm instance, in bytes. Defaults to 4GiB. | int | No |
| <a id="opt-settings-exclusive" href="#opt-settings-exclusive" title="#opt-settings-exclusive">`settings.exclusive`</a> | Dedicate a wasm instance to each in-flight request instead of sharing the instances between requests. See [wasm plugin execution](#wasm-plugin-execution). | bool | No |
| <a id="opt-settings-callTimeout" href="#opt-settings-callTimeout" title="#opt-settings-callTimeout">`settings.callTimeout`</a> | Maximum duration of a call to an http-wasm guest. When it is exceeded, the request gets a `500` response and the instance is replaced. Disabled by default. | duration | No |
| <a id="opt-settings-bodyPrefetchSize" href="#opt-settings-bodyPrefetchSize" title="#opt-settings-bodyPrefetchSize">`settings.bodyPrefetchSize`</a> | Bytes of the request body read before an http-wasm guest is called. Defaults to 65536. | int | No |
| <a id="opt-settings-allowOutbound" href="#opt-settings-allowOutbound" title="#opt-settings-allowOutbound">`settings.allowOutbound`</a> | Allow a plugin built with the [Traefik WASM Go SDK](https://github.com/traefik/wasm-go) to send outbound HTTP requests. Defaults to false. | bool | No |

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
| <a id="opt-settings-instances-2" href="#opt-settings-instances-2" title="#opt-settings-instances-2">`settings.instances`</a> | Number of wasm instances per middleware. Defaults to the number of CPUs, with a maximum of 8. | int | No |
| <a id="opt-settings-memoryLimit-2" href="#opt-settings-memoryLimit-2" title="#opt-settings-memoryLimit-2">`settings.memoryLimit`</a> | Maximum linear memory of a wasm instance, in bytes. Defaults to 4GiB. | int | No |
| <a id="opt-settings-exclusive-2" href="#opt-settings-exclusive-2" title="#opt-settings-exclusive-2">`settings.exclusive`</a> | Dedicate a wasm instance to each in-flight request instead of sharing the instances between requests. See [wasm plugin execution](#wasm-plugin-execution). | bool | No |
| <a id="opt-settings-callTimeout-2" href="#opt-settings-callTimeout-2" title="#opt-settings-callTimeout-2">`settings.callTimeout`</a> | Maximum duration of a call to an http-wasm guest. When it is exceeded, the request gets a `500` response and the instance is replaced. Disabled by default. | duration | No |
| <a id="opt-settings-bodyPrefetchSize-2" href="#opt-settings-bodyPrefetchSize-2" title="#opt-settings-bodyPrefetchSize-2">`settings.bodyPrefetchSize`</a> | Bytes of the request body read before an http-wasm guest is called. Defaults to 65536. | int | No |
| <a id="opt-settings-allowOutbound-2" href="#opt-settings-allowOutbound-2" title="#opt-settings-allowOutbound-2">`settings.allowOutbound`</a> | Allow a plugin built with the [Traefik WASM Go SDK](https://github.com/traefik/wasm-go) to send outbound HTTP requests. Defaults to false. | bool | No |

## Wasm Plugin Execution

A wasm middleware runs on a fixed number of guest instances (`settings.instances`), shared by all the requests and
all the routers that use the same middleware configuration.
A request is pinned to one instance for its `handle_request` and `handle_response` calls, but the instance is free
while the next handler of the chain runs: a slow upstream does not hold an instance.
The memory of a wasm middleware is therefore bounded by `settings.instances` times the memory of one instance,
whatever the number of in-flight requests.
When the routing configuration changes, the instances of a middleware configuration that is no longer used are
closed once their in-flight requests are done, or after one minute.

Because several requests are handled by the same instance between their `handle_request` and `handle_response`
calls, a plugin must keep its per-request state behind the request context it returns from `handle_request`,
as the http-wasm ABI requires.
A plugin that keeps per-request state in global variables can set `settings.exclusive` to get a dedicated
instance per in-flight request, at the cost of one instance memory per concurrent request.

An instance is busy while the guest runs `handle_request` or `handle_response`, including when it reads the request
body from the client.
Traefik reads the first `settings.bodyPrefetchSize` bytes of the body before calling the guest, so a guest that reads a
small body never waits for a slow client; raise it for plugins that inspect larger bodies.
`settings.callTimeout` bounds each guest call: a guest stuck in a loop gets its instance replaced instead of blocking
it forever.

A plugin built with the [Traefik WASM Go SDK](https://github.com/traefik/wasm-go) serves every request in its own
goroutine, inside one instance, so its global state must be synchronized as in any Go server.
Such a plugin has no file system, and reads the request body as a stream: `settings.mounts`, `settings.exclusive`,
`settings.callTimeout` and `settings.bodyPrefetchSize` are refused.
When an instance stops (for example on `os.Exit` or when it runs out of memory), its in-flight requests get a
`502 Bad Gateway` response and the instance is replaced.

Such a plugin follows the `wasi:http/middleware` world: it works on requests and responses, with streamed bodies and
trailers, and can call the next handler any number of times.
It cannot take over the connection, so WebSocket and other protocol upgrades do not work through it.
It reads its configuration by key with `wasiconfig`: each top-level key of the middleware configuration is a key, a
string value is given as is and any other value as JSON.

Such a plugin can send outbound HTTP requests with `wasihttp.DefaultClient`, for example to call an authorization
server, once `settings.allowOutbound` is set.
Traefik sends them on behalf of the plugin, and stops them when the instance stops.
