package magic2

// FCM + rANS primitives recovered from a static read of cls-magic2_x64
// (v22c4b). Image: /tmp/lolzpe/pe_3251605_x64_343040.bin
//
//	.text file 0x400 ↔ RVA 0x1000 ↔ VA 0x140001000
//	ImageBase 0x140000000
//
// Context hash (nibble model at 0x14001adb0 / 0x14001b200)
//
//	h0 = bitlen8( uint8( (abs32(w08-w0c) + abs32(w10-w14)) >> 1 ) )
//	h1 = bitlen8( uint8( abs32(w20-w24) ) )
//	idx = h0*9*32 + h1*32
//
//	bitlen8(x) = 0 if x==0 else floor(log2(x))+1   (bsr+1)
//	abs32 is cdq/xor/sub (two's complement abs).
//	w08..w24 are the IIR history dwords at FCM-state +0x08..+0x24.
//
// Mixer hash (8-sym at 0x140017410 / 0x1400179b0)
//
//	absb = abs_bytes( pack(w20,w10) - pack(w24,w14) )   // psubb + abs
//	h0 = bitlen8( b0|b1|b2 of absb )
//	h1 = bitlen8( byte5 of absb )
//	h2 = bitlen6( ((w20+w24)>>9) & 0x3f )
//	idx = (h0<<7) + (h1<<4) + (h2<<10)
//
// High-level o1 selectors (options, not the IIR hash). Defaults
// -pc2 -bc4 -bm4 -blr4 and published -blo8 -bll8:
//
//	pc  bits of pos,     bc  bits of prev in the mixer
//	blo bits of prev in literal-hi,  bll bits of prev in literal-lo
//	blr bits of rep0lit in literal-hi,  bm bits of rep0lit in match-flag
//
//	ctxHi  = (prev >> (8-blo)) | ((rep0lit >> (8-blr)) << blo) | ((pos & ((1<<pc)-1)) << (blo+blr))
//	ctxLo  = ((prev >> (8-bll)) << 4) | nibbleHi
//	ctxBM  = (rep0lit >> (8-bm)) | ((pos & ((1<<pc)-1)) << bm)
//	ctxBC  = prev >> (8-bc)
//
// rANS
//
//	L = 1<<23
//	slot = state & (M-1)     // never state % M
//	nibble / 8-sym: M = 1<<15, scale 15, CDF of uint16
//	binary:         M = 1<<14, scale 14, single p0 uint16
//
//	nibble find: first i in 1..16 where int16(cdf[i]) > int16(slot)
//	(pcmpgtw is signed; cdf words with the high bit set never win).
//	Sentinel i=16, cdf[16] treated as 0x8000 (LZNA-style 17th cell).
//	state' = (cdf[i]-cdf[i-1])*(state>>15) + slot - cdf[i-1]
//	adapt: cdf[j] += int16(target[sym][j]-cdf[j]) >> 7     // psraw $7
//	       target rows live at VA 0x140001c00 (16 x 16 u16)
//
//	8-sym: same find on 8 words, sentinel add 0x80, adapt >> 6
//	       toward VA 0x1400010e0
//
//	getBit (0x14001af4a / x86 0x415af5):
//	    slot = state & 0x3fff; quo = state >> 14; p = *p0
//	    if slot < p { state = quo*p + slot; p += (0x4000-p)>>4; bit=0 }
//	    else         { state = state - p*(quo+1); p -= p>>4;     bit=1 }
//	    *p0 = p
//
//	renorm: while state < 1<<23 { state = state<<8 | *src++ }

const (
	ransScaleNibble = 15
	ransMN          = 1 << ransScaleNibble // 32768
	ransScaleBit    = 14
	ransMB          = 1 << ransScaleBit // 16384
	adaptNibble     = 7
	adaptNibble9    = 6 // psraw $6 at 0x14001aecc
	adaptSym8       = 6
	adaptBit        = 4

	// Published / in-memory defaults. optionDefaults16 = {pc,bc,bm,blr}.
	optPC  = 2
	optBC  = 4
	optBM  = 4
	optBLR = 4
	optBLO = 8
	optBLL = 8
)

// bitlen returns bsr(x)+1, or 0 if x==0. Matches 0x14001adf8 / 0x140017485.
func bitlen(x uint32) int {
	if x == 0 {
		return 0
	}
	n := 0
	for x > 0 {
		x >>= 1
		n++
	}
	return n
}

func abs32(x int32) uint32 {
	if x < 0 {
		return uint32(-x)
	}
	return uint32(x)
}

// hashNibble is the nibble-FCM index from 0x14001adb0.
// Returns a byte offset (multiple of 32) into that model.
func hashNibble(w08, w0c, w10, w14, w20, w24 uint32) int {
	a := abs32(int32(w08) - int32(w0c))
	b := abs32(int32(w10) - int32(w14))
	h0 := bitlen(uint32(uint8((a + b) >> 1)))
	h1 := bitlen(uint32(uint8(abs32(int32(w20) - int32(w24)))))
	return h0*9*32 + h1*32
}

// hashMixer is the 8-sym mixer index from 0x140017410.
func hashMixer(w10, w14, w20, w24 uint32) int {
	// psubb + abs via pcmpgtb/pxor on pack(w20,w10) vs pack(w24,w14)
	var absb [8]byte
	p := [8]byte{
		byte(w10), byte(w10 >> 8), byte(w10 >> 16), byte(w10 >> 24),
		byte(w20), byte(w20 >> 8), byte(w20 >> 16), byte(w20 >> 24),
	}
	s := [8]byte{
		byte(w14), byte(w14 >> 8), byte(w14 >> 16), byte(w14 >> 24),
		byte(w24), byte(w24 >> 8), byte(w24 >> 16), byte(w24 >> 24),
	}
	for i := 0; i < 8; i++ {
		d := int(p[i]) - int(s[i])
		if d < 0 {
			d = -d
		}
		absb[i] = byte(d)
	}
	or3 := uint32(absb[0] | absb[1] | absb[2])
	h0 := bitlen(or3)
	h1 := bitlen(uint32(absb[5]))
	h2 := bitlen(((w20 + w24) >> 9) & 0x3f)
	return (h0 << 7) + (h1 << 4) + (h2 << 10)
}

func ctxLiteralHi(prev, rep0lit byte, pos int) int {
	hi := int(prev) >> (8 - optBLO)
	rep := int(rep0lit) >> (8 - optBLR)
	pc := pos & ((1 << optPC) - 1)
	return hi | rep<<optBLO | pc<<(optBLO+optBLR)
}

func ctxLiteralLo(prev, hi byte) int {
	return (int(prev)>>(8-optBLL))<<4 | int(hi)
}

func ctxMatchFlag(rep0lit byte, pos int) int {
	rep := int(rep0lit) >> (8 - optBM)
	pc := pos & ((1 << optPC) - 1)
	return rep | pc<<optBM
}

func ctxMixer(prev byte) int {
	return int(prev) >> (8 - optBC)
}

// getBit is binary rANS at scale 14, adapt >>4 (0x14001af4a / x86 0x415af5).
func (r *rANS) getBit(p0 *uint16) (int, error) {
	return r.getBitN(p0, ransScaleBit, adaptBit)
}

// getBitN is the PE/LZNA binary rANS: slot = state&(M-1), M=1<<nbits.
// if slot < p { state = quo*p+slot; p += (M-p)>>shift } else { state -= p*(quo+1); p -= p>>shift }.
// Other sites use nbits=13 shift=5 (0x1fff, like LZNA is_literal).
func (r *rANS) getBitN(p0 *uint16, nbits, shift uint) (int, error) {
	if r.off > len(r.buf) && r.x < ransL {
		return 0, errBitstream
	}
	m := uint32(1 << nbits)
	p := uint32(*p0)
	if p == 0 {
		p = 1
	}
	if p >= m {
		p = m - 1
	}
	slot := r.x & (m - 1)
	quo := r.x >> nbits
	if slot < p {
		r.x = quo*p + slot
		*p0 = uint16(p + (m-p)>>shift)
		r.renorm()
		return 0, nil
	}
	r.x = r.x - p*(quo+1)
	np := p - (p >> shift)
	if np == 0 {
		np = 1
	}
	*p0 = uint16(np)
	r.renorm()
	return 1, nil
}

// getNibble is 16-symbol CDF rANS. cdf is 16 little-endian uint16
// (cdf[0]==0, cdf[15] typically 0x7800 after init). Scale 15, adapt >>7
// toward nibbleTarget[sym]. 0x14001b9cf / LZNA LznaReadNibble sibling.
func (r *rANS) getNibble(cdf []uint16) (int, error) {
	if len(cdf) < 16 {
		return 0, errBitstream
	}
	if r.off > len(r.buf) && r.x < ransL {
		return 0, errBitstream
	}
	slot := r.x & (ransMN - 1)
	quo := r.x >> ransScaleNibble
	// pcmpgtw + packsswb + pmovmskb | 0x10000 + bsf
	i := 16
	for j := 1; j < 16; j++ {
		if int16(cdf[j]) > int16(slot) {
			i = j
			break
		}
	}
	start := uint32(cdf[i-1])
	end := uint32(0x8000)
	if i < 16 {
		end = uint32(cdf[i])
	}
	if end <= start {
		end = start + 1
	}
	r.x = (end-start)*quo + (slot - start)
	sym := i - 1
	adaptNibbleCDF(cdf, sym)
	r.renorm()
	return sym, nil
}

// getNibbleBSF is getNibble plus the 1-based bsf index (PE r12d).
func (r *rANS) getNibbleBSF(cdf []uint16) (sym, bsf int, err error) {
	sym, err = r.getNibble(cdf)
	if err != nil {
		return 0, 0, err
	}
	return sym, sym + 1, nil
}

// getNibble9 is the 9-symbol find at 0x14001adb0: pmovmskb + add 0x200
// + bsf. Alphabet is 0..8 (bitlen8 range). Adapt >>6 toward VA
// 0x140001aa0. cdf[9] is the sentinel high when no word 1..8 wins.
func (r *rANS) getNibble9(cdf []uint16) (sym, bsf int, err error) {
	if len(cdf) < 10 {
		return 0, 0, errBitstream
	}
	if r.off > len(r.buf) && r.x < ransL {
		return 0, 0, errBitstream
	}
	slot := r.x & (ransMN - 1)
	quo := r.x >> ransScaleNibble
	i := 9
	for j := 1; j <= 8; j++ {
		if int16(cdf[j]) > int16(slot) {
			i = j
			break
		}
	}
	start := uint32(cdf[i-1])
	end := uint32(cdf[i])
	if end <= start {
		end = start + 1
	}
	r.x = (end-start)*quo + (slot - start)
	sym = i - 1
	adaptNibble9CDF(cdf, sym)
	r.renorm()
	return sym, i, nil
}

func adaptNibble9CDF(cdf []uint16, sym int) {
	if sym < 0 {
		sym = 0
	}
	if sym > 8 {
		sym = 8
	}
	tgt := nibble9Target[sym]
	n := 16
	if n > len(cdf) {
		n = len(cdf)
	}
	for j := 0; j < n; j++ {
		d := int16(tgt[j]) - int16(cdf[j])
		cdf[j] = uint16(int16(cdf[j]) + d>>adaptNibble9)
	}
}

// getSym8 is the 8-symbol mixer/token rANS at 0x1400174e2.
// cdf is 8 uint16, sentinel bit 7 (add 0x80). Adapt >>6 toward sym8Target.
// Returns the 1-based bsf index (1 = first symbol), matching `cmp ecx,1`.
func (r *rANS) getSym8(cdf []uint16) (int, error) {
	if len(cdf) < 8 {
		return 0, errBitstream
	}
	if r.off > len(r.buf) && r.x < ransL {
		return 0, errBitstream
	}
	slot := r.x & (ransMN - 1)
	quo := r.x >> ransScaleNibble
	i := 7
	for j := 1; j < 7; j++ {
		if int16(cdf[j]) > int16(slot) {
			i = j
			break
		}
	}
	start := uint32(cdf[i-1])
	end := uint32(cdf[i])
	if end <= start {
		end = start + 1
	}
	r.x = (end-start)*quo + (slot - start)
	adaptSym8CDF(cdf, i-1)
	r.renorm()
	return i, nil
}

func adaptNibbleCDF(cdf []uint16, sym int) {
	if sym < 0 {
		sym = 0
	}
	if sym > 15 {
		sym = 15
	}
	tgt := nibbleTarget[sym]
	for j := 0; j < 16; j++ {
		d := int16(tgt[j]) - int16(cdf[j])
		cdf[j] = uint16(int16(cdf[j]) + d>>adaptNibble)
	}
}

func adaptSym8CDF(cdf []uint16, sym int) {
	if sym < 0 {
		sym = 0
	}
	if sym > 6 {
		sym = 6
	}
	tgt := sym8Target[sym]
	for j := 0; j < 8; j++ {
		d := int16(tgt[j]) - int16(cdf[j])
		cdf[j] = uint16(int16(cdf[j]) + d>>adaptSym8)
	}
}

func initNibbleCDF(dst []uint16) {
	for i := 0; i < 16 && i < len(dst); i++ {
		dst[i] = uint16(i * 0x800) // VA 0x140001bc0
	}
}

func initSym8CDF(dst []uint16) {
	for i := 0; i < 8 && i < len(dst); i++ {
		dst[i] = uint16(i * 0x1000)
	}
	if len(dst) >= 8 {
		dst[7] = 0x8000
	}
}

func initBit(p *uint16) { *p = ransMB / 2 }

// nibble9Target is VA 0x140001aa0, 9 rows of 16 u16. Used by the
// IIR nibble path (0x14001adb0). Row s is the adapt target for
// symbol s (bsf-1). Slot 9 is 0x8000.
var nibble9Target = [9][16]uint16{
	{0x0000, 0x8006, 0x800d, 0x8014, 0x801b, 0x8022, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
	{0x0000, 0x0007, 0x800d, 0x8014, 0x801b, 0x8022, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
	{0x0000, 0x0007, 0x000e, 0x8014, 0x801b, 0x8022, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
	{0x0000, 0x0007, 0x000e, 0x0015, 0x801b, 0x8022, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
	{0x0000, 0x0007, 0x000e, 0x0015, 0x001c, 0x8022, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
	{0x0000, 0x0007, 0x000e, 0x0015, 0x001c, 0x0023, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
	{0x0000, 0x0007, 0x000e, 0x0015, 0x001c, 0x0023, 0x002a, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
	{0x0000, 0x0007, 0x000e, 0x0015, 0x001c, 0x0023, 0x002a, 0x0031, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
	{0x0000, 0x0007, 0x000e, 0x0015, 0x001c, 0x0023, 0x002a, 0x0031, 0x0038, 0x8000, 0, 0, 0, 0, 0, 0},
}

// nibbleTarget is VA 0x140001c00, 16 rows of 16 u16. For symbol s the
// first s+1 cuts stay near 8,8,16,… and the rest sit at 0x8000+8k so
// psubw/psraw pulls the live CDF toward a one-hot at s.
var nibbleTarget = [16][16]uint16{
	{0x0000, 0x8007, 0x800f, 0x8017, 0x801f, 0x8027, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x800f, 0x8017, 0x801f, 0x8027, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x8017, 0x801f, 0x8027, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x801f, 0x8027, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x8027, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x0058, 0x805f, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x0058, 0x0060, 0x8067, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x0058, 0x0060, 0x0068, 0x806f, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x0058, 0x0060, 0x0068, 0x0070, 0x8077},
	{0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x0058, 0x0060, 0x0068, 0x0070, 0x0078},
}

// sym8Target is VA 0x1400010e0, 7 used rows of 8 u16.
var sym8Target = [7][8]uint16{
	{0x0000, 0x8008, 0x8011, 0x801a, 0x8023, 0x802c, 0x8035, 0x8000},
	{0x0000, 0x0009, 0x8011, 0x801a, 0x8023, 0x802c, 0x8035, 0x8000},
	{0x0000, 0x0009, 0x0012, 0x801a, 0x8023, 0x802c, 0x8035, 0x8000},
	{0x0000, 0x0009, 0x0012, 0x001b, 0x8023, 0x802c, 0x8035, 0x8000},
	{0x0000, 0x0009, 0x0012, 0x001b, 0x0024, 0x802c, 0x8035, 0x8000},
	{0x0000, 0x0009, 0x0012, 0x001b, 0x0024, 0x002d, 0x8035, 0x8000},
	{0x0000, 0x0009, 0x0012, 0x001b, 0x0024, 0x002d, 0x0036, 0x8000},
}
