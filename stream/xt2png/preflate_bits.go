package xt2png

import "math/bits"

// LSB-first bit IO matching preflate BitInputStream / BitOutputStream
// for the widths this reencoder actually uses. uint32 shifts follow the
// x86/wasm mask (count & 31).

func bitLength(v uint32) uint32 {
	if v == 0 {
		return 0
	}
	return 32 - uint32(bits.LeadingZeros32(v))
}

func bitReverse(v uint32, n uint32) uint32 {
	if n == 0 || n > 32 {
		return 0
	}
	return bits.Reverse32(v) >> (32 - n)
}

func shl32(v, s uint32) uint32 { return v << (s & 31) }

func shr32(v, s uint32) uint32 { return v >> (s & 31) }

type bitReader struct {
	data []byte
	pos  int
}

func newBitReader(b []byte) *bitReader {
	return &bitReader{data: b}
}

func (r *bitReader) get(n uint32) uint32 {
	var v uint32
	for i := uint32(0); i < n; i++ {
		var bit uint32
		bi := r.pos >> 3
		if bi >= 0 && bi < len(r.data) && (r.data[bi]>>(uint(r.pos)&7))&1 == 1 {
			bit = 1
		}
		if i < 32 {
			v |= bit << i
		}
		r.pos++
	}
	return v
}

func (r *bitReader) getReverse(n uint32) uint32 {
	return bitReverse(r.get(n), n)
}

func (r *bitReader) skipToByte() {
	if r.pos&7 != 0 {
		r.pos += 8 - (r.pos & 7)
	}
}

// getVLI matches BitInputStream::getVLI. bits still in the current byte
// are 8-(bitPos%8), and 8 when the position is byte-aligned.
func (r *bitReader) getVLI() uint64 {
	var result, o uint64
	var s uint32
	bitsRemaining := uint32(8 - (r.pos & 7))
	limit := uint32(1) << (bitsRemaining - 1)
	for {
		c := r.get(bitsRemaining)
		if c < limit {
			return result + o + (uint64(c) << s)
		}
		result += uint64(c&(limit-1)) << s
		s += bitsRemaining - 1
		o = (o + 1) << (bitsRemaining - 1)
		bitsRemaining = 8
		limit = 128
	}
}

func (r *bitReader) copyBytes(n uint64) []byte {
	if r.pos&7 != 0 {
		return []byte{}
	}
	off := r.pos >> 3
	if off < 0 || off > len(r.data) || n == 0 {
		return []byte{}
	}
	avail := uint64(len(r.data) - off)
	if n > avail {
		n = avail
	}
	end := off + int(n)
	out := append([]byte(nil), r.data[off:end]...)
	r.pos = end * 8
	return out
}

type bitWriter struct {
	buf []byte
	acc uint64
	n   uint32
}

func (w *bitWriter) put(v uint32, n uint32) {
	if n == 0 {
		return
	}
	if n < 32 {
		v &= (1 << n) - 1
	}
	w.acc |= uint64(v) << (w.n & 63)
	w.n += n
	for w.n >= 8 {
		w.buf = append(w.buf, byte(w.acc))
		w.acc >>= 8
		w.n -= 8
	}
}

func (w *bitWriter) fillByte() {
	if w.n&7 != 0 {
		w.put(0, 8-(w.n&7))
	}
}

func (w *bitWriter) flush() {
	if w.n > 0 {
		w.buf = append(w.buf, byte(w.acc))
		w.acc = 0
		w.n = 0
	}
}

func (w *bitWriter) putBytes(p []byte) {
	w.flush()
	w.buf = append(w.buf, p...)
}
