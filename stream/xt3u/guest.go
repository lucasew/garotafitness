package xt3u

import (
	"context"
	_ "embed"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/lucasew/garotafitness/internal/wasmrun"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed xt3udec.wasm
var guestWASM []byte

var instID atomic.Uint64

var compileCache = sync.OnceValue(wazero.NewCompilationCache)

type lz4guest struct {
	ctx    context.Context
	inst   *wasmrun.Instance
	mod    api.Module
	mem    api.Memory
	hc     api.Function
	fast   api.Function
	bound  api.Function
	zstd   api.Function
	malloc api.Function
	free   api.Function
}

func openGuest(ctx context.Context) (*lz4guest, error) {
	if len(guestWASM) == 0 {
		return nil, errGuest
	}
	if ctx == nil {
		ctx = context.Background()
	}
	inst, err := wasmrun.Open(ctx, compileCache(), guestWASM, "xt3u", wasmrun.Emscripten(nil), wazero.NewModuleConfig().
		WithName(fmt.Sprintf("xt3u-%d", instID.Add(1))))
	if err != nil {
		return nil, err
	}
	mod := inst.Mod
	g := &lz4guest{
		ctx:    ctx,
		inst:   inst,
		mod:    mod,
		mem:    mod.Memory(),
		hc:     mod.ExportedFunction("xt3u_lz4hc"),
		fast:   mod.ExportedFunction("xt3u_lz4"),
		bound:  mod.ExportedFunction("xt3u_lz4_bound"),
		zstd:   mod.ExportedFunction("xt3u_zstd_prefix"),
		malloc: mod.ExportedFunction("malloc"),
		free:   mod.ExportedFunction("free"),
	}
	if g.mem == nil || g.hc == nil || g.fast == nil || g.bound == nil || g.zstd == nil || g.malloc == nil || g.free == nil {
		inst.Close(ctx)
		return nil, errGuest
	}
	return g, nil
}

func (g *lz4guest) Close() error {
	if g == nil || g.inst == nil {
		return nil
	}
	err := g.inst.Close(g.ctx)
	g.inst = nil
	g.mod = nil
	return err
}

func (g *lz4guest) compressHC(src []byte, level, dstCap int) ([]byte, error) {
	return g.compress(g.hc, src, level, dstCap)
}

func (g *lz4guest) compressFast(src []byte, accel, dstCap int) ([]byte, error) {
	return g.compress(g.fast, src, accel, dstCap)
}

func (g *lz4guest) decodePrefix(patch, prefix []byte, dstCap, windowLog int) ([]byte, error) {
	if dstCap <= 0 {
		return nil, errGuest
	}
	pp, err := g.alloc(len(patch))
	if err != nil {
		return nil, err
	}
	defer g.dealloc(pp)
	pre, err := g.alloc(len(prefix))
	if err != nil {
		return nil, err
	}
	defer g.dealloc(pre)
	dp, err := g.alloc(dstCap)
	if err != nil {
		return nil, err
	}
	defer g.dealloc(dp)
	if len(patch) > 0 && !g.mem.Write(pp, patch) {
		return nil, errGuest
	}
	if len(prefix) > 0 && !g.mem.Write(pre, prefix) {
		return nil, errGuest
	}
	v, err := g.zstd.Call(g.ctx, uint64(pp), uint64(uint32(len(patch))), uint64(pre), uint64(uint32(len(prefix))), uint64(dp), uint64(uint32(dstCap)), uint64(uint32(windowLog)))
	if err != nil {
		return nil, fmt.Errorf("xt3u: zstd: %w", err)
	}
	if len(v) == 0 {
		return nil, errGuest
	}
	if int32(v[0]) <= 0 {
		return nil, fmt.Errorf("%w: zstd %d", errGuest, int32(v[0]))
	}
	n := int(int32(v[0]))
	out, ok := g.mem.Read(dp, uint32(n))
	if !ok {
		return nil, errGuest
	}
	return append([]byte(nil), out...), nil
}

func (g *lz4guest) compressBound(n int) (int, error) {
	v, err := g.bound.Call(g.ctx, uint64(uint32(n)))
	if err != nil {
		return 0, fmt.Errorf("xt3u: bound: %w", err)
	}
	if len(v) == 0 {
		return 0, errGuest
	}
	return int(int32(v[0])), nil
}

func (g *lz4guest) compress(fn api.Function, src []byte, param, dstCap int) ([]byte, error) {
	if dstCap <= 0 {
		n, err := g.compressBound(len(src))
		if err != nil {
			return nil, err
		}
		dstCap = n
	}
	if dstCap <= 0 {
		return nil, errGuest
	}
	sp, err := g.alloc(len(src))
	if err != nil {
		return nil, err
	}
	defer g.dealloc(sp)
	dp, err := g.alloc(dstCap)
	if err != nil {
		return nil, err
	}
	defer g.dealloc(dp)
	if len(src) > 0 && !g.mem.Write(sp, src) {
		return nil, errGuest
	}
	v, err := fn.Call(g.ctx, uint64(sp), uint64(uint32(len(src))), uint64(dp), uint64(uint32(dstCap)), uint64(uint32(param)))
	if err != nil {
		return nil, fmt.Errorf("xt3u: lz4: %w", err)
	}
	if len(v) == 0 || int32(v[0]) <= 0 {
		return nil, errGuest
	}
	n := int(int32(v[0]))
	out, ok := g.mem.Read(dp, uint32(n))
	if !ok {
		return nil, errGuest
	}
	return append([]byte(nil), out...), nil
}

func (g *lz4guest) alloc(n int) (uint32, error) {
	if n <= 0 {
		n = 1
	}
	v, err := g.malloc.Call(g.ctx, uint64(uint32(n)))
	if err != nil {
		return 0, fmt.Errorf("xt3u: malloc: %w", err)
	}
	if len(v) == 0 || v[0] == 0 {
		return 0, errGuest
	}
	return uint32(v[0]), nil
}

func (g *lz4guest) dealloc(p uint32) {
	if p == 0 {
		return
	}
	_, _ = g.free.Call(g.ctx, uint64(p))
}
