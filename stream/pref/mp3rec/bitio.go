package mp3rec

// Bit writer translated from packjpg bitops.cpp (abitwriter) and the
// packMP3 huffman_writer encode path.

type bitWriter struct {
	data  []byte
	cbyte int
	cbit  int
	fill  byte
}

func newBitWriter(size int) *bitWriter {
	if size <= 0 {
		size = 5 * 1024 * 1024
	}
	return &bitWriter{
		data: make([]byte, size),
		cbit: 8,
		fill: 1,
	}
}

func (w *bitWriter) grow() {
	n := make([]byte, len(w.data)*2+64)
	copy(n, w.data)
	w.data = n
}

func (w *bitWriter) ensure() {
	if w.cbyte+8 >= len(w.data) {
		w.grow()
	}
}

func rbits32(c uint32, n int32) uint32 {
	if n <= 0 {
		return 0
	}
	if n >= 32 {
		return c
	}
	return c & (0xFFFFFFFF >> uint(32-n))
}

func (w *bitWriter) write(val uint32, nbits int32) {
	if nbits <= 0 {
		return
	}
	w.ensure()
	for nbits >= int32(w.cbit) {
		top := rbits32(val, nbits) >> uint(nbits-int32(w.cbit))
		w.data[w.cbyte] |= byte(top)
		nbits -= int32(w.cbit)
		w.cbyte++
		w.cbit = 8
		w.ensure()
	}
	if nbits > 0 {
		w.data[w.cbyte] |= byte(rbits32(val, nbits) << uint(w.cbit-int(nbits)))
		w.cbit -= int(nbits)
	}
}

func (w *bitWriter) writeBit(bit byte) {
	w.ensure()
	w.cbit--
	if bit != 0 {
		w.data[w.cbyte] |= 1 << uint(w.cbit)
	}
	if w.cbit == 0 {
		w.cbyte++
		w.ensure()
		w.cbit = 8
	}
}

func (w *bitWriter) pad(fill byte) {
	for w.cbit < 8 {
		w.write(uint32(fill), 1)
	}
}

func (w *bitWriter) pos() int { return w.cbyte }

func (w *bitWriter) bytes() []byte {
	w.pad(w.fill)
	out := make([]byte, w.cbyte)
	copy(out, w.data[:w.cbyte])
	return out
}

type huffW struct {
	w     *bitWriter
	count int
}

func newHuffW() *huffW {
	return &huffW{w: newBitWriter(64 * 1024)}
}

func (h *huffW) writeBits(bits uint32, n int32) {
	h.w.write(bits, n)
	h.count += int(n)
}

func (h *huffW) writeBit(bit byte) {
	h.w.writeBit(bit)
	h.count++
}

func (h *huffW) reset() { h.count = 0 }

func (h *huffW) encodePair(rows [][]huffCode, vals []byte) error {
	v0 := int(vals[0])
	v1 := int(vals[1])
	if v0 < 0 || v0 >= len(rows) || v1 < 0 || v1 >= len(rows[v0]) {
		return errf("huffman pair %d,%d out of table", v0, v1)
	}
	hc := rows[v0][v1]
	h.writeBits(hc.code, int32(hc.len))
	return nil
}

func (h *huffW) encodeQuad(tab *[16]huffCode, vals []byte) {
	bits := 0
	for i := 0; i < 4; i++ {
		bits <<= 1
		if vals[i] != 0 {
			bits |= 1
		}
	}
	hc := tab[bits]
	h.writeBits(hc.code, int32(hc.len))
}
