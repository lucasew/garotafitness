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

// decodeBest runs the v22 8-symbol token + nibble-literal loop.
// ok is true only when bytes[2895:2901] hash to the steam_appid.txt CRC.
func decodeBest(src []byte) ([]byte, bool) {
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
