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
