package magic2

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"testing"
)

func TestBitlen(t *testing.T) {
	t.Parallel()
	if bitlen(0) != 0 || bitlen(1) != 1 || bitlen(2) != 2 || bitlen(0x80) != 8 {
		t.Fatalf("bitlen %d %d %d %d", bitlen(0), bitlen(1), bitlen(2), bitlen(0x80))
	}
}

func TestGetBitState(t *testing.T) {
	t.Parallel()
	// slot < p0 → bit 0, state = quo*p + slot, p += (M-p)>>4
	var p uint16 = 0x2000
	r := &rANS{x: 0x20000000} // slot = 0, quo = 0x20000000>>14
	bit, err := r.getBit(&p)
	if err != nil || bit != 0 {
		t.Fatalf("bit %d err %v", bit, err)
	}
	if p <= 0x2000 {
		t.Fatalf("p0 did not grow: %#x", p)
	}
	// slot >= p0 → bit 1
	p = 0x100
	r = &rANS{x: 0x8000 | 0x200} // slot 0x200 > 0x100
	bit, err = r.getBit(&p)
	if err != nil || bit != 1 {
		t.Fatalf("bit1 %d err %v p=%#x x=%#x", bit, err, p, r.x)
	}
	if p >= 0x100 {
		t.Fatalf("p0 did not shrink: %#x", p)
	}
}

func TestGetNibbleFind(t *testing.T) {
	t.Parallel()
	var cdf [16]uint16
	initNibbleCDF(cdf[:])
	// slot just below 0x1000 → should be symbol 1 (cdf[2]=0x1000 is first > slot)
	r := &rANS{x: 0x8000 | 0x0fff}
	sym, err := r.getNibble(cdf[:])
	if err != nil || sym != 1 {
		t.Fatalf("sym %d err %v cdf1=%#x cdf2=%#x", sym, err, cdf[1], cdf[2])
	}
}

func TestHashNibbleZero(t *testing.T) {
	t.Parallel()
	if hashNibble(0, 0, 0, 0, 0, 0) != 0 {
		t.Fatal("zero history")
	}
	if hashMixer(0, 0, 0, 0) != 0 {
		t.Fatal("zero mixer")
	}
}

func TestFG06FCMFirst16(t *testing.T) {
	src := fg06Payload(t)
	fmt.Printf("state_be=%08x rest=%x\n", binary.BigEndian.Uint32(src[:4]), src[4:16])

	iir, iirok := decodeIIR(src)
	print16(t, "iir-nibble", iir, iirok)

	v22, v22ok := decodeV22(src)
	print16(t, "v22-bit+nibble", v22, v22ok)

	out, ok := decodeFCM(src)
	print16(t, "tok8+2nibble", out, ok)

	lit := decodeNibblesOnly(src)
	print16(t, "nibble-only", lit, false)

	lzna := decodeLZNAstyle(src)
	print16(t, "lzna-style", lzna, false)

	if !ok {
		t.Log("steam_appid CRC not matched")
	}

	// Sweep start-offset and lit-bit polarity; log printable heads.
	for off := 0; off <= 8; off += 4 {
		for inv := 0; inv < 2; inv++ {
			got := probeLit(src, off, inv == 1)
			if printable(got, 8) {
				t.Logf("printable off=%d inv=%v %x %q", off, inv == 1, prefix(got, 16), prefix(got, 16))
			} else {
				t.Logf("off=%d inv=%v n=%d %x", off, inv == 1, len(got), prefix(got, 8))
			}
		}
	}
}

func probeLit(src []byte, off int, invert bool) []byte {
	if off+4 > len(src) {
		return nil
	}
	st := &rANS{buf: src, off: off + 4, x: binary.BigEndian.Uint32(src[off : off+4])}
	st.renorm()
	hiTab := make([]uint16, 256*16)
	loTab := make([]uint16, 256*16)
	for i := 0; i < 256; i++ {
		initNibbleCDF(hiTab[i*16 : i*16+16])
		initNibbleCDF(loTab[i*16 : i*16+16])
	}
	var p uint16
	initBit(&p)
	out := make([]byte, 0, 16)
	prev := byte(0)
	for len(out) < 16 {
		bit, err := st.getBitN(&p, 14, 5)
		if err != nil {
			break
		}
		if invert {
			bit ^= 1
		}
		if bit != 0 {
			break
		}
		hi, err := st.getNibble(nibbleAt(hiTab, int(prev)))
		if err != nil {
			break
		}
		lo, err := st.getNibble(nibbleAt(loTab, (int(prev)<<4)|hi))
		if err != nil {
			break
		}
		b := byte(hi<<4 | lo)
		out = append(out, b)
		prev = b
	}
	return out
}

func printable(b []byte, n int) bool {
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return false
	}
	ok := 0
	for _, c := range b[:n] {
		if c == '\t' || c == '\n' || c == '\r' || c >= 0x20 && c < 0x7f {
			ok++
		}
	}
	return ok*4 >= n*3
}

func prefix(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}

func print16(t *testing.T, name string, out []byte, ok bool) {
	t.Helper()
	n := 16
	if n > len(out) {
		n = len(out)
	}
	fmt.Printf("%s n=%d ok=%v first=%x", name, len(out), ok, out[:n])
	if n > 0 {
		fmt.Printf(" ascii=%q", out[:n])
	}
	fmt.Println()
	if len(out) >= emuSize+appidSize {
		got := crc32.ChecksumIEEE(out[emuSize : emuSize+appidSize])
		fmt.Printf("  appid crc=%08x want=%08x\n", got, appidCRC)
	}
}

func decodeNibblesOnly(src []byte) []byte {
	st := &rANS{buf: src, off: 4, x: binary.BigEndian.Uint32(src[:4])}
	st.renorm()
	hiTab := make([]uint16, 256*16)
	loTab := make([]uint16, 256*16)
	for i := 0; i < 256; i++ {
		initNibbleCDF(hiTab[i*16 : i*16+16])
		initNibbleCDF(loTab[i*16 : i*16+16])
	}
	out := make([]byte, 0, 16)
	prev := byte(0)
	for len(out) < 16 {
		hi, err := st.getNibble(nibbleAt(hiTab, int(prev)))
		if err != nil {
			break
		}
		lo, err := st.getNibble(nibbleAt(loTab, (int(prev)<<4)|hi))
		if err != nil {
			break
		}
		b := byte(hi<<4 | lo)
		out = append(out, b)
		prev = b
	}
	return out
}

func decodeLZNAstyle(src []byte) []byte {
	st := &rANS{buf: src, off: 4, x: binary.BigEndian.Uint32(src[:4])}
	st.renorm()
	const nHi = 256
	hiTab := make([]uint16, nHi*16)
	loTab := make([]uint16, nHi*16)
	for i := 0; i < nHi; i++ {
		initNibbleCDF(hiTab[i*16 : i*16+16])
		initNibbleCDF(loTab[i*16 : i*16+16])
	}
	litP := make([]uint16, 256)
	for i := range litP {
		litP[i] = 1 << 12 // half of 1<<13
	}
	out := make([]byte, 0, 64)
	prev := byte(0)
	for len(out) < 64 {
		bit, err := st.getBitN(&litP[prev], 13, 5)
		if err != nil {
			break
		}
		if bit == 0 {
			hi, err := st.getNibble(nibbleAt(hiTab, int(prev)))
			if err != nil {
				break
			}
			lo, err := st.getNibble(nibbleAt(loTab, (int(prev)<<4)|hi))
			if err != nil {
				break
			}
			b := byte(hi<<4 | lo)
			out = append(out, b)
			prev = b
			continue
		}
		// match: one extra nibble as a cheap length, copy last byte
		if len(out) == 0 {
			break
		}
		ln, err := st.getNibble(nibbleAt(loTab, 0x80))
		if err != nil {
			break
		}
		n := ln + 2
		for i := 0; i < n && len(out) < 64; i++ {
			out = append(out, out[len(out)-1])
		}
		prev = out[len(out)-1]
	}
	return out
}
