package jpgrec

// Arithmetic coder and bitwise writers translated from packjpg
// aricoder.cpp / bitops.cpp. C unsigned int is uint32; C int is int32.

import "errors"

const (
	coderUseBits  = 31
	coderLimit100 = uint32(1) << coderUseBits
	coderLimit025 = coderLimit100 / 4
	coderLimit050 = coderLimit100 / 2
	coderLimit075 = (coderLimit100 / 4) * 3
	escapeSymbol  = int32(coderLimit025)
)

type memReader struct {
	b []byte
	i int
}

func (r *memReader) readN(n int) int {
	if n < 0 || r.i >= len(r.b) {
		return 0
	}
	if r.i+n > len(r.b) {
		n = len(r.b) - r.i
	}
	r.i += n
	return n
}

func (r *memReader) readByte() (byte, bool) {
	if r.i >= len(r.b) {
		return 0, false
	}
	v := r.b[r.i]
	r.i++
	return v, true
}

func (r *memReader) u8() (byte, error) {
	b, ok := r.readByte()
	if !ok {
		return 0, errors.New("jpgrec: unexpected end of packJPG stream")
	}
	return b, nil
}

type bitWriter struct {
	data  []byte
	cbyte int
	cbit  int
	fill  byte
}

func newBitWriter() *bitWriter {
	return &bitWriter{data: make([]byte, 256), cbit: 8, fill: 1}
}

func (w *bitWriter) grow() {
	if w.cbyte+8 < len(w.data) {
		return
	}
	n := make([]byte, len(w.data)*2+64)
	copy(n, w.data)
	w.data = n
}

func rbits32(c uint32, n int32) uint32 {
	if n <= 0 {
		return 0
	}
	if n >= 32 {
		return c
	}
	return c & (0xFFFFFFFF >> uint(32-n))
}

func (w *bitWriter) write(val uint32, nbits int32) {
	if nbits <= 0 {
		return
	}
	w.grow()
	for nbits >= int32(w.cbit) {
		w.data[w.cbyte] |= byte(rbits32(val, nbits) >> uint(nbits-int32(w.cbit)))
		nbits -= int32(w.cbit)
		w.cbyte++
		w.cbit = 8
		w.grow()
	}
	if nbits > 0 {
		w.data[w.cbyte] |= byte(rbits32(val, nbits) << uint(w.cbit-int(nbits)))
		w.cbit -= int(nbits)
	}
}

func (w *bitWriter) pad(fill byte) {
	for w.cbit < 8 {
		w.write(uint32(fill), 1)
	}
}

func (w *bitWriter) pos() int { return w.cbyte }

func (w *bitWriter) bytes() []byte {
	w.pad(w.fill)
	out := make([]byte, w.cbyte)
	copy(out, w.data[:w.cbyte])
	return out
}

type byteBuf struct {
	data []byte
	n    int
}

func (b *byteBuf) write(v byte) {
	if b.n >= len(b.data) {
		n := make([]byte, len(b.data)*2+64)
		copy(n, b.data)
		b.data = n
	}
	b.data[b.n] = v
	b.n++
}

func (b *byteBuf) reset() { b.n = 0 }

func (b *byteBuf) peek() []byte { return b.data[:b.n] }

type symbol struct {
	low   uint32
	high  uint32
	scale uint32
}

type arith struct {
	r     *memReader
	bbyte byte
	cbit  uint8
	ccode uint32
	clow  uint32
	chigh uint32
	cstep uint32
}

func newArith(r *memReader) *arith {
	a := &arith{r: r, chigh: coderLimit100 - 1}
	for i := 0; i < coderUseBits; i++ {
		a.ccode = (a.ccode << 1) | uint32(a.readBit())
	}
	return a
}

func (a *arith) readBit() byte {
	if a.cbit == 0 {
		b, ok := a.r.readByte()
		if !ok {
			a.bbyte = 0
		} else {
			a.bbyte = b
		}
		a.cbit = 8
	}
	a.cbit--
	return (a.bbyte >> a.cbit) & 1
}

func (a *arith) decodeCount(s *symbol) (int32, error) {
	if s.scale == 0 {
		return 0, errors.New("jpgrec: arithmetic scale is 0")
	}
	a.cstep = (a.chigh - a.clow + 1) / s.scale
	if a.cstep == 0 {
		return 0, errors.New("jpgrec: arithmetic step is 0")
	}
	return int32((a.ccode - a.clow) / a.cstep), nil
}

func (a *arith) decode(s *symbol) {
	a.chigh = a.clow + a.cstep*s.high - 1
	a.clow = a.clow + a.cstep*s.low
	for a.clow >= coderLimit050 || a.chigh < coderLimit050 {
		if a.clow >= coderLimit050 {
			a.clow &= coderLimit050 - 1
			a.chigh &= coderLimit050 - 1
			a.ccode &= coderLimit050 - 1
		}
		a.clow <<= 1
		a.chigh <<= 1
		a.chigh++
		a.ccode <<= 1
		a.ccode |= uint32(a.readBit())
	}
	for a.clow >= coderLimit025 && a.chigh < coderLimit075 {
		a.clow &= coderLimit025 - 1
		a.chigh ^= coderLimit025 + coderLimit050
		a.ccode -= coderLimit025
		a.clow <<= 1
		a.chigh <<= 1
		a.chigh++
		a.ccode <<= 1
		a.ccode |= uint32(a.readBit())
	}
}

type tableS struct {
	counts    []uint16
	links     []*tableS
	lesser    *tableS
	maxCount  uint16
	maxSymbol int32
}

type modelS struct {
	totals    []uint32
	score     []byte
	sb0       int32
	ctx       []*tableS
	maxSymbol int32
	maxCtx    int32
	cur       int32
	maxOrder  int32
	maxCount  uint16
}

func newModelS(maxS, maxC, maxO int32) *modelS {
	if maxC < 1 {
		maxC = 1
	}
	m := &modelS{
		totals:    make([]uint32, maxS+2),
		score:     make([]byte, maxS),
		sb0:       maxS,
		maxSymbol: maxS,
		maxCtx:    maxC,
		maxOrder:  maxO,
		maxCount:  255,
		cur:       maxO,
	}
	nullT := &tableS{
		counts:    make([]uint16, maxS),
		maxCount:  1,
		maxSymbol: maxS,
	}
	for i := range nullT.counts {
		nullT.counts[i] = 1
	}
	start := &tableS{links: make([]*tableS, maxC), lesser: nullT}
	nullT.links = make([]*tableS, maxC)
	for i := range nullT.links {
		nullT.links[i] = start
	}
	n := int(maxO + 2)
	if n < 1 {
		n = 1
	}
	m.ctx = make([]*tableS, n)
	m.ctx[0] = nullT
	if maxO >= 0 {
		m.ctx[1] = start
	}
	for i := int32(1); i <= maxO; i++ {
		t := &tableS{lesser: m.ctx[i]}
		if i < maxO {
			t.links = make([]*tableS, maxC)
		}
		m.ctx[i].links[0] = t
		m.ctx[i+1] = t
	}
	return m
}

func (m *modelS) order(o int32) *tableS { return m.ctx[o+1] }

func (m *modelS) shift(c int32) {
	if m.maxOrder < 1 || c < 0 {
		return
	}
	for i := m.maxOrder; i > 0; i-- {
		parent := m.ctx[i] // C contexts[i-1]
		var context *tableS
		if int(c) < len(parent.links) {
			context = parent.links[c]
		}
		if context == nil {
			context = &tableS{}
			lesserParent := m.ctx[i-1] // C contexts[i-2]
			if int(c) < len(lesserParent.links) {
				context.lesser = lesserParent.links[c]
			}
			if i < m.maxOrder {
				context.links = make([]*tableS, m.maxCtx)
				m.ctx[i+2].lesser = context // C contexts[i+1]
			}
			if parent.links == nil {
				parent.links = make([]*tableS, m.maxCtx)
			}
			parent.links[c] = context
		}
		m.ctx[i+1] = context // C contexts[i]
	}
}

func (m *modelS) excludeAbove(c int32) {
	for x := c + 1; x < m.maxSymbol; x++ {
		if m.score[x] == 0 {
			m.score[x] = 1
			m.sb0--
		}
	}
}

func (m *modelS) totalize(context *tableS) error {
	if context.counts == nil {
		context.counts = make([]uint16, m.maxSymbol)
		m.totals[0] = 1
		m.totals[1] = 0
		return nil
	}
	local := m.sb0
	i := context.maxSymbol - 1
	if i+2 >= 0 && int(i+2) < len(m.totals) {
		m.totals[i+2] = 0
	}
	var curr uint32
	for ; i >= 0; i-- {
		if m.score[i] == 0 {
			cc := uint32(context.counts[i])
			if cc > 0 {
				curr += cc
				m.score[i] = 1
				m.sb0--
			}
		}
		m.totals[i+1] = curr
	}
	var esc uint32
	if local == m.sb0 {
		esc = 1
	} else if m.sb0 == 0 {
		esc = 0
	} else {
		esc = uint32(m.sb0 * (local - m.sb0))
		den := uint32(local) * uint32(context.maxCount)
		if den == 0 {
			return errors.New("jpgrec: arithmetic model count is 0")
		}
		esc /= den
		esc++
	}
	m.totals[0] = m.totals[1] + esc
	return nil
}

func (m *modelS) rescale(context *tableS, sh uint) {
	if context.counts == nil {
		return
	}
	lst := context.maxSymbol
	for i := int32(0); i < lst; i++ {
		if context.counts[i] > 0 {
			context.counts[i] >>= sh
		}
	}
	context.maxCount >>= sh
	i := lst - 1
	for ; i >= 0; i-- {
		if context.counts[i] > 0 {
			break
		}
	}
	context.maxSymbol = i + 1
}

func (m *modelS) flushRec(context *tableS, sh uint) {
	if context == nil {
		return
	}
	if context.links != nil {
		for i := range context.links {
			if context.links[i] != nil {
				m.flushRec(context.links[i], sh)
			}
		}
	}
	m.rescale(context, sh)
}

func (m *modelS) flush(sh uint) {
	if m.maxOrder < 0 || len(m.ctx) < 2 {
		return
	}
	m.flushRec(m.ctx[1], sh)
}

func (m *modelS) update(symbol int32) {
	if symbol >= 0 {
		local := m.cur
		if local < 0 {
			local = 0
		}
		for ; local <= m.maxOrder; local++ {
			context := m.order(local)
			if context.counts == nil {
				context.counts = make([]uint16, m.maxSymbol)
			}
			c := context.counts[symbol] + 1
			context.counts[symbol] = c
			if c > context.maxCount {
				context.maxCount = c
			}
			if symbol >= context.maxSymbol {
				context.maxSymbol = symbol + 1
			}
			if c >= m.maxCount {
				m.rescale(context, 1)
			}
		}
	}
	m.cur = m.maxOrder
	for i := int32(0); i < m.maxSymbol; i++ {
		m.score[i] = 0
	}
	m.sb0 = m.maxSymbol
}

func (m *modelS) fromCount(count int32, s *symbol) int32 {
	c := int32(1)
	for int(c) < len(m.totals) && count < int32(m.totals[c]) {
		c++
	}
	if int(c) >= len(m.totals) {
		c = int32(len(m.totals) - 1)
	}
	s.low = m.totals[c]
	s.high = m.totals[c-1]
	if c == 1 {
		m.cur--
		return escapeSymbol
	}
	return c - 2
}

func decodeS(a *arith, m *modelS) (int32, error) {
	for n := 0; n < 16; n++ {
		if err := m.totalize(m.order(m.cur)); err != nil {
			return 0, err
		}
		var s symbol
		s.scale = m.totals[0]
		count, err := a.decodeCount(&s)
		if err != nil {
			return 0, err
		}
		c := m.fromCount(count, &s)
		a.decode(&s)
		if c != escapeSymbol {
			m.update(c)
			return c, nil
		}
	}
	return 0, errors.New("jpgrec: arithmetic escape did not resolve")
}

type tableB struct {
	counts []uint16
	links  []*tableB
	lesser *tableB
	scale  uint32
}

type modelB struct {
	ctx      []*tableB
	maxCtx   int32
	maxOrder int32
	maxCount uint16
}

func newModelB(maxC, maxO int32) *modelB {
	if maxC < 1 {
		maxC = 1
	}
	m := &modelB{maxCtx: maxC, maxOrder: maxO, maxCount: 255}
	nullT := &tableB{counts: []uint16{1, 1}, scale: 2}
	start := &tableB{links: make([]*tableB, maxC), lesser: nullT}
	nullT.links = make([]*tableB, maxC)
	for i := range nullT.links {
		nullT.links[i] = start
	}
	n := int(maxO + 2)
	if n < 1 {
		n = 1
	}
	m.ctx = make([]*tableB, n)
	m.ctx[0] = nullT
	if maxO >= 0 {
		m.ctx[1] = start
	}
	for i := int32(1); i <= maxO; i++ {
		t := &tableB{lesser: m.ctx[i]}
		if i < maxO {
			t.links = make([]*tableB, maxC)
		}
		m.ctx[i].links[0] = t
		m.ctx[i+1] = t
	}
	return m
}

func (m *modelB) shift(c int32) {
	if m.maxOrder < 1 || c < 0 {
		return
	}
	for i := m.maxOrder; i > 0; i-- {
		parent := m.ctx[i]
		var context *tableB
		if int(c) < len(parent.links) {
			context = parent.links[c]
		}
		if context == nil {
			context = &tableB{}
			lesserParent := m.ctx[i-1]
			if int(c) < len(lesserParent.links) {
				context.lesser = lesserParent.links[c]
			}
			if i < m.maxOrder {
				context.links = make([]*tableB, m.maxCtx)
				m.ctx[i+2].lesser = context
			}
			if parent.links == nil {
				parent.links = make([]*tableB, m.maxCtx)
			}
			parent.links[c] = context
		}
		m.ctx[i+1] = context
	}
}

func (m *modelB) top() *tableB { return m.ctx[m.maxOrder+1] }

func checkCounts(context *tableB) {
	if context.counts == nil {
		context.counts = []uint16{1, 1}
		context.scale = 2
	}
}

func (m *modelB) rescale(context *tableB, sh uint) {
	if context.counts == nil {
		return
	}
	context.counts[0] >>= sh
	context.counts[1] >>= sh
	if context.counts[0] == 0 {
		context.counts[0] = 1
	}
	if context.counts[1] == 0 {
		context.counts[1] = 1
	}
	context.scale = uint32(context.counts[0]) + uint32(context.counts[1])
}

func (m *modelB) flushRec(context *tableB, sh uint) {
	if context == nil {
		return
	}
	if context.links != nil {
		for i := range context.links {
			if context.links[i] != nil {
				m.flushRec(context.links[i], sh)
			}
		}
	}
	m.rescale(context, sh)
}

func (m *modelB) flush(sh uint) {
	if m.maxOrder < 0 || len(m.ctx) < 2 {
		return
	}
	m.flushRec(m.ctx[1], sh)
}

func (m *modelB) update(symbol int32) {
	if symbol >= 0 && m.maxOrder >= 0 {
		context := m.top()
		checkCounts(context)
		context.counts[symbol]++
		context.scale++
		if context.counts[symbol] >= m.maxCount {
			m.rescale(context, 1)
		}
	}
}

func decodeB(a *arith, m *modelB) (int32, error) {
	context := m.top()
	checkCounts(context)
	var s symbol
	s.scale = context.scale
	countU, err := a.decodeCount(&s)
	if err != nil {
		return 0, err
	}
	count := uint32(countU)
	c0 := uint32(context.counts[0])
	var bit int32
	if count < c0 {
		s.low = 0
		s.high = c0
		bit = 0
	} else {
		s.low = c0
		s.high = s.scale
		bit = 1
	}
	a.decode(&s)
	m.update(bit)
	return bit, nil
}

func shiftS(m *modelS, cs ...int32) {
	for _, c := range cs {
		m.shift(c)
	}
}

func shiftB(m *modelB, cs ...int32) {
	for _, c := range cs {
		m.shift(c)
	}
}
