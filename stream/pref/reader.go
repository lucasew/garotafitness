// Package pref restores official Precomp PCF streams (FreeArc method pref).
//
// unpackcmd is `precomp.exe -ostdout -r stdin`. The guest calls
// recompress_file from schnaader/precomp-cpp 0.4.8.
package pref

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed prefdec.wasm
var guestWASM []byte

var (
	errNil   = errors.New("pref: nil reader")
	errGuest = errors.New("pref: guest")
	instID   atomic.Uint64
)

type engine struct {
	rt       wazero.Runtime
	compiled wazero.CompiledModule
}

var loadEngine = sync.OnceValues(func() (*engine, error) {
	ctx := context.Background()
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
	if _, err := rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(func(uint32) {}).Export("emscripten_notify_memory_growth").
		Instantiate(ctx); err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("pref: env: %w", err)
	}
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	compiled, err := rt.CompileModule(ctx, guestWASM)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("pref: compile guest: %w", err)
	}
	return &engine{rt: rt, compiled: compiled}, nil
})

// NewReader wraps an official PCF stream as compress/gzip does.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	in, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("pref: %w", err)
	}
	if len(in) < 7 || string(in[:3]) != "PCF" {
		return nil, fmt.Errorf("pref: bad magic")
	}
	out, err := restore(in)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(out)), nil
}

func restore(in []byte) ([]byte, error) {
	if len(guestWASM) == 0 {
		return nil, errGuest
	}
	eng, err := loadEngine()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "garotafitness-pref-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "in.pcf"), in, 0o600); err != nil {
		return nil, err
	}
	ctx := context.Background()
	mod, err := eng.rt.InstantiateModule(ctx, eng.compiled, wazero.NewModuleConfig().
		WithName(fmt.Sprintf("pref-%d", instID.Add(1))).
		WithStartFunctions("_initialize").
		WithFSConfig(wazero.NewFSConfig().WithDirMount(dir, "/")).
		WithStdout(io.Discard).
		WithStderr(io.Discard))
	if err != nil {
		return nil, fmt.Errorf("pref: instantiate: %w", err)
	}
	defer mod.Close(ctx)
	fn := mod.ExportedFunction("pref_restore")
	if fn == nil {
		return nil, errGuest
	}
	v, err := fn.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("pref: restore: %w", err)
	}
	if len(v) == 0 || v[0] != 0 {
		return nil, errGuest
	}
	out, err := os.ReadFile(filepath.Join(dir, "out.bin"))
	if err != nil {
		return nil, fmt.Errorf("pref: output: %w", err)
	}
	if len(out) == 0 {
		return nil, errGuest
	}
	return out, nil
}
