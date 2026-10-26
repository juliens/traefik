//go:build linux || darwin

package httpwasm

import (
	"context"
	"fmt"
	"os"

	"github.com/stealthrocket/wasi-go/imports"
	wazergo_wasip1 "github.com/stealthrocket/wasi-go/imports/wasi_snapshot_preview1"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	wazero_wasip1 "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// contextApplier adds to a call context what the WASI implementation needs.
type contextApplier func(ctx context.Context) context.Context

// instantiateWASI instantiates the WASI module according to the guest requirements (for now only socket extensions).
func instantiateWASI(ctx context.Context, rt wazero.Runtime, mod wazero.CompiledModule, envs, mounts []string) (contextApplier, error) {
	if extension := imports.DetectSocketsExtension(mod); extension != nil {
		builder := imports.NewBuilder().WithSocketsExtension("auto", mod)

		if len(envs) > 0 {
			values := make([]string, 0, len(envs))
			for _, env := range envs {
				values = append(values, fmt.Sprintf("%s=%s", env, os.Getenv(env)))
			}
			builder.WithEnv(values...)
		}

		if len(mounts) > 0 {
			builder.WithDirs(mounts...)
		}

		ctx, sys, err := builder.Instantiate(ctx, rt)
		if err != nil {
			return nil, fmt.Errorf("instantiating WASI with sockets extension: %w", err)
		}

		inst, err := wazergo.Instantiate(ctx, rt, wazergo_wasip1.NewHostModule(*extension), wazergo_wasip1.WithWASI(sys))
		if err != nil {
			return nil, fmt.Errorf("instantiating wazergo module: %w", err)
		}

		return func(ctx context.Context) context.Context {
			return wazergo.WithModuleInstance(ctx, inst)
		}, nil
	}

	if _, err := wazero_wasip1.Instantiate(ctx, rt); err != nil {
		return nil, fmt.Errorf("instantiating WASI: %w", err)
	}

	return func(ctx context.Context) context.Context {
		return ctx
	}, nil
}
