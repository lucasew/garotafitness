package magic2

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"hash/crc32"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed magic2dec.wasm
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
			return nil, fmt.Errorf("magic2: env: %w", err)
		}
		wasi_snapshot_preview1.MustInstantiate(ctx, rt)
		compiled, err := rt.CompileModule(ctx, guestWASM)
		if err != nil {
			rt.Close(ctx)
			return nil, fmt.Errorf("magic2: compile guest: %w", err)
		}
		return &engine{rt: rt, compiled: compiled}, nil
	})
)

func decodeWASM(src []byte) ([]byte, bool) {
	if len(src) < 4 || len(guestWASM) == 0 {
		return nil, false
	}
	eng, err := loadEngine()
	if err != nil {
		return nil, false
	}
	ctx := context.Background()
	mod, err := eng.rt.InstantiateModule(ctx, eng.compiled, wazero.NewModuleConfig().
		WithName(fmt.Sprintf("magic2-%d", instID.Add(1))).
		WithStartFunctions("_initialize").
		WithStdout(ioDiscard{}).
		WithStderr(ioDiscard{}))
	if err != nil {
		return nil, false
	}
	defer mod.Close(ctx)
	mem := mod.Memory()
	dec := mod.ExportedFunction("magic2_decode")
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")
	if mem == nil || dec == nil || malloc == nil || free == nil {
		return nil, false
	}
	sp, err := malloc.Call(ctx, uint64(len(src)))
	if err != nil || sp[0] == 0 {
		return nil, false
	}
	srcPtr := uint32(sp[0])
	defer free.Call(ctx, uint64(srcPtr))
	dp, err := malloc.Call(ctx, uint64(wantPlain))
	if err != nil || dp[0] == 0 {
		return nil, false
	}
	dstPtr := uint32(dp[0])
	defer free.Call(ctx, uint64(dstPtr))
	if !mem.Write(srcPtr, src) {
		return nil, false
	}
	res, err := dec.Call(ctx, uint64(srcPtr), uint64(len(src)), uint64(dstPtr), uint64(wantPlain))
	if err != nil || res[0] == 0 {
		return nil, false
	}
	n := uint32(res[0])
	out, ok := mem.Read(dstPtr, n)
	if !ok || len(out) < emuSize+appidSize {
		return nil, false
	}
	if crc32.ChecksumIEEE(out[emuSize:emuSize+appidSize]) != appidCRC {
		return nil, false
	}
	return bytes.Clone(out), true
}

// ioDiscard is a tiny io.Writer so guest.go does not pull extra names.
type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
