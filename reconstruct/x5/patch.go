// Package x5 applies the HDIFF13 patches used by the installer's x5 step.
package x5

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed hpatch.wasm
var guest []byte

// Apply reconstructs the target from old and an uncompressed HDIFF13 patch.
// The Guest is the official HDiffPatch 2.5.3 implementation.
func Apply(ctx context.Context, old, diff []byte) ([]byte, error) {
	const maxSize = 1 << 30
	if len(old) > maxSize || len(diff) > maxSize {
		return nil, fmt.Errorf("x5: input exceeds Guest memory limit")
	}
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
	defer rt.Close(context.Background())
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		return nil, fmt.Errorf("x5: wasi: %w", err)
	}
	if _, err := rt.NewHostModuleBuilder("env").NewFunctionBuilder().
		WithFunc(func(uint32) {}).Export("emscripten_notify_memory_growth").Instantiate(ctx); err != nil {
		return nil, fmt.Errorf("x5: env: %w", err)
	}
	mod, err := rt.InstantiateWithConfig(ctx, guest, wazero.NewModuleConfig().WithStartFunctions("_initialize"))
	if err != nil {
		return nil, fmt.Errorf("x5: instantiate: %w", err)
	}
	alloc := func(n int) (uint64, error) {
		res, err := mod.ExportedFunction("malloc").Call(ctx, uint64(max(n, 1)))
		if err != nil {
			return 0, fmt.Errorf("x5: allocate: %w", err)
		}
		if res[0] == 0 {
			return 0, fmt.Errorf("x5: allocation failed")
		}
		return res[0], nil
	}
	op, err := alloc(len(old))
	if err != nil {
		return nil, err
	}
	dp, err := alloc(len(diff))
	if err != nil {
		return nil, err
	}
	if !mod.Memory().Write(uint32(op), old) || !mod.Memory().Write(uint32(dp), diff) {
		return nil, fmt.Errorf("x5: input outside Guest memory")
	}
	sizes, err := mod.ExportedFunction("hpatch_size").Call(ctx, dp, uint64(len(diff)), uint64(len(old)))
	if err != nil {
		return nil, fmt.Errorf("x5: header: %w", err)
	}
	n := sizes[0]
	if n == 0 || n > maxSize || uint64(len(old))+uint64(len(diff))+n > (2<<30)-(64<<20) {
		return nil, fmt.Errorf("x5: invalid or unsupported patch header")
	}
	outp, err := alloc(int(n))
	if err != nil {
		return nil, err
	}
	res, err := mod.ExportedFunction("hpatch_apply").Call(ctx, op, uint64(len(old)), dp, uint64(len(diff)), outp, n)
	if err != nil {
		return nil, fmt.Errorf("x5: apply: %w", err)
	}
	if res[0] != 1 {
		return nil, fmt.Errorf("x5: corrupt patch")
	}
	out, ok := mod.Memory().Read(uint32(outp), uint32(n))
	if !ok {
		return nil, fmt.Errorf("x5: output outside Guest memory")
	}
	return bytes.Clone(out), nil
}
