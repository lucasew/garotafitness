package mp3rec

// Arithmetic coder and statistical models translated from packjpg
// aricoder.cpp. Counts use the packMP3 limit of 511.

const (
	coderUseBits  = 31
	coderLimit100 = uint32(1) << coderUseBits
	coderLimit025 = coderLimit100 / 4
	coderLimit050 = coderLimit100 / 2
	coderLimit075 = (coderLimit100 / 4) * 3
	escapeSymbol  = int32(coderLimit025)
	modelCountLim = 511
)

type memReader struct {
	b   []byte
	i   int
	eof bool
}

func (r *memReader) readExact(n int) ([]byte, error) {
	if n < 0 || r.i+n > len(r.b) {
		r.eof = true
		r.i = len(r.b)
		return nil, errUnexpected
	}
	out := r.b[r.i : r.i+n]
	r.i += n
	return out, nil
}

func (r *memReader) readByte() (byte, bool) {
	if r.i >= len(r.b) {
		return 0, false
	}
	v := r.b[r.i]
	r.i++
	return v, true
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

func (a *arith) decodeCount(scale uint32) (uint32, error) {
	if scale == 0 {
		return 0, errf("arithmetic scale is 0")
	}
	a.cstep = (a.chigh - a.clow + 1) / scale
	if a.cstep == 0 {
		return 0, errf("arithmetic step is 0")
	}
	return (a.ccode - a.clow) / a.cstep, nil
}

func (a *arith) decode(low, high uint32) {
	a.chigh = a.clow + a.cstep*high - 1
	a.clow = a.clow + a.cstep*low
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
	maxCount  int32
}

func newModelS(maxS, maxC, maxO int32) *modelS {
	m := &modelS{
		totals:    make([]uint32, maxS+2),
		score:     make([]byte, maxS),
		sb0:       maxS,
		maxSymbol: maxS,
		maxCtx:    maxC,
		maxOrder:  maxO,
		maxCount:  modelCountLim,
		cur:       maxO,
	}
	nullT := &tableS{
		counts:    make([]uint16, maxS),
		links:     make([]*tableS, maxC),
		maxCount:  1,
		maxSymbol: maxS,
	}
	for i := range nullT.counts {
		nullT.counts[i] = 1
	}
	start := &tableS{
		links:     make([]*tableS, maxC),
		lesser:    nullT,
		maxCount:  0,
		maxSymbol: 0,
	}
	for i := range nullT.links {
		nullT.links[i] = start
	}
	// Orders -1 .. maxO. Index of order o is o+1.
	m.ctx = make([]*tableS, maxO+2)
	m.ctx[0] = nullT
	m.ctx[1] = start
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

func (m *modelS) at(order int32) *tableS { return m.ctx[order+1] }

func (m *modelS) shift(c int32) error {
	if m.maxOrder < 1 || c < 0 {
		return nil
	}
	if c >= m.maxCtx {
		return errf("context index %d out of range %d", c, m.maxCtx)
	}
	for i := m.maxOrder; i > 0; i-- {
		parent := m.ctx[i] // order i-1
		if parent.links == nil || int(c) >= len(parent.links) {
			return errf("context index %d out of range %d", c, m.maxCtx)
		}
		context := parent.links[c]
		if context == nil {
			context = &tableS{}
			lesserParent := m.ctx[i-1] // order i-2
			if lesserParent.links != nil && int(c) < len(lesserParent.links) {
				context.lesser = lesserParent.links[c]
			}
			if i == m.maxOrder {
				context.links = nil
			} else {
				context.links = make([]*tableS, m.maxCtx)
				m.ctx[i+2].lesser = context // order i+1
			}
			parent.links[c] = context
		}
		m.ctx[i+1] = context // order i
	}
	return nil
}

func (m *modelS) totalize(context *tableS) error {
	if context.counts == nil {
		context.counts = make([]uint16, m.maxSymbol)
		m.totals[0] = 1
		m.totals[1] = 0
		return nil
	}
	local := m.sb0
	i := int(context.maxSymbol) - 1
	m.totals[i+2] = 0
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
		den := uint32(local) * uint32(context.maxCount)
		if den == 0 {
			return errf("arithmetic model count is 0")
		}
		esc = uint32(m.sb0)*uint32(local-m.sb0)/den + 1
	}
	m.totals[0] = m.totals[1] + esc
	return nil
}

func (m *modelS) rescale(context *tableS, sh uint) {
	if context.counts == nil {
		return
	}
	lst := int(context.maxSymbol)
	for i := 0; i < lst; i++ {
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
	context.maxSymbol = int32(i + 1)
}

func (m *modelS) flushRec(context *tableS, sh uint) {
	if context == nil {
		return
	}
	if context.links != nil {
		for i := int32(0); i < m.maxCtx && int(i) < len(context.links); i++ {
			if context.links[i] != nil {
				m.flushRec(context.links[i], sh)
			}
		}
	}
	m.rescale(context, sh)
}

func (m *modelS) flush(sh uint) {
	// contexts[0] is the start table, not the null table.
	m.flushRec(m.ctx[1], sh)
}

func (m *modelS) update(symbol int32) error {
	if symbol >= 0 {
		local := m.cur
		if local < 0 {
			local = 0
		}
		for ; local <= m.maxOrder; local++ {
			context := m.at(local)
			if context.counts == nil {
				return errf("model counts missing")
			}
			if int(symbol) >= len(context.counts) {
				return errf("symbol %d out of model", symbol)
			}
			context.counts[symbol]++
			c := context.counts[symbol]
			if c > context.maxCount {
				context.maxCount = c
			}
			if symbol >= context.maxSymbol {
				context.maxSymbol = symbol + 1
			}
			if int32(c) >= m.maxCount {
				m.rescale(context, 1)
			}
		}
	}
	m.cur = m.maxOrder
	for i := int32(0); i < m.maxSymbol; i++ {
		m.score[i] = 0
	}
	m.sb0 = m.maxSymbol
	return nil
}

func (m *modelS) fromCount(count int32) (sym int32, low, high uint32, err error) {
	c := 1
	for c < len(m.totals) && count < int32(m.totals[c]) {
		c++
	}
	if c >= len(m.totals) || c < 1 {
		return 0, 0, 0, errf("arithmetic totals exhausted")
	}
	low = m.totals[c]
	high = m.totals[c-1]
	if c == 1 {
		m.cur--
		return escapeSymbol, low, high, nil
	}
	return int32(c - 2), low, high, nil
}

func (a *arith) decodeS(m *modelS) (int32, error) {
	for {
		if m.cur < -1 {
			return 0, errf("arithmetic escape underrun")
		}
		if err := m.totalize(m.at(m.cur)); err != nil {
			return 0, err
		}
		count, err := a.decodeCount(m.totals[0])
		if err != nil {
			return 0, err
		}
		sym, low, high, err := m.fromCount(int32(count))
		if err != nil {
			return 0, err
		}
		a.decode(low, high)
		if sym != escapeSymbol {
			if err := m.update(sym); err != nil {
				return 0, err
			}
			return sym, nil
		}
	}
}

func shiftS(m *modelS, cs ...int32) error {
	for _, c := range cs {
		if err := m.shift(c); err != nil {
			return err
		}
	}
	return nil
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
	maxCount int32
}

func newModelB(maxC, maxO int32) *modelB {
	m := &modelB{maxCtx: maxC, maxOrder: maxO, maxCount: modelCountLim}
	nullT := &tableB{
		counts: []uint16{1, 1},
		links:  make([]*tableB, maxC),
		scale:  2,
	}
	start := &tableB{links: make([]*tableB, maxC), lesser: nullT}
	for i := range nullT.links {
		nullT.links[i] = start
	}
	m.ctx = make([]*tableB, maxO+2)
	m.ctx[0] = nullT
	m.ctx[1] = start
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

func (m *modelB) at(order int32) *tableB { return m.ctx[order+1] }

func (m *modelB) shift(c int32) error {
	if m.maxOrder < 1 || c < 0 {
		return nil
	}
	if c >= m.maxCtx {
		return errf("context index %d out of range %d", c, m.maxCtx)
	}
	for i := m.maxOrder; i > 0; i-- {
		parent := m.ctx[i]
		if parent.links == nil || int(c) >= len(parent.links) {
			return errf("context index %d out of range %d", c, m.maxCtx)
		}
		context := parent.links[c]
		if context == nil {
			context = &tableB{}
			lesserParent := m.ctx[i-1]
			if lesserParent.links != nil && int(c) < len(lesserParent.links) {
				context.lesser = lesserParent.links[c]
			}
			if i == m.maxOrder {
				context.links = nil
			} else {
				context.links = make([]*tableB, m.maxCtx)
				m.ctx[i+2].lesser = context
			}
			parent.links[c] = context
		}
		m.ctx[i+1] = context
	}
	return nil
}

func checkCountsB(context *tableB) {
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
		for i := int32(0); i < m.maxCtx && int(i) < len(context.links); i++ {
			if context.links[i] != nil {
				m.flushRec(context.links[i], sh)
			}
		}
	}
	m.rescale(context, sh)
}

func (m *modelB) flush(sh uint) { m.flushRec(m.ctx[1], sh) }

func (m *modelB) update(symbol int32) {
	if symbol >= 0 && m.maxOrder >= 0 {
		context := m.at(m.maxOrder)
		context.counts[symbol]++
		context.scale++
		if int32(context.counts[symbol]) >= m.maxCount {
			m.rescale(context, 1)
		}
	}
}

func (a *arith) decodeB(m *modelB) (int32, error) {
	context := m.at(m.maxOrder)
	checkCountsB(context)
	countU, err := a.decodeCount(context.scale)
	if err != nil {
		return 0, err
	}
	count := int32(countU)
	c0 := int32(context.counts[0])
	var bit int32
	var low, high uint32
	if count < c0 {
		low = 0
		high = uint32(c0)
		bit = 0
	} else {
		low = uint32(c0)
		high = context.scale
		bit = 1
	}
	a.decode(low, high)
	m.update(bit)
	return bit, nil
}

func shiftB(m *modelB, cs ...int32) error {
	for _, c := range cs {
		if err := m.shift(c); err != nil {
			return err
		}
	}
	return nil
}
