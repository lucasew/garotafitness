package rzw

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed rzwdec.wasm
var guestWASM []byte

type engine struct {
	rt       wazero.Runtime
	compiled wazero.CompiledModule
}

var (
	instID     atomic.Uint64
	loadEngine = sync.OnceValues(func() (*engine, error) {
		ctx := context.Background()
		rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
		if _, err := rt.NewHostModuleBuilder("env").
			NewFunctionBuilder().WithFunc(func(uint32) {}).Export("emscripten_notify_memory_growth").
			Instantiate(ctx); err != nil {
			rt.Close(ctx)
			return nil, fmt.Errorf("rzw: env: %w", err)
		}
		wasi_snapshot_preview1.MustInstantiate(ctx, rt)
		compiled, err := rt.CompileModule(ctx, guestWASM)
		if err != nil {
			rt.Close(ctx)
			return nil, fmt.Errorf("rzw: compile guest: %w", err)
		}
		return &engine{rt: rt, compiled: compiled}, nil
	})
)

func decodeWASM(src []byte, dcap uint32) ([]byte, error) {
	if len(src) == 0 || len(guestWASM) == 0 {
		return nil, errCodec
	}
	if dcap == 0 {
		return nil, errCodec
	}
	eng, err := loadEngine()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	mod, err := eng.rt.InstantiateModule(ctx, eng.compiled, wazero.NewModuleConfig().
		WithName(fmt.Sprintf("rzw-%d", instID.Add(1))).
		WithStartFunctions("_initialize").
		WithStdout(ioDiscard{}).
		WithStderr(ioDiscard{}))
	if err != nil {
		return nil, fmt.Errorf("rzw: instantiate: %w", err)
	}
	defer mod.Close(ctx)
	mem := mod.Memory()
	dec := mod.ExportedFunction("rzw_decode")
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")
	if mem == nil || dec == nil || malloc == nil || free == nil {
		return nil, errCodec
	}
	sp, err := malloc.Call(ctx, uint64(len(src)))
	if err != nil || sp[0] == 0 {
		return nil, errCodec
	}
	srcPtr := uint32(sp[0])
	defer free.Call(ctx, uint64(srcPtr))
	dp, err := malloc.Call(ctx, uint64(dcap))
	if err != nil || dp[0] == 0 {
		return nil, errCodec
	}
	dstPtr := uint32(dp[0])
	defer free.Call(ctx, uint64(dstPtr))
	if !mem.Write(srcPtr, src) {
		return nil, errCodec
	}
	res, err := dec.Call(ctx, uint64(srcPtr), uint64(len(src)), uint64(dstPtr), uint64(dcap))
	if err != nil {
		return nil, fmt.Errorf("rzw: decode: %w", errCodec)
	}
	n := uint32(res[0])
	if n == 0 || n > dcap {
		return nil, errCodec
	}
	out, ok := mem.Read(dstPtr, n)
	if !ok {
		return nil, errCodec
	}
	return bytes.Clone(out), nil
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
