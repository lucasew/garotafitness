package jpgrec

import (
	"errors"
)

// Bit reader, ANS, Huffman, and binary arithmetic translated from
// brunsli c/dec (64-bit little-endian BrunsliBitReader).

const (
	ansLogTabSize = 10
	ansTabSize    = 1 << ansLogTabSize
	ansSignature  = 0x13

	brBitSlack = 40
)

const (
	brStageSignature = iota
	brStageHeader
	brStageFallback
	brStageSection
	brStageBody
	brStageDone
	brStageError
)

type huffCode struct {
	bits  uint8
	value uint16
}

type brBitReader struct {
	val       uint64
	src       []byte
	pos       int
	available int
	bitPos    uint32
	tail      [brBitSlack * 2]byte
}

func brBitMask(n uint32) uint32 {
	if n == 0 {
		return 0
	}
	if n >= 32 {
		return 0xFFFFFFFF
	}
	return ^(uint32(0xFFFFFFFF) << n)
}

func newBrBitReader(data []byte) *brBitReader {
	// 64-bit little-endian window. The C reader copies the tail and
	// redirects into zero padding; a zero-padded buffer yields the same bytes.
	padded := make([]byte, len(data)+brBitSlack*2)
	copy(padded, data)
	br := &brBitReader{src: padded, available: len(data)}
	for i := 0; i < 8; i++ {
		br.val |= uint64(br.src[br.pos]) << (uint(i) * 8)
		br.pos++
		br.available--
	}
	return br
}

func (br *brBitReader) fill() {
	if br.bitPos >= 32 {
		br.val >>= 32
		br.bitPos ^= 32
		b0 := br.src[br.pos]
		b1 := br.src[br.pos+1]
		b2 := br.src[br.pos+2]
		b3 := br.src[br.pos+3]
		w := uint32(b0) | uint32(b1)<<8 | uint32(b2)<<16 | uint32(b3)<<24
		br.val |= uint64(w) << 32
		br.available -= 4
		br.pos += 4
	}
}

func (br *brBitReader) readMore() bool {
	return br.available > -8
}

func (br *brBitReader) readBits(n uint32) uint32 {
	br.fill()
	val := uint32(br.val>>br.bitPos) & brBitMask(n)
	br.bitPos += n
	return val
}

func (br *brBitReader) jump() int {
	nbits := br.bitPos & 7
	if nbits > 0 {
		br.readBits(8 - nbits)
	}
	return br.available + 8 - int(br.bitPos>>3)
}

func (br *brBitReader) readSymbol(table []huffCode) uint16 {
	br.fill()
	idx := int((br.val >> br.bitPos) & 0xFF)
	p := &table[idx]
	nbits := uint32(p.bits)
	if nbits > 8 {
		nbits -= 8
		br.bitPos += 8
		idx += int(p.value)
		idx += int((br.val >> br.bitPos) & ((uint64(1) << nbits) - 1))
		p = &table[idx]
	}
	br.bitPos += uint32(p.bits)
	return p.value
}

type brInput struct {
	data   []byte
	nwords int
	pos    int
	val    uint32
	bitPos int
	err    int
}

func newBrInput(data []byte) *brInput {
	return &brInput{data: data, nwords: len(data) >> 1, err: len(data) & 1}
}

func (in *brInput) nextWord() uint16 {
	var v uint16
	if in.pos < in.nwords {
		v = uint16(in.data[in.pos*2]) | uint16(in.data[in.pos*2+1])<<8
	} else {
		in.err = 1
	}
	in.pos++
	return v
}

func (in *brInput) initBits() { in.val = uint32(in.nextWord()) }

func (in *brInput) readBits(nbits int) int {
	if in.bitPos+nbits > 16 {
		in.val |= uint32(in.nextWord()) << 16
	}
	result := int((in.val >> uint(in.bitPos)) & uint32(kBrBitMask[nbits]))
	in.bitPos += nbits
	if in.bitPos > 16 {
		in.bitPos -= 16
		in.val >>= 16
	}
	return result
}

var kBrBitMask = [16]uint32{
	0, 1, 3, 7, 15, 31, 63, 127, 255, 511, 1023, 2047, 4095, 8191, 16383, 32767,
}

type prob struct {
	p8    uint8
	total uint8
	count uint16
}

func (p *prob) init(probability int) {
	p.p8 = uint8(probability)
	p.total = 3
	p.count = uint16(3 * probability)
}

func (p *prob) get() uint8 { return p.p8 }

func (p *prob) add(val int) {
	p.total++
	if val == 0 {
		p.count += 256
	} else {
		p.count++
	}
	p.p8 = uint8((uint32(kProbDivLut[p.total]) * uint32(p.count)) >> 17)
	if p.total == 254 {
		p.count >>= 1
		p.total = 127
	}
}

type arithDec struct {
	low, high, value uint32
}

func (a *arithDec) init(in *brInput) {
	a.low = 0
	a.high = ^uint32(0)
	a.value = uint32(in.nextWord())
	a.value = (a.value << 16) | uint32(in.nextWord())
}

func (a *arithDec) readBit(pr int, in *brInput) int {
	diff := a.high - a.low
	split := a.low + uint32((uint64(diff)*uint64(pr))>>8)
	var bit int
	if a.value > split {
		a.low = split + 1
		bit = 1
	} else {
		a.high = split
		bit = 0
	}
	if ((a.low ^ a.high) >> 16) == 0 {
		a.value = (a.value << 16) | uint32(in.nextWord())
		a.low <<= 16
		a.high <<= 16
		a.high |= 0xFFFF
	}
	return bit
}

type ansSym struct {
	off  uint16
	freq uint16
	sym  uint8
}

type ansCode struct {
	mp [ansTabSize]ansSym
}

type ansDec struct{ state uint32 }

func (a *ansDec) init(in *brInput) {
	a.state = uint32(in.nextWord())
	a.state = (a.state << 16) | uint32(in.nextWord())
}

func (a *ansDec) read(code *ansCode, in *brInput) int {
	res := a.state & (ansTabSize - 1)
	s := code.mp[res]
	a.state = uint32(s.freq)*(a.state>>ansLogTabSize) + uint32(s.off)
	if a.state < (1 << 16) {
		a.state = (a.state << 16) | uint32(in.nextWord())
	}
	return int(s.sym)
}

func (a *ansDec) check() bool { return a.state == (ansSignature << 16) }

func log2FloorNonZero(n uint32) int {
	r := 0
	for n > 1 {
		n >>= 1
		r++
	}
	return r
}

func nextHuffKey(key, length int) int {
	step := 1 << uint(length-1)
	for key&step != 0 {
		step >>= 1
	}
	return (key & (step - 1)) + step
}

func replicateHuff(table []huffCode, step, end int, code huffCode) {
	for end > 0 {
		end -= step
		table[end] = code
	}
}

func nextTableBitSize(count []uint16, length, rootBits int) int {
	left := 1 << uint(length-rootBits)
	for length < 15 {
		left -= int(count[length])
		if left <= 0 {
			break
		}
		length++
		left <<= 1
	}
	return length - rootBits
}

func buildHuffmanTable(root []huffCode, rootBits int, codeLengths []byte, count []uint16) int {
	const maxLength = 15
	if len(codeLengths) > 704 {
		return 0
	}
	var offset [maxLength + 1]uint16
	maxLen := 1
	var sum uint16
	for length := 1; length <= maxLength; length++ {
		offset[length] = sum
		if count[length] != 0 {
			sum += count[length]
			maxLen = length
		}
	}
	var sorted [704]int
	for symbol, cl := range codeLengths {
		if cl != 0 {
			sorted[offset[cl]] = symbol
			offset[cl]++
		}
	}
	tableBits := rootBits
	tableSize := 1 << uint(tableBits)
	totalSize := tableSize
	if offset[maxLength] == 1 {
		code := huffCode{bits: 0, value: uint16(sorted[0])}
		for key := 0; key < totalSize; key++ {
			root[key] = code
		}
		return totalSize
	}
	if tableBits > maxLen {
		tableBits = maxLen
		tableSize = 1 << uint(tableBits)
	}
	key := 0
	symbol := 0
	codeBits := 1
	step := 2
	for {
		for count[codeBits] != 0 {
			count[codeBits]--
			code := huffCode{bits: uint8(codeBits), value: uint16(sorted[symbol])}
			symbol++
			replicateHuff(root[key:], step, tableSize, code)
			key = nextHuffKey(key, codeBits)
		}
		step <<= 1
		codeBits++
		if codeBits > tableBits {
			break
		}
	}
	for totalSize != tableSize {
		if tableSize*2 > len(root) {
			return 0
		}
		copy(root[tableSize:tableSize*2], root[:tableSize])
		tableSize <<= 1
	}
	mask := totalSize - 1
	low := -1
	tablePos := 0
	step = 2
	for length := rootBits + 1; length <= maxLen; length++ {
		for count[length] != 0 {
			if (key & mask) != low {
				tablePos += tableSize
				tableBits = nextTableBitSize(count, length, rootBits)
				tableSize = 1 << uint(tableBits)
				totalSize += tableSize
				if tablePos+tableSize > len(root) {
					return 0
				}
				low = key & mask
				root[low].bits = uint8(tableBits + rootBits)
				root[low].value = uint16(tablePos - low)
			}
			count[length]--
			code := huffCode{bits: uint8(length - rootBits), value: uint16(sorted[symbol])}
			symbol++
			replicateHuff(root[tablePos+(key>>uint(rootBits)):], step, tableSize, code)
			key = nextHuffKey(key, length)
		}
		step <<= 1
	}
	return totalSize
}

var kCodeLenOrder = [18]byte{1, 2, 3, 4, 0, 5, 17, 6, 16, 7, 8, 9, 10, 11, 12, 13, 14, 15}

var kCodeLenHuff = [16]huffCode{
	{2, 0}, {2, 4}, {2, 3}, {3, 2}, {2, 0}, {2, 4}, {2, 3}, {4, 1},
	{2, 0}, {2, 4}, {2, 3}, {3, 2}, {2, 0}, {2, 4}, {2, 3}, {4, 5},
}

var kHistCountHuff = [64]huffCode{
	{2, 6}, {3, 7}, {3, 4}, {4, 1}, {2, 6}, {3, 8}, {3, 5}, {4, 3},
	{2, 6}, {3, 7}, {3, 4}, {4, 2}, {2, 6}, {3, 8}, {3, 5}, {5, 0},
	{2, 6}, {3, 7}, {3, 4}, {4, 1}, {2, 6}, {3, 8}, {3, 5}, {4, 3},
	{2, 6}, {3, 7}, {3, 4}, {4, 2}, {2, 6}, {3, 8}, {3, 5}, {6, 9},
	{2, 6}, {3, 7}, {3, 4}, {4, 1}, {2, 6}, {3, 8}, {3, 5}, {4, 3},
	{2, 6}, {3, 7}, {3, 4}, {4, 2}, {2, 6}, {3, 8}, {3, 5}, {5, 0},
	{2, 6}, {3, 7}, {3, 4}, {4, 1}, {2, 6}, {3, 8}, {3, 5}, {4, 3},
	{2, 6}, {3, 7}, {3, 4}, {4, 2}, {2, 6}, {3, 8}, {3, 5}, {6, 10},
}

func readHuffLens(codeLenLens []byte, numSymbols int, lens []byte, br *brBitReader) bool {
	symbol := 0
	prev := byte(8)
	repeat := 0
	repeatLen := byte(0)
	space := 32768
	var table [256]huffCode
	var counts [16]uint16
	for i := 0; i < 18; i++ {
		counts[codeLenLens[i]]++
	}
	if buildHuffmanTable(table[:], 5, codeLenLens, counts[:]) == 0 {
		return false
	}
	for symbol < numSymbols && space > 0 {
		if !br.readMore() {
			return false
		}
		br.fill()
		p := &table[(br.val>>br.bitPos)&31]
		br.bitPos += uint32(p.bits)
		codeLen := byte(p.value)
		if codeLen < 16 {
			repeat = 0
			lens[symbol] = codeLen
			symbol++
			if codeLen != 0 {
				prev = codeLen
				space -= 32768 >> codeLen
			}
		} else {
			extra := int(codeLen) - 14
			newLen := byte(0)
			if codeLen == 16 {
				newLen = prev
			}
			if repeatLen != newLen {
				repeat = 0
				repeatLen = newLen
			}
			old := repeat
			if repeat > 0 {
				repeat -= 2
				repeat <<= uint(extra)
			}
			repeat += int(br.readBits(uint32(extra))) + 3
			delta := repeat - old
			if symbol+delta > numSymbols {
				return false
			}
			for k := 0; k < delta; k++ {
				lens[symbol+k] = repeatLen
			}
			symbol += delta
			if repeatLen != 0 {
				space -= delta << uint(15-repeatLen)
			}
		}
	}
	if space != 0 {
		return false
	}
	for ; symbol < numSymbols; symbol++ {
		lens[symbol] = 0
	}
	return true
}

func readHuffTable(alphabet int, br *brBitReader) ([]huffCode, bool) {
	if !br.readMore() {
		return nil, false
	}
	lens := make([]byte, alphabet)
	ok := true
	simple := int(br.readBits(2))
	if simple == 1 {
		maxCounter := alphabet - 1
		maxBits := 0
		for maxCounter != 0 {
			maxCounter >>= 1
			maxBits++
		}
		nsym := int(br.readBits(2)) + 1
		var symbols [4]int
		for i := 0; i < nsym; i++ {
			symbols[i] = int(br.readBits(uint32(maxBits))) % alphabet
			lens[symbols[i]] = 2
		}
		lens[symbols[0]] = 1
		switch nsym {
		case 1:
		case 2:
			ok = symbols[0] != symbols[1]
			lens[symbols[1]] = 1
		case 3:
			ok = symbols[0] != symbols[1] && symbols[0] != symbols[2] && symbols[1] != symbols[2]
		case 4:
			ok = symbols[0] != symbols[1] && symbols[0] != symbols[2] && symbols[0] != symbols[3] &&
				symbols[1] != symbols[2] && symbols[1] != symbols[3] && symbols[2] != symbols[3]
			if br.readBits(1) != 0 {
				lens[symbols[2]] = 3
				lens[symbols[3]] = 3
			} else {
				lens[symbols[0]] = 2
			}
		default:
			return nil, false
		}
	} else {
		var clcl [18]byte
		space := 32
		numCodes := 0
		for i := simple; i < 18 && space > 0; i++ {
			idx := kCodeLenOrder[i]
			br.fill()
			p := &kCodeLenHuff[(br.val>>br.bitPos)&15]
			br.bitPos += uint32(p.bits)
			v := byte(p.value)
			clcl[idx] = v
			if v != 0 {
				space -= 32 >> v
				numCodes++
			}
		}
		ok = (numCodes == 1 || space == 0) && readHuffLens(clcl[:], alphabet, lens, br)
	}
	var counts [16]uint16
	for i := 0; i < alphabet; i++ {
		counts[lens[i]]++
	}
	table := make([]huffCode, 2048)
	size := 0
	if ok {
		size = buildHuffmanTable(table, 8, lens, counts[:])
	}
	if size <= 0 {
		return nil, false
	}
	return table, true
}

func readHistLength(br *brBitReader) int {
	lens := [16]byte{8, 8, 6, 6, 6, 5, 4, 3, 3, 3, 3, 3, 3, 4, 5, 7}
	var counts [16]uint16
	for i := 0; i < 16; i++ {
		counts[lens[i]]++
	}
	var table [256]huffCode
	buildHuffmanTable(table[:], 8, lens[:], counts[:])
	br.fill()
	p := &table[(br.val>>br.bitPos)&255]
	br.bitPos += uint32(p.bits)
	return int(p.value) + 3
}

func readHistogram(precisionBits, length int, counts []int, br *brBitReader) bool {
	if !br.readMore() {
		return false
	}
	if br.readBits(1) == 1 {
		maxCounter := length - 1
		maxBits := 0
		for maxCounter != 0 {
			maxCounter >>= 1
			maxBits++
		}
		nsym := int(br.readBits(1)) + 1
		for i := range counts[:length] {
			counts[i] = 0
		}
		var symbols [2]int
		for i := 0; i < nsym; i++ {
			symbols[i] = int(br.readBits(uint32(maxBits))) % length
		}
		if nsym == 1 {
			counts[symbols[0]] = 1 << uint(precisionBits)
		} else {
			if symbols[0] == symbols[1] {
				return false
			}
			counts[symbols[0]] = int(br.readBits(uint32(precisionBits)))
			counts[symbols[1]] = (1 << uint(precisionBits)) - counts[symbols[0]]
		}
		return true
	}
	realLen := readHistLength(br)
	for i := range counts[:length] {
		counts[i] = 0
	}
	if realLen > 18 || realLen > length {
		return false
	}
	var logCounts [18]int
	omitLog := -1
	omitPos := -1
	for i := 0; i < realLen; i++ {
		br.fill()
		p := &kHistCountHuff[(br.val>>br.bitPos)&63]
		br.bitPos += uint32(p.bits)
		logCounts[i] = int(p.value)
		if logCounts[i] > omitLog {
			omitLog = logCounts[i]
			omitPos = i
		}
	}
	if omitPos < 0 {
		return false
	}
	total := 0
	for i := 0; i < realLen; i++ {
		code := logCounts[i]
		if i == omitPos || code == 0 {
			continue
		}
		if code == 1 {
			counts[i] = 1
		} else {
			bitCount := (code - 1 + 1) >> 1
			counts[i] = (1 << uint(code-1)) + (int(br.readBits(uint32(bitCount))) << uint(code-1-bitCount))
		}
		total += counts[i]
	}
	if total >= (1 << uint(precisionBits)) {
		return false
	}
	counts[omitPos] = (1 << uint(precisionBits)) - total
	return true
}

func ansFromBits(alphabet int, br *brBitReader) (ansCode, bool) {
	var code ansCode
	counts := make([]int, alphabet)
	if !readHistogram(ansLogTabSize, alphabet, counts, br) {
		return code, false
	}
	pos := 0
	for i := 0; i < alphabet; i++ {
		for j := 0; j < counts[i]; j++ {
			if pos >= ansTabSize {
				return code, false
			}
			code.mp[pos] = ansSym{sym: uint8(i), freq: uint16(counts[i]), off: uint16(j)}
			pos++
		}
	}
	return code, pos == ansTabSize
}

func moveToFront(v []byte, index int) {
	value := v[index]
	for i := index; i > 0; i-- {
		v[i] = v[i-1]
	}
	v[0] = value
}

func inverseMTF(v []byte) {
	var mtf [256]byte
	for i := 0; i < 256; i++ {
		mtf[i] = byte(i)
	}
	for i := 0; i < len(v); i++ {
		index := v[i]
		v[i] = mtf[index]
		if index != 0 {
			moveToFront(mtf[:], int(index))
		}
	}
}

func decodeContextMap(numTrees, size int, ctx []byte, br *brBitReader) bool {
	if numTrees <= 1 {
		for i := 0; i < size; i++ {
			ctx[i] = 0
		}
		return true
	}
	maxRun := 0
	if br.readBits(1) != 0 {
		maxRun = int(br.readBits(4)) + 1
	}
	table, ok := readHuffTable(numTrees+maxRun, br)
	if !ok {
		return false
	}
	for i := 0; i < size; {
		if !br.readMore() {
			return false
		}
		code := int(br.readSymbol(table))
		if code == 0 {
			ctx[i] = 0
			i++
		} else if code <= maxRun {
			reps := 1 + (1 << uint(code)) + int(br.readBits(uint32(code)))
			for {
				reps--
				if reps == 0 {
					break
				}
				if i >= size {
					return false
				}
				ctx[i] = 0
				i++
			}
		} else {
			ctx[i] = byte(code - maxRun)
			i++
		}
	}
	if br.readBits(1) != 0 {
		inverseMTF(ctx[:size])
	}
	return true
}

func decodeVarLenUint8(br *brBitReader) uint32 {
	if br.readBits(1) != 0 {
		nbits := br.readBits(3)
		if nbits == 0 {
			return 1
		}
		return br.readBits(nbits) + (1 << nbits)
	}
	return 0
}

func decodeVarint(br *brBitReader, maxBits int) uint32 {
	var result uint32
	for b := 0; b < maxBits; b++ {
		if b+1 != maxBits && br.readBits(1) == 0 {
			break
		}
		result |= br.readBits(1) << uint(b)
	}
	return result
}

func decodeLimitedVarint(br *brBitReader, nbits, maxSymbols int) uint64 {
	var bits uint64
	shift := uint(0)
	for b := 0; b < maxSymbols && br.readBits(1) != 0; b++ {
		next := uint64(br.readBits(uint32(nbits)))
		bits |= next << shift
		shift += uint(nbits)
	}
	return bits
}

var errBr = errors.New("jpgrec: invalid brunsli stream")
