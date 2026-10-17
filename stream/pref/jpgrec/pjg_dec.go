package jpgrec

// PJG → JPEG restore, translated from packjpg.cpp (BUILD_LIB, F_PJG).
// USE_PLOCOI and DEV_BUILD are off, so DC uses dc_1ddct_predictor and the
// 8x8 IDCT table is not consulted on this path.

import (
	"errors"
	"fmt"
	"math"
)

const (
	pjgAppVersion = 25
	// jpgMaxMemory is JPG_MAX_MEMORY_SIZE from precomp_jpg.h (512 MiB).
	jpgMaxMemory = 512 * 1024 * 1024
	// mjpgDHTLen is MJPGDHT_LEN from precomp_jpg.h.
	mjpgDHTLen = 420
)

type huffTab struct {
	cval   [256]uint16
	clen   [256]uint16
	maxEob int32
}

type component struct {
	qIndex         int32
	huffDC, huffAC int32
	sfv, sfh       int32
	mbs            int32
	bcv, bch, bc   int32
	ncv, nch, nc   int32
	sid, jid       int32
}

type mcuSt struct {
	mcu, cmp, csc, sub, dpos, rstw int32
}

type pjg struct {
	qtables [4][64]uint16
	hcodes  [2][4]huffTab

	hdr    []byte
	huff   []byte
	grbg   []byte
	rstErr []byte
	rstp   []uint32
	scnp   []uint32
	rstc   int32
	scnc   int32
	rsti   int32
	padbit int32

	zdst  [4][]byte
	eobx  [4][]byte
	eoby  [4][]byte
	zdstX [4][]byte
	zdstY [4][]byte
	coll  [4][64][]int16
	zsrt  [4][64]byte

	cmp              [4]component
	cmpc             int32
	w, h             int32
	sfhm, sfvm       int32
	mcuv, mcuh, mcuc int32
	csCmpc           int32
	csCmp            [4]int32
	csFrom, csTo     int32
	csSah, csSal     int32
	jpegType         int32
	nois             [4]byte
	segm             [4]byte
	autoSet          bool
	errLevel         int
	errMsg           string
	adpt1x8, adpt8x1 [4][64]int32
}

func newPJG() *pjg {
	p := &pjg{autoSet: true, padbit: -1}
	for i := 0; i < 4; i++ {
		p.nois[i] = 6
		p.segm[i] = 10
		c := &p.cmp[i]
		c.qIndex = -1
		c.huffDC = -1
		c.huffAC = -1
		c.sfv = -1
		c.sfh = -1
		c.mbs = -1
		c.bcv = -1
		c.bch = -1
		c.bc = -1
		c.ncv = -1
		c.nch = -1
		c.nc = -1
		c.sid = -1
		c.jid = -1
	}
	return p
}

func (p *pjg) fail(level int, format string, args ...any) error {
	p.errLevel = level
	p.errMsg = fmt.Sprintf(format, args...)
	return errors.New("jpgrec: " + p.errMsg)
}

func bshort(a, b byte) int { return int(a)<<8 | int(b) }

func lbits(c byte, n uint) int { return int(c >> (8 - n)) }
func rbits(c byte, n uint) int { return int(c & (0xFF >> (8 - n))) }

func bitlenPos(v int32) int32 {
	if v <= 0 {
		return 0
	}
	var n int32
	u := uint32(v)
	for u > 0 {
		u >>= 1
		n++
	}
	return n
}

func bitlen2048(v int32) int32 {
	if v < 0 {
		v = -v
	}
	return bitlenPos(v)
}

func envli(s, v int32) uint32 {
	var r int32
	if v > 0 {
		r = v
	} else {
		r = (v - 1) + (int32(1) << uint(s))
	}
	return uint32(uint16(r))
}

func eEnvli(s, v int32) uint32 {
	r := v - (int32(1) << uint(s))
	return uint32(uint16(r))
}

func fdiv2(v, p int32) int32 {
	if p <= 0 {
		return v
	}
	if v < 0 {
		return -((-v) >> uint(p))
	}
	return v >> uint(p)
}

func clamped(lo, hi, v int32) int32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func bitn(c, n int32) int32 { return (c >> uint(n)) & 1 }

func dctRescale(v int32) int32 {
	if v > 0 {
		return (v + 4096) / 8192
	}
	return (v - 4096) / 8192
}

func ceilDivFF(num, den int32) int32 {
	q := float32(num) / float32(den)
	return int32(math.Ceil(float64(q)))
}

func ceilN(img, sf, sfm int32) int32 {
	ratio := float64(float32(sf)) / (8.0 * float64(sfm))
	return int32(math.Ceil(float64(float32(img)) * ratio))
}

func (p *pjg) q(cmp, bp int) uint16 {
	qi := p.cmp[cmp].qIndex
	if qi < 0 {
		return 0
	}
	return p.qtables[qi][bp]
}

func (p *pjg) maxV(cmp, bp int) int32 {
	qv := int32(p.q(cmp, bp))
	if qv > 0 {
		return (int32(freqmax[bp]) + qv - 1) / qv
	}
	return 0
}

func (r *memReader) readFull(n int) ([]byte, error) {
	if n < 0 || r.i+n > len(r.b) {
		return nil, errors.New("jpgrec: unexpected end of packJPG stream")
	}
	s := r.b[r.i : r.i+n]
	r.i += n
	return s, nil
}

func decodeGeneric(a *arith) ([]byte, error) {
	m := newModelS(257, 256, 1)
	bw := byteBuf{data: make([]byte, 256)}
	for {
		c, err := decodeS(a, m)
		if err != nil {
			return nil, err
		}
		if c == 256 {
			break
		}
		bw.write(byte(c))
		m.shift(c)
	}
	out := make([]byte, bw.n)
	copy(out, bw.data[:bw.n])
	return out, nil
}

func decodeBit(a *arith) (byte, error) {
	m := newModelB(1, -1)
	b, err := decodeB(a, m)
	if err != nil {
		return 0, err
	}
	return byte(b), nil
}

func (p *pjg) unpack(r *memReader) error {
	for {
		h, err := r.u8()
		if err != nil {
			return err
		}
		if h == 0x00 {
			nb, err := r.readFull(4)
			if err != nil {
				return err
			}
			copy(p.nois[:], nb)
			sb, err := r.readFull(4)
			if err != nil {
				return err
			}
			copy(p.segm[:], sb)
			p.autoSet = false
		} else if h >= 0x14 {
			if h != pjgAppVersion {
				return p.fail(2, "incompatible file, use packjpg v%d.%d", int(h)/10, int(h)%10)
			}
			break
		} else {
			return p.fail(2, "unknown header code, use newer version of packjpg")
		}
	}
	dec := newArith(r)
	hdr, err := decodeGeneric(dec)
	if err != nil {
		return err
	}
	p.hdr = hdr
	cb, err := decodeBit(dec)
	if err != nil {
		return err
	}
	p.padbit = int32(cb)
	cb, err = decodeBit(dec)
	if err != nil {
		return err
	}
	if cb == 1 {
		rst, err := decodeGeneric(dec)
		if err != nil {
			return err
		}
		p.rstErr = rst
	}
	if err := p.unoptimize(); err != nil {
		return err
	}
	if err := p.setup(); err != nil {
		return err
	}
	for cmp := 0; cmp < int(p.cmpc); cmp++ {
		if err := p.decZst(dec, cmp); err != nil {
			return err
		}
		if err := p.decZdstHigh(dec, cmp); err != nil {
			return err
		}
		if err := p.decACHigh(dec, cmp); err != nil {
			return err
		}
		if err := p.decZdstLow(dec, cmp); err != nil {
			return err
		}
		if err := p.decACLow(dec, cmp); err != nil {
			return err
		}
		if err := p.decDC(dec, cmp); err != nil {
			return err
		}
	}
	cb, err = decodeBit(dec)
	if err != nil {
		return err
	}
	if cb == 0 {
		p.grbg = nil
	} else {
		g, err := decodeGeneric(dec)
		if err != nil {
			return err
		}
		p.grbg = g
	}
	return nil
}

func (p *pjg) unoptimize() error {
	hdr := p.hdr
	hpos := 0
	for hpos < len(hdr) {
		if hpos+4 > len(hdr) {
			return p.fail(2, "truncated jpeg header")
		}
		typ := hdr[hpos+1]
		ln := 2 + bshort(hdr[hpos+2], hdr[hpos+3])
		if ln < 4 || hpos+ln > len(hdr) {
			return p.fail(2, "truncated jpeg header")
		}
		if typ == 0xC4 {
			fpos := hpos + ln
			hpos += 4
			for hpos < fpos {
				hpos++
				if hpos >= len(hdr) {
					return p.fail(2, "truncated dht")
				}
				if hdr[hpos] > 2 {
					i := int(hdr[hpos+1])
					if i < 0 || i >= len(stdHuffTables) {
						return p.fail(2, "bad standard huffman table id")
					}
					tab := stdHuffTables[i]
					if hpos+len(tab) > len(hdr) {
						return p.fail(2, "truncated standard huffman table")
					}
					copy(hdr[hpos:hpos+len(tab)], tab)
				}
				if hpos+16 > len(hdr) {
					return p.fail(2, "truncated dht")
				}
				skip := 16
				for i := 0; i < 16; i++ {
					skip += int(hdr[hpos+i])
				}
				hpos += skip
			}
		} else if typ == 0xDB {
			fpos := hpos + ln
			hpos += 4
			for hpos < fpos {
				if hpos >= len(hdr) {
					return p.fail(2, "truncated dqt")
				}
				prec := lbits(hdr[hpos], 4)
				hpos++
				if prec == 1 {
					hpos += 128
					continue
				}
				if hpos+64 > len(hdr) {
					return p.fail(2, "truncated dqt")
				}
				for spos := 1; spos < 64; spos++ {
					hdr[hpos+spos] += hdr[hpos+spos-1]
				}
				hpos += 64
			}
		} else {
			hpos += ln
		}
	}
	return nil
}

func (p *pjg) setup() error {
	hpos := 0
	for hpos < len(p.hdr) {
		if hpos+4 > len(p.hdr) {
			return p.fail(2, "truncated jpeg header")
		}
		typ := p.hdr[hpos+1]
		ln := 2 + bshort(p.hdr[hpos+2], p.hdr[hpos+3])
		if ln < 4 || hpos+ln > len(p.hdr) {
			return p.fail(2, "truncated jpeg header")
		}
		if typ != 0xDA && typ != 0xC4 && typ != 0xDD {
			if err := p.parse(p.hdr[hpos : hpos+ln]); err != nil {
				return err
			}
			if p.errLevel >= 1 {
				return errors.New("jpgrec: " + p.errMsg)
			}
		}
		hpos += ln
	}
	if p.cmpc == 0 {
		return p.fail(2, "header contains incomplete information")
	}
	for cmp := int32(0); cmp < p.cmpc; cmp++ {
		c := &p.cmp[cmp]
		if c.sfv == 0 || c.sfh == 0 || c.qIndex < 0 || p.qtables[c.qIndex][0] == 0 || p.jpegType == 0 {
			return p.fail(2, "header information is incomplete")
		}
	}
	for cmp := int32(0); cmp < p.cmpc; cmp++ {
		if p.cmp[cmp].sfh > p.sfhm {
			p.sfhm = p.cmp[cmp].sfh
		}
		if p.cmp[cmp].sfv > p.sfvm {
			p.sfvm = p.cmp[cmp].sfv
		}
	}
	p.mcuv = ceilDivFF(p.h, 8*p.sfhm)
	p.mcuh = ceilDivFF(p.w, 8*p.sfvm)
	p.mcuc = p.mcuv * p.mcuh
	for cmp := int32(0); cmp < p.cmpc; cmp++ {
		c := &p.cmp[cmp]
		c.mbs = c.sfv * c.sfh
		c.bcv = p.mcuv * c.sfh
		c.bch = p.mcuh * c.sfv
		c.bc = c.bcv * c.bch
		c.ncv = ceilN(p.h, c.sfh, p.sfhm)
		c.nch = ceilN(p.w, c.sfv, p.sfvm)
		c.nc = c.ncv * c.nch
		if c.bc < 0 || c.bch <= 0 {
			return p.fail(2, "bad block geometry")
		}
	}
	if p.cmpc <= 3 {
		for cmp := int32(0); cmp < p.cmpc; cmp++ {
			p.cmp[cmp].sid = cmp
		}
	} else {
		for cmp := int32(0); cmp < p.cmpc; cmp++ {
			p.cmp[cmp].sid = 0
		}
	}
	for cmp := int32(0); cmp < p.cmpc; cmp++ {
		bc := int(p.cmp[cmp].bc)
		for bpos := 0; bpos < 64; bpos++ {
			p.coll[cmp][bpos] = make([]int16, bc)
		}
		p.zdst[cmp] = make([]byte, bc)
		p.eobx[cmp] = make([]byte, bc)
		p.eoby[cmp] = make([]byte, bc)
		p.zdstX[cmp] = make([]byte, bc)
		p.zdstY[cmp] = make([]byte, bc)
	}
	if p.autoSet {
		for cmp := int32(0); cmp < p.cmpc; cmp++ {
			sid := p.cmp[cmp].sid
			if sid < 0 || int(sid) >= len(confSets[0]) {
				return p.fail(2, "bad statistical id")
			}
			bc := uint32(p.cmp[cmp].bc)
			i := 0
			for confSets[i][sid] > bc {
				i++
				if i >= len(confSets) {
					return p.fail(2, "no automatic setting")
				}
			}
			p.segm[cmp] = confSegm[i][sid]
			p.nois[cmp] = confNtrs[i][sid]
		}
	}
	return nil
}

func (p *pjg) parse(seg []byte) error {
	if len(seg) < 4 {
		return p.fail(2, "truncated marker")
	}
	typ := seg[1]
	hpos := 4
	switch typ {
	case 0xC4:
		for hpos < len(seg) {
			lval := lbits(seg[hpos], 4)
			rval := rbits(seg[hpos], 4)
			if lval < 0 || lval >= 2 || rval < 0 || rval >= 4 {
				break
			}
			hpos++
			if hpos+16 > len(seg) {
				return p.fail(2, "size mismatch in dht marker")
			}
			skip := 16
			for i := 0; i < 16; i++ {
				skip += int(seg[hpos+i])
			}
			if hpos+skip > len(seg) {
				return p.fail(2, "size mismatch in dht marker")
			}
			hc, err := buildHuff(seg[hpos:hpos+16], seg[hpos+16:hpos+skip])
			if err != nil {
				return err
			}
			p.hcodes[lval][rval] = hc
			hpos += skip
		}
		if hpos != len(seg) {
			return p.fail(2, "size mismatch in dht marker")
		}
		return nil
	case 0xDB:
		for hpos < len(seg) {
			lval := lbits(seg[hpos], 4)
			rval := rbits(seg[hpos], 4)
			if lval < 0 || lval >= 2 || rval < 0 || rval >= 4 {
				break
			}
			hpos++
			if lval == 0 {
				if hpos+64 > len(seg) {
					break
				}
				for i := 0; i < 64; i++ {
					p.qtables[rval][i] = uint16(seg[hpos+i])
					if p.qtables[rval][i] == 0 {
						break
					}
				}
				hpos += 64
			} else {
				if hpos+128 > len(seg) {
					break
				}
				for i := 0; i < 64; i++ {
					p.qtables[rval][i] = uint16(bshort(seg[hpos+2*i], seg[hpos+2*i+1]))
					if p.qtables[rval][i] == 0 {
						break
					}
				}
				hpos += 128
			}
		}
		if hpos != len(seg) {
			return p.fail(2, "size mismatch in dqt marker")
		}
		return nil
	case 0xDD:
		if hpos+2 > len(seg) {
			return p.fail(2, "truncated dri")
		}
		p.rsti = int32(bshort(seg[hpos], seg[hpos+1]))
		return nil
	case 0xDA:
		if hpos >= len(seg) {
			return p.fail(2, "truncated sos")
		}
		p.csCmpc = int32(seg[hpos])
		if p.csCmpc > p.cmpc {
			return p.fail(2, "%d components in scan, only %d are allowed", p.csCmpc, p.cmpc)
		}
		hpos++
		for i := int32(0); i < p.csCmpc; i++ {
			if hpos+2 > len(seg) {
				return p.fail(2, "truncated sos")
			}
			cmp := int32(0)
			for cmp < p.cmpc && seg[hpos] != byte(p.cmp[cmp].jid) {
				cmp++
			}
			if cmp == p.cmpc {
				return p.fail(2, "component id mismatch in start-of-scan")
			}
			p.csCmp[i] = cmp
			p.cmp[cmp].huffDC = int32(lbits(seg[hpos+1], 4))
			p.cmp[cmp].huffAC = int32(rbits(seg[hpos+1], 4))
			if p.cmp[cmp].huffDC < 0 || p.cmp[cmp].huffDC >= 4 || p.cmp[cmp].huffAC < 0 || p.cmp[cmp].huffAC >= 4 {
				return p.fail(2, "huffman table number mismatch")
			}
			hpos += 2
		}
		if hpos+3 > len(seg) {
			return p.fail(2, "truncated sos")
		}
		p.csFrom = int32(seg[hpos])
		p.csTo = int32(seg[hpos+1])
		p.csSah = int32(lbits(seg[hpos+2], 4))
		p.csSal = int32(rbits(seg[hpos+2], 4))
		if p.csFrom > p.csTo || p.csFrom > 63 || p.csTo > 63 {
			return p.fail(2, "spectral selection parameter out of range")
		}
		if p.csSah >= 12 || p.csSal >= 12 {
			return p.fail(2, "successive approximation parameter out of range")
		}
		return nil
	case 0xC0, 0xC1, 0xC2:
		if typ == 0xC2 {
			p.jpegType = 2
		} else {
			p.jpegType = 1
		}
		if hpos+6 > len(seg) {
			return p.fail(2, "truncated sof")
		}
		if seg[hpos] != 8 {
			return p.fail(2, "%d bit data precision is not supported", seg[hpos])
		}
		p.h = int32(bshort(seg[hpos+1], seg[hpos+2]))
		p.w = int32(bshort(seg[hpos+3], seg[hpos+4]))
		p.cmpc = int32(seg[hpos+5])
		if p.w == 0 || p.h == 0 {
			return p.fail(2, "resolution is %dx%d, possible malformed JPEG", p.w, p.h)
		}
		if p.cmpc > 4 {
			return p.fail(2, "image has %d components, max 4 are supported", p.cmpc)
		}
		hpos += 6
		for cmp := int32(0); cmp < p.cmpc; cmp++ {
			if hpos+3 > len(seg) {
				return p.fail(2, "truncated sof")
			}
			p.cmp[cmp].jid = int32(seg[hpos])
			p.cmp[cmp].sfv = int32(lbits(seg[hpos+1], 4))
			p.cmp[cmp].sfh = int32(rbits(seg[hpos+1], 4))
			if seg[hpos+2] < 4 {
				p.cmp[cmp].qIndex = int32(seg[hpos+2])
			}
			hpos += 3
		}
		return nil
	case 0xC3:
		return p.fail(2, "sof3 marker found, image is coded lossless")
	case 0xC5:
		return p.fail(2, "sof5 marker found, image is coded diff. sequential")
	case 0xC6:
		return p.fail(2, "sof6 marker found, image is coded diff. progressive")
	case 0xC7:
		return p.fail(2, "sof7 marker found, image is coded diff. lossless")
	case 0xC9:
		return p.fail(2, "sof9 marker found, image is coded arithm. sequential")
	case 0xCA:
		return p.fail(2, "sof10 marker found, image is coded arithm. progressive")
	case 0xCB:
		return p.fail(2, "sof11 marker found, image is coded arithm. lossless")
	case 0xCD:
		return p.fail(2, "sof13 marker found, image is coded arithm. diff. sequential")
	case 0xCE:
		return p.fail(2, "sof14 marker found, image is coded arithm. diff. progressive")
	case 0xCF:
		return p.fail(2, "sof15 marker found, image is coded arithm. diff. lossless")
	case 0xE0, 0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7, 0xE8, 0xE9, 0xEA, 0xEB, 0xEC, 0xED, 0xEE, 0xEF, 0xFE:
		return nil
	case 0xD0, 0xD1, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7:
		return p.fail(2, "rst marker found out of place")
	case 0xD8:
		return p.fail(2, "soi marker found out of place")
	case 0xD9:
		return p.fail(2, "eoi marker found out of place")
	default:
		p.errLevel = 1
		p.errMsg = fmt.Sprintf("unknown marker found: FF %2X", typ)
		return nil
	}
}

func buildHuff(counts, symbols []byte) (huffTab, error) {
	var hc huffTab
	if len(counts) < 16 {
		return hc, errors.New("jpgrec: truncated huffman table")
	}
	k := 0
	code := int32(0)
	for i := 0; i < 16; i++ {
		n := int(counts[i])
		for j := 0; j < n; j++ {
			if k >= len(symbols) {
				return hc, errors.New("jpgrec: truncated huffman symbols")
			}
			sym := int(symbols[k])
			hc.clen[sym] = uint16(1 + i)
			hc.cval[sym] = uint16(code)
			k++
			code++
		}
		code <<= 1
	}
	for i := 14; i >= 0; i-- {
		if hc.clen[i<<4] > 0 {
			hc.maxEob = (int32(2) << uint(i)) - 1
			break
		}
	}
	return hc, nil
}

func nnb(pos, w int) (a, b int) {
	if pos == 0 {
		return -1, -1
	}
	if w != 0 && pos%w == 0 {
		b = pos - w
		if pos >= (w << 1) {
			a = pos - (w << 1)
		} else {
			a = b
		}
		return a, b
	}
	if pos < w {
		a = pos - 1
		if pos >= 2 {
			b = pos - 2
		} else {
			b = a
		}
		return a, b
	}
	return pos - 1, pos - w
}

func (p *pjg) decZst(a *arith, cmp int) error {
	p.zsrt[cmp][0] = 0
	freq := stdscan
	m := newModelS(64, 64, 1)
	for i := 1; i < 64; i++ {
		m.excludeAbove(int32(64 - i))
		cpos, err := decodeS(a, m)
		if err != nil {
			return err
		}
		m.shift(cpos)
		if cpos == 0 {
			tpos := 0
			for ; i < 64; i++ {
				for {
					tpos++
					if tpos >= 64 {
						return p.fail(2, "zero sort scan overrun")
					}
					if freq[tpos] != 0 {
						break
					}
				}
				p.zsrt[cmp][i] = freq[tpos]
			}
			break
		}
		tpos := 0
		found := false
		for ; tpos < 64; tpos++ {
			if freq[tpos] != 0 {
				cpos--
			}
			if cpos == 0 {
				found = true
				break
			}
		}
		if !found {
			return p.fail(2, "zero sort scan position")
		}
		p.zsrt[cmp][i] = freq[tpos]
		freq[tpos] = 0
	}
	return nil
}

func (p *pjg) decZdstHigh(a *arith, cmp int) error {
	m := newModelS(49+1, 25+1, 1)
	z := p.zdst[cmp]
	w := int(p.cmp[cmp].bch)
	for dpos := 0; dpos < len(z); dpos++ {
		aa, bb := nnb(dpos, w)
		av, bv := 0, 0
		if aa >= 0 {
			av = int(z[aa])
		}
		if bb >= 0 {
			bv = int(z[bb])
		}
		m.shift(int32((av + bv + 2) / 4))
		c, err := decodeS(a, m)
		if err != nil {
			return err
		}
		z[dpos] = byte(c)
	}
	return nil
}

func (p *pjg) decZdstLow(a *arith, cmp int) error {
	m := newModelS(8, 8, 2)
	bc := len(p.zdst[cmp])
	for dpos := 0; dpos < bc; dpos++ {
		shiftS(m, int32((int(p.zdst[cmp][dpos])+3)/7), int32(p.eobx[cmp][dpos]))
		c, err := decodeS(a, m)
		if err != nil {
			return err
		}
		p.zdstX[cmp][dpos] = byte(c)
	}
	for dpos := 0; dpos < bc; dpos++ {
		shiftS(m, int32((int(p.zdst[cmp][dpos])+3)/7), int32(p.eoby[cmp][dpos]))
		c, err := decodeS(a, m)
		if err != nil {
			return err
		}
		p.zdstY[cmp][dpos] = byte(c)
	}
	return nil
}

func aavrg(store []uint16, w, pos, py, px, rx int32) int32 {
	var ctx, wctx int32
	add := func(wi int, off int32) {
		weight := absCtxWLum0[wi]
		ctx += int32(store[pos+off]) * weight
		wctx += weight
	}
	if py >= 2 {
		add(0, -2*w)
		add(2, -w)
		if px >= 2 {
			add(1, -1-w)
			add(4, -2)
			add(5, -1)
		} else if px == 1 {
			add(1, -1-w)
			add(5, -1)
		}
		if rx >= 1 {
			add(3, 1-w)
		}
	} else if py == 1 {
		add(2, -w)
		if px >= 2 {
			add(1, -1-w)
			add(4, -2)
			add(5, -1)
		} else if px == 1 {
			add(1, -1-w)
			add(5, -1)
		}
		if rx >= 1 {
			add(3, 1-w)
		}
	} else if px >= 2 {
		add(4, -2)
		add(5, -1)
	} else if px == 1 {
		add(5, -1)
	}
	if wctx != 0 {
		return (ctx + wctx/2) / wctx
	}
	return 0
}

func (p *pjg) segTab(cmp int) []byte {
	return segmTables[int(p.segm[cmp])-1][:]
}

func modelCtx(segm, min int32) int32 {
	if segm > min {
		return segm
	}
	return min
}

func (p *pjg) decDC(a *arith, cmp int) error {
	if p.segm[cmp] == 0 || int(p.segm[cmp]) > len(segmTables) {
		return p.fail(2, "bad segment count")
	}
	seg := p.segTab(cmp)
	maxVal := p.maxV(cmp, 0)
	maxLen := bitlenPos(maxVal)
	modLen := newModelS(maxLen+1, modelCtx(int32(p.segm[cmp]), maxLen+1), 2)
	resCtx := int32(16)
	if p.segm[cmp] >= 16 {
		resCtx = int32(p.segm[cmp])
	}
	modRes := newModelB(resCtx, 2)
	modSgn := newModelB(1, 0)
	bc := int(p.cmp[cmp].bc)
	w := int32(p.cmp[cmp].bch)
	absStore := make([]uint16, bc)
	coeffs := p.coll[cmp][0]
	z := p.zdst[cmp]
	for dpos := 0; dpos < bc; dpos++ {
		py := int32(dpos) / w
		px := int32(dpos) % w
		rx := w - (px + 1)
		snum := int32(seg[z[dpos]])
		ctxAvr := aavrg(absStore, w, int32(dpos), py, px, rx)
		ctxLen := bitlenPos(ctxAvr)
		shiftS(modLen, ctxLen, snum)
		clen, err := decodeS(a, modLen)
		if err != nil {
			return err
		}
		if clen == 0 {
			continue
		}
		absv := int32(1)
		for bp := clen - 2; bp >= 0; bp-- {
			shiftB(modRes, snum, bp)
			bt, err := decodeB(a, modRes)
			if err != nil {
				return err
			}
			absv <<= 1
			if bt != 0 {
				absv |= 1
			}
		}
		sgn, err := decodeB(a, modSgn)
		if err != nil {
			return err
		}
		if sgn == 0 {
			coeffs[dpos] = int16(absv)
		} else {
			coeffs[dpos] = int16(-absv)
		}
		absStore[dpos] = uint16(absv)
	}
	return nil
}

func (p *pjg) decACHigh(a *arith, cmp int) error {
	if p.segm[cmp] == 0 || int(p.segm[cmp]) > len(segmTables) {
		return p.fail(2, "bad segment count")
	}
	seg := p.segTab(cmp)
	modLen := newModelS(11, modelCtx(int32(p.segm[cmp]), 11), 2)
	resCtx := int32(16)
	if p.segm[cmp] >= 16 {
		resCtx = int32(p.segm[cmp])
	}
	modRes := newModelB(resCtx, 2)
	modSgn := newModelB(9, 1)
	bc := int(p.cmp[cmp].bc)
	w := int32(p.cmp[cmp].bch)
	absStore := make([]uint16, bc)
	sgnStore := make([]byte, bc)
	zdst := make([]byte, bc)
	copy(zdst, p.zdst[cmp])
	for i := 1; i < 64; i++ {
		bpos := int(p.zsrt[cmp][i])
		bx := int32(unzigzag[bpos] % 8)
		by := int32(unzigzag[bpos] / 8)
		if bx == 0 || by == 0 {
			continue
		}
		for k := range absStore {
			absStore[k] = 0
			sgnStore[k] = 0
		}
		coeffs := p.coll[cmp][bpos]
		maxLen := bitlenPos(p.maxV(cmp, bpos))
		for dpos := 0; dpos < bc; dpos++ {
			if zdst[dpos] == 0 {
				continue
			}
			py := int32(dpos) / w
			px := int32(dpos) % w
			rx := w - (px + 1)
			snum := int32(seg[zdst[dpos]])
			ctxAvr := aavrg(absStore, w, int32(dpos), py, px, rx)
			ctxLen := bitlenPos(ctxAvr)
			shiftS(modLen, ctxLen, snum)
			modLen.excludeAbove(maxLen)
			clen, err := decodeS(a, modLen)
			if err != nil {
				return err
			}
			if clen == 0 {
				continue
			}
			absv := int32(1)
			for bp := clen - 2; bp >= 0; bp-- {
				shiftB(modRes, snum, bp)
				bt, err := decodeB(a, modRes)
				if err != nil {
					return err
				}
				absv <<= 1
				if bt != 0 {
					absv |= 1
				}
			}
			ctxSgn := int32(0)
			if px > 0 {
				ctxSgn = int32(sgnStore[dpos-1])
			}
			if py > 0 {
				ctxSgn += 3 * int32(sgnStore[dpos-int(w)])
			}
			modSgn.shift(ctxSgn)
			sgn, err := decodeB(a, modSgn)
			if err != nil {
				return err
			}
			if sgn == 0 {
				coeffs[dpos] = int16(absv)
			} else {
				coeffs[dpos] = int16(-absv)
			}
			absStore[dpos] = uint16(absv)
			sgnStore[dpos] = byte(sgn + 1)
			zdst[dpos]--
			if bx > int32(p.eobx[cmp][dpos]) {
				p.eobx[cmp][dpos] = byte(bx)
			}
			if by > int32(p.eoby[cmp][dpos]) {
				p.eoby[cmp][dpos] = byte(by)
			}
		}
		modLen.flush(1)
		modRes.flush(1)
		modSgn.flush(1)
	}
	return nil
}

func lakh(rows [][]int16, pred *[8]int32, pos, off int) int32 {
	var pv int32
	for k := 1; k <= 7; k++ {
		ck := int32(rows[k][pos])
		ak := int32(rows[k][pos+off])
		if k%2 == 1 {
			pv -= (ck + ak) * pred[k]
		} else {
			pv -= (ck - ak) * pred[k]
		}
	}
	half := pred[0] / 2
	if pv > 0 {
		pv = (pv + half) / pred[0]
	} else {
		pv = (pv - half) / pred[0]
	}
	pv += int32(rows[0][pos+off])
	return pv
}

func (p *pjg) decACLow(a *arith, cmp int) error {
	modLen := newModelS(11, modelCtx(int32(p.segm[cmp]), 11), 2)
	modRes := newModelB(1<<4, 2)
	topC := int32(1 << 4)
	if p.nois[cmp] > 4 {
		if p.nois[cmp] >= 31 {
			return p.fail(2, "noise threshold out of range")
		}
		topC = int32(1) << p.nois[cmp]
	}
	modTop := newModelB(topC, 3)
	modSgn := newModelB(11, 1)
	bc := int(p.cmp[cmp].bc)
	w := int(p.cmp[cmp].bch)
	for i := 2; i < 16; i++ {
		bx := 0
		by := 0
		if i%2 == 0 {
			bx = i / 2
		} else {
			by = i / 2
		}
		bpos := int(zigzag[bx+8*by])
		var rows [8][]int16
		var pred [8]int32
		var zdst []byte
		off := 0
		if bx == 0 {
			for x := 0; x < 8; x++ {
				zp := int(zigzag[x+8*by])
				rows[x] = p.coll[cmp][zp]
				pred[x] = icosBase8x8[x*8] * int32(p.q(cmp, zp))
			}
			zdst = p.zdstY[cmp]
			off = -1
		} else {
			for y := 0; y < 8; y++ {
				zp := int(zigzag[bx+8*y])
				rows[y] = p.coll[cmp][zp]
				pred[y] = icosBase8x8[y*8] * int32(p.q(cmp, zp))
			}
			zdst = p.zdstX[cmp]
			off = -w
		}
		coeffs := p.coll[cmp][bpos]
		maxValP := p.maxV(cmp, bpos)
		maxValN := -maxValP
		maxLen := bitlenPos(maxValP)
		thrs := int32(0)
		if maxLen > int32(p.nois[cmp]) {
			thrs = maxLen - int32(p.nois[cmp])
		}
		for dpos := 0; dpos < bc; dpos++ {
			if zdst[dpos] == 0 {
				continue
			}
			py := dpos / w
			px := dpos % w
			edge := py
			if bx == 0 {
				edge = px
			}
			ctxLak := int32(0)
			if edge > 0 {
				if pred[0] == 0 {
					return p.fail(2, "quantization is zero")
				}
				ctxLak = lakh(rows[:], &pred, dpos, off)
			}
			ctxLak = clamped(maxValN, maxValP, ctxLak)
			ctxLen := bitlen2048(ctxLak)
			shiftS(modLen, ctxLen, int32(zdst[dpos]))
			modLen.excludeAbove(maxLen)
			clen, err := decodeS(a, modLen)
			if err != nil {
				return err
			}
			if clen == 0 {
				continue
			}
			bp := clen - 2
			ctxRes := int32(0)
			if bp >= thrs {
				ctxRes = 1
			}
			ctxAbs := ctxLak
			if ctxAbs < 0 {
				ctxAbs = -ctxAbs
			}
			ctxSgn := int32(0)
			if ctxLak > 0 {
				ctxSgn = 1
			} else if ctxLak < 0 {
				ctxSgn = 2
			}
			for ; bp >= thrs; bp-- {
				shiftB(modTop, ctxAbs>>uint(thrs), ctxRes, clen-thrs)
				bt, err := decodeB(a, modTop)
				if err != nil {
					return err
				}
				ctxRes <<= 1
				if bt != 0 {
					ctxRes |= 1
				}
			}
			absv := ctxRes
			if ctxRes == 0 {
				absv = 1
			}
			for ; bp >= 0; bp-- {
				shiftB(modRes, int32(zdst[dpos]), bp)
				bt, err := decodeB(a, modRes)
				if err != nil {
					return err
				}
				absv <<= 1
				if bt != 0 {
					absv |= 1
				}
			}
			shiftB(modSgn, int32(zdst[dpos]), ctxSgn)
			sgn, err := decodeB(a, modSgn)
			if err != nil {
				return err
			}
			if sgn == 0 {
				coeffs[dpos] = int16(absv)
			} else {
				coeffs[dpos] = int16(-absv)
			}
			zdst[dpos]--
		}
		modLen.flush(1)
		modRes.flush(1)
		modTop.flush(1)
		modSgn.flush(1)
	}
	return nil
}

func (p *pjg) adapt() {
	for cmp := int32(0); cmp < p.cmpc; cmp++ {
		var quant [64]uint16
		for ipos := 0; ipos < 64; ipos++ {
			quant[ipos] = p.q(int(cmp), int(zigzag[ipos]))
			if quant[ipos] >= 2048 {
				quant[ipos] = 0
			}
		}
		for ipos := 0; ipos < 64; ipos++ {
			p.adpt1x8[cmp][ipos] = icosIDCT1x8[ipos] * int32(quant[(ipos%8)*8])
			p.adpt8x1[cmp][ipos] = icosIDCT1x8[ipos] * int32(quant[ipos%8])
		}
	}
}

func (p *pjg) idct8x1(cmp, dpos, ix int) int32 {
	ixy := ix << 3
	tab := p.adpt8x1[cmp]
	c := p.coll[cmp]
	var id int32
	id += int32(c[0][dpos]) * tab[ixy+0]
	id += int32(c[1][dpos]) * tab[ixy+1]
	id += int32(c[5][dpos]) * tab[ixy+2]
	id += int32(c[6][dpos]) * tab[ixy+3]
	id += int32(c[14][dpos]) * tab[ixy+4]
	id += int32(c[15][dpos]) * tab[ixy+5]
	id += int32(c[27][dpos]) * tab[ixy+6]
	id += int32(c[28][dpos]) * tab[ixy+7]
	return id
}

func (p *pjg) idct1x8(cmp, dpos, iy int) int32 {
	ixy := iy << 3
	tab := p.adpt1x8[cmp]
	c := p.coll[cmp]
	var id int32
	id += int32(c[0][dpos]) * tab[ixy+0]
	id += int32(c[2][dpos]) * tab[ixy+1]
	id += int32(c[3][dpos]) * tab[ixy+2]
	id += int32(c[9][dpos]) * tab[ixy+3]
	id += int32(c[10][dpos]) * tab[ixy+4]
	id += int32(c[20][dpos]) * tab[ixy+5]
	id += int32(c[21][dpos]) * tab[ixy+6]
	id += int32(c[35][dpos]) * tab[ixy+7]
	return id
}

func (p *pjg) dcPred(cmp, dpos int) int32 {
	w := int(p.cmp[cmp].bch)
	px := dpos % w
	py := dpos / w
	swap := p.coll[cmp][0][dpos]
	p.coll[cmp][0][dpos] = 0
	var pred int32
	if px > 0 && py > 0 {
		pa := p.idct8x1(cmp, dpos-1, 7)
		pb := p.idct1x8(cmp, dpos-w, 7)
		xa := p.idct8x1(cmp, dpos, 0)
		xb := p.idct1x8(cmp, dpos, 0)
		pred = ((pa - xa) + (pb - xb)) * (8 / 2)
	} else if px > 0 {
		pa := p.idct8x1(cmp, dpos-1, 7)
		xa := p.idct8x1(cmp, dpos, 0)
		pred = (pa - xa) * 8
	} else if py > 0 {
		pb := p.idct1x8(cmp, dpos-w, 7)
		xb := p.idct1x8(cmp, dpos, 0)
		pred = (pb - xb) * 8
	}
	p.coll[cmp][0][dpos] = swap
	pred = clamped(-(1024 * 8192), 1016*8192, pred)
	qv := int32(p.q(cmp, 0))
	pred = pred / qv
	return dctRescale(pred)
}

func (p *pjg) unpredict() {
	for cmp := 0; cmp < int(p.cmpc); cmp++ {
		absMax := p.maxV(cmp, 0)
		absMin := -absMax
		corr := 2*absMax + 1
		coefs := p.coll[cmp][0]
		for dpos := 1; dpos < int(p.cmp[cmp].bc); dpos++ {
			sum := int32(coefs[dpos]) + p.dcPred(cmp, dpos)
			v := int16(sum)
			if int32(v) > absMax {
				v = int16(int32(v) - corr)
			} else if int32(v) < absMin {
				v = int16(int32(v) + corr)
			}
			coefs[dpos] = v
		}
	}
}

func (w *bitWriter) putCode(t *huffTab, sym int32) {
	if sym < 0 || sym >= 256 {
		return
	}
	w.write(uint32(t.cval[sym]), int32(t.clen[sym]))
}

func (w *bitWriter) encodeEOB(ac *huffTab, eobrun *int32) {
	if *eobrun <= 0 {
		return
	}
	for *eobrun > ac.maxEob {
		w.putCode(ac, 0xE0)
		w.write(eEnvli(14, 32767), 14)
		*eobrun -= ac.maxEob
	}
	s := bitlenPos(*eobrun)
	s--
	n := eEnvli(s, *eobrun)
	w.putCode(ac, s<<4)
	w.write(n, s)
	*eobrun = 0
}

func (w *bitWriter) encodeSeq(dc, ac *huffTab, block *[64]int16) {
	s := bitlen2048(int32(block[0]))
	w.putCode(dc, s)
	w.write(envli(s, int32(block[0])), s)
	z := int32(0)
	for bpos := 1; bpos < 64; bpos++ {
		if block[bpos] != 0 {
			for z >= 16 {
				w.putCode(ac, 0xF0)
				z -= 16
			}
			s = bitlen2048(int32(block[bpos]))
			hc := (z << 4) + s
			w.putCode(ac, hc)
			w.write(envli(s, int32(block[bpos])), s)
			z = 0
		} else {
			z++
		}
	}
	if z > 0 {
		w.putCode(ac, 0)
	}
}

func (w *bitWriter) encodeDCFS(dc *huffTab, block *[64]int16) {
	s := bitlen2048(int32(block[0]))
	w.putCode(dc, s)
	w.write(envli(s, int32(block[0])), s)
}

func (w *bitWriter) encodeACFS(ac *huffTab, block *[64]int16, eobrun *int32, from, to int32) {
	z := int32(0)
	for bpos := from; bpos <= to; bpos++ {
		if block[bpos] != 0 {
			w.encodeEOB(ac, eobrun)
			for z >= 16 {
				w.putCode(ac, 0xF0)
				z -= 16
			}
			s := bitlen2048(int32(block[bpos]))
			hc := (z << 4) + s
			w.putCode(ac, hc)
			w.write(envli(s, int32(block[bpos])), s)
			z = 0
		} else {
			z++
		}
	}
	if z > 0 {
		*eobrun++
		if *eobrun == ac.maxEob {
			w.encodeEOB(ac, eobrun)
		}
	}
}

func (w *bitWriter) crbits(stor *byteBuf) {
	for _, b := range stor.peek() {
		w.write(uint32(b), 1)
	}
	stor.reset()
}

func (w *bitWriter) encodeACSA(ac *huffTab, stor *byteBuf, block *[64]int16, eobrun *int32, from, to int32) {
	eob := from
	for bpos := to; bpos >= from; bpos-- {
		if block[bpos] == 1 || block[bpos] == -1 {
			eob = bpos + 1
			break
		}
	}
	if eob > from && *eobrun > 0 {
		w.encodeEOB(ac, eobrun)
		w.crbits(stor)
	}
	z := int32(0)
	bpos := from
	for ; bpos < eob; bpos++ {
		if block[bpos] == 0 {
			z++
			if z == 16 {
				w.putCode(ac, 0xF0)
				w.crbits(stor)
				z = 0
			}
		} else if block[bpos] == 1 || block[bpos] == -1 {
			s := bitlen2048(int32(block[bpos]))
			hc := (z << 4) + s
			w.putCode(ac, hc)
			w.write(envli(s, int32(block[bpos])), s)
			w.crbits(stor)
			z = 0
		} else {
			stor.write(byte(block[bpos] & 1))
		}
	}
	for ; bpos <= to; bpos++ {
		if block[bpos] != 0 {
			stor.write(byte(block[bpos] & 1))
		}
	}
	if eob <= to {
		*eobrun++
		if *eobrun == ac.maxEob {
			w.encodeEOB(ac, eobrun)
			w.crbits(stor)
		}
	}
}

func (p *pjg) nextMCU(s *mcuSt) int32 {
	sta := int32(0)
	s.sub++
	if s.sub >= p.cmp[s.cmp].mbs {
		s.sub = 0
		s.csc++
		if s.csc >= p.csCmpc {
			s.csc = 0
			s.cmp = p.csCmp[0]
			s.mcu++
			if s.mcu >= p.mcuc {
				sta = 2
			} else if p.rsti > 0 {
				s.rstw--
				if s.rstw == 0 {
					sta = 1
				}
			}
		} else {
			s.cmp = p.csCmp[s.csc]
		}
	}
	ci := &p.cmp[s.cmp]
	if ci.sfh > 1 {
		d := (s.mcu/p.mcuh)*ci.sfh + s.sub/ci.sfv
		d *= ci.bch
		d += (s.mcu%p.mcuh)*ci.sfv + s.sub%ci.sfv
		s.dpos = d
	} else if ci.sfv > 1 {
		s.dpos = s.mcu*ci.mbs + s.sub
	} else {
		s.dpos = s.mcu
	}
	return sta
}

func (p *pjg) nextMCUn(cmp, dpos, rstw *int32) int32 {
	*dpos++
	ci := &p.cmp[*cmp]
	if ci.bch != ci.nch {
		if *dpos%ci.bch == ci.nch {
			*dpos += ci.bch - ci.nch
		}
	}
	if ci.bcv != ci.ncv {
		if *dpos/ci.bch == ci.ncv {
			*dpos = ci.bc
		}
	}
	if *dpos >= ci.bc {
		return 2
	} else if p.rsti > 0 {
		*rstw--
		if *rstw == 0 {
			return 1
		}
	}
	return 0
}

func (p *pjg) growScnp() {
	for len(p.scnp) < int(p.scnc)+2 {
		p.scnp = append(p.scnp, 0)
	}
}

func (p *pjg) growRstp(n int) {
	for len(p.rstp) < n {
		p.rstp = append(p.rstp, 0)
	}
}

func (p *pjg) recode() error {
	w := newBitWriter()
	w.fill = byte(p.padbit)
	stor := &byteBuf{data: make([]byte, 64)}
	hpos := 0
	p.scnc = 0
	p.rstc = 0
	for {
		typ := byte(0)
		for typ != 0xDA {
			if hpos >= len(p.hdr) {
				break
			}
			if hpos+4 > len(p.hdr) {
				return p.fail(2, "truncated jpeg header")
			}
			typ = p.hdr[hpos+1]
			ln := 2 + bshort(p.hdr[hpos+2], p.hdr[hpos+3])
			if ln < 4 || hpos+ln > len(p.hdr) {
				return p.fail(2, "truncated jpeg header")
			}
			if typ == 0xC4 || typ == 0xDA || typ == 0xDD {
				if err := p.parse(p.hdr[hpos : hpos+ln]); err != nil {
					return err
				}
				if p.errLevel >= errTol() {
					return errors.New("jpgrec: " + p.errMsg)
				}
			}
			hpos += ln
		}
		if typ != 0xDA {
			break
		}
		p.growScnp()
		if p.rsti > 0 {
			var extra int32
			if p.csCmpc > 1 {
				extra = p.mcuc / p.rsti
			} else {
				extra = p.cmp[p.csCmp[0]].bc / p.rsti
			}
			p.growRstp(int(p.rstc+extra) + 1)
		}
		s := mcuSt{cmp: p.csCmp[0]}
		p.scnp[p.scnc] = uint32(w.pos())
		for {
			var lastdc [4]int32
			sta := int32(0)
			eobrun := int32(0)
			s.rstw = p.rsti
			var block [64]int16
			if p.csCmpc > 1 {
				if p.jpegType == 1 {
					for sta == 0 {
						cmp := int(s.cmp)
						for bpos := 0; bpos < 64; bpos++ {
							block[bpos] = p.coll[cmp][bpos][s.dpos]
						}
						block[0] = int16(int32(block[0]) - lastdc[cmp])
						lastdc[cmp] = int32(p.coll[cmp][0][s.dpos])
						// Sequential encode uses the AC table id for both DC and AC codes.
						dc := &p.hcodes[0][p.cmp[cmp].huffAC]
						ac := &p.hcodes[1][p.cmp[cmp].huffAC]
						w.encodeSeq(dc, ac, &block)
						sta = p.nextMCU(&s)
					}
				} else if p.csSah == 0 {
					for sta == 0 {
						cmp := int(s.cmp)
						tmp := int32(p.coll[cmp][0][s.dpos]) >> uint(p.csSal)
						block[0] = int16(tmp - lastdc[cmp])
						lastdc[cmp] = tmp
						w.encodeDCFS(&p.hcodes[0][p.cmp[cmp].huffDC], &block)
						sta = p.nextMCU(&s)
					}
				} else {
					for sta == 0 {
						cmp := int(s.cmp)
						block[0] = int16(bitn(int32(p.coll[cmp][0][s.dpos]), p.csSal))
						w.write(uint32(block[0]), 1)
						sta = p.nextMCU(&s)
					}
				}
			} else if p.jpegType == 1 {
				for sta == 0 {
					cmp := int(s.cmp)
					for bpos := 0; bpos < 64; bpos++ {
						block[bpos] = p.coll[cmp][bpos][s.dpos]
					}
					block[0] = int16(int32(block[0]) - lastdc[cmp])
					lastdc[cmp] = int32(p.coll[cmp][0][s.dpos])
					dc := &p.hcodes[0][p.cmp[cmp].huffAC]
					ac := &p.hcodes[1][p.cmp[cmp].huffAC]
					w.encodeSeq(dc, ac, &block)
					sta = p.nextMCUn(&s.cmp, &s.dpos, &s.rstw)
				}
			} else if p.csTo == 0 {
				if p.csSah == 0 {
					for sta == 0 {
						cmp := int(s.cmp)
						tmp := int32(p.coll[cmp][0][s.dpos]) >> uint(p.csSal)
						block[0] = int16(tmp - lastdc[cmp])
						lastdc[cmp] = tmp
						w.encodeDCFS(&p.hcodes[0][p.cmp[cmp].huffDC], &block)
						sta = p.nextMCUn(&s.cmp, &s.dpos, &s.rstw)
					}
				} else {
					for sta == 0 {
						cmp := int(s.cmp)
						block[0] = int16(bitn(int32(p.coll[cmp][0][s.dpos]), p.csSal))
						w.write(uint32(block[0]), 1)
						sta = p.nextMCUn(&s.cmp, &s.dpos, &s.rstw)
					}
				}
			} else if p.csSah == 0 {
				for sta == 0 {
					cmp := int(s.cmp)
					for bpos := p.csFrom; bpos <= p.csTo; bpos++ {
						block[bpos] = int16(fdiv2(int32(p.coll[cmp][bpos][s.dpos]), p.csSal))
					}
					ac := &p.hcodes[1][p.cmp[cmp].huffAC]
					w.encodeACFS(ac, &block, &eobrun, p.csFrom, p.csTo)
					sta = p.nextMCUn(&s.cmp, &s.dpos, &s.rstw)
				}
				cmp := int(s.cmp)
				w.encodeEOB(&p.hcodes[1][p.cmp[cmp].huffAC], &eobrun)
			} else {
				for sta == 0 {
					cmp := int(s.cmp)
					for bpos := p.csFrom; bpos <= p.csTo; bpos++ {
						block[bpos] = int16(fdiv2(int32(p.coll[cmp][bpos][s.dpos]), p.csSal))
					}
					ac := &p.hcodes[1][p.cmp[cmp].huffAC]
					w.encodeACSA(ac, stor, &block, &eobrun, p.csFrom, p.csTo)
					sta = p.nextMCUn(&s.cmp, &s.dpos, &s.rstw)
				}
				cmp := int(s.cmp)
				ac := &p.hcodes[1][p.cmp[cmp].huffAC]
				w.encodeEOB(ac, &eobrun)
				w.crbits(stor)
			}
			w.pad(byte(p.padbit))
			if sta == -1 {
				return p.fail(2, "encode error in scan%d", p.scnc)
			} else if sta == 2 {
				p.scnc++
				break
			} else if sta == 1 && p.rsti > 0 {
				p.growRstp(int(p.rstc) + 1)
				p.rstp[p.rstc] = uint32(w.pos() - 1)
				p.rstc++
			}
		}
	}
	p.huff = w.bytes()
	p.growScnp()
	p.scnp[p.scnc] = uint32(len(p.huff))
	if p.rstp != nil {
		p.growRstp(int(p.rstc) + 1)
		p.rstp[p.rstc] = uint32(len(p.huff))
	}
	return nil
}

func errTol() int { return 1 }

func (p *pjg) merge() ([]byte, error) {
	o := []byte{0xFF, 0xD8}
	hpos := 0
	scan := 1
	rpos := 0
	for {
		tmp := hpos
		typ := byte(0)
		for typ != 0xDA {
			if hpos >= len(p.hdr) {
				break
			}
			if hpos+4 > len(p.hdr) {
				return nil, p.fail(2, "truncated jpeg header")
			}
			typ = p.hdr[hpos+1]
			ln := 2 + bshort(p.hdr[hpos+2], p.hdr[hpos+3])
			if ln < 2 || hpos+ln > len(p.hdr) {
				return nil, p.fail(2, "truncated jpeg header")
			}
			hpos += ln
		}
		o = append(o, p.hdr[tmp:hpos]...)
		if typ != 0xDA {
			break
		}
		if scan >= len(p.scnp) || scan-1 < 0 {
			return nil, p.fail(2, "missing scan position")
		}
		from := int(p.scnp[scan-1])
		to := int(p.scnp[scan])
		if from < 0 || to > len(p.huff) || from > to {
			return nil, p.fail(2, "bad scan range")
		}
		cpos := 0
		for ipos := from; ipos < to; ipos++ {
			o = append(o, p.huff[ipos])
			if p.huff[ipos] == 0xFF {
				o = append(o, 0x00)
			}
			if p.rstp != nil && rpos < len(p.rstp) && uint32(ipos) == p.rstp[rpos] {
				o = append(o, 0xFF, byte(0xD0+(cpos%8)))
				rpos++
				cpos++
			}
		}
		if p.rstErr != nil && scan-1 < len(p.rstErr) {
			for p.rstErr[scan-1] > 0 {
				o = append(o, 0xFF, byte(0xD0+(cpos%8)))
				cpos++
				p.rstErr[scan-1]--
			}
		}
		scan++
	}
	o = append(o, 0xFF, 0xD9)
	if len(p.grbg) > 0 {
		o = append(o, p.grbg...)
	}
	return o, nil
}

func restorePJG(payload []byte) ([]byte, error) {
	if len(payload) < 2 || payload[0] != 'J' || payload[1] != 'S' {
		return nil, errors.New("jpgrec: packJPG magic is not JS")
	}
	p := newPJG()
	r := &memReader{b: payload[2:]}
	if err := p.unpack(r); err != nil {
		return nil, err
	}
	if p.errLevel >= errTol() {
		return nil, errors.New("jpgrec: " + p.errMsg)
	}
	p.adapt()
	p.unpredict()
	if err := p.recode(); err != nil {
		return nil, err
	}
	out, err := p.merge()
	if err != nil {
		return nil, err
	}
	if len(out) > jpgMaxMemory {
		return nil, fmt.Errorf("jpgrec: output length %d exceeds JPG_MAX_MEMORY_SIZE", len(out))
	}
	return out, nil
}
