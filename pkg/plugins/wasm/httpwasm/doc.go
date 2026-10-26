// Package httpwasm runs http-wasm guests (https://http-wasm.io) on a fixed set of long-lived instances.
//
// The host functions, the request state and the net/http host come from github.com/http-wasm/http-wasm-host-go
// v0.7.0 (Apache License 2.0, Copyright 2022 the http-wasm Authors), see NOTICE.
// The fork replaces the per-request guest pool of the upstream library: a request is pinned to one instance for its
// handle_request and handle_response calls, both made under the instance lock, but the instance is free while the
// next handler runs, so the number of instances no longer grows with the number of in-flight requests.
package httpwasm
