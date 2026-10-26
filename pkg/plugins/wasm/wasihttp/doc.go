// Package wasihttp runs reactor guests built with the Traefik Go SDK (github.com/traefik/wasm-go).
//
// The guest is called once, on its traefik_serve export, and never returns: it is an event loop that serves every
// request in its own goroutine, inside one wasm instance. The host delivers the requests and their bodies as readable
// virtual file descriptors, and the guest waits for them with poll_oneoff, which blocks the instance goroutine of the
// host until an event arrives. One instance therefore serves any number of in-flight requests in one memory.
package wasihttp
