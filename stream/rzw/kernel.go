package rzw

// RAZOR 1.00: dual u32 rANS, 16-bit refill, binary scale 12, CDF scale 14.
// Decoder 0x4022b0, token jump table 0x42a000, constructor 0x40c610.

const (
	ransLimit = 0xffff
	binScale  = 12
	binMask   = 1<<binScale - 1
	nibScale  = 14
	nibMask   = 1<<nibScale - 1
	nibSyms   = 16
	rolzSyms  = 32
	farSyms   = 22
	imgSyms   = 20
	adaptSh   = 7
	kindLit   = 4
	kindTok   = 0
	kindRolz  = 2
	kindFar   = 3
	kindRep   = 1
)

// Byte-class tables at 0x42a140 / 0x42a160 (rz 1.00 .rdata).
var classA = [256]byte{
	0, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 2,
	2, 4, 5, 5, 5, 4, 5, 5, 5, 4, 5, 5, 5, 5, 5, 4,
	5, 5, 5, 5, 5, 4, 5, 5, 5, 5, 5, 6, 6, 6, 6, 6,
	6, 7, 8, 8, 8, 7, 8, 8, 8, 7, 8, 8, 8, 8, 8, 7,
	8, 8, 8, 8, 8, 7, 8, 8, 8, 8, 8, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
	10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
}

var classB = [256]byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 2,
	2, 4, 5, 5, 5, 4, 5, 5, 5, 4, 5, 5, 5, 5, 5, 4,
	5, 5, 5, 5, 5, 4, 5, 5, 5, 5, 5, 6, 6, 6, 6, 6,
	6, 7, 8, 8, 8, 7, 8, 8, 8, 7, 8, 8, 8, 8, 8, 7,
	8, 8, 8, 8, 8, 7, 8, 8, 8, 8, 8, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
	10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
	12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12,
	12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12,
}

// 0x42a100 / 0x42a120 / 0x42a130 — recent-match slot maps.
var repDelta = [16]int16{0, 0, 0, 0, -1, 1, -2, 2, -3, 3, -1, 1, -2, 2, -3, 3}
var repIdx = [16]byte{0, 1, 2, 3, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1}
var repKind = [16]byte{0, 1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3}

// 0x42a0a0: a8 next-class, 14-byte rows indexed by ebx kind (0x403120).
var a8Tab = [5][14]byte{
	{13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13},
	{7, 7, 7, 7, 7, 7, 7, 10, 10, 10, 10, 10, 10, 10},
	{8, 8, 8, 8, 8, 8, 8, 11, 11, 11, 11, 11, 11, 11},
	{9, 9, 9, 9, 9, 9, 9, 12, 12, 12, 12, 12, 12, 12},
	{0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 4, 5, 6, 0},
}

// extraBits15 is 0x42b850: extra bit counts for a 16-symbol length.
var extraBits15 = [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 0}

// rolzExtra is 0x42b780 (obj 0x431880 / 0x42c140): extra[i]=i, 27 used of 32.
var rolzExtra = [32]byte{
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
	16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 0, 0, 0, 0, 0,
}

// resExtra16 is 0x42b740 (obj 0x431720 / +0xf618): signed-byte residual.
// widths sum to 256; bias 128 maps onto int8.
var resExtra16 = [16]byte{0, 0, 1, 2, 3, 4, 5, 6, 6, 5, 4, 3, 2, 1, 0, 0}

// imgExtra20 is 0x42b720 (1.03.7 file 0x29c60 / VA 0x42c460).
// ImagePred residual (0x4047ca / 1.03.7 0x404ae1); first 5 symbols extra-0, then 1..15.
var imgExtra20 = [20]byte{0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

// 0x42b7c0 / reconstructed bases for far-distance first symbol (ctor 0x4319e0).
var farExtra = [22]byte{8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29}
var farBase [22]int
var rolzBase [32]int
var resBase16 [16]int
var imgBase20 [20]int

// FSM next/remap from ctor 0x428110 (loop-back reconstruction of the
// 0x44f72f tail). next[s][0/1] at 0x431f60+2*s; remap at +512.
var fsmNext0 = [256]byte{
	0, 1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
	15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
	31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46,
	47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
	63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78,
	79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94,
	95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110,
	111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126,
	127, 128, 129, 130, 131, 132, 133, 134, 134, 135, 136, 137, 138, 139, 140, 141,
	142, 143, 144, 144, 145, 146, 147, 148, 149, 150, 151, 152, 152, 153, 154, 155,
	156, 157, 158, 158, 159, 160, 161, 162, 162, 163, 164, 165, 166, 166, 167, 168,
	169, 170, 170, 171, 172, 173, 173, 174, 175, 176, 176, 177, 178, 178, 179, 180,
	180, 181, 182, 182, 183, 184, 184, 185, 186, 186, 187, 187, 188, 188, 189, 190,
	190, 191, 191, 192, 192, 193, 193, 194, 194, 195, 195, 196, 196, 197, 197, 198,
	198, 198, 199, 199, 200, 200, 200, 201, 201, 201, 202, 202, 203, 203, 203, 203,
	204, 204, 204, 205, 205, 205, 206, 206, 206, 206, 207, 207, 207, 207, 207, 208,
}

var fsmNext1 = [256]byte{
	0, 48, 49, 49, 49, 49, 49, 50, 50, 50, 50, 51, 51, 51, 52, 52,
	52, 53, 53, 53, 53, 54, 54, 55, 55, 55, 56, 56, 56, 57, 57, 58,
	58, 58, 59, 59, 60, 60, 61, 61, 62, 62, 63, 63, 64, 64, 65, 65,
	66, 66, 67, 68, 68, 69, 69, 70, 70, 71, 72, 72, 73, 74, 74, 75,
	76, 76, 77, 78, 78, 79, 80, 80, 81, 82, 83, 83, 84, 85, 86, 86,
	87, 88, 89, 90, 90, 91, 92, 93, 94, 94, 95, 96, 97, 98, 98, 99,
	100, 101, 102, 103, 104, 104, 105, 106, 107, 108, 109, 110, 111, 112, 112, 113,
	114, 115, 116, 117, 118, 119, 120, 121, 122, 122, 123, 124, 125, 126, 127, 128,
	129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139, 140, 141, 142, 143, 144,
	145, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155, 156, 157, 158, 159, 160,
	161, 162, 163, 164, 165, 166, 167, 168, 169, 170, 171, 172, 173, 174, 175, 176,
	177, 178, 179, 180, 181, 182, 183, 184, 185, 186, 187, 188, 189, 190, 191, 192,
	193, 194, 195, 196, 197, 198, 199, 200, 201, 202, 203, 204, 205, 206, 207, 208,
	209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224,
	225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 235, 236, 237, 238, 239, 240,
	241, 242, 243, 244, 245, 246, 247, 248, 249, 250, 251, 252, 253, 254, 255, 255,
}

var fsmRemap = [256]byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 4, 4, 4, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5,
	5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5,
	5, 5, 5, 5, 5, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 6, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
}

type rans struct {
	s0, s1 uint32
	src    []byte
	off    int // initial state bytes; refills do not move this
	end    int // next unread u16 is src[end-2:end] (PE 0x418310)

	frames [][]byte
}

func (r *rans) refill() {
	// PE 0x418310: count--, s0 = s0<<16 | buf[count] (u16s from the tail).
	if r.end == 0 && len(r.frames) != 0 {
		p := r.frames[0]
		r.frames = r.frames[1:]
		if len(p) < 8 || len(p)%2 != 0 {
			r.off = len(r.src) + 1
			return
		}
		r.src = p
		r.end = len(p) - 8
		r.s0 = uint32(p[len(p)-4]) | uint32(p[len(p)-3])<<8 | uint32(p[len(p)-2])<<16 | uint32(p[len(p)-1])<<24
		r.s1 = uint32(p[len(p)-8]) | uint32(p[len(p)-7])<<8 | uint32(p[len(p)-6])<<16 | uint32(p[len(p)-5])<<24
		return
	}
	if r.end < 2 {
		r.off = len(r.src) + 1
		return
	}
	r.end -= 2
	r.s0 = r.s0<<16 | uint32(r.src[r.end]) | uint32(r.src[r.end+1])<<8
}

func (r *rans) swap() {
	r.s0, r.s1 = r.s1, r.s0
	if r.s0 <= ransLimit {
		r.refill()
	}
}

func (r *rans) ok() bool {
	return r.off <= len(r.src) && r.end >= 0
}

func (r *rans) bits(n int) uint32 {
	if n > 16 {
		hi := r.bits(n - 16)
		return hi<<16 | r.bits(16)
	}
	r.swap()
	if !r.ok() || n <= 0 {
		return 0
	}
	mask := uint32(1<<uint(n) - 1)
	v := r.s0 & mask
	r.s0 >>= uint(n)
	return v
}

func lengthFromExtra(r *rans, extra []byte, sym int) int {
	if sym < 0 {
		sym = 0
	}
	if sym >= len(extra) {
		sym = len(extra) - 1
	}
	base := 2
	for i := 0; i < sym; i++ {
		base += 1 << extra[i]
	}
	n := base
	if e := int(extra[sym]); e > 0 {
		n += int(r.bits(e))
	}
	if n < 1 {
		n = 1
	}
	return n
}

func lengthFromSym(r *rans, sym int) int {
	return lengthFromExtra(r, extraBits15[:], sym)
}

type binModel uint16

func (m *binModel) bitSh(r *rans, sh uint) int {
	r.swap()
	if !r.ok() {
		return 0
	}
	freq := uint32(*m) >> 4
	if freq == 0 {
		freq = 1
	}
	if freq > binMask {
		freq = binMask
	}
	slot := r.s0 & binMask
	quo := r.s0 >> binScale
	if slot < freq {
		r.s0 = freq*quo + slot
		neg := uint16(-int16(*m))
		*m = binModel(uint16(*m) + neg>>sh)
		if *m == 0 {
			*m = 1 << 4
		}
		return 1
	}
	r.s0 = r.s0 - freq*(quo+1)
	*m = binModel(uint16(*m) - uint16(*m)>>sh)
	if *m < 1<<4 {
		*m = 1 << 4
	}
	return 0
}

func (m *binModel) bit(r *rans) int {
	// PE 0x4023ef: cmp freq, slot; jbe match. Literal only when slot < freq.
	return m.bitSh(r, adaptSh)
}

type nibModel [16]uint16

func (m *nibModel) init() {
	for i := range m {
		m[i] = uint16(i << (nibScale - 4))
	}
}

func (m *nibModel) sym(r *rans) int {
	r.swap()
	if !r.ok() {
		return 0
	}
	slot := int(r.s0&nibMask) + 1
	s := 0
	for i := 15; i >= 0; i-- {
		if int(m[i]) < slot {
			s = i
			break
		}
	}
	start := uint32(m[s])
	end := uint32(m[(s+1)&15])
	freq := (end - start) & nibMask
	if freq == 0 {
		freq = 1
	}
	r.s0 = (r.s0>>nibScale)*freq + ((r.s0 - start) & nibMask)
	m.adapt(s)
	return s
}

var nibTgt16 [16][16]uint16
var rolzTgt32 [32][32]uint16
var farTgt24 [24][24]uint16

func fillTgt(dst []uint16, s, n int) {
	for i := 0; i < n; i++ {
		if i <= s {
			dst[i] = uint16(i)
		} else {
			dst[i] = uint16(0x4000 + 0x7f - n + i)
		}
	}
}

func init() {
	for s := 0; s < 16; s++ {
		fillTgt(nibTgt16[s][:], s, 16)
	}
	for s := 0; s < 32; s++ {
		fillTgt(rolzTgt32[s][:], s, 32)
	}
	for s := 0; s < 24; s++ {
		fillTgt(farTgt24[s][:], s, 22)
		farTgt24[s][22] = 16384
		farTgt24[s][23] = 16384
	}
	b := 0
	for i := range farBase {
		farBase[i] = b
		b += 1 << farExtra[i]
	}
	b = 0
	for i := range rolzBase {
		rolzBase[i] = b
		b += 1 << rolzExtra[i]
	}
	b = 0
	for i := range resBase16 {
		resBase16[i] = b
		b += 1 << resExtra16[i]
	}
	b = 0
	for i := range imgBase20 {
		imgBase20[i] = b
		b += 1 << imgExtra20[i]
	}
}

func (m *nibModel) adapt(s int) {
	if s < 0 {
		s = 0
	}
	if s > 15 {
		s = 15
	}
	tgt := nibTgt16[s]
	for i := range m {
		d := int16(tgt[i]) - int16(m[i])
		m[i] = uint16(int16(m[i]) + (d >> adaptSh))
	}
}

type rolzModel [32]uint16

func (m *rolzModel) init() {
	for i := range m {
		m[i] = uint16(i << (nibScale - 5))
	}
}

func (m *rolzModel) adapt(s int) {
	if s < 0 {
		s = 0
	}
	if s > 31 {
		s = 31
	}
	tgt := rolzTgt32[s]
	for i := range m {
		d := int16(tgt[i]) - int16(m[i])
		m[i] = uint16(int16(m[i]) + (d >> adaptSh))
	}
}

func (m *rolzModel) sym(r *rans) int {
	r.swap()
	if !r.ok() {
		return 0
	}
	slot := int(r.s0&nibMask) + 1
	s := 0
	for i := 31; i >= 0; i-- {
		if int(m[i]) < slot {
			s = i
			break
		}
	}
	start := uint32(m[s])
	end := uint32(m[(s+1)&31])
	freq := (end - start) & nibMask
	if freq == 0 {
		freq = 1
	}
	r.s0 = (r.s0>>nibScale)*freq + ((r.s0 - start) & nibMask)
	m.adapt(s)
	return s
}

type farModel [24]uint16

func (m *farModel) init() {
	weights := [...]byte{4, 4, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	total, sum := 0, 0
	for _, w := range weights {
		total += int(w)
	}
	for i, w := range weights {
		m[i] = uint16(sum * 16384 / total)
		sum += int(w)
	}
	m[22] = 16384
	m[23] = 16384
}

func (m *farModel) adapt(s int) {
	if s < 0 {
		s = 0
	}
	if s > farSyms-1 {
		s = farSyms - 1
	}
	tgt := farTgt24[s]
	for i := range m {
		d := int16(tgt[i]) - int16(m[i])
		m[i] = uint16(int16(m[i]) + (d >> adaptSh))
	}
}

func (m *farModel) sym(r *rans) int {
	r.swap()
	if !r.ok() {
		return 0
	}
	slot := int(r.s0&nibMask) + 1
	s := 0
	for i := farSyms - 1; i >= 0; i-- {
		if int(m[i]) < slot {
			s = i
			break
		}
	}
	start := uint32(m[s])
	end := uint32(m[(s+1)%farSyms])
	freq := (end - start) & nibMask
	if freq == 0 {
		freq = 1
	}
	r.s0 = (r.s0>>nibScale)*freq + ((r.s0 - start) & nibMask)
	m.adapt(s)
	return s
}

type dec struct {
	overflow      bool
	mono8         audio8
	stereo8       audio8
	mono16        audio8
	stereo16      audio8
	color         [2][3][24]uint16
	alpha         nibModel
	literalRun    [17]nibModel
	literalRunEnd int
	matches       [256][]matchRecord
	matchLen      [48]farModel
	farLow        [22]nibModel
	farHigh       [22]nibModel
	farLen        [56]nibModel
	repSlot       [14]nibModel
	repLen        [28]nibModel
	u8            [4][8]nibModel
	u16x2         [4]rolzModel
	u16           [2]rolzModel
	u32           [2]rolzModel
	r             rans
	out           []byte
	pos           int
	a8            int
	reps          [4]int
	bin           []binModel
	nibLit        []nibModel // 0x11c0: [13][81] unmatched+predicted
	tok           nibModel   // 0x9ac0 token type

	far24 farModel // 0x9ae0 24-sym first far symbol

	rolz32 []rolzModel // 0xb5d0: 32-sym per prev byte

	img imgPred

	stRec [320][4]byte

	why string
}

func newDec(src []byte, cap int) *dec {
	d := &dec{
		out:    make([]byte, 0, cap),
		bin:    make([]binModel, 0x300),
		nibLit: make([]nibModel, 13*81),
		rolz32: make([]rolzModel, 256),
	}
	d.r.src = src
	d.r.frames = [][]byte{src}
	for i := range d.bin {
		d.bin[i] = 0x8000
	}
	for a := 0; a < 8; a++ {
		for b := 0; b < 8; b++ {
			d.bin[0x270+a*8+b] = binModel((a + b + 1) * 0x1000)
		}
	}

	for i := range d.nibLit {
		d.nibLit[i].init()
	}
	// The constructor initializes the token CDF uniformly (0x40ca38).
	d.tok.init()
	d.mono8.init()
	d.stereo8.init()
	d.mono16.init()
	d.stereo16.init()
	d.img.init()
	for i := range d.color {
		for j := range d.color[i] {
			for k := range d.color[i][j] {
				d.color[i][j][k] = uint16(min(k*16384/17, 16384))
			}
		}
	}
	d.alpha.init()
	for i := range d.literalRun {
		d.literalRun[i].init()
	}

	d.far24.init()
	for i := range d.farLow {
		d.farLow[i].init()
		d.farHigh[i].init()
	}
	for i := range d.farLen {
		d.farLen[i].initWeights([]byte{1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4})
	}
	for i := range d.rolz32 {
		initCDF(d.rolz32[i][:], []byte{8, 8, 8, 8, 8, 8, 8, 8, 4, 4, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	}

	for i := range d.matchLen {
		initCDF(d.matchLen[i][:], []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 8, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	}
	for i := range d.stRec {
		d.stRec[i] = [4]byte{0x80, 0x80, 0x80, 0x80}
	}
	// The final constructor pass replaces these uniform CDFs with priors
	// (0x40ddd1 and 0x40de41), after initializing the other token models.
	for i := range d.repSlot {
		d.repSlot[i].initWeights([]byte{64, 1, 1, 1, 8, 8, 8, 8, 4, 4, 1, 1, 1, 1, 1, 1})
	}
	for i := range d.repLen {
		d.repLen[i].initWeights([]byte{1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4})
	}
	for i := range d.u8 {
		for j := range d.u8[i] {
			d.u8[i][j].init()
		}
	}
	for i := range d.u16x2 {
		d.u16x2[i].init()
	}
	for i := range d.u16 {
		d.u16[i].init()
	}
	for i := range d.u32 {
		for j := range d.u32[i] {
			d.u32[i][j] = uint16(j * 16384 / 31)
		}
	}
	d.reps = [4]int{1, 2, 3, 4}
	return d
}

func (d *dec) prev() byte {
	if d.pos == 0 {
		return 0
	}
	return d.out[d.pos-1]
}

func (d *dec) prevSlot() *[4]byte {
	return &d.stRec[int(d.prev())]
}

func (d *dec) posSlot() *[4]byte {
	a8 := d.a8
	if a8 < 0 {
		a8 = 0
	}
	if a8 > 13 {
		a8 = 13
	}
	idx := 0x100 + (d.pos & 3) + a8*4
	if idx >= len(d.stRec) {
		idx = 0x100
	}
	return &d.stRec[idx]
}

func (d *dec) binAt(sa, sb byte) *binModel {
	idx := 0x270 + int(fsmRemap[sb]) + int(fsmRemap[sa])*8
	if idx < 0 || idx >= len(d.bin) {
		idx = 0x270
	}
	return &d.bin[idx]
}

func saNext(s byte, bit int) byte {
	if bit != 0 {
		return fsmNext1[s]
	}
	return fsmNext0[s]
}

func (d *dec) stepA8(kind int) {
	if kind < 0 || kind > 4 {
		kind = 0
	}
	a := d.a8
	if a < 0 {
		a = 0
	}
	if a > 13 {
		a = 13
	}
	d.a8 = int(a8Tab[kind][a])

}

func (d *dec) decodeByte() bool {
	if !d.r.ok() {
		d.why = "rANS exhausted"
		return false
	}
	if d.pos >= cap(d.out) {
		d.why = "output limit"
		return false
	}
	ps := d.prevSlot()
	qs := d.posSlot()
	m := d.binAt(ps[0], qs[0])
	off0 := d.r.off
	s00, s10 := d.r.s0, d.r.s1
	lit := m.bit(&d.r)
	if !d.r.ok() {
		d.r.off, d.r.s0, d.r.s1 = off0, s00, s10
		d.why = "rANS bit"
		return false
	}
	ps[0] = saNext(ps[0], lit)
	qs[0] = saNext(qs[0], lit)
	if lit == 1 {

		if !d.literal() {
			d.why = "lit"
			return false
		}

		d.stepA8(kindLit)
		return true
	}

	if !d.match() {
		if d.why == "" {
			d.why = "match"
		}
		return false
	}
	return true
}

func (d *dec) literal() bool {
	prev := d.prev()
	cls := int(classB[prev])
	if cls > 12 {
		cls = 12
	}
	ca := int(classA[d.a8&255])
	var b byte
	if ca == 0 {
		// Unmatched: [0]=hi, [1..16]=lo[hi]. PE 0x4024ec / +0x8f.
		hi := d.nibLit[cls*81].sym(&d.r)
		if !d.r.ok() {
			return false
		}
		lo := d.nibLit[cls*81+1+hi].sym(&d.r)
		if !d.r.ok() {
			return false
		}
		b = byte(hi<<4 | lo)
	} else {
		// Predicted: [17..32]/[33..48] hi, [49..64]/[65..80] lo.
		// Output is (hi<<4)|lo; XOR vs pred only selects the lo model.
		pred := byte(0)
		if dist := d.reps[0]; dist > 0 && d.pos-dist >= 0 {
			pred = d.out[d.pos-dist]
		}
		pcls := ca - 1
		if pcls > 1 {
			pcls = 1
		}
		hi := d.nibLit[cls*81+17+pcls*16+int(pred>>4)].sym(&d.r)
		if !d.r.ok() {
			return false
		}
		mix := pred ^ byte(hi<<4)
		var loIdx int
		if mix <= 0x0f {
			loIdx = cls*81 + 49 + pcls*16 + int(pred&0xf)
		} else {
			loIdx = cls*81 + 1 + hi
		}
		if loIdx >= len(d.nibLit) {
			loIdx = cls * 81
		}
		lo := d.nibLit[loIdx].sym(&d.r)
		if !d.r.ok() {
			return false
		}
		b = byte(hi<<4 | lo)
	}
	d.emit(b)
	return true
}

func (d *dec) match() bool {
	ps, qs := d.prevSlot(), d.posSlot()
	a := d.binAt(ps[1], qs[1]).bit(&d.r)
	ps[1] = saNext(ps[1], a)
	qs[1] = saNext(qs[1], a)
	j := a + 2
	b := d.binAt(ps[j], qs[j]).bit(&d.r)
	ps[j] = saNext(ps[j], b)
	qs[j] = saNext(qs[j], b)
	if !d.r.ok() {
		return false
	}
	switch 2*a + b {
	case 0:
		return d.token()
	case 1:
		return d.repMatch()
	case 2:
		return d.rolzMatch()
	default:
		return d.farMatch()
	}
}

func (d *dec) repMatch() bool {
	slot := d.repSlot[d.a8].sym(&d.r)
	dist := d.reps[repIdx[slot]] + int(repDelta[slot])
	k := int(repKind[slot]) ^ 1
	old := d.reps[k]
	if k > 1 {
		d.reps[3] = old
		d.reps[2] = d.reps[1]
		old = d.reps[0]
	}
	d.reps[1] = old
	d.reps[0] = dist
	ctx := 2 * d.a8
	if slot != 0 {
		ctx++
	}
	sym := d.repLen[ctx].sym(&d.r)
	n := lengthFromSym(&d.r, sym) - 1

	ok := d.copyHistory(dist, n)
	d.stepA8(kindRep)
	return ok && d.r.ok()
}

func (d *dec) copyHistory(dist, n int) bool {
	if dist < 1 || dist > d.pos+0xff0 || n < 1 || n > cap(d.out)-d.pos {
		d.why = "invalid match"
		return false
	}
	for i := 0; i < n; i++ {
		var b byte
		if d.pos >= dist {
			b = d.out[d.pos-dist]
		}
		d.emit(b)
	}
	return true
}

func (d *dec) farMatch() bool {
	slot := d.far24.sym(&d.r)
	low := d.farLow[slot].sym(&d.r)
	high := d.farHigh[slot].sym(&d.r)
	v := uint32(high) << uint(slot)
	if slot != 0 {
		v |= d.r.bits(slot)
	}
	dist := farBase[slot] + 1 + low + int(v<<4)
	d.reps = [4]int{dist, d.reps[0], d.reps[1], d.reps[2]}
	log := 0
	for n := dist; n > 1; n >>= 1 {
		log++
	}
	sym := d.farLen[d.a8*4+log/8].sym(&d.r)
	widths := [...]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6}
	n := lengthFromExtra(&d.r, widths[:], sym)

	start := d.pos
	ok := d.copyHistory(dist, n)
	if ok {
		d.recordMatch(start, n)
		d.recordMatch(start+1, n-1)
	}
	d.stepA8(kindFar)
	return ok && d.r.ok()
}

func (d *dec) token() bool {

	t := d.tok.sym(&d.r)
	if !d.r.ok() {
		d.why = "tok-sym"
		return false
	}

	ok := false
	switch t {
	case tokDelta1xU8:
		ok = d.tokDeltaU8(1)
	case tokDelta2xU8:
		ok = d.tokDeltaU8(2)
	case tokDelta3xU8:
		ok = d.tokDeltaU8(3)
	case tokDelta4xU8:
		ok = d.tokDeltaU8(4)
	case tokDelta1xU16:
		ok = d.delta16()
	case tokDelta2xU16:
		ok = d.delta16x2()
	case tokDelta1xU32:
		ok = d.delta32()
	case tokRGB:
		ok = d.tokRGB(3)
	case tokRGBA:
		ok = d.tokRGB(4)
	case tokImagePred:
		ok = d.tokImagePred()
	case tokMono8:
		ok = d.audioPCM8(1)
	case tokStereo8:
		ok = d.audioPCM8(2)
	case tokMono16:
		ok = d.audioPCM16(1)
	case tokStereo16:
		ok = d.audioPCM16(2)
	case tokLiterals32:
		ok = d.tokLit32()
	case tokRawBytes:
		ok = d.tokRaw()
	default:
		d.why = "tok-id"
		return false
	}
	if ok {
		d.stepA8(kindTok)
	}
	return ok && !d.overflow
}

func (d *dec) tokDeltaU8(step int) bool {
	n := 8
	if step == 3 {
		n = 9
	}
	for i := 0; i < n; i++ {
		prev := byte(d.wordBefore(1, step))
		sign := (prev - byte(d.wordBefore(1, 2*step))) >> 7
		ctx := 2*((n-1-i)%step) + int(sign)
		sym := d.u8[step-1][ctx].sym(&d.r)
		v := resBase16[sym]
		if e := resExtra16[sym]; e != 0 {
			v += int(d.r.bits(int(e)))
		}
		d.emit(prev + byte(v))
	}
	return d.r.ok()
}

func (d *dec) tokRGB(ch int) bool {
	// rz 1.00: 0x40404f / 0x404363. Three RGB or two RGBA pixels.
	widths := [2][17]byte{
		{7, 6, 5, 4, 3, 2, 1, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7},
		{8, 7, 6, 5, 4, 3, 2, 1, 0, 1, 2, 3, 4, 5, 6, 7, 8},
	}
	for i := 0; i < 12-ch; i += ch {
		var v [3]int
		for c := range v {
			w := widths[min(c, 1)]
			sym := decodeCDF(&d.r, d.color[ch-3][c][:], 17)
			v[c] = -255
			if c != 0 {
				v[c] = -510
			}
			for j := 0; j < sym; j++ {
				v[c] += 1 << w[j]
			}
			if w[sym] != 0 {
				v[c] += int(d.r.bits(int(w[sym])))
			}
		}
		g := v[0] - (v[2] >> 1)
		b := g - (v[1] >> 1)
		delta := [3]int{b + v[1], g + v[2], b}
		for _, v := range delta {
			d.emit(byte(d.wordBefore(1, ch)) + byte(v))
		}
		if ch == 4 {
			s := d.alpha.sym(&d.r)
			v := resBase16[s]
			if resExtra16[s] != 0 {
				v += int(d.r.bits(int(resExtra16[s])))
			}
			d.emit(byte(d.wordBefore(1, ch)) + byte(v))
		}
	}
	return d.r.ok()
}

func unzig(v int) int {
	// 0x404af7: v>>1, not; cmove if (v&1)==0 → even: v>>1, odd: ^(v>>1).
	if v&1 == 0 {
		return v >> 1
	}
	return ^(v >> 1)
}

func (d *dec) tokLit32() bool {
	start := max(d.literalRunEnd, d.pos-256)
	for _, b := range d.out[start:d.pos] {
		d.literalRun[0].adapt(int(b >> 4))
		d.literalRun[1+int(b>>4)].adapt(int(b & 15))
	}
	for i := 0; i < 32; i++ {
		hi := d.literalRun[0].sym(&d.r)
		lo := d.literalRun[1+hi].sym(&d.r)
		d.emit(byte(hi<<4 | lo))
	}
	d.literalRunEnd = d.pos
	return d.r.ok()
}

func (d *dec) tokRaw() bool {
	for i := 0; i < 16; i++ {
		v := d.r.bits(16)
		d.emit(byte(v))
		d.emit(byte(v >> 8))
	}
	return d.r.ok()
}

func (d *dec) emit(b byte) {
	if d.pos >= cap(d.out) {
		d.why = "output limit"
		d.overflow = true
		return
	}
	d.out = append(d.out, b)
	d.pos++
}
