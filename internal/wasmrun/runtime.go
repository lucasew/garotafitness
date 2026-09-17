package wasmrun

import (
	"context"
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Host installs env exports on a fresh Runtime. Guests stay single-threaded;
// Go creates one Runtime per stream so Call does not share a store lock.
type Host func(ctx context.Context, rt wazero.Runtime) error

// Instance is one Guest on its own Runtime.
type Instance struct {
	RT  wazero.Runtime
	Mod api.Module
	lw  io.WriteCloser
}

// Open compiles wasm through cache and instantiates it on a new Runtime.
func Open(ctx context.Context, cache wazero.CompilationCache, wasm []byte, name string, host Host, cfg wazero.ModuleConfig) (*Instance, error) {
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true).
		WithCompilationCache(cache))
	if host != nil {
		if err := host(ctx, rt); err != nil {
			rt.Close(context.WithoutCancel(ctx))
			return nil, err
		}
	}
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		rt.Close(context.WithoutCancel(ctx))
		return nil, err
	}
	compiled, err := rt.CompileModule(ctx, wasm)
	if err != nil {
		rt.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("%s: compile guest: %w", name, err)
	}
	var lw io.WriteCloser
	if cfg == nil {
		lw = taskgroup.LineWriterFrom(ctx)
		cfg = wazero.NewModuleConfig().WithStdout(io.Discard).WithStderr(lw)
	}
	mod, err := rt.InstantiateModule(ctx, compiled, cfg.WithStartFunctions("_initialize"))
	if err != nil {
		if lw != nil {
			lw.Close()
		}
		rt.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("%s: instantiate: %w", name, err)
	}
	return &Instance{RT: rt, Mod: mod, lw: lw}, nil
}

func (in *Instance) Close(ctx context.Context) error {
	if in == nil {
		return nil
	}
	ctx = context.WithoutCancel(ctx)
	var err error
	if in.lw != nil {
		err = in.lw.Close()
		in.lw = nil
	}
	if in.Mod != nil {
		if e := in.Mod.Close(ctx); e != nil && err == nil {
			err = e
		}
		in.Mod = nil
	}
	if in.RT != nil {
		if e := in.RT.Close(ctx); e != nil && err == nil {
			err = e
		}
		in.RT = nil
	}
	return err
}

// Emscripten notifies growth. Extra host functions are optional.
func Emscripten(extra func(wazero.HostModuleBuilder)) Host {
	return func(ctx context.Context, rt wazero.Runtime) error {
		b := rt.NewHostModuleBuilder("env").
			NewFunctionBuilder().WithFunc(func(uint32) {}).Export("emscripten_notify_memory_growth")
		if extra != nil {
			extra(b)
		}
		_, err := b.Instantiate(ctx)
		return err
	}
}
