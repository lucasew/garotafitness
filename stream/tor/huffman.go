package tor

// huffTree is HuffmanTree in EntropyCoder.cpp (decoder direction).
type huffTree struct {
	n       int
	counter []uint32
	bits    []uint8
	maxbits int
	maxSeen int
	fast    []int
	index   []uint16
}

func newHuff(n int) *huffTree {
	h := &huffTree{
		n:       n,
		counter: make([]uint32, n),
		bits:    make([]uint8, n),
		fast:    make([]int, 1<<fastBits),
	}
	for i := range h.counter {
		h.counter[i] = 1
	}
	if err := h.build(0); err != nil {
		panic(err)
	}
	return h
}

type node struct {
	cnt         uint32
	code        uint32
	left, right uint16
	bits        uint8
}

func (h *huffTree) build(mode int) error {
	const chunks = 250
	buf := make([]node, 2*maxHuf+4)
	places := make([]int, chunks+1)
	b := 0
	for i := 0; i < h.n; i++ {
		if h.counter[i] < chunks {
			places[h.counter[i]+1]++
		}
		b++
	}
	for i := 0; i < chunks; i++ {
		places[i+1] += places[i]
	}
	rest := make([]uint32, 0, h.n)
	for i := 0; i < h.n; i++ {
		c := h.counter[i]
		if c < chunks {
			p := places[c]
			places[c]++
			buf[p].cnt = c
			buf[p].left = uint16(i)
		} else {
			rest = append(rest, c*maxHuf+uint32(i))
		}
	}
	// Node0 keys are unique (count, index), so a straight sort matches qsort.
	for i := 1; i < len(rest); i++ {
		v := rest[i]
		j := i
		for j > 0 && rest[j-1] > v {
			rest[j] = rest[j-1]
			j--
		}
		rest[j] = v
	}
	for i, v := range rest {
		p := i + places[chunks]
		buf[p].cnt = v / maxHuf
		buf[p].left = uint16(v % maxHuf)
	}
	for i := 0; i < b+4; i++ {
		buf[b+i].cnt = 1<<31 - 1 // INT_MAX fence
	}
	p1, p2, p3 := 0, b+2, b+2
	for !(p1 == b && p3-p2 == 1) {
		if buf[p1+1].cnt < buf[p2].cnt {
			buf[p3].cnt = buf[p1].cnt + buf[p1+1].cnt
			buf[p3].left = uint16(p1)
			buf[p3].right = uint16(p1 + 1)
			p1 += 2
		} else if buf[p1].cnt > buf[p2+1].cnt {
			buf[p3].cnt = buf[p2].cnt + buf[p2+1].cnt
			buf[p3].left = uint16(p2)
			buf[p3].right = uint16(p2 + 1)
			p2 += 2
		} else {
			buf[p3].cnt = buf[p1].cnt + buf[p2].cnt
			buf[p3].left = uint16(p1)
			buf[p3].right = uint16(p2)
			p1++
			p2++
		}
		p3++
	}
	buf[p2].bits = 0
	buf[p2].code = 0
	for i := p2; i >= b+2; i-- {
		parentBits := buf[i].bits
		parentCode := buf[i].code
		L := buf[i].left
		R := buf[i].right
		buf[L].bits = parentBits + 1
		buf[L].code = parentCode
		buf[R].bits = parentBits + 1
		buf[R].code = parentCode + (uint32(1) << parentBits)
	}
	h.maxbits = int(buf[0].bits)
	if h.maxbits < 0 || h.maxbits > 28 {
		return errFormat
	}
	if h.maxbits > h.maxSeen {
		h.maxSeen = h.maxbits
		idx := make([]uint16, 1<<h.maxbits)
		copy(idx, h.index)
		h.index = idx
	}
	for i := 0; i < b; i++ {
		s := int(buf[i].left)
		sbits := int(buf[i].bits)
		scode := buf[i].code
		if s < 0 || s >= h.n {
			return errFormat
		}
		h.bits[s] = uint8(sbits)
		if sbits <= fastBits {
			n := 1 << (fastBits - sbits)
			for j := 0; j < n; j++ {
				h.fast[scode+uint32(j<<sbits)] = s
			}
		} else {
			h.fast[scode&((1<<fastBits)-1)] = -1
			n := 1 << (h.maxbits - sbits)
			for j := 0; j < n; j++ {
				h.index[scode+uint32(j<<sbits)] = uint16(s)
			}
		}
	}
	factor := uint32(2)
	switch mode {
	case 0:
		factor = 2
	case 1:
		factor = 3
	case 2:
		factor = 4
	case 3:
		factor = 6
	case 4:
		factor = 8
	case 5:
		factor = 10
	case 6:
		factor = 12
	case 7:
		factor = 16
	}
	for s, c := range h.counter {
		var sub uint32
		if c > 1 && c < factor {
			sub = 1
		} else {
			sub = c / factor
		}
		h.counter[s] = c - sub
	}
	return nil
}

func (h *huffTree) sym(code uint32) int {
	x := h.fast[code&((1<<fastBits)-1)]
	if x >= 0 {
		return x
	}
	if int(code) >= len(h.index) {
		return -1
	}
	return int(h.index[code])
}

type huffDec struct {
	bit  *bitIn
	tree *huffTree
	fail error
}

func newHuffDec(src []byte, n int) *huffDec {
	return &huffDec{bit: newBitIn(src), tree: newHuff(n)}
}

func (h *huffDec) decodeSym() uint32 {
	if h.fail != nil {
		return 0
	}
	for {
		nb := h.bit.needbits(h.tree.maxbits)
		s := h.tree.sym(nb)
		if s < 0 || s >= h.tree.n {
			h.fail = errFormat
			return 0
		}
		h.bit.dumpbits(int(h.tree.bits[s]))
		if s != eobCode {
			h.tree.counter[s]++
			return uint32(s)
		}
		mode := int(h.bit.getbits(3))
		if err := h.tree.build(mode); err != nil {
			h.fail = err
			return 0
		}
	}
}

func (h *huffDec) decode() uint32 { return h.decodeSym() }

func (h *huffDec) getbits(n int) uint32 { return h.bit.getbits(n) }

func (h *huffDec) failed() error { return h.fail }
