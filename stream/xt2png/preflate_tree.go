package xt2png

// Tree predictor and block writer. Stored and static blocks do not advance
// the tree input; only dynamic blocks consume bytes via token statistics.

type freqIdx struct {
	freq uint32
	idx  uint32
}

type treeNode struct {
	parent uint32
	idx    uint32
}

func pqSmaller(a, b freqIdx, depth []byte) bool {
	return a.freq < b.freq || (a.freq == b.freq && depth[a.idx] <= depth[b.idx])
}

func pqDown(ptr []freqIdx, index, n int, depth []byte) {
	k := index
	v := ptr[k]
	j := k*2 + 1
	for j < n {
		if j+1 < n && pqSmaller(ptr[j+1], ptr[j], depth) {
			j++
		}
		if pqSmaller(v, ptr[j], depth) {
			break
		}
		ptr[k] = ptr[j]
		k = j
		j = k*2 + 1
	}
	ptr[k] = v
}

func pqMake(ptr []freqIdx, depth []byte) {
	n := len(ptr)
	for i := (n-1)/2 + 1; i > 0; i-- {
		pqDown(ptr, i-1, n, depth)
	}
}

func pqRemove(ptr []freqIdx, depth []byte) (freqIdx, []freqIdx) {
	result := ptr[0]
	n := len(ptr) - 1
	ptr[0] = ptr[n]
	ptr = ptr[:n]
	pqDown(ptr, 0, len(ptr), depth)
	return result, ptr
}

func calcBitLengths(sym []byte, freq []uint32, symCount, maxBits, minMax int) int {
	const capN = litLenCount*2 + 1
	toSort := make([]freqIdx, 0, symCount)
	nodeBit := make([]byte, capN)
	nodeDepth := make([]byte, capN)
	maxCode := 0
	for i := 0; i < symCount; i++ {
		if freq[i] != 0 {
			maxCode = i
			toSort = append(toSort, freqIdx{freq: freq[i], idx: uint32(i)})
		}
	}
	if len(toSort) < 2 {
		for i := 0; i < symCount && i < len(sym); i++ {
			sym[i] = 0
		}
		if maxCode < len(sym) {
			sym[maxCode] = 1
		}
		if maxCode < 2 {
			maxCode++
			if maxCode < len(sym) {
				sym[maxCode] = 1
			}
		} else if len(sym) > 0 {
			sym[0] = 1
		}
		if minMax > maxCode+1 {
			return minMax
		}
		return maxCode + 1
	}
	pqMake(toSort, nodeDepth)
	nodes := make([]treeNode, 0, symCount*2)
	nodeID := uint32(symCount)
	for len(toSort) > 1 {
		var least freqIdx
		least, toSort = pqRemove(toSort, nodeDepth)
		second := toSort[0]
		toSort[0] = freqIdx{freq: least.freq + second.freq, idx: nodeID}
		nodes = append(nodes, treeNode{nodeID, least.idx}, treeNode{nodeID, second.idx})
		d := nodeDepth[least.idx]
		if nodeDepth[second.idx] > d {
			d = nodeDepth[second.idx]
		}
		nodeDepth[nodeID] = d + 1
		pqDown(toSort, 0, len(toSort), nodeDepth)
		nodeID++
	}
	overflow := uint32(0)
	var blCount [16]uint32
	for i := len(nodes); i > 0; {
		i--
		newLen := int(nodeBit[nodes[i].parent]) + 1
		if newLen > maxBits {
			newLen = maxBits
			overflow++
		}
		idx := int(nodes[i].idx)
		nodeBit[idx] = byte(newLen)
		if idx < symCount {
			blCount[newLen]++
		}
	}
	if overflow != 0 {
		for {
			bitsN := maxBits - 1
			for blCount[bitsN] == 0 {
				bitsN--
				if bitsN <= 0 {
					break
				}
			}
			if bitsN <= 0 {
				break
			}
			blCount[bitsN]--
			blCount[bitsN+1] += 2
			blCount[maxBits]--
			overflow -= 2
			if overflow == 0 {
				break
			}
		}
		bitsN := maxBits
		for i := len(nodes); i > 0; {
			i--
			idx := int(nodes[i].idx)
			if idx >= symCount {
				continue
			}
			for blCount[bitsN] == 0 {
				bitsN--
				if bitsN <= 0 {
					break
				}
			}
			nodeBit[idx] = byte(bitsN)
			if bitsN > 0 {
				blCount[bitsN]--
			}
		}
	}
	n := symCount
	if n > len(sym) {
		n = len(sym)
	}
	copy(sym[:n], nodeBit[:n])
	if minMax > maxCode+1 {
		return minMax
	}
	return maxCode + 1
}

func predictType(sym []byte, off, count int, first bool) int {
	if off < 0 || off >= len(sym) {
		return 0
	}
	code := sym[off]
	if code == 0 {
		cur := 1
		maxC := count
		if maxC > 11 {
			maxC = 11
		}
		for cur < maxC && off+cur < len(sym) && sym[off+cur] == 0 {
			cur++
		}
		if cur >= 11 {
			return 3
		}
		if cur >= 3 {
			return 2
		}
		return 0
	}
	if !first && off > 0 && code == sym[off-1] {
		cur := 1
		maxC := count
		if maxC > 3 {
			maxC = 3
		}
		for cur < maxC && off+cur < len(sym) && sym[off+cur] == code {
			cur++
		}
		if cur >= 3 {
			return 1
		}
	}
	return 0
}

func predictData(sym []byte, off, typ, count int) int {
	if off < 0 || off >= len(sym) {
		return 0
	}
	code := int(sym[off])
	switch typ {
	case 1:
		cur := 3
		maxC := count
		if maxC > 6 {
			maxC = 6
		}
		for cur < maxC && off+cur < len(sym) && int(sym[off+cur]) == code {
			cur++
		}
		return cur
	case 2, 3:
		cur := 3
		maxC := 10
		if typ == 3 {
			cur = 11
			maxC = 138
		}
		if count < maxC {
			maxC = count
		}
		for cur < maxC && off+cur < len(sym) && sym[off+cur] == 0 {
			cur++
		}
		return cur
	default:
		return code
	}
}

var (
	repMin  = [4]uint32{0, 3, 3, 11}
	repSpan = [4]uint32{0, 3, 7, 127}
	repOff  = [3]uint32{3, 3, 11}
	repBits = [3]uint32{2, 3, 7}
)

type treePred struct {
	data []byte
	pos  int
	fail bool
}

func newTreePred(data []byte, offset int) *treePred {
	if offset < 0 {
		offset = 0
	}
	if offset > len(data) {
		offset = len(data)
	}
	return &treePred{data: data, pos: offset}
}

func dcode(dist uint32) int {
	if dist == 0 {
		return 0
	}
	var idx uint32
	if dist <= 256 {
		idx = dist - 1
	} else {
		idx = 256 + ((dist - 1) >> 7)
	}
	if int(idx) >= len(distCode) {
		return 0
	}
	return int(distCode[idx])
}

func lcode(length uint32) int {
	if length < minMatch {
		return 0
	}
	idx := int(length) - minMatch
	if idx >= len(lengthCode) {
		return 0
	}
	return int(lengthCode[idx])
}

func (tp *treePred) collect(block *pfBlock, lc, dc []uint32) {
	for i := range lc {
		lc[i] = 0
	}
	for i := range dc {
		dc[i] = 0
	}
	for _, t := range block.tokens {
		if t.length == 1 {
			if tp.pos < 0 || tp.pos >= len(tp.data) {
				tp.fail = true
				return
			}
			lc[tp.data[tp.pos]]++
			tp.pos++
			continue
		}
		lc[nonLenCount+lcode(uint32(t.length))]++
		dc[dcode(uint32(t.dist))]++
		tp.pos += int(t.length)
	}
	lc[litSymCount] = 1
}

func (tp *treePred) reconstruct(m *predModel, d *arithDec, freq []uint32, target, sym []byte, lCount, dCount int) int {
	for i := range freq {
		freq[i] = 0
	}
	osize := 0
	count1, count2 := lCount, dCount
	off := 0
	first := true
	for count1+count2 > 0 {
		kind := predictType(sym, off, count1, first)
		if kind >= 0 && kind < 4 {
			nt := m.tree.derived[kind].decode(d)
			switch nt {
			case 0, 1, 2, 3:
				kind = int(nt)
			}
		}
		raw := predictData(sym, off, kind, count1)
		first = false
		var data byte
		if kind != 0 {
			data = byte(m.tree.rep.decode(d, uint32(raw), repMin[kind], repMin[kind]+repSpan[kind]))
		} else {
			data = byte(m.tree.ldBits.decode(d, uint32(raw), 0, 15))
		}
		l := 1
		if kind != 0 {
			if kind+15 < len(freq) {
				freq[kind+15]++
			}
			l = int(data)
			if osize+2 > len(target) {
				tp.fail = true
				break
			}
			target[osize] = byte(kind + 15)
			target[osize+1] = data
			osize += 2
		} else {
			if int(data) < len(freq) {
				freq[data]++
			}
			if osize >= len(target) {
				tp.fail = true
				break
			}
			target[osize] = data
			osize++
		}
		off += l
		if count1 > l {
			count1 -= l
		} else {
			count1 += count2
			count2 = 0
			first = true
			if count1 >= l {
				count1 -= l
			} else {
				tp.fail = true
				break
			}
		}
	}
	if count1+count2 != 0 {
		tp.fail = true
	}
	if tp.fail {
		return 0
	}
	return osize
}

func (tp *treePred) decodeBlock(block *pfBlock, m *predModel, d *arithDec) bool {
	if block.kind != blockDynamic {
		return true
	}
	lc := make([]uint32, litLenCount)
	dc := make([]uint32, distSymCount)
	tp.collect(block, lc, dc)
	if tp.fail {
		return false
	}
	bitLen := make([]byte, litDistCount+16)
	nlen := calcBitLengths(bitLen, lc, litLenCount, 15, nonLenCount)
	if m.tree.lCount.decode(d) != 0 {
		nlen = int(d.decodeBits(5)) + nonLenCount
	}
	block.nlen = nlen
	if nlen < 0 || nlen > len(bitLen) {
		tp.fail = true
		return false
	}
	ndist := calcBitLengths(bitLen[nlen:], dc, distSymCount, 15, 0)
	if m.tree.dCount.decode(d) != 0 {
		ndist = int(d.decodeBits(5))
	}
	block.ndist = ndist
	freq := make([]uint32, codeTreeCount)
	comp := make([]byte, litDistCount)
	osize := tp.reconstruct(m, d, freq, comp, bitLen, nlen, ndist)
	if tp.fail {
		return false
	}
	simple := make([]byte, codeTreeCount)
	calcBitLengths(simple, freq, codeTreeCount, 7, 0)
	nc := codeTreeCount
	for nc > 4 && simple[treeCodeOrder[nc-1]] == 0 {
		nc--
	}
	if m.tree.tcCount.decode(d) != 0 {
		nc = int(d.decodeBits(4)) + 4
	}
	block.ncode = nc
	if nc < 0 || nc > codeTreeCount {
		tp.fail = true
		return false
	}
	block.trees = make([]byte, 0, nc+osize)
	for i := 0; i < nc; i++ {
		pred := uint32(simple[treeCodeOrder[i]])
		block.trees = append(block.trees, byte(m.tree.tcBits.decode(d, pred, 0, 7)))
	}
	block.trees = append(block.trees, comp[:osize]...)
	return true
}

func writeBlock(w *bitWriter, b pfBlock, data []byte, pos *int, last bool) bool {
	var fin uint32
	if last {
		fin = 1
	}
	w.put(fin, 1)
	switch b.kind {
	case blockDynamic:
		if b.ncode < 4 || b.ncode > codeTreeCount || len(b.trees) < b.ncode ||
			b.nlen < nonLenCount || b.nlen > litLenCount ||
			b.ndist < 1 || b.ndist > distSymCount {
			return false
		}
		w.put(2, 2)
		w.put(uint32(b.nlen-nonLenCount), 5)
		w.put(uint32(b.ndist-1), 5)
		w.put(uint32(b.ncode-4), 4)
		lit, dist, ok := writeDyn(w, b)
		if !ok {
			return false
		}
		return writeTokens(w, b.tokens, data, pos, lit, dist)
	case blockStatic:
		w.put(1, 2)
		if staticLitEnc.err || staticDistEnc.err {
			return false
		}
		return writeTokens(w, b.tokens, data, pos, staticLitEnc, staticDistEnc)
	case blockStored:
		w.put(0, 2)
		w.put(uint32(b.padV), uint32(b.padN))
		w.fillByte()
		w.put(b.uncLen, 16)
		w.put(^b.uncLen, 16)
		if *pos < 0 || *pos > len(data) || int(b.uncLen) > len(data)-*pos {
			return false
		}
		w.putBytes(data[*pos : *pos+int(b.uncLen)])
		*pos += int(b.uncLen)
		return true
	default:
		return true
	}
}

func writeDyn(w *bitWriter, b pfBlock) (*huffEnc, *huffEnc, bool) {
	var tcBits [codeTreeCount]byte
	var ld [litDistCount]byte
	for i := 0; i < b.ncode; i++ {
		tc := uint32(b.trees[i])
		w.put(tc, 3)
		tcBits[treeCodeOrder[i]] = b.trees[i]
	}
	tcEnc := newHuff(tcBits[:], true)
	if tcEnc.err {
		return nil, nil, false
	}
	o := 0
	maxo := b.nlen + b.ndist
	for i := b.ncode; i < len(b.trees); {
		code := b.trees[i]
		i++
		if code > 18 {
			return nil, nil, false
		}
		if !tcEnc.encode(w, uint(code)) {
			return nil, nil, false
		}
		if code < 16 {
			if o >= maxo {
				return nil, nil, false
			}
			ld[o] = code
			o++
			continue
		}
		if i >= len(b.trees) || (code == 16 && o == 0) {
			return nil, nil, false
		}
		ln := b.trees[i]
		i++
		var tocopy byte
		if code == 16 {
			tocopy = ld[o-1]
		}
		delta := int32(ln) - int32(repOff[code-16])
		w.put(uint32(delta), repBits[code-16])
		if o+int(ln) > maxo {
			return nil, nil, false
		}
		for k := 0; k < int(ln); k++ {
			ld[o+k] = tocopy
		}
		o += int(ln)
	}
	if o != maxo || ld[litSymCount] == 0 {
		return nil, nil, false
	}
	lit := newHuff(ld[:b.nlen], true)
	if lit.err {
		return nil, nil, false
	}
	dist := newHuff(ld[b.nlen:b.nlen+b.ndist], true)
	if dist.err {
		return nil, nil, false
	}
	return lit, dist, true
}

func writeTokens(w *bitWriter, tokens []pfToken, data []byte, pos *int, lit, dist *huffEnc) bool {
	for _, t := range tokens {
		if t.length == 1 {
			if *pos < 0 || *pos >= len(data) {
				return false
			}
			if !lit.encode(w, uint(data[*pos])) {
				return false
			}
			*pos++
			continue
		}
		if t.irreg {
			if !lit.encode(w, litLenCount-2) {
				return false
			}
			w.put(31, 5)
		} else {
			lc := lcode(uint32(t.length))
			if !lit.encode(w, uint(nonLenCount+lc)) {
				return false
			}
			extra := uint32(lengthExtra[lc])
			if extra != 0 {
				base := uint32(lengthBase[lc])
				w.put(uint32(t.length)-minMatch-base, extra)
			}
		}
		dc := dcode(uint32(t.dist))
		if !dist.encode(w, uint(dc)) {
			return false
		}
		extra := uint32(distExtra[dc])
		if extra != 0 {
			w.put(uint32(t.dist)-1-uint32(distBase[dc]), extra)
		}
		*pos += int(t.length)
	}
	return lit.encode(w, litSymCount)
}
