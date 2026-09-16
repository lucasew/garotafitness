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

	"github.com/lucasew/garotafitness/internal/wasmrun"
	"github.com/tetratelabs/wazero"
)

//go:embed prefdec.wasm
var guestWASM []byte

var (
	errNil   = errors.New("pref: nil reader")
	errGuest = errors.New("pref: guest")
	instID   atomic.Uint64
)

var compileCache = sync.OnceValue(wazero.NewCompilationCache)

func prefHost(ctx context.Context, rt wazero.Runtime) error {
	return wasmrun.Emscripten(func(b wazero.HostModuleBuilder) {
		b.NewFunctionBuilder().WithFunc(func(int32, int32, int32) int32 { return 0 }).Export("__syscall_unlinkat")
		b.NewFunctionBuilder().WithFunc(func(int32) int32 { return 0 }).Export("__syscall_rmdir")
		b.NewFunctionBuilder().WithFunc(func() int32 { return 0 }).Export("pref_extra_threads")
	})(ctx, rt)
}

// NewReader wraps an official PCF stream as compress/gzip does.
func NewReader(ctx context.Context, r io.Reader) (io.ReadCloser, error) {
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
	out, err := restore(ctx, in)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(out)), nil
}

func restore(ctx context.Context, in []byte) ([]byte, error) {
	if len(guestWASM) == 0 {
		return nil, errGuest
	}
	var stdio bytes.Buffer
	inst, err := wasmrun.Open(ctx, compileCache(), guestWASM, "pref", prefHost, wazero.NewModuleConfig().
		WithName(fmt.Sprintf("pref-%d", instID.Add(1))).
		WithStdout(&stdio).
		WithStderr(&stdio))
	if err != nil {
		return nil, err
	}
	defer inst.Close(ctx)
	mod := inst.Mod
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
