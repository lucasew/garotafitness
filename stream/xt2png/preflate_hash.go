package xt2png

// Hash and sequence chains. totalShift starts at (unsigned)-8 so chain
// indices are pos+8. The hash constructor primes bytes 0 and 1 when the
// buffer is longer than 2; the token predictor primes those two bytes
// again. Both steps are required.

const (
	chainWin   = 1 << 16
	chainDelta = 0x7e00
)

type hashChain struct {
	data       []byte
	pos        uint32
	head       []uint16
	prev       []uint16
	chainDepth []uint32
	hashBits   uint8
	hashShift  uint8
	running    uint16
	hashMask   uint16
	totalShift uint32
}

func newHash(data []byte, memLevel uint8) *hashChain {
	hb := uint32(memLevel) + 7
	h := &hashChain{
		data:       data,
		hashBits:   uint8(hb),
		hashShift:  uint8((hb + 2) / 3),
		hashMask:   uint16((uint32(1) << hb) - 1),
		totalShift: 0xfffffff8,
	}
	h.head = make([]uint16, int(h.hashMask)+1)
	h.prev = make([]uint16, chainWin)
	h.chainDepth = make([]uint32, chainWin)
	if len(data) > 2 {
		h.updateRunning(data[0])
		h.updateRunning(data[1])
	}
	return h
}

func (h *hashChain) updateRunning(b byte) {
	h.running = uint16((uint32(h.running) << h.hashShift) ^ uint32(b))
}

func (h *hashChain) next1(b byte) uint32 {
	return (uint32(h.running) << h.hashShift) ^ uint32(b)
}

func (h *hashChain) next2(b1, b2 byte) uint32 {
	return (((uint32(h.running) << h.hashShift) ^ uint32(b1)) << h.hashShift) ^ uint32(b2)
}

func (h *hashChain) reshift() {
	delta := uint16(chainDelta)
	for i := range h.head {
		if h.head[i] < delta {
			h.head[i] = 0
		} else {
			h.head[i] -= delta
		}
	}
	for i := chainDelta + 8; i < chainWin; i++ {
		v := h.prev[i]
		if v < delta {
			v = 0
		} else {
			v -= delta
		}
		h.prev[i-chainDelta] = v
	}
	const keep = chainWin - 8 - chainDelta
	copy(h.chainDepth[8:8+keep], h.chainDepth[8+chainDelta:8+chainDelta+keep])
	h.totalShift += chainDelta
}

func (h *hashChain) updateHash(l uint32) bool {
	if l > 0x180 {
		for l > 0 {
			blk := l
			if blk > 0x180 {
				blk = 0x180
			}
			if !h.updateHash(blk) {
				return false
			}
			l -= blk
		}
		return true
	}
	if int(h.pos) > len(h.data) {
		return false
	}
	pos := h.pos
	if pos-h.totalShift >= 0xfe08 {
		h.reshift()
	}
	rem := uint32(len(h.data)) - h.pos
	limit := l + 2
	if limit > rem {
		limit = rem
	}
	for i := uint32(2); i < limit; i++ {
		h.updateRunning(h.data[h.pos+i])
		hh := uint32(h.running) & uint32(h.hashMask)
		p := (pos + i - 2) - h.totalShift
		if p >= chainWin || int(hh) >= len(h.head) {
			return false
		}
		h.chainDepth[p] = h.chainDepth[h.head[hh]] + 1
		h.prev[p] = h.head[hh]
		h.head[hh] = uint16(p)
	}
	h.pos += l
	return true
}

func (h *hashChain) skipHash(l uint32) bool {
	if int(h.pos) > len(h.data) {
		return false
	}
	pos := h.pos
	if pos-h.totalShift >= 0xfe08 {
		h.reshift()
	}
	rem := uint32(len(h.data)) - h.pos
	if rem > 2 {
		if h.pos+2 >= uint32(len(h.data)) {
			return false
		}
		h.updateRunning(h.data[h.pos+2])
		hh := uint32(h.running) & uint32(h.hashMask)
		p := pos - h.totalShift
		if p >= chainWin || int(hh) >= len(h.head) {
			return false
		}
		h.chainDepth[p] = h.chainDepth[h.head[hh]] + 1
		h.prev[p] = h.head[hh]
		h.head[hh] = uint16(p)
		for i := uint32(1); i < l; i++ {
			p = (pos + i) - h.totalShift
			if p >= chainWin {
				return false
			}
			h.chainDepth[p] = 0xffff8000
		}
		if rem > l {
			if h.pos+l >= uint32(len(h.data)) {
				return false
			}
			h.updateRunning(h.data[h.pos+l])
			if rem > l+1 {
				if h.pos+l+1 >= uint32(len(h.data)) {
					return false
				}
				h.updateRunning(h.data[h.pos+l+1])
			}
		}
	}
	h.pos += l
	return true
}

func (h *hashChain) getHead(hash uint32) uint32 {
	return uint32(h.head[hash&uint32(h.hashMask)])
}

func (h *hashChain) relDepth(refPos, head uint32) uint32 {
	return h.chainDepth[head] - h.chainDepth[refPos-h.totalShift]
}

type hashIt struct {
	prev    []uint16
	depth   []uint32
	refPos  uint32
	maxDist uint32
	curPos  uint32
	curDist uint32
}

func (h *hashChain) iterNode(node, refPos, maxDist uint32) hashIt {
	rel := refPos - h.totalShift
	return hashIt{
		prev: h.prev, depth: h.chainDepth,
		refPos: rel, maxDist: maxDist,
		curPos: node, curDist: rel - node,
	}
}

func (it *hashIt) next() bool {
	if it.curPos >= chainWin {
		it.curPos = 0
		it.curDist = it.refPos
		return false
	}
	it.curPos = uint32(it.prev[it.curPos])
	it.curDist = it.refPos - it.curPos
	return it.curPos > 0 && it.curDist <= it.maxDist
}

type seqEnt struct {
	distToNext uint16
	length     uint16
}

type seqChain struct {
	data       []byte
	pos        uint32
	prev       []seqEnt
	heads      [256]uint16
	totalShift uint32
	curPos     uint32
}

func newSeq(data []byte) *seqChain {
	s := &seqChain{
		data:       data,
		prev:       make([]seqEnt, chainWin),
		totalShift: 0xfffffff8,
	}
	n := uint32(len(data))
	if n > chainWin-8 {
		n = chainWin - 8
	}
	s.build(8, n)
	return s
}

func (s *seqChain) build(off0, size uint32) {
	if size == 0 {
		return
	}
	base := s.pos
	if base+size > uint32(len(s.data)) {
		return
	}
	cur := s.data[base]
	startOff := off0
	s.prev[off0] = seqEnt{distToNext: 0xffff}
	if off0 > 8 && base >= 1 && cur == s.data[base-1] {
		startOff--
		if base >= 2 && cur == s.data[base-2] {
			startOff--
			if base >= 3 && cur == s.data[base-3] {
				startOff = uint32(s.heads[cur])
				s.prev[off0-2].distToNext = uint16(off0 - startOff - 2)
				s.prev[off0-1].distToNext = uint16(off0 - startOff - 1)
				s.prev[off0].distToNext = uint16(off0 - startOff)
				s.prev[off0].length = 1
			} else {
				s.prev[startOff].distToNext = uint16(startOff - uint32(s.heads[cur]))
				s.prev[startOff+1].distToNext = 1
				s.prev[startOff+2].distToNext = 2
				s.prev[startOff+2].length = 1
				s.heads[cur] = uint16(startOff)
			}
		} else {
			s.prev[startOff+1].distToNext = 1
			s.prev[startOff+1].length = 1
		}
	}
	s.prev[startOff].length++
	prev := cur
	for i := uint32(1); i < size; i++ {
		cur = s.data[base+i]
		if prev == cur {
			s.prev[startOff].length++
			if s.prev[startOff].length == 3 {
				s.prev[startOff].distToNext = uint16(startOff - uint32(s.heads[prev]))
				s.heads[prev] = uint16(startOff)
			}
			s.prev[off0+i].distToNext = uint16(off0 + i - startOff)
			s.prev[off0+i].length = 1
		} else {
			if s.prev[startOff].length >= 2 {
				if s.prev[startOff].length >= 3 {
					s.prev[off0+i-2].distToNext = 0xffff
				}
				s.prev[off0+i-1].distToNext = 0xffff
			}
			s.prev[off0+i] = seqEnt{distToNext: 0xffff}
			startOff = off0 + i
			s.prev[startOff].length++
		}
		prev = cur
	}
	if s.prev[startOff].length >= 2 {
		if s.prev[startOff].length >= 3 {
			s.prev[off0+size-2].distToNext = 0xffff
		}
		s.prev[off0+size-1].distToNext = 0xffff
	}
	s.pos += size
}

func (s *seqChain) reshift() {
	const delta = uint32(chainDelta)
	remaining := uint32(chainWin) - (delta + 8)
	e := &s.prev[delta+8]
	if e.distToNext != 0xffff && e.length < minMatch {
		d := uint32(e.distToNext)
		e.distToNext = 0xffff
		e.length = s.prev[delta+8-d].length - uint16(d)
		for i := uint32(3); i < uint32(e.length); i++ {
			s.prev[delta+8+i-2].distToNext -= uint16(d)
		}
		if s.pos >= remaining && int(s.pos-remaining) < len(s.data) {
			c := s.data[s.pos-remaining]
			if uint32(s.heads[c]) == delta+8-d {
				s.heads[c] += uint16(d)
			} else {
				for i := uint32(e.length); i < remaining; i++ {
					if uint32(s.prev[delta+8+i].distToNext) == i+d {
						s.prev[delta+8+i].distToNext -= uint16(d)
						break
					}
				}
			}
		}
	}
	for i := 0; i < 256; i++ {
		if uint32(s.heads[i]) < delta {
			s.heads[i] = 0
		} else {
			s.heads[i] -= uint16(delta)
		}
	}
	copy(s.prev[8:], s.prev[delta+8:delta+8+remaining])
	s.totalShift += delta
	sz := delta
	if s.pos > uint32(len(s.data)) {
		sz = 0
	} else if rem := uint32(len(s.data)) - s.pos; rem < sz {
		sz = rem
	}
	s.build(8+remaining, sz)
}

func (s *seqChain) updateSeq(l uint32) {
	s.curPos += l
	for s.curPos-s.totalShift >= 0xfe08 {
		s.reshift()
	}
}

func (s *seqChain) valid(refPos uint32) bool {
	idx := refPos - s.totalShift
	if idx >= chainWin {
		return false
	}
	return s.prev[idx].distToNext != 0xffff
}

func (s *seqChain) seqLen(refPos uint32) uint16 {
	return s.prev[refPos-s.totalShift].length
}

type seqIt struct {
	chain   []seqEnt
	refPos  uint32
	curDist uint32
}

func (s *seqChain) iterate(refPos uint32) seqIt {
	idx := refPos - s.totalShift
	return seqIt{chain: s.prev, refPos: idx, curDist: uint32(s.prev[idx].distToNext)}
}

func (it seqIt) ok() bool { return it.curDist <= it.refPos-8 }

func (it seqIt) dist() uint32 { return it.curDist }

func (it seqIt) length() uint16 { return it.chain[it.refPos-it.curDist].length }

func (it *seqIt) next() bool {
	it.curDist += uint32(it.chain[it.refPos-it.curDist].distToNext)
	return it.ok()
}
