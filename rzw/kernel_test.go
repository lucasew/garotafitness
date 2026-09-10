package rzw

import (
	"hash/crc32"
	"os"
	"testing"
)

func TestFG05TgtSweep(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(rimworld + "/fg-05.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	packed := raw[0x1F+4+17 : 0x1F+4+17+230542]
	for s := 0; s < 16; s++ {
		d := newDec(packed, 1<<20)
		d.noChunk = true
		for i := range d.nibLit {
			d.nibLit[i].initTgt(s)
		}
		if !d.decodeByte() {
			continue
		}
		for i := 0; i < 8 && d.r.ok(); i++ {
			if !d.literal() {
				break
			}
		}
		if len(d.out) < 16 {
			t.Logf("s=%d short %d", s, len(d.out))
			continue
		}
		ch := uint32(d.out[8]) | uint32(d.out[9])<<8 | uint32(d.out[10])<<16 | uint32(d.out[11])<<24
		tag := uint32(d.out[12]) | uint32(d.out[13])<<8 | uint32(d.out[14])<<16 | uint32(d.out[15])<<24
		looks := tag == 0xC71B3AE1 || tag == 0xC71B3AE2 || tag == 0x26351817 || ch == 0x4000
		t.Logf("tgt%d dest[8:16]=%x ch=%08x tag=%08x looks=%v", s, d.out[8:16], ch, tag, looks)
	}
}

func TestFG05FirstLits(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(rimworld + "/fg-05.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	packed := raw[0x1F+4+17 : 0x1F+4+17+230542]
	d := newDec(packed, 1<<20)
	if !d.decodeByte() || (d.pos != 8 && d.pos != 12) {
		t.Fatalf("t15: pos=%d why=%s ev=%v", d.pos, d.why, d.ev)
	}
	t.Logf("afterT15 s0=%08x s1=%08x off=%d a8=%d ca=%d cb=%d dest=%x",
		d.r.s0, d.r.s1, d.r.off, d.a8, classA[d.a8&255], classB[d.prev()], d.out[:8])
	for i := 0; i < 8; i++ {
		ps, qs := d.prevSlot(), d.posSlot()
		m := d.binAt(ps[0], qs[0])
		freq := uint32(*m) >> 4
		// peek slot after the swap the bit() will do
		s0, s1, off, end := d.r.s0, d.r.s1, d.r.off, d.r.end
		d.r.swap()
		slot := d.r.s0 & binMask
		d.r.s0, d.r.s1, d.r.off, d.r.end = s0, s1, off, end
		pos0 := d.pos
		if !d.decodeByte() {
			t.Logf("i=%d decode fail why=%s ev=%v", i, d.why, d.ev)
			break
		}
		t.Logf("i=%d bit slot=%03x freq=%03x lit=%v a8=%d ca=%d cb=%d wrote=%x s0=%08x s1=%08x off=%d ev=%v",
			i, slot, freq, d.pos == pos0+1, d.a8, classA[d.a8&255], classB[d.prev()],
			d.out[pos0:d.pos], d.r.s0, d.r.s1, d.r.off, d.ev)
	}
	if len(d.out) >= 16 {
		ch := uint32(d.out[8]) | uint32(d.out[9])<<8 | uint32(d.out[10])<<16 | uint32(d.out[11])<<24
		tag := uint32(d.out[12]) | uint32(d.out[13])<<8 | uint32(d.out[14])<<16 | uint32(d.out[15])<<24
		t.Logf("dest[8:16]=%x ch=%08x legal=%v tag=%08x", d.out[8:16], ch, ch >= 1 && ch <= 8<<20, tag)
	}

	// PE: first 4 dest data bytes from classA/classB literals, even if bit4 is match.
	d2 := newDec(packed, 1<<20)
	if !d2.decodeByte() {
		t.Fatal("t15")
	}
	for i := 0; i < 4; i++ {
		if !d2.literal() {
			t.Logf("forced lit %d fail", i)
			break
		}
	}
	if len(d2.out) >= 16 {
		ch := uint32(d2.out[8]) | uint32(d2.out[9])<<8 | uint32(d2.out[10])<<16 | uint32(d2.out[11])<<24
		tag := uint32(d2.out[12]) | uint32(d2.out[13])<<8 | uint32(d2.out[14])<<16 | uint32(d2.out[15])<<24
		t.Logf("force4lit dest[8:16]=%x ch=%08x legal=%v tag=%08x", d2.out[8:16], ch, ch >= 1 && ch <= 8<<20, tag)
	}

	// consume each binary bit, then always emit a nibble literal (ignore match).
	d3 := newDec(packed, 1<<20)
	if !d3.decodeByte() {
		t.Fatal("t15")
	}
	for i := 0; i < 4; i++ {
		ps, qs := d3.prevSlot(), d3.posSlot()
		m := d3.binAt(ps[0], qs[0])
		lit := m.bit(&d3.r)
		ps[0] = saNext(ps[0], lit)
		qs[0] = saNext(qs[0], lit)
		if !d3.literal() {
			t.Logf("bit+lit %d fail litbit=%d", i, lit)
			break
		}
		d3.stepA8(kindLit)
		t.Logf("bit+lit i=%d bit=%d b=%02x a8=%d", i, lit, d3.out[8+i], d3.a8)
	}
	if len(d3.out) >= 16 {
		ch := uint32(d3.out[8]) | uint32(d3.out[9])<<8 | uint32(d3.out[10])<<16 | uint32(d3.out[11])<<24
		tag := uint32(d3.out[12]) | uint32(d3.out[13])<<8 | uint32(d3.out[14])<<16 | uint32(d3.out[15])<<24
		t.Logf("bit+4lit dest[8:16]=%x ch=%08x legal=%v tag=%08x", d3.out[8:16], ch, ch >= 1 && ch <= 8<<20, tag)
	}

	type vcfg struct {
		name   string
		zero   bool
		noswap bool
		pred   int // 0=a8, 1=prev, 2=force predicted
		rev    bool
		noinc  bool
		n      int
	}
	cfgs := []vcfg{
		{"base", false, false, 0, false, false, 8},
		{"zero", true, false, 0, false, false, 8},
		{"nsw", false, true, 0, false, false, 8},
		{"prev", false, false, 1, false, false, 8},
		{"pred", false, false, 2, false, false, 8},
		{"rev", false, false, 0, true, false, 8},
		{"noinc", false, false, 0, false, true, 8},
		{"zero+nsw", true, true, 0, false, false, 8},
		{"nsw+pred", false, true, 2, false, false, 8},
		{"zero+pred", true, false, 2, false, false, 8},
		{"prev+rev", false, false, 1, true, false, 8},
	}
	for _, c := range cfgs {
		d := newDec(packed, 1<<20)
		if !d.decodeByte() {
			continue
		}
		if c.zero {
			for i := range d.nibLit {
				for j := range d.nibLit[i] {
					d.nibLit[i][j] = 0
				}
			}
		}
		for i := 0; i < c.n; i++ {
			ps, qs := d.prevSlot(), d.posSlot()
			m := d.binAt(ps[0], qs[0])
			lit := m.bit(&d.r)
			ps[0] = saNext(ps[0], lit)
			qs[0] = saNext(qs[0], lit)
			prev := d.prev()
			cls := int(classB[prev])
			if cls > 12 {
				cls = 12
			}
			ca := int(classA[d.a8&255])
			if c.pred == 1 {
				ca = int(classA[prev])
			}
			if c.pred == 2 {
				ca = 1
			}
			var b byte
			if ca == 0 && c.pred != 2 {
				var hi, lo int
				if c.noswap {
					hi = nibPeek(&d.nibLit[cls*81], &d.r, c.noinc)
					lo = nibPeek(&d.nibLit[cls*81+1+hi], &d.r, c.noinc)
				} else {
					hi = d.nibLit[cls*81].sym(&d.r)
					lo = d.nibLit[cls*81+1+hi].sym(&d.r)
				}
				if c.rev {
					b = byte(lo<<4 | hi)
				} else {
					b = byte(hi<<4 | lo)
				}
			} else {
				if !d.literal() {
					break
				}
				d.stepA8(kindLit)
				continue
			}
			d.emit(b)
			d.stepA8(kindLit)
		}
		if len(d.out) < 16 {
			t.Logf("var %s short %d", c.name, len(d.out))
			continue
		}
		ch := uint32(d.out[8]) | uint32(d.out[9])<<8 | uint32(d.out[10])<<16 | uint32(d.out[11])<<24
		tag := uint32(d.out[12]) | uint32(d.out[13])<<8 | uint32(d.out[14])<<16 | uint32(d.out[15])<<24
		legal := ch >= 1 && ch <= 8<<20
		looks := tag == 0xC71B3AE1 || tag == 0xC71B3AE2 || tag == 0x26351817
		t.Logf("var %s dest[8:16]=%x ch=%08x legal=%v tag=%08x looks=%v", c.name, d.out[8:16], ch, legal, tag, looks)
	}
}

func nibPeek(m *nibModel, r *rans, noinc bool) int {
	if !r.ok() {
		return 0
	}
	slot := int(r.s0 & nibMask)
	if !noinc {
		slot++
	}
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

func TestFG05AfterHeader(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(rimworld + "/fg-05.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	packed := raw[0x1F+4+17 : 0x1F+4+17+230542]
	t.Logf("packed[0:16]=%x", packed[:16])

	type trial struct {
		name string
		fn   func(*dec)
	}
	trials := []trial{
		{"kernel", func(*dec) {}},
		{"d8", func(d *dec) { d.tokDeltaU8(1) }},
		{"d32", func(d *dec) { d.tokDeltaU8(4) }},
		{"a8one", func(d *dec) { d.a8 = 1 }},
		{"predlit", func(d *dec) {
			for i := 0; i < 8; i++ {
				d.a8 = 1
				if !d.literal() {
					break
				}
				d.stepA8(kindLit)
			}
		}},
		{"nostep", func(d *dec) { d.a8 = 0; d.lastKind = 0 }},
		{"dumpbin", func(d *dec) {
			t.Logf("after t15 pos=%d a8=%d s0=%08x s1=%08x end=%d dest=%x",
				d.pos, d.a8, d.r.s0, d.r.s1, d.r.end, head(d.out, 16))
		}},
		{"swap", func(d *dec) { d.r.swap() }},
		{"renorm", func(d *dec) { d.renormS0() }},
		{"swapren", func(d *dec) { d.r.swap(); d.renormS0() }},
		{"imgu32", func(d *dec) {
			v := uint32(d.imgU(&d.imgWM))
			d.emit(byte(v))
			d.emit(byte(v >> 8))
			d.emit(byte(v >> 16))
			d.emit(byte(v >> 24))
		}},
		{"imgu16x2", func(d *dec) {
			for i := 0; i < 2; i++ {
				w := uint16(d.imgU(&d.imgWM))
				d.emit(byte(w))
				d.emit(byte(w >> 8))
			}
		}},
		{"res4", func(d *dec) {
			for i := 0; i < 4; i++ {
				d.emit(d.resByte(i))
			}
		}},
		{"toklen", func(d *dec) {
			v := uint32(d.tokLen())
			d.emit(byte(v))
			d.emit(byte(v >> 8))
			d.emit(byte(v >> 16))
			d.emit(byte(v >> 24))
		}},
		{"bits32", func(d *dec) {
			v := d.r.bits(32)
			d.emit(byte(v))
			d.emit(byte(v >> 8))
			d.emit(byte(v >> 16))
			d.emit(byte(v >> 24))
		}},
		{"nolitstep", func(d *dec) {
			for i := 0; i < 4 && d.r.ok(); i++ {
				if !d.literal() {
					break
				}
			}
		}},
		{"bin0", func(d *dec) {
			for i := range d.bin {
				d.bin[i] = 0
			}
		}},
		{"s1u16", func(d *dec) {
			w := d.rawSwapNoRefill()
			d.emit(byte(w))
			d.emit(byte(w >> 8))
		}},
		{"copy8", func(d *dec) {
			// PE-plausible: first match copies dest[0:4] as chunk=ds.
			for i := 0; i < 4; i++ {
				d.emit(d.out[i])
			}
		}},
	}
	for _, tr := range trials {
		d := newDec(packed, 1<<20)
		if !d.decodeByte() {
			t.Logf("%s: first tok failed %s", tr.name, d.why)
			continue
		}
		s0, s1, off := d.r.s0, d.r.s1, d.r.off
		tr.fn(d)
		for d.pos < 1<<20 {
			if d.lim > 0 && d.pos >= d.lim {
				break
			}
			if !d.decodeByte() {
				break
			}
		}
		if d.lim > 0 && len(d.out) > d.lim {
			d.out = d.out[:d.lim]
		}
		ch := uint32(0)
		if len(d.out) >= 12 {
			ch = uint32(d.out[8]) | uint32(d.out[9])<<8 | uint32(d.out[10])<<16 | uint32(d.out[11])<<24
		}
		crc := crc32.ChecksumIEEE(d.out)
		tag := uint32(0)
		if len(d.out) >= 16 {
			tag = uint32(d.out[12]) | uint32(d.out[13])<<8 | uint32(d.out[14])<<16 | uint32(d.out[15])<<24
		}
		t.Logf("%s afterT15 s0=%08x s1=%08x off=%d pos=%d len=%d crc=%08x dest8=%08x dest12=%08x srep=%v ok=%v head=%x ev=%v",
			tr.name, s0, s1, off, d.pos, len(d.out), crc, ch, tag, tag == 0x26351817, ch >= 1 && ch <= 8<<20, head(d.out, 20), d.ev)
	}
}

func TestFG05NativeProbe(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(rimworld + "/fg-05.bin")
	if err != nil {
		t.Skip("corpus not mounted")
	}
	// solid at 0x1F: u32 size + CM( header 17 + packed
	off := 0x1F + 4 + 17
	packed := raw[off : off+230542]
	want := uint32(0xe77aa130)
	d0 := newDec(packed, 1<<20)
	out, err := runDec(d0, 1<<20)
	if err != nil {
		t.Fatalf("decodeNative: %v", err)
	}
	got := crc32.ChecksumIEEE(out)
	t.Logf("len=%d crc=%08x want=%08x head=%x tail=%x in=%d/%d lit=%d mat=%d tok=%d imgW=%d why=%s ev=%v",
		len(out), got, want, head(out, 32), tail(out, 16), d0.r.off, len(packed), d0.nLit, d0.nMat, d0.nTok, d0.imgW, d0.why, d0.ev)
	if len(out) >= 8 {
		ds := uint32(out[0]) | uint32(out[1])<<8 | uint32(out[2])<<16 | uint32(out[3])<<24
		ts := uint32(out[4]) | uint32(out[5])<<8 | uint32(out[6])<<16 | uint32(out[7])<<24
		span := 8 + 3*int(ts) + int(ds)
		plausible := ds >= 1024 && ds <= 2<<20 && ts < 0x7fffffff && ts%4 == 0
		t.Logf("delta dataSize=%#x (%d) tableSize=%#x (%d) span=%d dest=%d plausible=%v valid=%v",
			ds, ds, ts, ts, span, len(out), plausible, ds < 0x7fffffff && ts < 0x7fffffff && ts%4 == 0)
	}
	if len(out) >= 12 {
		ch := uint32(out[8]) | uint32(out[9])<<8 | uint32(out[10])<<16 | uint32(out[11])<<24
		t.Logf("dest[8:12]=%08x (%d) dispack=%v dest[8:16]=%x",
			ch, ch, ch >= 1 && ch <= 8<<20, out[8:min(16, len(out))])
	}
	if len(out) >= 28 {
		t.Logf("dest[12:28]=%x", out[12:28])
	}
	if len(out) >= 16 {
		tag := uint32(out[12]) | uint32(out[13])<<8 | uint32(out[14])<<16 | uint32(out[15])<<24
		t.Logf("dest[12:16]=%08x srep=%v dispackTag=%v dest[12:32]=%x",
			tag, tag == 0x26351817, tag == 0xC71B3AE1 || tag == 0xC71B3AE2, head(out[12:], 20))
		hits := []string{}
		for i := 0; i+4 <= len(out); i++ {
			v := uint32(out[i]) | uint32(out[i+1])<<8 | uint32(out[i+2])<<16 | uint32(out[i+3])<<24
			switch v {
			case 0x26351817:
				hits = append(hits, "srep@"+itoa(i))
			case 0xC71B3AE1:
				hits = append(hits, "tagData@"+itoa(i))
			case 0xC71B3AE2:
				hits = append(hits, "tagEXE@"+itoa(i))
			case 0x4000:
				if i >= 8 {
					hits = append(hits, "16kb@"+itoa(i))
				}
			}
			if len(hits) >= 12 {
				break
			}
		}
		t.Logf("magics n=%d %v", len(hits), hits)
	if wplain, werr := decodeWASM(packed, 1<<20); werr != nil {
		t.Logf("wasm: %v", werr)
	} else {
		wcrc := crc32.ChecksumIEEE(wplain)
		t.Logf("wasm len=%d crc=%08x head=%x", len(wplain), wcrc, head(wplain, 32))
		whits := 0
		for i := 0; i+4 <= len(wplain); i++ {
			v := uint32(wplain[i]) | uint32(wplain[i+1])<<8 | uint32(wplain[i+2])<<16 | uint32(wplain[i+3])<<24
			if v == 0x26351817 || v == 0xC71B3AE1 || v == 0xC71B3AE2 {
				t.Logf("wasm magic %#x at %d", v, i)
				whits++
				if whits >= 6 {
					break
				}
			}
		}
	}
	}
	t.Logf("packed head=%x tail=%x s0=%08x s1=%08x end=%d",
		head(packed, 16), tail(packed, 16), d0.r.s0, d0.r.s1, d0.r.end)
	if got == want {
		return
	}
	// try alt inits
	alts := []struct {
		name string
		fn   func([]byte, int) ([]byte, error)
	}{
		{"be32", decodeAltBE},
		{"u16", decodeAltU16},
		{"swap", decodeAltSwap},
		{"fill", decodeAltFill},
		{"resnib", decodeAltResNib},
		{"img8", decodeAltImg8},
		{"tok0cdf", decodeAltTokZero},
		{"memcpy32", decodeAltMemcpy32},
		{"nswren", decodeAltNoSwapRenorm},
		{"t15lit", decodeAltT15ThenLit},
		{"t15nib", decodeAltT15ThenNib},
		{"t15bits", decodeAltT15ThenBits},
		{"bin0", decodeAltBin0},
		{"s1u16", decodeAltS1u16},
		{"copy8", decodeAltCopy8},
		{"t15d8", decodeAltT15Delta},
		{"t15d32", decodeAltT15Delta4},
		{"nochk", decodeAltNoChunk},
	}
	for _, a := range alts {
		o, e := a.fn(packed, 1<<20)
		if e != nil {
			t.Logf("%s: %v", a.name, e)
			continue
		}
		c := crc32.ChecksumIEEE(o)
		ch := uint32(0)
		if len(o) >= 12 {
			ch = uint32(o[8]) | uint32(o[9])<<8 | uint32(o[10])<<16 | uint32(o[11])<<24
		}
		t.Logf("%s len=%d crc=%08x head=%x dest8=%08x dispack=%v",
			a.name, len(o), c, head(o, 16), ch, ch >= 1 && ch <= 8<<20)
		if c == want {
			t.Fatalf("alt %s matched", a.name)
		}
	}
	d1 := newDec(packed, 1<<20)
	d1.altImg = 1
	o1, _ := runDec(d1, 1<<20)
	t.Logf("img8d len=%d crc=%08x head=%x ev=%v why=%s",
		len(o1), crc32.ChecksumIEEE(o1), head(o1, 16), d1.ev, d1.why)
	d2 := newDec(packed, 1<<20)
	d2.altImg = 1
	d2.altRes = 1
	o2, _ := runDec(d2, 1<<20)
	t.Logf("img8nib len=%d crc=%08x head=%x ev=%v why=%s",
		len(o2), crc32.ChecksumIEEE(o2), head(o2, 16), d2.ev, d2.why)

	for _, mode := range []int{rawNoSwap, rawFirstNS, rawBits16, rawBinLSB, rawBinMSB, rawImgU, rawNib} {
		d := newDec(packed, 1<<20)
		d.altRaw = mode
		o, e := runDec(d, 1<<20)
		if e != nil {
			t.Logf("raw%d: %v", mode, e)
			continue
		}
		ds, ts := uint32(0), uint32(0)
		if len(o) >= 8 {
			ds = uint32(o[0]) | uint32(o[1])<<8 | uint32(o[2])<<16 | uint32(o[3])<<24
			ts = uint32(o[4]) | uint32(o[5])<<8 | uint32(o[6])<<16 | uint32(o[7])<<24
		}
		ok := ds >= 1024 && ds <= 2<<20 && ts < 0x7fffffff && ts%4 == 0
		t.Logf("raw%d len=%d crc=%08x head=%x ds=%#x ts=%#x plaus=%v ev=%v",
			mode, len(o), crc32.ChecksumIEEE(o), head(o, 16), ds, ts, ok, d.ev)
	}
}

func head(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}
func tail(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[len(b)-n:]
}

func decodeAltBE(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	d.r.s0 = uint32(src[0])<<24 | uint32(src[1])<<16 | uint32(src[2])<<8 | uint32(src[3])
	d.r.s1 = uint32(src[4])<<24 | uint32(src[5])<<16 | uint32(src[6])<<8 | uint32(src[7])
	d.r.off = 8
	return runDec(d, dcap)
}
func decodeAltU16(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	d.r.s0 = uint32(src[0]) | uint32(src[1])<<8
	d.r.s1 = uint32(src[2]) | uint32(src[3])<<8
	d.r.off = 4
	return runDec(d, dcap)
}
func decodeAltSwap(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	d.r.s0, d.r.s1 = d.r.s1, d.r.s0
	return runDec(d, dcap)
}
func decodeAltResNib(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	d.altRes = 1
	return runDec(d, dcap)
}
func decodeAltImg8(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	d.altImg = 1
	return runDec(d, dcap)
}
func decodeAltTokZero(src []byte, dcap int) ([]byte, error) {
	// PE calloc leaves +0x9ac0 token CDF at 0; first sym is t15 (bsr).
	d := newDec(src, dcap)
	for i := range d.tok {
		d.tok[i] = 0
	}
	return runDec(d, dcap)
}

func decodeAltMemcpy32(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	_ = d.tok.sym(&d.r)
	n := 32
	if d.r.off+n > len(src) {
		n = len(src) - d.r.off
	}
	if n < 1 {
		return nil, errCodec
	}
	out := append([]byte(nil), src[d.r.off:d.r.off+n]...)
	return out, nil
}
func decodeAltNoSwapRenorm(src []byte, dcap int) ([]byte, error) {
	// All 16 u16s no-swap, then refill s0 so later tokens are not zeros.
	d := newDec(src, dcap)
	d.altRaw = rawNoSwap
	for d.pos < dcap {
		if d.lim > 0 && d.pos >= d.lim {
			break
		}
		if !d.decodeByte() {
			break
		}
		if d.rawN == 16 && d.r.s0 <= ransLimit {
			for k := 0; k < 2 && d.r.s0 <= ransLimit; k++ {
				if d.r.off+2 > len(d.r.src) {
					d.r.off = len(d.r.src) + 1
					break
				}
				d.r.s0 = d.r.s0<<16 | uint32(d.r.src[d.r.off]) | uint32(d.r.src[d.r.off+1])<<8
				d.r.off += 2
			}
			d.rawN = 0
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

// first t15 leftover header, then 12 u16s from the literal kernel.
func decodeAltT15ThenLit(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	if !d.decodeByte() {
		return nil, errCodec
	}
	if d.pos != 8 && d.pos != 12 && d.pos != 32 {
		return nil, errCodec
	}
	for d.pos < 32 {
		if !d.literal() {
			break
		}
	}
	for d.pos < dcap {
		if d.lim > 0 && d.pos >= d.lim {
			break
		}
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

// first t15 leftover header, then 12 u16s from a nibble model (not raw s0).
func decodeAltT15ThenNib(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	if !d.decodeByte() {
		return nil, errCodec
	}
	if d.pos != 8 && d.pos != 12 && d.pos != 32 {
		return nil, errCodec
	}
	for i := 0; i < 12 && d.r.ok(); i++ {
		hi := d.nibLit[0].sym(&d.r)
		lo := d.nibLit[1].sym(&d.r)
		d.emit(byte(hi<<4 | lo))
		d.emit(byte(d.nibLit[2].sym(&d.r)<<4 | d.nibLit[3].sym(&d.r)))
	}
	for d.pos < dcap {
		if d.lim > 0 && d.pos >= d.lim {
			break
		}
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

// first t15 leftover header, then 12 u16s via bits(16) (swap path).
func decodeAltT15ThenBits(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	if !d.decodeByte() {
		return nil, errCodec
	}
	if d.pos != 8 && d.pos != 12 && d.pos != 32 {
		return nil, errCodec
	}
	for i := 0; i < 12 && d.r.ok(); i++ {
		w := uint16(d.r.bits(16))
		d.emit(byte(w))
		d.emit(byte(w >> 8))
	}
	for d.pos < dcap {
		if d.lim > 0 && d.pos >= d.lim {
			break
		}
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

func decodeAltBin0(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	for i := range d.bin {
		d.bin[i] = 0
	}
	return runDec(d, dcap)
}

func decodeAltS1u16(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	if !d.decodeByte() {
		return nil, errCodec
	}
	w := d.rawSwapNoRefill()
	d.emit(byte(w))
	d.emit(byte(w >> 8))
	return runDec(d, dcap)
}

func decodeAltS1lane(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	if !d.decodeByte() {
		return nil, errCodec
	}
	for i := 0; i < 12; i++ {
		w := d.rawSwapNoRefill()
		d.emit(byte(w))
		d.emit(byte(w >> 8))
	}
	return runDec(d, dcap)
}

func decodeAltCopy8(src []byte, dcap int) ([]byte, error) {
	// tokRaw already writes dest[8:12] = DisPack 16KiB CHUNK_SIZE.
	return decodeNative(src, dcap)
}

func decodeAltT15Delta(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	d.noChunk = true
	if !d.decodeByte() {
		return nil, errCodec
	}
	if !d.tokDeltaU8(1) {
		return nil, errCodec
	}
	return runDec(d, dcap)
}

func decodeAltT15Delta4(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	d.noChunk = true
	if !d.decodeByte() {
		return nil, errCodec
	}
	if !d.tokDeltaU8(4) {
		return nil, errCodec
	}
	return runDec(d, dcap)
}

func decodeAltNoChunk(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	d.noChunk = true
	return runDec(d, dcap)
}

func decodeAltFill(src []byte, dcap int) ([]byte, error) {
	d := newDec(src, dcap)
	d.r.s0, d.r.s1, d.r.off = 0, 0, 0
	d.r.swap()
	d.r.swap()
	return runDec(d, dcap)
}
func runDec(d *dec, dcap int) ([]byte, error) {
	for d.pos < dcap {
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
