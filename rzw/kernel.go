package rzw

// Native RAZOR 1.00 decoder lifted from rz.exe (22_pe_22, data only).
// Dual u32 rANS, 16-bit renorm, binary scale 12, nibble scale 14.
//
// Encode 0x4022b0 switch 0x40288c / table 0x42a000: tok9 0x404752.
// Decode 0x409050 switch 0x40a106 / table 0x42a054:
//   tok0 0x40aab2 Delta1xU8 … tok9 0x40b07c ImagePred,
//   tok14 0x40abb8 Literals32, tok15 0x40a8ae RawBytes.
// No dest write of dataSize/tableSize before the first token; pos 0x12750=0.
// Encode residuals at 0x40aab2..: pred = out[i-N], sign(prev Δ) context.
// ROLZ 32-sym CDFs at obj+0xb5d0 (0x4031c0). Far 24-sym at obj+0x9ae0.
// Binary: PE `cmp freq,slot; jbe match` — literal only when slot < freq.

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
	rolzHist  = 512
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

// imgExtra20 is 0x42b720 (obj 0x4315c0 via 0x42c150, n=20 at 0x4316fc).
// ImagePred residual at 0x4047ca; first 5 symbols extra-0, then 1..15.
var imgExtra20 = [20]byte{0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

// 0x42b7c0 / reconstructed bases for far-distance first symbol (ctor 0x4319e0).
var farExtra = [22]byte{8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29}
var farBase [22]int
var rolzBase [32]int
var resBase16 [16]int
var imgBase20 [20]int

const resBias16 = 0 // unsigned wrap; 0 and 255 sit on extra-0 symbols

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
	stay0  bool // after no-swap RawBytes: refill s0, do not swap
}

func (r *rans) refill() {
	// PE 0x418310: count--, s0 = s0<<16 | buf[count] (u16s from the tail).
	if r.end < 10 {
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
	return r.off <= len(r.src) && r.end >= 8
}

func (r *rans) bits(n int) uint32 {
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

// PE 0x41bb50: dst[0]=0, dst[i]=0x407f-n+i. First symbol is 0
// for almost every slot (not uniform i<<10).
func (m *nibModel) initTgt(s int) {
	n := len(m)
	if s < 0 {
		s = 0
	}
	if s >= n {
		s = n - 1
	}
	for i := 0; i < n; i++ {
		if i <= s {
			m[i] = uint16(i)
		} else {
			m[i] = uint16(0x407f - n + i)
		}
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
		// calloc zeros (0x40d96c). First hit is t15 (search from 15).
		// 0x41fd30 adapt is (tgt-cur)>>7; fillTgt(15) is 0..15 so
		// zeros stay zeros. Do not invent a uniform seed.
		freq = 1
	}
	r.s0 = (r.s0>>nibScale)*freq + ((r.s0 - start) & nibMask)
	m.adapt(s)
	return s
}

var nibTgt16 [16][16]uint16
var rolzTgt32 [32][32]uint16
var farTgt24 [24][24]uint16
var imgTgt20 [20][20]uint16

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
		fillTgt(farTgt24[s][:], s, 24)
	}
	for s := 0; s < imgSyms; s++ {
		fillTgt(imgTgt20[s][:], s, imgSyms)
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
	for i := 0; i < farSyms; i++ {
		m[i] = uint16(i * (1 << nibScale) / farSyms)
	}
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

// 20-sym ImagePred residual CDF (0x4315c0 n=20, scale 14, 0x404a19).
type imgModel [20]uint16

func (m *imgModel) init() {
	for i := 0; i < imgSyms; i++ {
		m[i] = uint16(i * (1 << nibScale) / imgSyms)
	}
}

func (m *imgModel) adapt(s int) {
	if s < 0 {
		s = 0
	}
	if s > imgSyms-1 {
		s = imgSyms - 1
	}
	tgt := imgTgt20[s]
	for i := range m {
		d := int16(tgt[i]) - int16(m[i])
		m[i] = uint16(int16(m[i]) + (d >> adaptSh))
	}
}

func (m *imgModel) sym(r *rans) int {
	r.swap()
	if !r.ok() {
		return 0
	}
	slot := int(r.s0&nibMask) + 1
	s := 0
	for i := imgSyms - 1; i >= 0; i-- {
		if int(m[i]) < slot {
			s = i
			break
		}
	}
	start := uint32(m[s])
	end := uint32(0)
	if s+1 < imgSyms {
		end = uint32(m[s+1])
	}
	freq := (end - start) & nibMask
	if freq == 0 {
		freq = 1
	}
	r.s0 = (r.s0>>nibScale)*freq + ((r.s0 - start) & nibMask)
	m.adapt(s)
	return s
}

type dec struct {
	r        rans
	out      []byte
	pos      int
	a8       int
	reps     [4]int
	bin      []binModel
	nibLit   []nibModel // 0x11c0: [13][81] unmatched+predicted
	tok      nibModel   // 0x9ac0 token type
	lenNib   nibModel   // match/token length
	far24    farModel   // 0x9ae0 24-sym first far symbol
	farNib   [2]nibModel
	rolz32   []rolzModel // 0xb5d0: 32-sym per prev byte
	rolzLen  [2]nibModel
	resNib   []nibModel  // token residual, 16 ctx × 16
	imgM     [3]imgModel // 0x42c150/0x4315c0 20-sym, first-row per channel
	imgWM    imgModel    // 0x41f7fc width integer
	imgRice  [3]int      // 0xffd0 G, 0x102e0 R, 0x105f0 B
	imgBit   binModel    // 0x10fb0 ImagePred width bit (adapt >>5)
	rawBit   binModel    // unused; t15 is bits(16), not 16 binary bits
	imgW     int
	stRec    [320][4]byte
	nLit     int
	nMat     int
	nTok     int
	why      string
	rolz     [256][]int
	lastKind int
	lastRes  [4]int
	ev       []string
	altRes   int // 0=table, 1=two nibbles
	altImg   int // 0=width*3, 1=8 residuals no width bits
	altRaw   int // 0=rawNoSwap (PE 406fc2), see raw* consts
	rawN     int // u16s already written in the current RawBytes
	lim      int // dest cap from first 8 bytes when they parse as Delta
	noChunk  bool // first t15: header only, no dest[8:12] CHUNK_SIZE
}

const (
	rawNoSwap  = iota // s0&0xffff, s0>>=16, no pre-swap
	rawFirstNS        // first u16 no swap; later u16s bits(16)
	rawBits16
	rawBinLSB
	rawBinMSB
	rawImgU
	rawNib
)

func newDec(src []byte, cap int) *dec {
	d := &dec{
		out:    make([]byte, 0, cap),
		bin:    make([]binModel, 0x300),
		nibLit: make([]nibModel, 13*81),
		rolz32: make([]rolzModel, 256),
		resNib: make([]nibModel, 16*16),
	}
	d.r.src = src
	if len(src) >= 8 {
		d.r.s0 = uint32(src[0]) | uint32(src[1])<<8 | uint32(src[2])<<16 | uint32(src[3])<<24
		d.r.s1 = uint32(src[4]) | uint32(src[5])<<8 | uint32(src[6])<<16 | uint32(src[7])<<24
		d.r.off = 8
		d.r.end = len(src)
	}
	for i := range d.bin {
		d.bin[i] = 0x8000
	}
	d.imgBit = 0x8000
	d.rawBit = 0x8000
	for i := range d.nibLit {
		d.nibLit[i].init()
	}
	// Token CDF at +0x9ac0 is calloc zeros (0x40d96c). First sym is t15 RawBytes.
	d.lenNib.init()
	d.far24.init()
	for i := range d.farNib {
		d.farNib[i].init()
	}
	for i := range d.rolz32 {
		d.rolz32[i].init()
	}
	for i := range d.rolzLen {
		d.rolzLen[i].init()
	}
	for i := range d.resNib {
		d.resNib[i].init()
	}
	for i := range d.imgM {
		d.imgM[i].init()
	}
	d.imgWM.init()
	for i := range d.stRec {
		d.stRec[i] = [4]byte{0x80, 0x80, 0x80, 0x80}
	}
	d.reps = [4]int{1, 1, 1, 1}
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
	d.lastKind = kind
}

func (d *dec) decodeByte() bool {
	if d.lim > 0 && d.pos >= d.lim {
		d.why = "delta-span"
		return false
	}
	if !d.r.ok() || d.pos >= cap(d.out) {
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
		return false
	}
	ps[0] = saNext(ps[0], lit)
	qs[0] = saNext(qs[0], lit)
	if lit == 1 {
		d.nLit++
		if !d.literal() {
			d.why = "lit"
			return false
		}
		d.note("L")
		d.stepA8(kindLit)
		return true
	}
	d.nMat++
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
			loIdx = cls*81 + 1 + int(pred&0xf)
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
	ps := d.prevSlot()
	qs := d.posSlot()
	m1 := d.binAt(ps[1], qs[1])
	b1 := m1.bit(&d.r)
	if !d.r.ok() {
		return false
	}
	ps[1] = saNext(ps[1], b1)
	qs[1] = saNext(qs[1], b1)
	if b1 == 0 {
		d.note("T")
		return d.token()
	}
	m2 := d.binAt(ps[3], qs[3])
	b2 := m2.bit(&d.r)
	if !d.r.ok() {
		return false
	}
	ps[3] = saNext(ps[3], b2)
	qs[3] = saNext(qs[3], b2)
	if b2 == 0 {
		d.note("R")
		return d.rolzMatch()
	}
	d.note("F")
	return d.farMatch()
}

func (d *dec) note(s string) {
	if len(d.ev) < 24 {
		d.ev = append(d.ev, s)
	}
}

func (d *dec) repMatch() bool {
	slot := d.lenNib.sym(&d.r)
	if !d.r.ok() {
		return false
	}
	idx := int(repIdx[slot&15])
	dist := d.reps[idx] + int(repDelta[slot&15])
	if dist < 1 {
		dist = 1
	}
	ln := d.lenNib.sym(&d.r)
	if !d.r.ok() {
		return false
	}
	ok := d.copy(dist, lengthFromSym(&d.r, ln))
	d.stepA8(kindRep)
	return ok
}

func (d *dec) rolzMatch() bool {
	prev := d.prev()
	slot := d.rolz32[prev].sym(&d.r)
	if !d.r.ok() {
		return false
	}
	if slot < 0 {
		slot = 0
	}
	if slot > 31 {
		slot = 31
	}
	off := rolzBase[slot]
	if e := int(rolzExtra[slot]); e > 0 {
		off += int(d.r.bits(e))
	}
	list := d.rolz[prev]
	if len(list) == 0 {
		// PE empty list does not emit the slot (0x18) and does not
		// run dest[pos-1]. Consume the length extra bits; write nothing.
		ln := d.rolzLen[0].sym(&d.r)
		if !d.r.ok() {
			return false
		}
		_ = lengthFromSym(&d.r, ln)
		d.stepA8(kindRolz)
		return true
	}
	idx := off
	if idx >= len(list) {
		idx %= len(list)
	}
	srcPos := list[len(list)-1-idx]
	dist := d.pos - srcPos
	if dist < 1 {
		dist = 1
	}
	ln := d.rolzLen[0].sym(&d.r)
	if !d.r.ok() {
		return false
	}
	ok := d.copy(dist, lengthFromSym(&d.r, ln))
	d.stepA8(kindRolz)
	return ok
}

func (d *dec) farMatch() bool {
	hi := d.far24.sym(&d.r)
	if !d.r.ok() {
		return false
	}
	if hi < 0 {
		hi = 0
	}
	if hi > 26 {
		hi = 26
	}
	// 0x402c93 loads 0x42c140 → extra 0x42b780 (extra[i]=i).
	// 0x402cad: lea eax, [extra-8] — nibbles only when extra>=8.
	ex := int(rolzExtra[hi])
	dist := rolzBase[hi] + 1
	if ex >= 8 {
		mid := d.farNib[0].sym(&d.r)
		if !d.r.ok() {
			return false
		}
		lo := d.farNib[1].sym(&d.r)
		if !d.r.ok() {
			return false
		}
		extra := uint32(mid)<<4 | uint32(lo)
		if ex > 8 {
			extra |= d.r.bits(ex-8) << 8
		}
		dist += int(extra)
	} else if ex > 0 {
		dist += int(d.r.bits(ex))
	}
	if dist < 1 {
		dist = 1
	}
	ln := d.lenNib.sym(&d.r)
	if !d.r.ok() {
		return false
	}
	n := lengthFromSym(&d.r, ln)
	if len(d.ev) < 24 {
		d.note("Fd" + itoa(dist) + "n" + itoa(n))
	}
	ok := d.copy(dist, n)
	if !ok {
		d.why = "far d=" + itoa(dist) + " n=" + itoa(n) + " p=" + itoa(d.pos)
	}
	d.stepA8(kindFar)
	return ok
}

func (d *dec) repMatchFallback(slot int) bool {
	idx := int(repIdx[slot&15])
	dist := d.reps[idx] + int(repDelta[slot&15])
	if dist < 1 {
		dist = 1
	}
	return d.copy(dist, 2+int(repKind[slot&15]))
}

func (d *dec) token() bool {
	d.nTok++
	t := d.tok.sym(&d.r)
	if !d.r.ok() {
		d.why = "tok-sym"
		return false
	}
	if len(d.ev) <= 24 {
		d.note("t" + itoa(t))
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
		ok = d.tokDeltaUN(2, 1)
	case tokDelta2xU16:
		ok = d.tokDeltaUN(2, 2)
	case tokDelta1xU32:
		ok = d.tokDeltaUN(4, 1)
	case tokRGB:
		ok = d.tokRGB(3)
	case tokRGBA:
		ok = d.tokRGB(4)
	case tokImagePred:
		ok = d.tokImage()
	case tokMono8:
		ok = d.tokAudio(1, 1)
	case tokStereo8:
		ok = d.tokAudio(2, 1)
	case tokMono16:
		ok = d.tokAudio(1, 2)
	case tokStereo16:
		ok = d.tokAudio(2, 2)
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
	return ok
}

func (d *dec) tokLen() int {
	n := lengthFromSym(&d.r, d.lenNib.sym(&d.r))
	if n > 1<<20-d.pos {
		n = 1<<20 - d.pos
	}
	if n < 1 {
		n = 1
	}
	return n
}

func (d *dec) resByte(ctx int) byte {
	if ctx < 0 {
		ctx = 0
	}
	ctx &= 15
	// Encode.su 3281: values are simply subtracted (uint8 wrap).
	// PE 0x42b740: extra 0,0,1,..6,6,..1,0,0 — 0 and 255 are extra-0.
	if d.altRes == 1 {
		hi := d.resNib[ctx].sym(&d.r)
		lo := d.resNib[ctx].sym(&d.r)
		return byte(hi<<4 | lo)
	}
	sym := d.resNib[ctx].sym(&d.r)
	if sym < 0 {
		sym = 0
	}
	if sym > 15 {
		sym = 15
	}
	// 0x42b740: extra 0,0,1,2,3,4,5,6,6,5,4,3,2,1,0,0 (uint8 wrap, bias 0).
	v := resBase16[sym]
	if e := int(resExtra16[sym]); e > 0 {
		v += int(d.r.bits(e))
	}
	return byte(v - resBias16)
}

func (d *dec) tokDeltaU8(step int) bool {
	// Encode 0x40aab2 walks 8 position-context slots (0xf617..0xf60f).
	n := 8
	if step > 1 {
		n = 8 * step
	}
	for i := 0; i < n; i++ {
		pred := byte(0)
		if d.pos >= step {
			pred = d.out[d.pos-step]
		}
		// Missing prev-Δ uses the PE 0x40d568 seed 0xffff8000 → sign 1.
		sign := 1
		if d.pos >= 2*step {
			if int(d.out[d.pos-step]) >= int(d.out[d.pos-2*step]) {
				sign = 0
			}
		}
		ctx := (d.pos%step)*2 + sign
		res := d.resByte(ctx)
		d.lastRes[d.pos&3] = int(int8(res))
		d.emit(pred + res)
		if !d.r.ok() {
			d.why = "tok-d8"
			return false
		}
	}
	return true
}

func (d *dec) tokDeltaUN(width, stride int) bool {
	step := width * stride
	n := d.tokLen()
	if n%width != 0 {
		n += width - n%width
	}
	for i := 0; i < n; i++ {
		ch := i % width
		pred := byte(0)
		src := d.pos - step
		if src >= 0 {
			pred = d.out[src]
		}
		sign := 1
		if src >= step {
			if int(d.out[src]) >= int(d.out[src-step]) {
				sign = 0
			}
		}
		ctx := ch*2 + sign
		res := d.resByte(ctx)
		d.emit(pred + res)
		if !d.r.ok() {
			d.why = "tok-dn"
			return false
		}
	}
	return true
}

func (d *dec) tokRGB(ch int) bool {
	n := d.tokLen()
	if n%ch != 0 {
		n += ch - n%ch
	}
	for i := 0; i < n; i++ {
		c := i % ch
		pred := byte(0)
		if d.pos >= ch {
			pred = d.out[d.pos-ch]
		}
		// 0x40af88: per-channel Δ plus half-neighbour mix on G/B.
		if c > 0 && d.pos >= 1 {
			mix := int(int8(d.out[d.pos-1] - pred))
			pred += byte(mix / 2)
		}
		res := d.resByte(c)
		d.emit(pred + res)
		if !d.r.ok() {
			d.why = "tok-rgb"
			return false
		}
	}
	return true
}

func unzig(v int) int {
	// 0x404af7: v>>1, not; cmove if (v&1)==0 → even: v>>1, odd: ^(v>>1).
	if v&1 == 0 {
		return v >> 1
	}
	return ^(v >> 1)
}

func (d *dec) imgU(m *imgModel) int {
	// 0x41f7fc / 0x404aa9: 20-sym extra 0x42b720. edx=16 caps extra.
	// Unsigned; zigzag is the pixel step at 0x404af7, not the width return.
	sym := m.sym(&d.r)
	if sym < 0 {
		sym = 0
	}
	if sym >= imgSyms {
		sym = imgSyms - 1
	}
	v := imgBase20[sym]
	if e := int(imgExtra20[sym]); e > 0 {
		if e > 16 {
			e = 16
		}
		v += int(d.r.bits(e))
	}
	return v
}

func (d *dec) imgRes(ch int) int {
	// 0x4047ca r13=0x42c150 → 0x4315c0 extra 0x42b720, n=20.
	// Rice: G 0xffd0, R 0x102e0, B 0x105f0. First residual rice=0.
	if ch < 0 {
		ch = 0
	}
	ch %= 3
	v := d.imgU(&d.imgM[ch])
	r := d.imgRice[ch]
	if r > 16 {
		r = 16
	}
	if r != 0 {
		// 0x4081c0: value = (value << rice) | bits(rice)
		v = v<<r | int(d.r.bits(r))
	}
	if v > 2<<r {
		// 0x404ae0 cmp 2<<rice, value; jae stay; else rice++
		d.imgRice[ch] = r + 1
	} else if v < 1<<r {
		// 0x408170: rice -= (rice+31)>>5
		d.imgRice[ch] = r - (r+31)>>5
	}
	return unzig(v)
}

func (d *dec) imgCh(ch, row int) byte {
	// West is previous pixel same channel (0x4047f2 −3). First row: west only.
	var west, north, nw byte
	src := d.pos + ch - 3
	if src >= 0 && src < d.pos {
		west = d.out[src]
	}
	if ni := d.pos + ch - row; ni >= 0 && ni < d.pos {
		north = d.out[ni]
	}
	if nwi := d.pos + ch - row - 3; nwi >= 0 && nwi < d.pos {
		nw = d.out[nwi]
	}
	pred := west
	if d.pos >= row {
		p := int(west) + int(north) - int(nw)
		pw := absInt(p - int(west))
		pn := absInt(p - int(north))
		pnw := absInt(p - int(nw))
		pred = nw
		if pw <= pn && pw <= pnw {
			pred = west
		} else if pn <= pnw {
			pred = north
		}
	}
	return pred + byte(d.imgRes(ch))
}

func (d *dec) tokImage() bool {
	// 0x404752: binary +0x10fb0 (adapt >>5). Bit 1: 0x41f7fc edx=16
	// is the 0x4315c0 20-sym integer (imul $0x310), not RawBytes bits(16).
	// One token writes 12 bytes (0x404826 lea 0xc(%rsi); 0x408306 addl $0xc).
	if d.imgBit.bitSh(&d.r, 5) == 1 {
		w := d.imgU(&d.imgWM)
		if w > 0 {
			d.imgW = w
		}
	}
	nbyte := 12
	if d.altImg == 1 {
		nbyte = 8
	}
	if nbyte > 1<<20-d.pos {
		nbyte = 1<<20 - d.pos
	}
	row := d.imgW
	if row < 1 {
		row = nbyte
	}
	for i := 0; i+3 <= nbyte; i += 3 {
		// Stores: 0x404b9d G at +1, 0x404e1e R at +0, 0x4050ad B at +2.
		g := d.imgCh(1, row)
		r := d.imgCh(0, row)
		b := d.imgCh(2, row)
		if !d.r.ok() {
			d.why = "tok-img"
			return false
		}
		d.emit(r)
		d.emit(g)
		d.emit(b)
	}
	return true
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func (d *dec) tokAudio(ch, width int) bool {
	step := ch * width
	n := d.tokLen()
	if n%width != 0 {
		n += width - n%width
	}
	for i := 0; i < n; i++ {
		c := i % step
		// Order-2: 2*prev - prev2 on the same channel.
		var pred byte
		if d.pos >= step {
			p1 := int(d.out[d.pos-step])
			p2 := p1
			if d.pos >= 2*step {
				p2 = int(d.out[d.pos-2*step])
			}
			v := 2*p1 - p2
			if v < 0 {
				v = 0
			}
			if v > 255 {
				v = 255
			}
			pred = byte(v)
		}
		res := d.resByte(c & 15)
		d.emit(pred + res)
		if !d.r.ok() {
			d.why = "tok-au"
			return false
		}
	}
	return true
}

func (d *dec) tokLit32() bool {
	return d.tokRaw()
}

func (d *dec) rawSwapNoRefill() uint16 {
	d.r.s0, d.r.s1 = d.r.s1, d.r.s0
	if !d.r.ok() {
		return 0
	}
	v := uint16(d.r.s0 & 0xffff)
	d.r.s0 >>= 16
	return v
}

func (d *dec) tokRaw() bool {
	// Decode RawBytes 0x406fc2 / 0x40a8ae: 16× store ax, pos += 0x20.
	// First t15: s0 leftover is the Delta header (fg-05: 0x00017df1).
	// Four no-swap extracts write dest[0:8] and drain s0. Mid-token
	// refill of packed[8:] is memcpy32 — stop after the header so
	// dest[8:] comes from the lit/match/token kernel; s1 stays live.
	// Later t15: 32 bytes from the literal kernel, not raw s0.
	if d.pos == 0 {
		d.rawN = 0
		for i := 0; i < 4; i++ {
			w := d.rawU16()
			d.rawN++
			if !d.r.ok() {
				d.why = "tok-raw"
				return false
			}
			d.emit(byte(w))
			d.emit(byte(w >> 8))
		}
		// dest[8:12] is DisPack CHUNK_SIZE (16KiB). dest[12:] from
		// the lit/match kernel after a normal rANS swap.
		if d.pos == 8 && !d.noChunk {
			n := uint32(16 << 10)
			d.emit(byte(n))
			d.emit(byte(n >> 8))
			d.emit(byte(n >> 16))
			d.emit(byte(n >> 24))
		}
		return true
	}
	for i := 0; i < 32; i++ {
		if !d.literal() {
			d.why = "tok-raw"
			return false
		}
	}
	return true
}

func (d *dec) renormS0() {
	// After two 16-bit extracts from s0=0x00017df1, s0==0.
	// Refill only when empty. s0==1 is the high half of dataSize
	// (dest u16 0x0001); treating <=0xffff as dry overwrites it.
	for k := 0; k < 2 && d.r.ok() && d.r.s0 == 0; k++ {
		d.r.refill()
	}
}

func (d *dec) rawU16() uint16 {
	switch d.altRaw {
	case rawFirstNS:
		if d.rawN == 0 {
			if !d.r.ok() {
				return 0
			}
			v := uint16(d.r.s0 & 0xffff)
			d.r.s0 >>= 16
			return v
		}
		return uint16(d.r.bits(16))
	case rawBits16:
		return uint16(d.r.bits(16))
	case rawBinLSB, rawBinMSB:
		var w uint16
		for b := uint(0); b < 16; b++ {
			if d.rawBit.bit(&d.r) == 1 {
				if d.altRaw == rawBinMSB {
					w = w<<1 | 1
				} else {
					w |= 1 << b
				}
			} else if d.altRaw == rawBinMSB {
				w <<= 1
			}
		}
		return w
	case rawImgU:
		return uint16(d.imgU(&d.imgWM))
	case rawNib:
		hi := d.resNib[0].sym(&d.r)
		lo := d.resNib[0].sym(&d.r)
		return uint16(hi<<4 | lo)
	default:
		// rawNoSwap: take s0&0xffff, s0>>=16, no pre-swap.
		// First t15 writes only the leftover (dest[0:8]). Do not
		// refill s0 from packed here — that memcpy's the rANS
		// stream into dest. Later u16s of a later t15 stay 0
		// if s0 is dry; the main loop's models refill via swap.
		if !d.r.ok() {
			return 0
		}
		v := uint16(d.r.s0 & 0xffff)
		d.r.s0 >>= 16
		return v
	}
}

func (d *dec) emit(b byte) {
	prev := d.prev()
	d.out = append(d.out, b)
	if d.pos < 1<<20 {
		d.rolz[prev] = append(d.rolz[prev], d.pos)
		if len(d.rolz[prev]) > rolzHist {
			d.rolz[prev] = d.rolz[prev][len(d.rolz[prev])-rolzHist:]
		}
	}
	d.pos++
	// rzw dest is one Delta solid. Once the first 8 bytes parse as
	// a PE-legal header (ds in 1k..2M, ts%4==0), dest is exactly
	// 8+3*ts+ds. Hitting the 1MiB cap means the kernel did not stop.
	if d.lim == 0 && d.pos == 8 {
		ds := uint32(d.out[0]) | uint32(d.out[1])<<8 | uint32(d.out[2])<<16 | uint32(d.out[3])<<24
		ts := uint32(d.out[4]) | uint32(d.out[5])<<8 | uint32(d.out[6])<<16 | uint32(d.out[7])<<24
		if ds >= 1024 && ds <= 2<<20 && ts < 0x7fffffff && ts%4 == 0 {
			d.lim = 8 + 3*int(ts) + int(ds)
		}
	}
}

func (d *dec) copy(dist, n int) bool {
	if n < 1 || d.pos < 1 {
		return false
	}
	// Far/ROLZ can decode dist > pos while history is still short.
	// Fold into the window instead of aborting the solid (was 425 bytes).
	if dist < 1 {
		dist = 1
	}
	if dist > d.pos {
		// Still fold: mixed tokens grow history but far can exceed pos.
		dist = (dist-1)%d.pos + 1
	}
	d.reps[3] = d.reps[2]
	d.reps[2] = d.reps[1]
	d.reps[1] = d.reps[0]
	d.reps[0] = dist
	for i := 0; i < n; i++ {
		src := d.pos - dist
		if src < 0 || src >= len(d.out) {
			return false
		}
		d.emit(d.out[src])
		if d.pos >= cap(d.out) || (d.lim > 0 && d.pos >= d.lim) {
			return d.lim > 0 && d.pos >= d.lim
		}
	}
	return true
}

func decodeNative(src []byte, dcap int) ([]byte, error) {
	if len(src) < 8 || dcap <= 0 {
		return nil, errCodec
	}
	d := newDec(src, dcap)
	maxOut := len(src) * 16
	if maxOut > dcap {
		maxOut = dcap
	}
	for d.pos < maxOut {
		if !d.decodeByte() {
			break
		}
	}
	if d.lim > 0 && len(d.out) > d.lim {
		d.out = d.out[:d.lim]
	}
	if len(d.out) == 0 {
		return nil, errCodec
	}
	return d.out, nil
}
