//go:build !linux && !darwin

package httpwasm

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"
	wazero_wasip1 "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// contextApplier adds to a call context what the WASI implementation needs.
type contextApplier func(ctx context.Context) context.Context

// instantiateWASI instantiates the wazero WASI module; the sockets extension is not supported on this platform.
func instantiateWASI(ctx context.Context, rt wazero.Runtime, _ wazero.CompiledModule, _, _ []string) (contextApplier, error) {
	if _, err := wazero_wasip1.Instantiate(ctx, rt); err != nil {
		return nil, fmt.Errorf("instantiating WASI: %w", err)
	}

	return func(ctx context.Context) context.Context {
		return ctx
	}, nil
}
