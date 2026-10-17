package xt2png

import "sort"

// Submodels follow preflate_statistical_codec.cpp. N == 0 is the empty
// PreflateSubModel specialization (decode returns 0, never reads bits).

type subModel struct {
	n         int
	bounds    []uint32
	scaled    []uint32
	ids       []uint16
	rids      []uint16
	scaleDown uint32
	isFixed   bool
}

func (sm *subModel) alloc(n int) {
	sm.n = n
	sm.bounds = make([]uint32, n+1)
	sm.scaled = make([]uint32, n+1)
	sm.ids = make([]uint16, n+1)
	sm.rids = make([]uint16, 32)
	sm.scaleDown = 0
	sm.isFixed = false
}

func (sm *subModel) scale() {
	boundBits := uint32(0xFFFF0000)
	for i := 0; i <= sm.n; i++ {
		boundBits |= sm.bounds[i]
	}
	z := uint32(bitsTrailing(boundBits))
	sm.scaleDown = 16 - z
	for i := 0; i <= sm.n; i++ {
		sm.scaled[i] = sm.bounds[i] >> z
	}
	sm.isFixed = sm.bounds[sm.n-1] == 0
}

func bitsTrailing(v uint32) int {
	if v == 0 {
		return 32
	}
	n := 0
	for v&0xf == 0 {
		v >>= 4
		n += 4
	}
	return n + int(trailing4[v&0xf])
}

var trailing4 = [16]uint8{4, 0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0}

func (sm *subModel) decode(d *arithDec) uint32 {
	if sm.n == 0 || sm.isFixed {
		if sm.n == 0 {
			return 0
		}
		return uint32(sm.ids[sm.n-1])
	}
	val := d.decodeShiftScale(sm.scaleDown, sm.scaled, sm.n)
	if int(val) >= len(sm.ids) {
		return 0
	}
	return uint32(sm.ids[val])
}

func (sm *subModel) buildDefault(n int, def uint32) {
	if n <= 0 {
		sm.n = 0
		sm.isFixed = true
		return
	}
	sm.alloc(n)
	sm.bounds[n] = 0x10000
	if int(def) < len(sm.ids) {
		sm.ids[n-1] = uint16(def)
	}
	if int(def) < len(sm.rids) {
		sm.rids[def] = uint16(n - 1)
	}
	sm.scale()
}

func (sm *subModel) extract(dst []uint32) {
	for i := 0; i < sm.n && i < len(dst); i++ {
		ri := uint32(sm.rids[i])
		dst[i] = sm.bounds[ri+1] - sm.bounds[ri]
	}
}

func sumU(a []uint32, n int) uint32 {
	var s uint32
	for i := 0; i < n; i++ {
		s += a[i]
	}
	return s
}

func (sm *subModel) build(arr []uint32) {
	n := len(arr)
	if n == 0 {
		sm.n = 0
		sm.isFixed = true
		return
	}
	sm.alloc(n)
	order := make([]uint16, n)
	for i := range order {
		order[i] = uint16(i)
	}
	sort.Slice(order, func(a, b int) bool {
		if arr[order[a]] != arr[order[b]] {
			return arr[order[a]] < arr[order[b]]
		}
		return order[a] < order[b]
	})
	for i := 0; i < n; i++ {
		sm.bounds[i] = arr[order[i]]
		sm.ids[i] = order[i]
		sm.rids[order[i]] = uint16(i)
	}
	sum := sumU(sm.bounds, n)
	if sum == 0 {
		sm.buildDefault(n, 0)
		return
	}
	prev := sm.bounds[0]
	sm.bounds[0] = 0
	var acc uint32
	for i := 0; i < n; i++ {
		if prev != 0 {
			acc += prev
			prev = sm.bounds[i+1]
			raw := ((uint64(acc) << 16) / uint64(sum)) - uint64(sm.bounds[i])
			diff := int32(uint32(raw))
			diffBits := bitLength(uint32(diff))
			if diff > 0 && diffBits > 5 {
				diff = int32(uint32(diff) & (uint32(31) << (diffBits - 5)))
			}
			sm.bounds[i+1] = sm.bounds[i] + uint32(diff)
			if sm.bounds[i+1] <= sm.bounds[i] {
				sm.bounds[i+1] = sm.bounds[i] + 1
			}
		} else {
			prev = sm.bounds[i+1]
			sm.bounds[i+1] = sm.bounds[i]
		}
	}
	if sm.bounds[n] > 0 {
		sm.bounds[n] = 1 << 16
	}
	sm.scale()
}

func decodeProb(d *arithDec) uint32 {
	b := d.decodeBits(4) + 1
	if b >= 5 {
		return (d.decodeBits(4) | 0x10) << (b - 5)
	}
	return d.decodeBits(b-1) | (1 << (b - 1))
}

func decodeID(d *arithDec, count int) uint32 {
	return d.decodeBits(bitLength(uint32(count - 1)))
}

func (sm *subModel) read(d *arithDec, n int) {
	if n <= 0 {
		sm.n = 0
		sm.isFixed = true
		return
	}
	sm.alloc(n)
	zeros := d.decodeBits(bitLength(uint32(n - 1)))
	for i := uint32(1) + zeros; i < uint32(n); i++ {
		sm.bounds[i] = decodeProb(d) + sm.bounds[i-1]
	}
	sm.bounds[n] = 1 << 16
	for i := zeros; i < uint32(n); i++ {
		id := uint16(decodeID(d, n))
		if int(i) < len(sm.ids) {
			sm.ids[i] = id
		}
		if int(id) < len(sm.rids) {
			sm.rids[id] = uint16(i)
		}
	}
	sm.scale()
}

// corrModel is PreflateCorrectionSubModel<NEG,POS>. NEG == 0 keeps an
// empty negative submodel: decode returns 0 and (int)(NEG-1) == -1, so
// the negative arm always takes the wide decodeBits path.
type corrModel struct {
	negN int
	posN int
	sign subModel
	pos  subModel
	neg  subModel
}

func (c *corrModel) buildDefault(def uint32) {
	signDef := uint32(0)
	if def != 0 {
		signDef = 1
	}
	c.sign.buildDefault(3, signDef)
	posDef := uint32(0)
	if c.posN > 0 {
		posDef = uint32(c.posN - 1)
		if def > 0 && def <= uint32(c.posN) {
			posDef = def - 1
		}
		c.pos.buildDefault(c.posN, posDef)
	}
	if c.negN > 0 {
		c.neg.buildDefault(c.negN, uint32(c.negN-1))
	}
}

func (c *corrModel) read(d *arithDec) {
	c.sign.read(d, 3)
	if c.posN > 0 {
		c.pos.read(d, c.posN)
	}
	if c.negN > 0 {
		c.neg.read(d, c.negN)
	}
}

func (c *corrModel) decode(d *arithDec, ref, minV, maxV uint32) uint32 {
	s := c.sign.decode(d)
	if s == 0 {
		return ref
	}
	if s == 1 {
		diff := int32(c.pos.decode(d))
		lim := int32(c.posN - 1)
		if diff >= lim {
			return ref + d.decodeBits(bitLength(maxV-uint32(c.posN)-ref)) + uint32(c.posN)
		}
		return ref + uint32(diff) + 1
	}
	diff := int32(c.neg.decode(d))
	lim := int32(c.negN - 1)
	if diff >= lim {
		return ref - d.decodeBits(bitLength(ref-uint32(c.negN)-minV)) - uint32(c.negN)
	}
	return ref - uint32(diff) - 1
}

type modelCodec struct {
	nonDefault          subModel
	blockFull, treeFull bool
	tokenFull           bool
	total               uint32
}

func (mc *modelCodec) readFrom(d *arithDec) {
	mc.blockFull = d.decodeBits(1) != 0
	mc.treeFull = d.decodeBits(1) != 0
	mc.tokenFull = d.decodeBits(1) != 0
	mc.total = 0
	if !mc.blockFull {
		mc.total += 3
	}
	if !mc.treeFull {
		mc.total += 11
	}
	if !mc.tokenFull {
		mc.total += 6
	}
	def := d.decodeBits(bitLength(mc.total))
	if mc.total != 0 {
		mc.nonDefault.build([]uint32{def, mc.total - def})
	}
}

func takeSub(sm *subModel, full bool, mc *modelCodec, d *arithDec, n int, def uint32) {
	if full || mc.nonDefault.decode(d) == 0 {
		sm.buildDefault(n, def)
		return
	}
	sm.read(d, n)
}

func takeCorr(c *corrModel, full bool, mc *modelCodec, d *arithDec, def uint32) {
	if full || mc.nonDefault.decode(d) == 0 {
		c.buildDefault(def)
		return
	}
	c.read(d)
}

type blockModel struct {
	blockType subModel
	eob       subModel
	nz        subModel
}

func (m *blockModel) readFrom(mc *modelCodec, d *arithDec) {
	takeSub(&m.blockType, mc.blockFull, mc, d, 3, blockDynamic)
	takeSub(&m.eob, mc.blockFull, mc, d, 2, 0)
	takeSub(&m.nz, mc.blockFull, mc, d, 2, 0)
}

type treeModel struct {
	tcCount subModel
	lCount  subModel
	dCount  subModel
	ldMis   [4]subModel
	ldBase  subModel
	tcBits  corrModel
	ldBits  corrModel
	rep     corrModel
	derived [4]subModel
}

func (m *treeModel) readFrom(mc *modelCodec, d *arithDec) {
	takeSub(&m.tcCount, mc.treeFull, mc, d, 2, 0)
	takeSub(&m.lCount, mc.treeFull, mc, d, 2, 0)
	takeSub(&m.dCount, mc.treeFull, mc, d, 2, 0)
	for i := 0; i < 4; i++ {
		takeSub(&m.ldMis[i], mc.treeFull, mc, d, 2, 0)
	}
	takeSub(&m.ldBase, mc.treeFull, mc, d, 4, 0)
	m.tcBits.negN, m.tcBits.posN = 3, 3
	m.ldBits.negN, m.ldBits.posN = 4, 4
	m.rep.negN, m.rep.posN = 1, 1
	takeCorr(&m.tcBits, mc.treeFull, mc, d, 0)
	takeCorr(&m.ldBits, mc.treeFull, mc, d, 0)
	takeCorr(&m.rep, mc.treeFull, mc, d, 0)
	m.derive()
}

func (m *treeModel) derive() {
	var arr [4]uint32
	var mp [2]uint32
	m.ldBase.extract(arr[:])
	for i := 0; i < 4; i++ {
		m.ldMis[i].extract(mp[:])
		if mp[1] == 0 {
			m.derived[i].buildDefault(4, uint32(i))
			continue
		}
		if mp[0] == 0 {
			mp[1] = 1
		}
		var hit, miss [4]uint32
		var sumhit uint32
		for j := 0; j < 4; j++ {
			hit[j] = arr[j] * mp[0]
			miss[j] = arr[j] * mp[1]
			sumhit += hit[j]
		}
		miss[i] = sumhit - hit[i]
		for j := 0; j < 4; j++ {
			if miss[j] > 0 && miss[j] < 16 {
				miss[j] = 1
			} else {
				miss[j] >>= 4
			}
		}
		m.derived[i].build(miss[:])
	}
}

type tokenModel struct {
	lit     subModel
	ref     subModel
	lenCorr corrModel
	distAL  corrModel
	distO   corrModel
	irr     subModel
}

func (m *tokenModel) readFrom(mc *modelCodec, d *arithDec) {
	takeSub(&m.lit, mc.tokenFull, mc, d, 2, 0)
	takeSub(&m.ref, mc.tokenFull, mc, d, 2, 0)
	m.lenCorr.negN, m.lenCorr.posN = 6, 6
	m.distAL.negN, m.distAL.posN = 0, 3
	m.distO.negN, m.distO.posN = 0, 3
	takeCorr(&m.lenCorr, mc.tokenFull, mc, d, 0)
	takeCorr(&m.distAL, mc.tokenFull, mc, d, 0)
	takeCorr(&m.distO, mc.tokenFull, mc, d, 0)
	takeSub(&m.irr, mc.tokenFull, mc, d, 2, 0)
}

type predModel struct {
	block blockModel
	tree  treeModel
	token tokenModel
}

func (m *predModel) readFrom(mc *modelCodec, d *arithDec) {
	m.block.readFrom(mc, d)
	m.tree.readFrom(mc, d)
	m.token.readFrom(mc, d)
}
