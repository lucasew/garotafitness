package mpz

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/lucasew/garotafitness/internal/wasmrun"
	"github.com/tetratelabs/wazero"
)

//go:embed mpzdec.wasm
var guestWASM []byte

var (
	instID       atomic.Uint64
	compileCache = sync.OnceValue(wazero.NewCompilationCache)
)

func decodeWASM(ctx context.Context, src []byte, h Header) ([]byte, error) {
	if len(src) == 0 || len(guestWASM) == 0 {
		return nil, errGuest
	}
	orig := h.Orig
	if orig == 0 {
		return nil, errGuest
	}
	inst, err := wasmrun.Open(ctx, compileCache(), guestWASM, "mpz", wasmrun.Emscripten(nil), wazero.NewModuleConfig().
		WithName(fmt.Sprintf("mpz-%d", instID.Add(1))))
	if err != nil {
		return nil, err
	}
	defer inst.Close(ctx)
	mod := inst.Mod
	mem := mod.Memory()
	dec := mod.ExportedFunction("mpz_decode")
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")
	if mem == nil || dec == nil || malloc == nil || free == nil {
		return nil, errGuest
	}
	sp, err := malloc.Call(ctx, uint64(len(src)+headerLen))
	if err != nil || sp[0] == 0 {
		return nil, errGuest
	}
	srcPtr := uint32(sp[0])
	defer free.Call(ctx, uint64(srcPtr))
	dp, err := malloc.Call(ctx, uint64(orig))
	if err != nil || dp[0] == 0 {
		return nil, errGuest
	}
	dstPtr := uint32(dp[0])
	defer free.Call(ctx, uint64(dstPtr))
	var tag [headerLen]byte
	binary.LittleEndian.PutUint32(tag[0:4], h.Version)
	binary.LittleEndian.PutUint32(tag[4:8], h.Orig)
	binary.LittleEndian.PutUint32(tag[8:12], h.Frames)
	binary.LittleEndian.PutUint32(tag[12:16], h.Extra)
	if !mem.Write(srcPtr, tag[:]) || !mem.Write(srcPtr+headerLen, src) {
		return nil, errGuest
	}
	res, err := dec.Call(ctx, uint64(srcPtr), uint64(len(src)+headerLen), uint64(dstPtr), uint64(orig))
	if err != nil {
		return nil, fmt.Errorf("mpz: decode: %w", err)
	}
	n := uint32(res[0])
	if n != orig {
		return nil, errGuest
	}
	out, ok := mem.Read(dstPtr, n)
	if !ok {
		return nil, errGuest
	}
	return bytes.Clone(out), nil
}
