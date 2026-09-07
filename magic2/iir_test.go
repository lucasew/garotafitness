package magic2

import "testing"

func TestHashNibbleRange(t *testing.T) {
	t.Parallel()
	if hashNibble(0, 0, 0, 0, 0, 0) != 0 {
		t.Fatal("zero")
	}
	// max bitlen8 is 8
	off := hashNibble(0xff, 0, 0, 0, 0xff, 0)
	if off/32 >= nibbleRows || off < 0 || off%32 != 0 {
		t.Fatalf("off %d", off)
	}
}

func TestMixIIR(t *testing.T) {
	t.Parallel()
	a, b := mixIIR(0, 0, 0, 0)
	if a != 0 || b != 0 {
		t.Fatalf("zero mix %d %d", a, b)
	}
	a, b = mixIIR(0, 0, 5, 0)
	if a == 0 && b == 0 {
		t.Fatal("nonzero sample should move")
	}
}

func TestExtraSampleZero(t *testing.T) {
	t.Parallel()
	var h iirHist
	st := &rANS{x: 0x20000000}
	bits := newExtraBits()
	a, b, err := h.extraSample(st, bits, 1, 0)
	if err != nil || a != 0 || b != 0 {
		t.Fatalf("bsf1 %d %d %v", a, b, err)
	}
	if st.off != 0 {
		t.Fatalf("bsf1 consumed input %d", st.off)
	}
}

func TestExtraBitOff(t *testing.T) {
	t.Parallel()
	// h1=0, sym-1=0, level 0, rdx 0 → 0
	if extraBitOff(0, 0, 0, 0) != 0 {
		t.Fatal("zero")
	}
	// h1=1 is +2048 bytes = +1024 u16
	if extraBitOff(1, 0, 0, 0) != 1024 {
		t.Fatalf("h1 %d", extraBitOff(1, 0, 0, 0))
	}
	// (sym-1)<<8 bytes = 128 u16
	if extraBitOff(0, 1, 0, 0) != 128 {
		t.Fatalf("sym %d", extraBitOff(0, 1, 0, 0))
	}
	// level*32 + 8*rdx bytes
	if extraBitOff(0, 0, 1, 3) != (32+24)/2 {
		t.Fatalf("walk %d", extraBitOff(0, 0, 1, 3))
	}
}

func TestIIRRowMoves(t *testing.T) {
	t.Parallel()
	var h iirHist
	if h.row() != 0 {
		t.Fatal("start row")
	}
	h.afterNibble(7)
	if h.w20 == 0 && h.w24 == 0 {
		t.Fatal("history stuck")
	}
}
