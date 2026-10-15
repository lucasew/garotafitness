package xdelta

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/adler32"
	"io"
)

// VCDIFF window decode follows third_party/xdelta3 v3.1.0 built with
// XD3_USE_LARGESIZET=0: sizes are uint32, copy offsets are uint64, the
// default RFC 3284 code table is the only one, and secondary compressors
// are off. Checksums are verified when the window asks for them.

const (
	opNoop = 0
	opAdd  = 1
	opRun  = 2
	opCpy  = 3

	vcdSecondary = 1 << 0
	vcdCodeTable = 1 << 1
	vcdAppHeader = 1 << 2

	vcdSource = 1 << 0
	vcdTarget = 1 << 1
	vcdAdler  = 1 << 2

	hardMaxWin = 1 << 26

	nearModes = 4
	sameModes = 3
	cpyModes  = 2 + nearModes + sameModes

	u32Oflow = 0xfe000000
	u64Oflow = 0xfe00000000000000
)

type codeEntry struct {
	t1, s1, t2, s2 byte
}

type halfInst struct {
	typ  byte
	size uint32
	addr uint32
}

type addrCache struct {
	near     [nearModes]uint32
	same     [sameModes * 256]uint32
	nextSlot int
}

var codeTable = buildCodeTable()

func buildCodeTable() [256]codeEntry {
	var tbl [256]codeEntry
	i := 0
	tbl[i].t1 = opRun
	i++
	tbl[i].t1 = opAdd
	i++
	for size := 1; size <= 17; size++ {
		tbl[i].t1 = opAdd
		tbl[i].s1 = byte(size)
		i++
	}
	for mode := 0; mode < cpyModes; mode++ {
		tbl[i].t1 = opCpy + byte(mode)
		i++
		for size := 4; size < 4+15; size++ {
			tbl[i].t1 = opCpy + byte(mode)
			tbl[i].s1 = byte(size)
			i++
		}
	}
	for mode := 0; mode < cpyModes; mode++ {
		max := 4
		if mode < 2+nearModes {
			max = 6
		}
		for add := 1; add <= 4; add++ {
			for size := 4; size <= max; size++ {
				tbl[i].t1 = opAdd
				tbl[i].s1 = byte(add)
				tbl[i].t2 = opCpy + byte(mode)
				tbl[i].s2 = byte(size)
				i++
			}
		}
	}
	for mode := 0; mode < cpyModes; mode++ {
		tbl[i].t1 = opCpy + byte(mode)
		tbl[i].s1 = 4
		tbl[i].t2 = opAdd
		tbl[i].s2 = 1
		i++
	}
	if i != 256 {
		panic("xdelta: code table length")
	}
	return tbl
}

// ApplyStream decodes a VCDIFF patch. oldSize 0 means the patch has no
// source, matching the guest (it called xd3_set_source only when the source
// length was non-zero). One target window is buffered, at most 64 MiB.
func ApplyStream(ctx context.Context, old io.ReaderAt, oldSize int64, diff io.ReaderAt, diffSize int64, out io.WriterAt) error {
	if diffSize <= 0 {
		return fmt.Errorf("xdelta: empty patch")
	}
	if oldSize < 0 {
		oldSize = 0
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	p := cursor{r: diff, n: diffSize}
	if err := readHeader(&p); err != nil {
		return err
	}
	var outPos int64
	for p.off < p.n {
		if err := ctx.Err(); err != nil {
			return err
		}
		win, err := readWindow(&p)
		if err != nil {
			return err
		}
		if win.winInd&vcdSource != 0 && (old == nil || oldSize == 0) {
			return fmt.Errorf("xdelta: source input required")
		}
		buf := make([]byte, win.tgtLen)
		if err := emitWindow(buf, &win, old, oldSize); err != nil {
			return err
		}
		if win.winInd&vcdAdler != 0 && adler32.Checksum(buf) != win.adler {
			return fmt.Errorf("xdelta: target window checksum mismatch")
		}
		if len(buf) > 0 {
			n, err := out.WriteAt(buf, outPos)
			if err != nil {
				return err
			}
			if n != len(buf) {
				return fmt.Errorf("xdelta: short write")
			}
		}
		if outPos > int64(^uint64(0)>>1)-int64(len(buf)) {
			return fmt.Errorf("xdelta: decoder file offset overflow")
		}
		outPos += int64(len(buf))
	}
	return nil
}

type cursor struct {
	r   io.ReaderAt
	n   int64
	off int64
}

func (c *cursor) readFull(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	if c.off > c.n || int64(len(b)) > c.n-c.off {
		return io.ErrUnexpectedEOF
	}
	n, err := c.r.ReadAt(b, c.off)
	c.off += int64(n)
	if n == len(b) {
		return nil
	}
	if err != nil {
		return err
	}
	return io.ErrUnexpectedEOF
}

func (c *cursor) u8() (byte, error) {
	var b [1]byte
	if err := c.readFull(b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

func (c *cursor) u32() (uint32, error) {
	v, err := c.uint(u32Oflow)
	return uint32(v), err
}

func (c *cursor) u64() (uint64, error) {
	return c.uint(u64Oflow)
}

func (c *cursor) uint(oflow uint64) (uint64, error) {
	var val uint64
	for {
		if c.off >= c.n {
			return 0, io.ErrUnexpectedEOF
		}
		if val&oflow != 0 {
			return 0, fmt.Errorf("xdelta: overflow in read_integer")
		}
		b, err := c.u8()
		if err != nil {
			return 0, err
		}
		val = (val << 7) | uint64(b&127)
		if b < 128 {
			return val, nil
		}
	}
}

func readHeader(c *cursor) error {
	var magic [4]byte
	if err := c.readFull(magic[:]); err != nil {
		if err == io.ErrUnexpectedEOF {
			return fmt.Errorf("xdelta: unsupported VCDIFF header")
		}
		return err
	}
	if magic != [4]byte{0xd6, 0xc3, 0xc4, 0} {
		return fmt.Errorf("xdelta: unsupported VCDIFF header")
	}
	ind, err := c.u8()
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			return fmt.Errorf("xdelta: unsupported VCDIFF header")
		}
		return err
	}
	if ind&^vcdAppHeader != 0 || ind&vcdSecondary != 0 || ind&vcdCodeTable != 0 {
		return fmt.Errorf("xdelta: unsupported VCDIFF header")
	}
	if ind&vcdAppHeader == 0 {
		return nil
	}
	n, err := c.u32()
	if err != nil {
		return err
	}
	if int64(n) > c.n-c.off {
		return io.ErrUnexpectedEOF
	}
	c.off += int64(n)
	return nil
}

type window struct {
	winInd byte
	cpyLen uint32
	cpyOff uint64
	tgtLen uint32
	adler  uint32
	data   []byte
	inst   []byte
	addr   []byte
}

func readWindow(c *cursor) (window, error) {
	var w window
	ind, err := c.u8()
	if err != nil {
		return w, err
	}
	w.winInd = ind
	if ind&^7 != 0 || ind&3 == 3 {
		return w, fmt.Errorf("xdelta: invalid window flags")
	}
	if ind&vcdTarget != 0 {
		return w, fmt.Errorf("xdelta: VCD_TARGET not implemented")
	}
	if ind&vcdSource != 0 {
		n, err := c.u32()
		if err != nil {
			return w, err
		}
		w.cpyLen = n
		w.cpyOff, err = c.u64()
		if err != nil {
			return w, err
		}
		if uint64(w.cpyLen) > ^uint64(0)-w.cpyOff {
			return w, fmt.Errorf("xdelta: decoder copy window overflows a file offset")
		}
	}
	encLen, err := c.u32()
	if err != nil {
		return w, err
	}
	w.tgtLen, err = c.u32()
	if err != nil {
		return w, err
	}
	if w.tgtLen > hardMaxWin {
		return w, fmt.Errorf("xdelta: hard window size exceeded")
	}
	if uint64(w.cpyLen) > uint64(^uint32(0))-uint64(w.tgtLen) {
		return w, fmt.Errorf("xdelta: decoder target window overflows a usize_t")
	}
	del, err := c.u8()
	if err != nil {
		return w, err
	}
	if del != 0 {
		return w, fmt.Errorf("xdelta: invalid delta indicator bits set")
	}
	dataLen, err := c.u32()
	if err != nil {
		return w, err
	}
	instLen, err := c.u32()
	if err != nil {
		return w, err
	}
	addrLen, err := c.u32()
	if err != nil {
		return w, err
	}
	if ind&vcdAdler != 0 {
		var sum [4]byte
		if err := c.readFull(sum[:]); err != nil {
			return w, err
		}
		w.adler = binary.BigEndian.Uint32(sum[:])
	}
	want := uint64(1)
	want += uint64(sizeof32(w.tgtLen) + sizeof32(dataLen) + sizeof32(instLen) + sizeof32(addrLen))
	want += uint64(dataLen) + uint64(instLen) + uint64(addrLen)
	if ind&vcdAdler != 0 {
		want += 4
	}
	if uint64(encLen) != want {
		return w, fmt.Errorf("xdelta: incorrect encoding length")
	}
	rest := uint64(dataLen) + uint64(instLen) + uint64(addrLen)
	if c.off > c.n || rest > uint64(c.n-c.off) {
		return w, io.ErrUnexpectedEOF
	}
	w.data = make([]byte, dataLen)
	w.inst = make([]byte, instLen)
	w.addr = make([]byte, addrLen)
	if err := c.readFull(w.data); err != nil {
		return w, err
	}
	if err := c.readFull(w.inst); err != nil {
		return w, err
	}
	if err := c.readFull(w.addr); err != nil {
		return w, err
	}
	return w, nil
}

func sizeof32(n uint32) uint32 {
	if n < 1<<7 {
		return 1
	}
	if n < 1<<14 {
		return 2
	}
	if n < 1<<21 {
		return 3
	}
	if n < 1<<28 {
		return 4
	}
	return 5
}

func emitWindow(out []byte, w *window, old io.ReaderAt, oldSize int64) error {
	var cache addrCache
	here := w.cpyLen
	maxPos := w.cpyLen + w.tgtLen
	ip, dp, ap := 0, 0, 0
	dst := 0
	var h1, h2 halfInst
	for ip < len(w.inst) || h1.typ != opNoop || h2.typ != opNoop {
		if h1.typ == opNoop && h2.typ == opNoop {
			e := codeTable[w.inst[ip]]
			ip++
			h1 = halfInst{typ: e.t1, size: uint32(e.s1)}
			h2 = halfInst{typ: e.t2, size: uint32(e.s2)}
			if h1.typ != opNoop {
				if err := parseHalf(&h1, &here, w.cpyLen, maxPos, w.inst, &ip, w.addr, &ap, &cache); err != nil {
					return err
				}
			}
			if h2.typ != opNoop {
				if err := parseHalf(&h2, &here, w.cpyLen, maxPos, w.inst, &ip, w.addr, &ap, &cache); err != nil {
					return err
				}
			}
		}
		if h1.typ != opNoop {
			n, err := outputHalf(out, dst, &h1, w, &dp, old, oldSize)
			if err != nil {
				return err
			}
			dst += n
			h1.typ = opNoop
		}
		if h2.typ != opNoop {
			n, err := outputHalf(out, dst, &h2, w, &dp, old, oldSize)
			if err != nil {
				return err
			}
			dst += n
			h2.typ = opNoop
		}
	}
	if dst != len(out) {
		return fmt.Errorf("xdelta: wrong window length")
	}
	if dp != len(w.data) {
		return fmt.Errorf("xdelta: extra data section")
	}
	if ap != len(w.addr) {
		return fmt.Errorf("xdelta: extra address section")
	}
	return nil
}

func parseHalf(h *halfInst, here *uint32, cpyLen, maxPos uint32, inst []byte, ip *int, addr []byte, ap *int, cache *addrCache) error {
	if h.size == 0 {
		sz, n, err := readU32(inst[*ip:])
		if err != nil {
			return err
		}
		*ip += n
		h.size = sz
	}
	if h.typ >= opCpy {
		mode := uint32(h.typ - opCpy)
		a, n, err := decodeAddress(cache, *here, mode, addr[*ap:])
		if err != nil {
			return err
		}
		*ap += n
		h.addr = a
		if a >= *here {
			return fmt.Errorf("xdelta: address too large")
		}
		if a < cpyLen && uint64(a)+uint64(h.size) > uint64(cpyLen) {
			return fmt.Errorf("xdelta: size too large")
		}
	}
	if uint64(*here)+uint64(h.size) > uint64(maxPos) {
		return fmt.Errorf("xdelta: size too large")
	}
	*here += h.size
	return nil
}

func decodeAddress(cache *addrCache, here, mode uint32, addr []byte) (uint32, int, error) {
	sameStart := uint32(2 + nearModes)
	var val uint32
	n := 0
	if mode < sameStart {
		v, used, err := readU32(addr)
		if err != nil {
			return 0, 0, err
		}
		val = v
		n = used
		switch mode {
		case 0:
		case 1:
			val = here - val
		default:
			val += cache.near[mode-2]
		}
	} else {
		if len(addr) == 0 {
			return 0, 0, fmt.Errorf("xdelta: address underflow")
		}
		mode -= sameStart
		if mode >= sameModes {
			return 0, 0, fmt.Errorf("xdelta: address underflow")
		}
		val = cache.same[mode*256+uint32(addr[0])]
		n = 1
	}
	cache.near[cache.nextSlot] = val
	cache.nextSlot = (cache.nextSlot + 1) % nearModes
	cache.same[val%uint32(len(cache.same))] = val
	return val, n, nil
}

func readU32(b []byte) (uint32, int, error) {
	var val uint32
	for i, next := range b {
		if val&u32Oflow != 0 {
			return 0, 0, fmt.Errorf("xdelta: overflow in read_integer")
		}
		val = (val << 7) | uint32(next&127)
		if next < 128 {
			return val, i + 1, nil
		}
	}
	return 0, 0, io.ErrUnexpectedEOF
}

func outputHalf(out []byte, dst int, h *halfInst, w *window, dp *int, old io.ReaderAt, oldSize int64) (int, error) {
	take := int(h.size)
	if take > len(out)-dst {
		return 0, fmt.Errorf("xdelta: size too large")
	}
	switch h.typ {
	case opRun:
		if *dp >= len(w.data) {
			return 0, fmt.Errorf("xdelta: data underflow")
		}
		b := w.data[*dp]
		*dp++
		for i := 0; i < take; i++ {
			out[dst+i] = b
		}
	case opAdd:
		if take > len(w.data)-*dp {
			return 0, fmt.Errorf("xdelta: data underflow")
		}
		copy(out[dst:dst+take], w.data[*dp:*dp+take])
		*dp += take
	default:
		if h.addr < w.cpyLen {
			if take == 0 {
				return 0, nil
			}
			pos := w.cpyOff + uint64(h.addr)
			end := pos + uint64(take)
			if end < pos || pos > uint64(^uint64(0)>>1) || end > uint64(oldSize) {
				return 0, fmt.Errorf("xdelta: source file too short")
			}
			n, err := old.ReadAt(out[dst:dst+take], int64(pos))
			if n != take {
				if err != nil {
					return 0, err
				}
				return 0, fmt.Errorf("xdelta: source file too short")
			}
		} else {
			src := int(h.addr - w.cpyLen)
			for i := 0; i < take; i++ {
				out[dst+i] = out[src+i]
			}
		}
	}
	return take, nil
}

type sliceAt struct {
	buf []byte
	n   int
}

func (s *sliceAt) WriteAt(p []byte, pos int64) (int, error) {
	if pos != int64(s.n) {
		return 0, fmt.Errorf("xdelta: non-sequential write at %d want %d", pos, s.n)
	}
	if int(pos) > len(s.buf) || len(p) > len(s.buf)-int(pos) {
		return 0, fmt.Errorf("xdelta: target size mismatch")
	}
	copy(s.buf[pos:], p)
	s.n += len(p)
	return len(p), nil
}
