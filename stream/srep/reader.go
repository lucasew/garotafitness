package srep

import (
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/lucasew/garotafitness/internal/wasmrun"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed srepdec.wasm
var guestWASM []byte

const maxBlock = 8 << 20

var (
	errNil      = errors.New("srep: nil reader")
	errClosed   = errors.New("srep: closed")
	errTooLarge = errors.New("srep: block too large")
	errBroken   = errors.New("srep: broken block")
	errGuest    = errors.New("srep: guest")
	instID      atomic.Uint64
)

var (
	compileCache = sync.OnceValue(wazero.NewCompilationCache)
	bufPool      sync.Pool
)

func getBuf(n int) []byte {
	if n == 0 {
		return nil
	}
	b, _ := bufPool.Get().([]byte)
	if cap(b) < n {
		return make([]byte, n)
	}
	return b[:n]
}

func putBuf(b []byte) {
	if b == nil || cap(b) > maxBlock {
		return
	}
	bufPool.Put(b[:0])
}

func srepHost(ctx context.Context, rt wazero.Runtime) error {
	return wasmrun.Emscripten(func(b wazero.HostModuleBuilder) {
		b.NewFunctionBuilder().WithFunc(func(int32, int32, int32) int32 { return 0 }).Export("__syscall_unlinkat")
		b.NewFunctionBuilder().WithFunc(func(int32) int32 { return 0 }).Export("__syscall_rmdir")
	})(ctx, rt)
}

// NewReader wraps an official SREP v3 stream as compress/gzip does.
func NewReader(ctx context.Context, r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	h, err := ParseHeader(r)
	if err != nil {
		return nil, err
	}
	if h.Format != FormatFutureLZ {
		return nil, fmt.Errorf("srep: format %d", h.Format)
	}
	if h.Seed > 0 {
		if _, err := io.CopyN(io.Discard, r, int64(h.Seed)); err != nil {
			return nil, fmt.Errorf("srep: seed: %w", err)
		}
	}
	inst, err := wasmrun.Open(ctx, compileCache(), guestWASM, "srep", srepHost, wazero.NewModuleConfig().
		WithName(fmt.Sprintf("srep-%d", instID.Add(1))))
	if err != nil {
		return nil, err
	}
	mod := inst.Mod
	open := mod.ExportedFunction("srep_open")
	rd := &reader{
		src:    r,
		hdr:    h,
		ctx:    ctx,
		inst:   inst,
		mod:    mod,
		mem:    mod.Memory(),
		block:  mod.ExportedFunction("srep_block"),
		cls:    mod.ExportedFunction("srep_close"),
		malloc: mod.ExportedFunction("malloc"),
		free:   mod.ExportedFunction("free"),
	}
	if rd.mem == nil || open == nil || rd.block == nil || rd.cls == nil || rd.malloc == nil || rd.free == nil {
		inst.Close(ctx)
		return nil, errGuest
	}
	if _, err := open.Call(ctx, uint64(h.BaseLen)); err != nil {
		inst.Close(ctx)
		return nil, fmt.Errorf("srep: open: %w", err)
	}
	return rd, nil
}

type reader struct {
	src    io.Reader
	hdr    Header
	ctx    context.Context
	inst   *wasmrun.Instance
	mod    api.Module
	mem    api.Memory
	block  api.Function
	cls    api.Function
	malloc api.Function
	free   api.Function
	buf    []byte
	off    int
	start  uint64
	err    error
	eof    bool
}

func (r *reader) Read(p []byte) (int, error) {
	if r.err != nil && r.off >= len(r.buf) {
		return 0, r.err
	}
	for r.off >= len(r.buf) {
		if r.eof {
			return 0, io.EOF
		}
		block, err := r.next()
		if err == io.EOF {
			r.eof = true
			return 0, io.EOF
		}
		if err != nil {
			r.err = err
			return 0, err
		}
		putBuf(r.buf)
		r.buf = block
		r.off = 0
	}
	n := copy(p, r.buf[r.off:])
	r.off += n
	return n, nil
}

func (r *reader) Close() error {
	if r.inst == nil {
		return nil
	}
	if r.cls != nil {
		_, _ = r.cls.Call(r.ctx)
	}
	err := r.inst.Close(r.ctx)
	r.mod = nil
	r.err = errClosed
	putBuf(r.buf)
	r.buf = nil
	return err
}

func (r *reader) next() ([]byte, error) {
	hdrSize := 12 + r.hdr.hashLen()
	hdr := getBuf(hdrSize)
	n, err := io.ReadFull(r.src, hdr)
	if n == 0 && (err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF)) {
		putBuf(hdr)
		return nil, io.EOF
	}
	if err != nil {
		putBuf(hdr)
		return nil, fmt.Errorf("srep: block header: %w", err)
	}
	dataSize := binary.LittleEndian.Uint32(hdr[0:4])
	origSize := binary.LittleEndian.Uint32(hdr[4:8])
	statSize := binary.LittleEndian.Uint32(hdr[8:12])
	if dataSize == 0 && origSize == 0 {
		putBuf(hdr)
		return nil, io.EOF
	}
	if origSize > maxBlock || dataSize > maxBlock || statSize > maxBlock {
		putBuf(hdr)
		// fg-01: last literal block is followed by a FreeArc trailer
		// that is not an SREP header. After a successful stream, stop.
		if r.start > 0 {
			return nil, io.EOF
		}
		return nil, errTooLarge
	}
	putBuf(hdr)
	stat := getBuf(int(statSize))
	if statSize > 0 {
		if _, err := io.ReadFull(r.src, stat); err != nil {
			putBuf(stat)
			return nil, fmt.Errorf("srep: stat: %w", err)
		}
	}
	lits := getBuf(int(dataSize))
	if dataSize > 0 {
		if _, err := io.ReadFull(r.src, lits); err != nil {
			putBuf(stat)
			putBuf(lits)
			return nil, fmt.Errorf("srep: lits: %w", err)
		}
	}
	out, err := r.decode(stat, lits, origSize)
	putBuf(stat)
	putBuf(lits)
	if err != nil {
		return nil, err
	}
	r.start += uint64(origSize)
	return out, nil
}

func (r *reader) decode(stat, lits []byte, orig uint32) ([]byte, error) {
	statPtr, err := r.alloc(uint32(len(stat)))
	if err != nil {
		return nil, err
	}
	defer r.drop(statPtr)
	litPtr, err := r.alloc(uint32(len(lits)))
	if err != nil {
		return nil, err
	}
	defer r.drop(litPtr)
	outPtr, err := r.alloc(orig)
	if err != nil {
		return nil, err
	}
	defer r.drop(outPtr)
	if len(stat) > 0 && !r.mem.Write(statPtr, stat) {
		return nil, errGuest
	}
	if len(lits) > 0 && !r.mem.Write(litPtr, lits) {
		return nil, errGuest
	}
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	res, err := r.block.Call(r.ctx,
		uint64(statPtr), uint64(len(stat)),
		uint64(litPtr), uint64(len(lits)),
		uint64(outPtr), uint64(orig),
		r.start)
	if err != nil {
		return nil, fmt.Errorf("srep: block: %w", err)
	}
	if res[0] != 0 {
		return nil, errBroken
	}
	src, ok := r.mem.Read(outPtr, orig)
	if !ok {
		return nil, errGuest
	}
	out := getBuf(int(orig))
	copy(out, src)
	return out, nil
}

func (r *reader) alloc(n uint32) (uint32, error) {
	if n == 0 {
		return 0, nil
	}
	res, err := r.malloc.Call(r.ctx, uint64(n))
	if err != nil {
		return 0, fmt.Errorf("srep: malloc: %w", err)
	}
	ptr := uint32(res[0])
	if ptr == 0 {
		return 0, errGuest
	}
	return ptr, nil
}

func (r *reader) drop(ptr uint32) {
	if ptr == 0 {
		return
	}
	_, _ = r.free.Call(r.ctx, uint64(ptr))
}

func (h Header) hashLen() int {
	return int(h.HashExtra+16) & 255
}
