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
//	ones-complement if neg, por {1,1,1,1} at 0x14000a3e0,
//	paddd old twice, psrld 1, keep low byte, store at +0x18.
//
// Callee never writes +0x20. Caller 3-stage shift of the packed
// qwords: w08←w10←w20←(+0x18). applySample does that by using
// +0x18 as the temp and copying it onto +0x20.

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

// extraBits is the depth-2 FCM tree at 0x14001af1b..0x14001b14f.
// After the 9-sym nibble (add r12d, 0x200 + bsf):
//
//	bsf==1 (sym 0): r10=r13=0, skip the loop
//	else base = (h1<<11) + ((sym-1)<<8)
//	     rdx = 0
//	     for i := 0; i < sym; i++ {
//	         p0 = base + i*32 + 8*rdx
//	         bit0 = getBit(p0)
//	         bit1 = getBit(p0 + 2 + 2*bit0)
//	         r10 = r10*2 + bit0
//	         r13 = r13*2 + bit1
//	         rdx = bit1 + 2*bit0
//	     }
//
// Those two integers are the IIR sample packed as r13<<32|r10.
const extraBitsBytes = 9 << 11 // h1 is 0..8

func extraBitOff(h1, symMinus1, level, rdx int) int {
	return (h1<<11 + symMinus1<<8 + level*32 + 8*rdx) / 2
}

func (h *iirHist) extraSample(st *rANS, bits []uint16, bsf, h1 int) (int, int, error) {
	if bsf <= 1 {
		h.applySample(0, 0)
		return 0, 0, nil
	}
	sym := bsf - 1
	if sym > 8 {
		sym = 8
	}
	if h1 < 0 {
		h1 = 0
	}
	if h1 > 8 {
		h1 = 8
	}
	r10, r13 := 0, 0
	rdx := 0
	for i := 0; i < sym; i++ {
		off := extraBitOff(h1, sym-1, i, rdx)
		if off < 0 || off+2 >= len(bits) {
			return 0, 0, errBitstream
		}
		b0, err := st.getBit(&bits[off])
		if err != nil {
			return 0, 0, err
		}
		b1, err := st.getBit(&bits[off+1+b0])
		if err != nil {
			return 0, 0, err
		}
		r10 = r10*2 + b0
		r13 = r13*2 + b1
		rdx = b1 + 2*b0
	}
	h.applySample(uint32(r10), uint32(r13))
	return r10, r13, nil
}

// applySample is the callee store at +0x18 plus the caller 3-stage
// qword rotate: w08←w10←w20←mix.
func (h *iirHist) applySample(n0, n1 uint32) {
	mixed0, mixed1 := mixIIR(h.w20, h.w24, n0, n1)
	h.w08, h.w0c = h.w10, h.w14
	h.w10, h.w14 = h.w20, h.w24
	h.w18, h.w1c = mixed0, mixed1
	h.w20, h.w24 = mixed0, mixed1
}

func (h *iirHist) h1() int {
	return bitlen(uint32(uint8(abs32(int32(h.w20) - int32(h.w24)))))
}

func newExtraBits() []uint16 {
	p := make([]uint16, extraBitsBytes/2)
	for i := range p {
		initBit(&p[i])
	}
	return p
}
