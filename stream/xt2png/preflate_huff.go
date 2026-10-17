package xt2png

// HuffmanEncoder from support/huffman_encoder.cpp. Lengths are stored
// shifted by one (blCount[bitlen+1]) so a zero-length symbol can be
// dropped without a hole in the canonical codes.

const huffMaxBL = 25

type huffEnc struct {
	lookup []uint32
	err    bool
}

func countSymbols(lengths []byte, disableZero bool) (next []uint32, minL, maxL int, ok bool) {
	n := len(lengths)
	if n < 1 || n >= 1024 {
		return nil, 0, 0, false
	}
	blCount := make([]uint16, huffMaxBL+2)
	for _, L := range lengths {
		l := int(L) + 1
		if l > huffMaxBL+1 || l < 0 {
			return nil, 0, 0, false
		}
		blCount[l]++
	}
	minL = 1
	for minL <= huffMaxBL+1 && blCount[minL] == 0 {
		minL++
	}
	maxL = huffMaxBL + 1
	for maxL >= minL && blCount[maxL] == 0 {
		maxL--
	}
	if minL > maxL {
		return nil, 0, 0, false
	}
	blCount[0] = 0
	if disableZero {
		blCount[1] = 0
	}
	next = make([]uint32, huffMaxBL+2)
	code := uint32(0)
	for i := minL; i <= maxL; i++ {
		code = (code + uint32(blCount[i-1])) << 1
		next[i] = code
	}
	if minL == maxL && blCount[maxL] == 1 {
		return next, minL, maxL, true
	}
	if next[maxL]+uint32(blCount[maxL]) != 1<<uint(maxL-1) {
		return nil, 0, 0, false
	}
	return next, minL, maxL, true
}

func newHuff(lengths []byte, disableZero bool) *huffEnc {
	h := &huffEnc{lookup: make([]uint32, len(lengths))}
	next, _, _, ok := countSymbols(lengths, disableZero)
	if !ok {
		h.err = true
		return h
	}
	minKeep := 1
	if disableZero {
		minKeep = 2
	}
	for i := range lengths {
		l := int(lengths[i]) + 1
		if l < minKeep {
			h.lookup[i] = 0
			continue
		}
		k := uint32(l - 1)
		code := bitReverse(next[l], k)
		next[l]++
		h.lookup[i] = (code << 5) | k
	}
	return h
}

func (h *huffEnc) encode(w *bitWriter, sym uint) bool {
	if h == nil || h.err || int(sym) >= len(h.lookup) {
		return false
	}
	v := h.lookup[sym]
	w.put(v>>5, v&0x1f)
	return true
}

var staticLitEnc, staticDistEnc *huffEnc

func init() {
	var lit [288]byte
	for i := 0; i < 144; i++ {
		lit[i] = 8
	}
	for i := 144; i < 256; i++ {
		lit[i] = 9
	}
	for i := 256; i < 280; i++ {
		lit[i] = 7
	}
	for i := 280; i < 288; i++ {
		lit[i] = 8
	}
	staticLitEnc = newHuff(lit[:], true)
	var dist [32]byte
	for i := range dist {
		dist[i] = 5
	}
	staticDistEnc = newHuff(dist[:], true)
}
