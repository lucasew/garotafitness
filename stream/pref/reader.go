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
		NewFunctionBuilder().WithFunc(func(int32, int32, int32) int32 { return 0 }).Export("__syscall_unlinkat").
		NewFunctionBuilder().WithFunc(func(int32) int32 { return 0 }).Export("__syscall_rmdir").
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
	ctx := context.Background()
	var stdio bytes.Buffer
	mod, err := eng.rt.InstantiateModule(ctx, eng.compiled, wazero.NewModuleConfig().
		WithName(fmt.Sprintf("pref-%d", instID.Add(1))).
		WithStartFunctions("_initialize").
		WithStdout(&stdio).
		WithStderr(&stdio))
	if err != nil {
		return nil, fmt.Errorf("pref: instantiate: %w", err)
	}
	defer mod.Close(ctx)
	fn := mod.ExportedFunction("pref_restore")
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")
	outPtr := mod.ExportedFunction("pref_out_ptr")
	outLen := mod.ExportedFunction("pref_out_len")
	mem := mod.Memory()
	if fn == nil || malloc == nil || free == nil || outPtr == nil || outLen == nil || mem == nil {
		return nil, errGuest
	}
	p, err := malloc.Call(ctx, uint64(len(in)))
	if err != nil || len(p) == 0 {
		return nil, fmt.Errorf("pref: malloc: %w", err)
	}
	if !mem.Write(uint32(p[0]), in) {
		return nil, fmt.Errorf("%w: write input", errGuest)
	}
	v, err := fn.Call(ctx, p[0], uint64(len(in)))
	if _, err2 := free.Call(ctx, p[0]); err == nil && err2 != nil {
		return nil, fmt.Errorf("pref: free: %w", err2)
	}
	msg := bytes.TrimSpace(stdio.Bytes())
	if err != nil {
		return nil, fmt.Errorf("pref: restore: %w %s", err, msg)
	}
	if len(v) == 0 || v[0] != 0 {
		return nil, fmt.Errorf("%w: restore %v %s", errGuest, v, msg)
	}
	op, err := outPtr.Call(ctx)
	if err != nil || len(op) == 0 {
		return nil, fmt.Errorf("pref: out ptr: %w", err)
	}
	ol, err := outLen.Call(ctx)
	if err != nil || len(ol) == 0 || ol[0] == 0 {
		return nil, fmt.Errorf("%w: empty output %s", errGuest, msg)
	}
	out, ok := mem.Read(uint32(op[0]), uint32(ol[0]))
	if !ok {
		return nil, fmt.Errorf("%w: read output", errGuest)
	}
	cp := make([]byte, len(out))
	copy(cp, out)
	return cp, nil
}
