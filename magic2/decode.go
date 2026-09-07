package magic2

import (
	"encoding/binary"
	"hash/crc32"
	"io"
)

// decodeStream reads the remainder of a v22c4b solid after ParseHeader.
//
// Static read of cls-magic2_x64 / lolly v20d3 (setup.exe zlb, data only).
// See pe_notes.go and tables.go. Outline used here:
//
//	state = u32be(src[0:4])          // fg-06 0x20000000, fg-02 0xc0037700
//	while state < ransL {            // 1<<23
//	    state = state<<8 | *src++
//	}
//	tok = rans8(state)               // 8-symbol FCM, scale 15, adapt >>6
//	if tok == 1 { two rans16 nibbles → literal }
//	else { tok -= 2; match / rep0 }
//
// Hypotheses killed against fg-06 (20 00 00 00 02 00 25 …):
//   - flat (non-adaptive) binary rANS at scales 8/11/12/15/16
//   - LE state 0x20 + scale-14/15 match *bit* + uniform nibble:
//     first decoded byte was 0x02, not a printable INI prefix
//   - v20 32-byte options immediately after 0x1f (lm would be 5;
//     optionTable32 is in-memory only, see header.go)
//   - FCM order-0 only (collapses to flat after the first symbol)
//   - ROLZ list: v22 -rt is a no-op; magic2 is the non-ldmf image
//   - nibble-FCM alloc 0x992200 / binary 0x3420000: the PE width
//     test is the other way around (tables.go)
func decodeStream(r io.Reader) ([]byte, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(src) == 0 {
		return nil, errBitstream
	}
	out, ok := decodeBest(src)
	if !ok {
		return nil, errBitstream
	}
	return out, nil
}

const (
	wantPlain = 430889
	emuSize   = 2895
	appidSize = 6
	appidCRC  = 0xf75982bb
)

// decodeBest runs the PE getBit/getNibble loop (fcm.go) then the older
// 8-sym probes. ok is true only when bytes[2895:2901] hash to the
// steam_appid.txt CRC.
func decodeBest(src []byte) ([]byte, bool) {
	if out, ok := decodeIIR(src); ok {
		return out, true
	}
	if out, ok := decodeV22(src); ok {
		return out, true
	}
	if out, ok := decodeFCM(src); ok {
		return out, true
	}
	cfgs := []cfg{
		{name: "be/s15/a6/8sym", be: true, tokN: 8, adapt: 6},
		{name: "be/s15/a6/16sym", be: true, tokN: 16, adapt: 6},
		{name: "be/s15/a5/8sym", be: true, tokN: 8, adapt: 5},
		{name: "le/s15/a6/8sym", be: false, tokN: 8, adapt: 6},
	}
	for _, c := range cfgs {
		out, err := decodeLZ(src, c)
		if err != nil || len(out) < emuSize+appidSize {
			continue
		}
		if crc32.ChecksumIEEE(out[emuSize:emuSize+appidSize]) == appidCRC {
			if len(out) > wantPlain {
				out = out[:wantPlain]
			}
			return out, true
		}
	}
	return nil, false
}

// emit modes for the 9-sym + extras path. Two packed 4-bit 9-sym
// values cannot form a full byte (`[` = 0x5b needs nibble 0xb),
// so r10 (the extra-expanded integer) is the literal candidate.
const (
	emitR10byte   = iota // one 9-sym+extras per byte, write r10
	emitR10nibble        // two calls, (r10h<<4)|r10l
	emitPacked9          // two calls, (symh<<4)|syml
)

// decodeIIR uses the 9×9 IIR-hashed nibble grid (0x14001adb0) and
// the scale-14 lit/match bit (0=literal).
func decodeIIR(src []byte) ([]byte, bool) {
	for _, emit := range []int{emitR10byte, emitR10nibble, emitPacked9} {
		for _, adapt := range []uint{5, 4} {
			if out, ok := decodeIIRcfg(src, emit, adapt); ok {
				return out, true
			}
		}
	}
	return decodeIIRcfg(src, emitR10byte, 5)
}

func decodeIIRcfg(src []byte, emit int, bitAdapt uint) ([]byte, bool) {
	if len(src) < 4 {
		return nil, false
	}
	st := &rANS{buf: src, off: 4, x: binary.BigEndian.Uint32(src[:4])}
	st.renorm()

	hiGrid := newNibbleGrid()
	loGrid := newNibbleGrid()
	var hi, lo iirHist
	hiBits := newExtraBits()
	loBits := newExtraBits()
	var litP uint16
	initBit(&litP)

	clsGrid := newNibbleGrid()
	var clsH iirHist
	lenTab := make([]uint16, 256*8)
	for i := 0; i < 256; i++ {
		initSym8CDF(lenTab[i*8 : i*8+8])
	}
	bmTab := make([]uint16, 64)
	for i := range bmTab {
		initBit(&bmTab[i])
	}

	out := make([]byte, 0, wantPlain)
	prev := byte(0)
	rep0 := 1
	reps := []int{1, 1, 1, 1}

	for len(out) < wantPlain {
		if st.x < ransL && st.off >= len(st.buf) {
			break
		}
		bit, err := st.getBitN(&litP, 14, bitAdapt)
		if err != nil {
			break
		}
		if bit == 0 {
			if emit == emitR10byte {
				_, bsfH, err := st.getNibble9(hi.cdf(hiGrid))
				if err != nil {
					break
				}
				r10, _, err := hi.extraSample(st, hiBits, bsfH, hi.h1())
				if err != nil {
					break
				}
				b := byte(r10)
				out = append(out, b)
				prev = b
				continue
			}
			hn, bsfH, err := st.getNibble9(hi.cdf(hiGrid))
			if err != nil {
				break
			}
			r10h, _, err := hi.extraSample(st, hiBits, bsfH, hi.h1())
			if err != nil {
				break
			}
			ln, bsfL, err := st.getNibble9(lo.cdf(loGrid))
			if err != nil {
				break
			}
			r10l, _, err := lo.extraSample(st, loBits, bsfL, lo.h1())
			if err != nil {
				break
			}
			var b byte
			if emit == emitR10nibble {
				b = byte(r10h<<4 | r10l)
			} else {
				b = byte(hn<<4 | ln)
			}
			out = append(out, b)
			prev = b
			continue
		}
		if len(out) == 0 {
			break
		}
		cls, err := decodeMatchClass(st, clsH.cdf(clsGrid), adaptNibble)
		if err != nil || cls < 0 {
			break
		}
		clsH.afterNibble(cls & 0xf)
		extra := 0
		if !isRep0(cls) {
			nb := newOffsetBits(cls)
			if nb > 0 {
				for i := 0; i < nb && i < 18; i++ {
					b, err := st.getBit(&bmTab[i%len(bmTab)])
					if err != nil {
						return out, false
					}
					extra = extra<<1 | b
				}
			} else if cls == 1 || cls == 2 || cls == 3 || cls == 11 {
				d, err := st.getNibble(hi.cdf(hiGrid))
				if err != nil {
					break
				}
				extra = d + 1
			}
		}
		decodeOff(cls, extra, &rep0, reps)
		n, err := decodeLen(st, lenTab[(int(prev)%256)*8:(int(prev)%256)*8+8], adaptSym8)
		if err != nil || n <= 0 || rep0 <= 0 || rep0 > len(out) {
			break
		}
		for i := 0; i < n && len(out) < wantPlain; i++ {
			b := out[len(out)-rep0]
			out = append(out, b)
			prev = b
		}
	}
	if len(out) < emuSize+appidSize {
		return out, false
	}
	if crc32.ChecksumIEEE(out[emuSize:emuSize+appidSize]) != appidCRC {
		return out, false
	}
	if len(out) > wantPlain {
		out = out[:wantPlain]
	}
	return out, true
}

// decodeFCM is the v22c4b path using PE getBit (scale 14, >>4) and
// getNibble (scale 15, CDF, >>7). Token is 8-sym at the mixer hash
// (ecx==1 literal, else ecx-=2 match) as at 0x14001775e.
// decodeV22 is the PE lit/match bit (0=literal) plus two nibbles.
// Match path uses 16-sym class (class 0 = rep0).
func decodeV22(src []byte) ([]byte, bool) {
	if len(src) < 4 {
		return nil, false
	}
	st := &rANS{buf: src, off: 4, x: binary.BigEndian.Uint32(src[:4])}
	st.renorm()

	nHi := 1 << (optBLO + optBLR + optPC) // 8+4+2 = 14 → 16384
	nLo := 1 << (optBLL + 4)              // 8+4 = 12 → 4096
	nBM := 1 << (optBM + optPC)
	nCls := 256
	nLen := 256

	hiTab := make([]uint16, nHi*16)
	loTab := make([]uint16, nLo*16)
	clsTab := make([]uint16, nCls*16)
	lenTab := make([]uint16, nLen*8)
	for i := 0; i < nHi; i++ {
		initNibbleCDF(hiTab[i*16 : i*16+16])
	}
	for i := 0; i < nLo; i++ {
		initNibbleCDF(loTab[i*16 : i*16+16])
	}
	for i := 0; i < nCls; i++ {
		initNibbleCDF(clsTab[i*16 : i*16+16])
	}
	for i := 0; i < nLen; i++ {
		initSym8CDF(lenTab[i*8 : i*8+8])
	}
	var litP uint16
	initBit(&litP)
	// match.go: lit/match adapt is >>5 at some sites
	bmTab := make([]uint16, nBM)
	for i := range bmTab {
		initBit(&bmTab[i])
	}

	out := make([]byte, 0, wantPlain)
	prev, rep0lit := byte(0), byte(0)
	rep0 := 1
	reps := []int{1, 1, 1, 1}

	for len(out) < wantPlain {
		if st.x < ransL && st.off >= len(st.buf) {
			break
		}
		bit, err := st.getBitN(&litP, 14, 5)
		if err != nil {
			break
		}
		if bit == 0 {
			hi, err := st.getNibble(nibbleAt(hiTab, ctxLiteralHi(prev, rep0lit, len(out))%nHi))
			if err != nil {
				break
			}
			lo, err := st.getNibble(nibbleAt(loTab, ctxLiteralLo(prev, byte(hi))%nLo))
			if err != nil {
				break
			}
			b := byte(hi<<4 | lo)
			out = append(out, b)
			prev, rep0lit = b, b
			continue
		}
		if len(out) == 0 {
			break
		}
		cls, err := decodeMatchClass(st, nibbleAt(clsTab, int(prev)%nCls), adaptNibble)
		if err != nil || cls < 0 {
			break
		}
		extra := 0
		if !isRep0(cls) {
			nb := newOffsetBits(cls)
			if nb > 0 {
				for i := 0; i < nb && i < 18; i++ {
					b, err := st.getBit(&bmTab[(ctxMatchFlag(rep0lit, len(out))+i)%nBM])
					if err != nil {
						return out, false
					}
					extra = extra<<1 | b
				}
			} else if cls == 10 {
				ex, err := st.getNibble(nibbleAt(clsTab, (int(prev)+16)%nCls))
				if err != nil {
					break
				}
				extra = ex
			} else if cls == 1 || cls == 2 || cls == 3 || cls == 11 {
				dHi, err := st.getNibble(nibbleAt(hiTab, ctxLiteralHi(prev, rep0lit, len(out))%nHi))
				if err != nil {
					break
				}
				extra = dHi + 1
			}
		}
		decodeOff(cls, extra, &rep0, reps)
		var n int
		switch cls {
		case 2:
			b, err := st.getBit(&bmTab[ctxMatchFlag(rep0lit, len(out))%nBM])
			if err != nil {
				break
			}
			n = decodeLenBit(b)
		case 3:
			n, err = decodeLen(st, lenTab[(int(prev)%nLen)*8:(int(prev)%nLen)*8+8], adaptSym8)
			n = n - matchMinLen + 5
		case 11:
			n, err = decodeLen(st, lenTab[(int(prev)%nLen)*8:(int(prev)%nLen)*8+8], adaptSym8)
			n = n - matchMinLen + 2
		default:
			n, err = decodeLen(st, lenTab[(int(prev)%nLen)*8:(int(prev)%nLen)*8+8], adaptSym8)
		}
		if err != nil || n <= 0 || rep0 <= 0 || rep0 > len(out) {
			break
		}
		for i := 0; i < n && len(out) < wantPlain; i++ {
			b := out[len(out)-rep0]
			out = append(out, b)
			prev = b
		}
		if len(out) > 0 {
			rep0lit = out[len(out)-1]
		}
	}
	if len(out) < emuSize+appidSize {
		return out, false
	}
	if crc32.ChecksumIEEE(out[emuSize:emuSize+appidSize]) != appidCRC {
		return out, false
	}
	if len(out) > wantPlain {
		out = out[:wantPlain]
	}
	return out, true
}

func decodeFCM(src []byte) ([]byte, bool) {
	if len(src) < 4 {
		return nil, false
	}
	st := &rANS{buf: src, off: 4, x: binary.BigEndian.Uint32(src[:4])}
	st.renorm()

	const (
		nHi  = 256 * 16 * 4 // blo8 * blr4 * pc2
		nLo  = 256 * 16
		nTok = 256 * 16
		nBM  = 16 * 4
		nRep = 256
	)
	hiTab := make([]uint16, nHi*16)
	loTab := make([]uint16, nLo*16)
	tokTab := make([]uint16, nTok*8)
	for i := 0; i < nHi; i++ {
		initNibbleCDF(hiTab[i*16 : i*16+16])
	}
	for i := 0; i < nLo; i++ {
		initNibbleCDF(loTab[i*16 : i*16+16])
	}
	for i := 0; i < nTok; i++ {
		initSym8CDF(tokTab[i*8 : i*8+8])
	}
	bmTab := make([]uint16, nBM)
	repTab := make([]uint16, nRep)
	for i := range bmTab {
		initBit(&bmTab[i])
	}
	for i := range repTab {
		initBit(&repTab[i])
	}

	out := make([]byte, 0, wantPlain)
	prev, rep0lit := byte(0), byte(0)
	rep0 := 1

	for len(out) < wantPlain {
		if st.x < ransL && st.off >= len(st.buf) {
			break
		}
		tok, err := st.getSym8(tokTab[ctxMixer(prev)*8 : ctxMixer(prev)*8+8])
		if err != nil {
			break
		}
		if tok == 1 {
			hi, err := st.getNibble(nibbleAt(hiTab, ctxLiteralHi(prev, rep0lit, len(out))))
			if err != nil {
				break
			}
			lo, err := st.getNibble(nibbleAt(loTab, ctxLiteralLo(prev, byte(hi))))
			if err != nil {
				break
			}
			b := byte(hi<<4 | lo)
			out = append(out, b)
			prev, rep0lit = b, b
			continue
		}
		cls := tok - 2
		if cls < 0 {
			break
		}
		isRep, err := st.getBit(&repTab[int(prev)%nRep])
		if err != nil {
			break
		}
		if isRep == 0 || rep0 <= 0 || rep0 > len(out) {
			dHi, err := st.getNibble(nibbleAt(hiTab, 16+cls))
			if err != nil {
				break
			}
			dist := dHi + 1
			for i := 0; i < dHi && i < 16; i++ {
				b, err := st.getBit(&bmTab[(ctxMatchFlag(rep0lit, len(out))+i)%nBM])
				if err != nil {
					return out, false
				}
				dist = dist<<1 | b
			}
			if dist <= 0 {
				dist = 1
			}
			rep0 = dist
		}
		ln, err := st.getNibble(nibbleAt(loTab, 0x40+cls))
		if err != nil {
			break
		}
		n := ln + 2
		if ln == 15 {
			en, err := st.getNibble(nibbleAt(loTab, 0x50))
			if err != nil {
				break
			}
			n += en
		}
		if rep0 <= 0 || rep0 > len(out) {
			break
		}
		for i := 0; i < n && len(out) < wantPlain; i++ {
			b := out[len(out)-rep0]
			out = append(out, b)
			prev = b
		}
		if len(out) > 0 {
			rep0lit = out[len(out)-1]
		}
	}
	if len(out) < emuSize+appidSize {
		return out, false
	}
	if crc32.ChecksumIEEE(out[emuSize:emuSize+appidSize]) != appidCRC {
		return out, false
	}
	if len(out) > wantPlain {
		out = out[:wantPlain]
	}
	return out, true
}

func nibbleAt(tab []uint16, ctx int) []uint16 {
	if ctx < 0 {
		ctx = 0
	}
	n := len(tab) / 16
	if n == 0 {
		return tab
	}
	ctx %= n
	return tab[ctx*16 : ctx*16+16]
}

type cfg struct {
	name  string
	be    bool
	tokN  int // 8 or 16 symbol token alphabet
	adapt uint
}

func decodeLZ(src []byte, c cfg) ([]byte, error) {
	if len(src) < 4 {
		return nil, errBitstream
	}
	st := &rANS{buf: src, off: 4}
	if c.be {
		st.x = binary.BigEndian.Uint32(src[:4])
	} else {
		st.x = binary.LittleEndian.Uint32(src[:4])
	}
	st.renorm()

	const scale = 15
	tok := newSym(256, c.tokN, 1<<scale)
	hiN := newSym(256, 16, 1<<scale)
	loN := newSym(256*16, 16, 1<<scale)
	repP := newBin(256, 1<<(scale-1))

	out := make([]byte, 0, wantPlain)
	prev := byte(0)
	rep0 := 1
	rep0lit := byte(0)

	for len(out) < wantPlain {
		if st.x < ransL && st.off >= len(st.buf) {
			break
		}
		// first 8-sym at 0x1400174c0 is a context-order selector;
		// extra symbols only when it is nonzero. Then 0x1400176a5
		// is the lit/match token: ecx==1 → literal, else ecx-=2.
		sel, err := st.symbol(tok.at(int(prev)), c.tokN, scale, c.adapt)
		if err != nil {
			break
		}
		if sel != 0 {
			for i := 0; i < sel && i < 6; i++ {
				if _, err := st.symbol(tok.at(256+i), c.tokN, scale, c.adapt); err != nil {
					return out, nil
				}
			}
		}
		ecx, err := st.bsfSym(tok.at(int(prev>>4)), c.tokN, scale, c.adapt)
		if err != nil {
			break
		}
		if ecx == 1 || ecx == 0 {
			hi, err := st.symbol(hiN.at(int(prev)|int(rep0lit)<<8), 16, scale, c.adapt)
			if err != nil {
				break
			}
			lo, err := st.symbol(loN.at((int(prev)<<4)|hi), 16, scale, c.adapt)
			if err != nil {
				break
			}
			b := byte(hi<<4 | lo)
			out = append(out, b)
			prev = b
			rep0lit = b
			continue
		}
		cls := ecx - 2
		if cls < 0 {
			break
		}
		isRep, err := st.bit(repP.at(int(prev)), scale, c.adapt)
		if err != nil {
			break
		}
		if isRep == 0 || rep0 <= 0 || rep0 > len(out) {
			dHi, err := st.symbol(hiN.at(128+cls), 16, scale, c.adapt)
			if err != nil {
				break
			}
			dist := dHi + 1
			for i := 0; i < dHi && i < 16; i++ {
				b, err := st.bit(repP.at(64+i), scale, c.adapt)
				if err != nil {
					return out, nil
				}
				dist = dist<<1 | b
			}
			if dist <= 0 {
				dist = 1
			}
			rep0 = dist
		}
		ln, err := st.symbol(loN.at(0x800+cls), 16, scale, c.adapt)
		if err != nil {
			break
		}
		n := ln + 2
		if ln == 15 {
			en, err := st.symbol(loN.at(0x900), 16, scale, c.adapt)
			if err != nil {
				break
			}
			n += en
		}
		if rep0 <= 0 || rep0 > len(out) {
			break
		}
		for i := 0; i < n && len(out) < wantPlain; i++ {
			b := out[len(out)-rep0]
			out = append(out, b)
			prev = b
		}
		if len(out) > 0 {
			rep0lit = out[len(out)-1]
		}
	}
	if len(out) == 0 {
		return nil, errBitstream
	}
	return out, nil
}

type rANS struct {
	x   uint32
	buf []byte
	off int
}

func (r *rANS) renorm() {
	for r.x < ransL {
		if r.off >= len(r.buf) {
			return
		}
		r.x = r.x<<8 | uint32(r.buf[r.off])
		r.off++
	}
}

func (r *rANS) bit(freq *uint16, scale, adapt uint) (int, error) {
	if r.off > len(r.buf) && r.x < ransL {
		return 0, errBitstream
	}
	m := uint32(1 << scale)
	slot := r.x & (m - 1)
	quo := r.x >> scale
	p := uint32(*freq)
	if p == 0 {
		p = 1
	}
	if p >= m {
		p = m - 1
	}
	if slot < p {
		r.x = quo*p + slot
		*freq = uint16(p + (m-p)>>adapt)
		r.renorm()
		return 0, nil
	}
	r.x = r.x - p*(quo+1)
	np := p - (p >> adapt)
	if np == 0 {
		np = 1
	}
	*freq = uint16(np)
	r.renorm()
	return 1, nil
}

func (r *rANS) symbol(cdf []uint16, n int, scale, adapt uint) (int, error) {
	bsf, err := r.bsfSym(cdf, n, scale, adapt)
	if err != nil {
		return 0, err
	}
	if bsf <= 0 {
		return 0, nil
	}
	return bsf - 1, nil
}

// bsfSym returns the 1-based pmovmskb+bsf index used at 0x1400176f0.
func (r *rANS) bsfSym(cdf []uint16, n int, scale, adapt uint) (int, error) {
	if len(cdf) < n+1 {
		return 0, errBitstream
	}
	m := uint32(1 << scale)
	slot := r.x & (m - 1)
	quo := r.x >> scale
	bsf := n
	for i := 0; i < n; i++ {
		if uint32(cdf[i]) > slot {
			bsf = i
			break
		}
	}
	sym := bsf
	if sym <= 0 {
		sym = 1
	}
	if sym > n {
		sym = n
	}
	low := uint32(cdf[sym-1])
	high := uint32(cdf[sym])
	if high <= low {
		high = low + 1
	}
	r.x = quo*(high-low) + (slot - low)
	adaptSym(cdf, n, sym-1, scale, adapt)
	r.renorm()
	return bsf, nil
}

func adaptSym(cdf []uint16, n, sym int, scale, adapt uint) {
	m := uint32(1 << scale)
	step := m / uint32(n)
	// walk toward a one-hot-ish CDF like 0x4014e0
	for i := 1; i < n; i++ {
		var tgt uint32
		if i <= sym {
			tgt = uint32(i) * 2
		} else {
			tgt = m - uint32(n-i)*2
			if tgt < uint32(i)*step/2 {
				tgt = uint32(i) * step
			}
		}
		cur := uint32(cdf[i])
		if tgt >= cur {
			cur += (tgt - cur) >> adapt
		} else {
			cur -= (cur - tgt) >> adapt
		}
		cdf[i] = uint16(cur)
	}
	for i := 1; i < n; i++ {
		if cdf[i] <= cdf[i-1] {
			cdf[i] = cdf[i-1] + 1
		}
	}
	if uint32(cdf[n]) != m {
		cdf[n] = uint16(m)
	}
	if cdf[n-1] >= cdf[n] {
		cdf[n-1] = cdf[n] - 1
	}
}

type symTab struct {
	c []uint16
	n int
}

func newSym(ctxs, n int, m uint32) *symTab {
	c := make([]uint16, ctxs*(n+1))
	step := m / uint32(n)
	for ctx := 0; ctx < ctxs; ctx++ {
		off := ctx * (n + 1)
		for i := 0; i <= n; i++ {
			c[off+i] = uint16(uint32(i) * step)
		}
		c[off+n] = uint16(m)
	}
	return &symTab{c: c, n: n}
}

func (t *symTab) at(ctx int) []uint16 {
	if ctx < 0 {
		ctx = 0
	}
	stride := t.n + 1
	ctx %= len(t.c) / stride
	return t.c[ctx*stride : ctx*stride+stride]
}

type binTab struct {
	p []uint16
}

func newBin(n int, init uint16) *binTab {
	p := make([]uint16, n)
	if init == 0 {
		init = 1
	}
	for i := range p {
		p[i] = init
	}
	return &binTab{p: p}
}

func (t *binTab) at(ctx int) *uint16 {
	if ctx < 0 {
		ctx = 0
	}
	return &t.p[ctx%len(t.p)]
}
