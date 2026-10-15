package xt2png

const (
	blockStored = iota
	blockDynamic
	blockStatic

	minMatch     = 3
	maxMatch     = 258
	minLookahead = maxMatch + minMatch + 1
	tooFar       = 4096

	litSymCount   = 256
	nonLenCount   = 257
	litLenCount   = 286
	distSymCount  = 30
	litDistCount  = litLenCount + distSymCount
	codeTreeCount = 19
)

type parserCfg struct {
	good, lazy, nice, chain uint16
}

var fastCfg = [3]parserCfg{
	{4, 4, 8, 4},
	{4, 5, 16, 8},
	{4, 6, 32, 32},
}

var slowCfg = [6]parserCfg{
	{4, 4, 16, 16},
	{8, 16, 32, 32},
	{8, 16, 128, 128},
	{8, 32, 128, 256},
	{32, 128, 258, 1024},
	{32, 258, 258, 4096},
}

type pfParams struct {
	comp, mem, wbits uint8
	zlib             bool
	veryFar          bool
	toStart          bool
	log2             uint8
}

func (p pfParams) fast() bool { return p.comp >= 1 && p.comp <= 3 }
func (p pfParams) slow() bool { return p.comp >= 4 && p.comp <= 9 }

func (p pfParams) cfg() parserCfg {
	if p.fast() {
		return fastCfg[p.comp-1]
	}
	if p.slow() {
		return slowCfg[p.comp-4]
	}
	return slowCfg[5]
}

type pfToken struct {
	length uint16
	irreg  bool
	dist   uint16
}

func litToken() pfToken { return pfToken{length: 1} }

func refTok(length, dist uint32) pfToken {
	return pfToken{length: uint16(length) & 0x1ff, dist: uint16(dist)}
}

type pfBlock struct {
	kind        int
	uncLen      uint32
	nlen, ndist int
	ncode       int
	padN, padV  uint8
	trees       []byte
	tokens      []pfToken
}

type predState struct {
	hash      *hashChain
	seq       *seqChain
	window    uint32
	maxTokens uint32
	cfg       parserCfg
}

func newState(h *hashChain, s *seqChain, p pfParams) predState {
	return predState{
		hash:      h,
		seq:       s,
		window:    1 << p.wbits,
		maxTokens: (1 << (6 + p.mem)) - 1,
		cfg:       p.cfg(),
	}
}

func (s *predState) avail() uint32 {
	if s.hash.pos > uint32(len(s.hash.data)) {
		return 0
	}
	return uint32(len(s.hash.data)) - s.hash.pos
}

func (s *predState) curHash() uint32 {
	return s.hash.next1(s.hash.data[s.hash.pos+2])
}

func (s *predState) curHashNext() uint32 {
	return s.hash.next2(s.hash.data[s.hash.pos+2], s.hash.data[s.hash.pos+3])
}

type matchHelp struct {
	start             uint32
	maxLen            uint32
	hop0, hop1        uint32
	maxChain, niceLen uint32
}

func (s *predState) help(prevLen, start uint32, veryFar, toStart bool, maxDepth uint32) (matchHelp, bool) {
	var h matchHelp
	if start > uint32(len(s.hash.data)) {
		return h, false
	}
	h.maxLen = uint32(len(s.hash.data)) - start
	if h.maxLen > maxMatch {
		h.maxLen = maxMatch
	}
	need := prevLen + 1
	if need < minMatch {
		need = minMatch
	}
	if h.maxLen < need {
		return h, false
	}
	h.start = start
	maxToStart := start - 1
	if toStart {
		maxToStart = start
	}
	if veryFar {
		h.hop0 = maxToStart
		if s.window < h.hop0 {
			h.hop0 = s.window
		}
		h.hop1 = h.hop0
	} else {
		maxDist := s.window - minLookahead
		h.hop0 = maxToStart
		if maxDist < h.hop0 {
			h.hop0 = maxDist
		}
		hop1cap := maxDist - 1
		h.hop1 = maxToStart
		if hop1cap < h.hop1 {
			h.hop1 = hop1cap
		}
	}
	if maxDepth > 0 {
		h.maxChain = maxDepth
		h.niceLen = h.maxLen
	} else {
		h.maxChain = uint32(s.cfg.chain)
		h.niceLen = uint32(s.cfg.nice)
		if h.niceLen > h.maxLen {
			h.niceLen = h.maxLen
		}
		if prevLen >= uint32(s.cfg.good) {
			h.maxChain >>= 2
		}
	}
	return h, true
}

func prefixCompare(data []byte, a, b, best, max int) int {
	if a < 0 || b < 0 || best < 0 || max < 1 {
		return 0
	}
	if a+best >= len(data) || b+best >= len(data) {
		return 0
	}
	if data[a+best] != data[b+best] {
		return 0
	}
	if a+2 >= len(data) || b+2 >= len(data) || data[a] != data[b] || data[a+1] != data[b+1] || data[a+2] != data[b+2] {
		return 0
	}
	limit := max
	if a+limit > len(data) {
		limit = len(data) - a
	}
	if b+limit > len(data) {
		limit = len(data) - b
	}
	scan := 3
	for scan < limit && data[a+scan] == data[b+scan] {
		scan++
	}
	return scan
}

func suffixCompare(data []byte, a, b, best, max int) int {
	if a < 0 || b < 0 || best < 0 || a+best >= len(data) || b+best >= len(data) {
		return 0
	}
	if data[a+best] != data[b+best] {
		return 0
	}
	limit := max
	if limit < 0 {
		return 0
	}
	if a+limit > len(data) {
		limit = len(data) - a
	}
	if b+limit > len(data) {
		limit = len(data) - b
	}
	n := 0
	for n < limit && data[a+n] == data[b+n] {
		n++
	}
	return n
}

func (s *predState) match(hashHead, prevLen, offset uint32, veryFar, toStart bool, maxDepth uint32) pfToken {
	best := pfToken{}
	h, ok := s.help(prevLen, s.hash.pos+offset, veryFar, toStart, maxDepth)
	if !ok {
		return best
	}
	it := s.hash.iterNode(hashHead, h.start, h.hop1)
	if it.curDist > h.hop0 {
		return best
	}
	data := s.hash.data
	in := int(s.hash.pos + offset)
	bestLen := prevLen
	for {
		ml := prefixCompare(data, in-int(it.curDist), in, int(bestLen), int(h.maxLen))
		if uint32(ml) > bestLen {
			bestLen = uint32(ml)
			best = refTok(bestLen, it.curDist)
			if bestLen >= h.niceLen {
				break
			}
		}
		if !it.next() || h.maxChain <= 1 {
			break
		}
		h.maxChain--
	}
	return best
}

func (s *predState) seqMatch(startPos, hashHead, prevLen uint32, veryFar, toStart bool, maxDepth uint32) pfToken {
	best := pfToken{}
	h, ok := s.help(prevLen, startPos, veryFar, toStart, maxDepth)
	if !ok {
		return best
	}
	chainIt := s.seq.iterate(startPos)
	if !chainIt.ok() {
		return best
	}
	curSeq := uint32(s.seq.seqLen(startPos))
	if curSeq > h.maxLen {
		curSeq = h.maxLen
	}
	curMax := h.hop1
	bestLen := prevLen
	if curSeq < minMatch {
		sub := uint32(chainIt.length()) - chainIt.dist()
		curSeq = sub
		if curSeq > h.maxLen {
			curSeq = h.maxLen
		}
		if curSeq > prevLen && h.hop0 >= 1 {
			bestLen = curSeq
			best = refTok(curSeq, 1)
		}
		if bestLen >= h.niceLen || !chainIt.next() {
			return best
		}
		if chainIt.dist() > h.hop1+uint32(chainIt.length())-minMatch {
			return best
		}
	} else {
		minOff := uint32(chainIt.length()) - minMatch
		if chainIt.dist() > h.hop1+minOff {
			if chainIt.dist() > h.hop0+minOff {
				return best
			}
			latest := h.start - chainIt.dist() + minOff
			depth := s.hash.relDepth(latest, hashHead)
			if depth == 0 {
				curMax = h.hop0
			}
		}
	}
	inputAt := int(s.hash.pos)
	in := inputAt + int(startPos) - inputAt
	bestSeq := curSeq
	if bestLen < bestSeq {
		bestSeq = bestLen
	}
	data := s.hash.data
	for {
		if uint32(chainIt.length()) >= bestSeq {
			oldBest := bestSeq
			bestSeq = curSeq
			if uint32(chainIt.length()) < bestSeq {
				bestSeq = uint32(chainIt.length())
			}
			if h.niceLen < bestSeq {
				bestSeq = h.niceLen
			}
			bestDist := chainIt.dist() - uint32(chainIt.length()) + bestSeq
			errn := uint32(0)
			if bestDist > curMax {
				errn = bestDist - curMax
				if errn > uint32(chainIt.length())-minMatch {
					break
				}
			}
			depth := s.hash.relDepth(h.start-bestDist+errn, hashHead)
			if depth >= h.maxChain {
				errn += depth - h.maxChain + 1
				if errn > uint32(chainIt.length())-minMatch {
					break
				}
			}
			if errn != 0 {
				floor := oldBest
				if floor < minMatch-1 {
					floor = minMatch - 1
				}
				if bestSeq > floor+errn {
					best = refTok(bestSeq-errn, bestDist-errn)
				}
				break
			}
			if bestSeq == h.maxLen {
				best = refTok(bestSeq, bestDist)
				break
			}
			base := bestSeq
			absIn := in + int(base)
			absM := in - int(bestDist) + int(base)
			extraBest := int(bestLen)
			if int(bestSeq) > extraBest {
				extraBest = int(bestSeq)
			}
			ml := int(base) + suffixCompare(data, absM, absIn, extraBest-int(base), int(h.maxLen-base))
			if uint32(ml) > bestLen {
				bestLen = uint32(ml)
				best = refTok(bestLen, bestDist)
				if bestLen >= h.niceLen {
					break
				}
			}
			curMax = h.hop1
		}
		if !chainIt.next() {
			break
		}
	}
	return best
}

func (s *predState) firstMatch(length uint32) uint32 {
	maxLen := s.avail()
	if maxLen > maxMatch {
		maxLen = maxMatch
	}
	need := length
	if need < minMatch {
		need = minMatch
	}
	if maxLen < need {
		return 0
	}
	cur := s.hash.pos
	curMax := cur
	if s.window < curMax {
		curMax = s.window
	}
	it := s.hash.iterNode(s.hash.getHead(s.curHash()), cur, curMax)
	if it.curDist > curMax {
		return 0
	}
	data := s.hash.data
	in := int(cur)
	for {
		ml := prefixCompare(data, in-int(it.curDist), in, int(length)-1, int(length))
		if uint32(ml) >= length {
			return it.curDist
		}
		if !it.next() {
			return 0
		}
	}
}

func (s *predState) hopMatch(tok pfToken, hops uint32) uint32 {
	if hops == 0 {
		return uint32(tok.dist)
	}
	cur := s.hash.pos
	maxLen := s.avail()
	if maxLen > maxMatch {
		maxLen = maxMatch
	}
	if maxLen < uint32(tok.length) {
		return 0
	}
	curMax := cur
	if s.window < curMax {
		curMax = s.window
	}
	it := s.hash.iterNode(cur-uint32(tok.dist)-s.hash.totalShift, cur, curMax)
	if it.curDist > curMax {
		return 0
	}
	data := s.hash.data
	in := int(cur)
	best := int(tok.length)
	for todo := hops; todo > 0; {
		if !it.next() {
			break
		}
		ml := prefixCompare(data, in-int(it.curDist), in-int(tok.dist), best-1, best)
		if ml >= best {
			todo--
			if todo == 0 {
				return it.curDist
			}
		}
	}
	return 0
}

type tokenPred struct {
	st      predState
	hash    *hashChain
	seq     *seqChain
	params  pfParams
	fail    bool
	fast    bool
	prevLen uint32
	pending pfToken
	count   uint32
}

func newTokenPred(p pfParams, data []byte, offset int) *tokenPred {
	h := newHash(data, p.mem)
	s := newSeq(data)
	tp := &tokenPred{
		st:     newState(h, s, p),
		hash:   h,
		seq:    s,
		params: p,
		fast:   p.fast(),
	}
	if tp.st.avail() >= 2 {
		h.updateRunning(h.data[h.pos])
		h.updateRunning(h.data[h.pos+1])
		s.updateSeq(2)
	}
	if offset < 0 || !h.updateHash(uint32(offset)) {
		tp.fail = true
		return tp
	}
	s.updateSeq(uint32(offset))
	return tp
}

func (tp *tokenPred) predictEOB() bool {
	return tp.st.avail() == 0 || tp.count == tp.st.maxTokens
}

func (tp *tokenPred) commit(t pfToken) bool {
	n := uint32(t.length)
	if tp.fast && n > uint32(tp.st.cfg.lazy) {
		if !tp.hash.skipHash(n) {
			return false
		}
	} else if !tp.hash.updateHash(n) {
		return false
	}
	tp.seq.updateSeq(n)
	return true
}

func (tp *tokenPred) depth1() uint32 {
	if tp.params.zlib {
		return 0
	}
	return 1 << tp.params.log2
}

func (tp *tokenPred) depth2() uint32 {
	if tp.params.zlib {
		return 0
	}
	return 2 << tp.params.log2
}

func (tp *tokenPred) predictToken() pfToken {
	if tp.st.hash.pos == 0 || tp.st.avail() < minMatch {
		return litToken()
	}
	hash := tp.st.curHash()
	var match pfToken
	if tp.pending.length > 1 {
		match = tp.pending
	} else {
		head := tp.hash.getHead(hash)
		if !tp.fast && tp.seq.valid(tp.st.hash.pos) {
			match = tp.st.seqMatch(tp.st.hash.pos, head, tp.prevLen, tp.params.veryFar, tp.params.toStart, tp.depth1())
		} else {
			match = tp.st.match(head, tp.prevLen, 0, tp.params.veryFar, tp.params.toStart, tp.depth1())
		}
	}
	tp.prevLen = 0
	tp.pending = pfToken{}
	if match.length < minMatch {
		return litToken()
	}
	if tp.fast {
		return match
	}
	if match.length == 3 && match.dist > tooFar {
		return litToken()
	}
	if uint32(match.length) < uint32(tp.st.cfg.lazy) && tp.st.avail() >= uint32(match.length)+2 {
		hashNext := tp.st.curHashNext()
		headNext := tp.hash.getHead(hashNext)
		var matchNext pfToken
		if !tp.fast && tp.seq.valid(tp.st.hash.pos+1) {
			matchNext = tp.st.seqMatch(tp.st.hash.pos+1, headNext, uint32(match.length), tp.params.veryFar, tp.params.toStart, tp.depth2())
		} else {
			matchNext = tp.st.match(headNext, uint32(match.length), 1, tp.params.veryFar, tp.params.toStart, tp.depth2())
			if ((hashNext ^ hash) & uint32(tp.hash.hashMask)) == 0 {
				maxSize := tp.st.avail() - 1
				if maxSize > maxMatch {
					maxSize = maxMatch
				}
				c := tp.hash.data[tp.hash.pos:]
				b := c[0]
				var rle uint32
				for rle < maxSize && c[1+rle] == b {
					rle++
				}
				if rle > uint32(match.length) && rle >= uint32(matchNext.length) {
					matchNext.length = uint16(rle) & 0x1ff
					matchNext.dist = 1
				}
			}
		}
		if matchNext.length > match.length {
			tp.prevLen = uint32(match.length)
			tp.pending = matchNext
			if !tp.params.zlib {
				tp.prevLen = 0
				tp.pending = pfToken{}
			}
			return litToken()
		}
	}
	return match
}

func (tp *tokenPred) repredictRef() (pfToken, bool) {
	if tp.st.hash.pos == 0 || tp.st.avail() < minMatch {
		return pfToken{}, false
	}
	head := tp.hash.getHead(tp.st.curHash())
	match := tp.st.match(head, 0, 0, tp.params.veryFar, tp.params.toStart, 2<<tp.params.log2)
	tp.prevLen = 0
	tp.pending = pfToken{}
	if match.length < minMatch {
		return pfToken{}, false
	}
	return match, true
}

func (tp *tokenPred) decodeBlock(m *predModel, d *arithDec) (pfBlock, bool) {
	var block pfBlock
	tp.count = 0
	tp.prevLen = 0
	tp.pending = pfToken{}
	bt := m.block.blockType.decode(d)
	switch bt {
	case blockStored:
		block.kind = blockStored
		block.uncLen = d.decodeBits(16)
		if m.block.nz.decode(d) != 0 {
			block.padN = uint8(d.decodeBits(3))
			if block.padN > 0 {
				extra := d.decodeBits(uint32(block.padN - 1))
				block.padV = uint8((uint32(1) << (block.padN - 1)) + extra)
			}
		}
		if !tp.hash.updateHash(block.uncLen) {
			tp.fail = true
			return pfBlock{}, false
		}
		tp.seq.updateSeq(block.uncLen)
		return block, true
	case blockStatic:
		block.kind = blockStatic
	case blockDynamic:
		block.kind = blockDynamic
	}
	var blockSize uint32
	checkEOB := true
	if m.block.eob.decode(d) != 0 {
		bitsN := d.decodeBits(5)
		if bitsN >= 2 {
			blockSize = d.decodeBits(bitsN)
		} else {
			blockSize = bitsN
		}
		checkEOB = false
	}
	for (checkEOB && !tp.predictEOB()) || (!checkEOB && tp.count < blockSize) {
		pred := tp.predictToken()
		if pred.length == 1 {
			if m.token.lit.decode(d) == 0 {
				block.tokens = append(block.tokens, pred)
				if !tp.commit(pred) {
					tp.fail = true
					return pfBlock{}, false
				}
				tp.count++
				continue
			}
			var ok bool
			pred, ok = tp.repredictRef()
			if !ok {
				tp.fail = true
				return pfBlock{}, false
			}
		} else if m.token.ref.decode(d) != 0 {
			pred = litToken()
			block.tokens = append(block.tokens, pred)
			if !tp.commit(pred) {
				tp.fail = true
				return pfBlock{}, false
			}
			tp.count++
			continue
		}
		newLen := m.token.lenCorr.decode(d, uint32(pred.length), 3, 258)
		if newLen != uint32(pred.length) {
			hops := m.token.distAL.decode(d, 0, 0, 32767)
			pred.length = uint16(newLen) & 0x1ff
			pred.dist = uint16(tp.st.firstMatch(uint32(pred.length)))
			if hops != 0 {
				pred.dist = uint16(tp.st.hopMatch(pred, hops))
			}
			if pred.length < 3 || pred.length > 258 || pred.dist == 0 {
				tp.fail = true
				return pfBlock{}, false
			}
		} else {
			hops := m.token.distO.decode(d, 0, 0, 32767)
			if hops != 0 {
				pred.dist = uint16(tp.st.hopMatch(pred, hops))
				if pred.dist == 0 {
					tp.fail = true
					return pfBlock{}, false
				}
			}
		}
		if pred.length == 258 {
			pred.irreg = m.token.irr.decode(d) != 0
		}
		block.tokens = append(block.tokens, pred)
		if !tp.commit(pred) {
			tp.fail = true
			return pfBlock{}, false
		}
		tp.count++
	}
	return block, !tp.fail
}

func (tp *tokenPred) decodeEOF(m *predModel, d *arithDec) bool {
	if tp.st.avail() == 0 {
		return d.decodeBits(1) == 0
	}
	return false
}

func (tp *tokenPred) inputEOF() bool { return tp.st.avail() == 0 }
