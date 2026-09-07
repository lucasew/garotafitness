package magic2

// IIR history + 9×9 nibble CDF grid from cls-magic2_x64 0x14001adb0
// and the epilogue at 0x14001b15f.
//
// hashNibble returns a byte offset: h0*9*32 + h1*32. Each row is 16
// little-endian uint16 (32 bytes). h0,h1 are 0..8.
//
// After a nibble, 0x14001b15f mixes a packed (r10, r13<<32) sample
// into the qword at +0x20 (w20,w24):
//
//	abs32 each dword, por 1, paddd old twice, psrld 1, keep low byte
//	store at +0x18
//
// The caller is assumed to shift +0x18 into the longer history
// (w08←w10←w18, w0c←w14←w1c) and copy +0x18 onto +0x20.

const nibbleRows = 9 * 9

type iirHist struct {
	w08, w0c, w10, w14 uint32
	w18, w1c           uint32
	w20, w24           uint32
}

func (h *iirHist) row() int {
	off := hashNibble(h.w08, h.w0c, h.w10, h.w14, h.w20, h.w24)
	r := off / 32
	if r < 0 {
		return 0
	}
	if r >= nibbleRows {
		return nibbleRows - 1
	}
	return r
}

// mixIIR is 0x14001b15f: abs(new)|1 mixed with old, low byte kept.
func mixIIR(old0, old1, new0, new1 uint32) (uint32, uint32) {
	a0 := abs32(int32(new0)) | 1
	a1 := abs32(int32(new1)) | 1
	// paddd old twice, psrld 1 → (new + 2*old) >> 1
	o0 := (a0 + 2*old0) >> 1
	o1 := (a1 + 2*old1) >> 1
	return o0 & 0xff, o1 & 0xff
}

// afterNibble feeds the decoded nibble as the low sample (high=0),
// matching the simple path that zeros r13 and uses the symbol in r10
// before the xor-to-zero on the bsf==1 path we still apply the nibble
// itself so the history moves when the symbol is nonzero.
func (h *iirHist) afterNibble(sym int) {
	h.shift(uint32(uint8(int8(sym<<4)>>4)), 0)
}

func (h *iirHist) afterNibbleU(sym int) {
	h.shift(uint32(sym&0xf), 0)
}

func (h *iirHist) shift(n0, n1 uint32) {
	h.w08, h.w0c = h.w10, h.w14
	h.w10, h.w14 = h.w18, h.w1c
	h.w18, h.w1c = mixIIR(h.w20, h.w24, n0, n1)
	h.w20, h.w24 = h.w18, h.w1c
}

// afterByte feeds a full reconstructed byte as two stacked nibbles.
func (h *iirHist) afterByte(b byte) {
	h.afterNibble(int(b >> 4))
	h.afterNibble(int(b & 0xf))
}

func newNibbleGrid() []uint16 {
	g := make([]uint16, nibbleRows*16)
	for i := 0; i < nibbleRows; i++ {
		initNibbleCDF(g[i*16 : i*16+16])
	}
	return g
}

func (h *iirHist) cdf(grid []uint16) []uint16 {
	r := h.row()
	return grid[r*16 : r*16+16]
}
