package xt3u

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

// zstd 1.5.5 frame decoder for xt3u_zstd_prefix (raw refPrefix, not a dictionary frame).
// 64-bit bitstream, Huffman X1, literals kept in a side buffer.

const (
	zMagic          = 0xFD2FB528
	zSkippableMagic = 0x184D2A50
	zSkippableMask  = 0xFFFFFFF0
	zBlockSizeMax   = 128 << 10
	zWindowLogMin   = 10
	zWindowLogMax   = 31
	zWindowDefault  = 27

	zGeneric     zcode = 1
	zPrefix      zcode = 10
	zUnsupported zcode = 14
	zWindow      zcode = 16
	zCorruption  zcode = 20
	zChecksum    zcode = 22
	zLitHeader   zcode = 24
	zDictCorrupt zcode = 30
	zDictWrong   zcode = 32
	zParamBound  zcode = 42
	zTableLog    zcode = 44
	zMaxSymSmall zcode = 48
	zDstSmall    zcode = 70
	zSrcSize     zcode = 72

	btRaw        = 0
	btRLE        = 1
	btCompressed = 2
	btReserved   = 3

	setBasic      = 0
	setRLE        = 1
	setCompressed = 2
	setRepeat     = 3

	bitUnfinished  = 0
	bitEndOfBuffer = 1
	bitCompleted   = 2
	bitOverflow    = 3

	accReloadAt = 57 - (9 + 9 + 8) // STREAM_ACCUMULATOR_MIN_64-(LL+ML+OF FSE logs)
)

type zcode int

func (z zcode) Error() string { return fmt.Sprintf("zstd %d", int(z)) }

func zstdGuest(n int) error {
	return fmt.Errorf("%w: zstd %d", errGuest, n)
}

func guestFrom(err error) error {
	c, ok := err.(zcode)
	if !ok {
		return err
	}
	return zstdGuest(-int(c) - 1)
}

type bitReader struct {
	src          []byte
	ptr          int
	bits         uint64
	bitsConsumed uint32
}

func (b *bitReader) init(src []byte) error {
	*b = bitReader{src: src}
	if len(src) < 1 {
		return zSrcSize
	}
	if len(src) >= 8 {
		b.ptr = len(src) - 8
		b.bits = binary.LittleEndian.Uint64(src[b.ptr:])
		last := src[len(src)-1]
		if last == 0 {
			return zGeneric
		}
		b.bitsConsumed = 8 - highbit32(uint32(last))
		return nil
	}
	var c uint64
	for i := 0; i < len(src); i++ {
		c |= uint64(src[i]) << (uint(i) * 8)
	}
	b.bits = c
	last := src[len(src)-1]
	if last == 0 {
		return zCorruption
	}
	b.bitsConsumed = 8 - highbit32(uint32(last))
	b.bitsConsumed += uint32(8-len(src)) * 8
	return nil
}

func (b *bitReader) lookBits(nb uint32) uint64 {
	if nb == 0 {
		return 0
	}
	start := (64 - b.bitsConsumed - nb) & 63
	return (b.bits >> start) & ((uint64(1) << nb) - 1)
}

func (b *bitReader) lookBitsFast(nb uint32) uint64 {
	return (b.bits << (b.bitsConsumed & 63)) >> ((64 - nb) & 63)
}

func (b *bitReader) readBits(nb uint32) uint64 {
	v := b.lookBits(nb)
	b.bitsConsumed += nb
	return v
}

func (b *bitReader) readBitsFast(nb uint32) uint64 {
	v := b.lookBitsFast(nb)
	b.bitsConsumed += nb
	return v
}

func (b *bitReader) reload() int {
	if b.bitsConsumed > 64 {
		return bitOverflow
	}
	if b.ptr >= 8 {
		return b.reloadFast()
	}
	if b.ptr == 0 {
		if b.bitsConsumed < 64 {
			return bitEndOfBuffer
		}
		return bitCompleted
	}
	nbBytes := b.bitsConsumed >> 3
	result := bitUnfinished
	if b.ptr < int(nbBytes) {
		nbBytes = uint32(b.ptr)
		result = bitEndOfBuffer
	}
	b.ptr -= int(nbBytes)
	b.bitsConsumed -= nbBytes * 8
	if b.ptr < 0 || b.ptr+8 > len(b.src) {
		return bitOverflow
	}
	b.bits = binary.LittleEndian.Uint64(b.src[b.ptr:])
	return result
}

func (b *bitReader) reloadFast() int {
	if b.ptr < 8 {
		return bitOverflow
	}
	b.ptr -= int(b.bitsConsumed >> 3)
	b.bitsConsumed &= 7
	if b.ptr < 0 || b.ptr+8 > len(b.src) {
		return bitOverflow
	}
	b.bits = binary.LittleEndian.Uint64(b.src[b.ptr:])
	return bitUnfinished
}

func (b *bitReader) ended() bool {
	return b.ptr == 0 && b.bitsConsumed == 64
}

func highbit32(v uint32) uint32 {
	return uint32(bits.Len32(v) - 1)
}

type fseSym struct {
	base uint32
	add  uint8
	nb   uint8
	next uint16
}

type fseTable struct {
	log  uint32
	fast bool
	cell []fseSym
}

func buildFSE(norm []int16, maxSym, tableLog int, base []uint32, add []uint8) (fseTable, error) {
	if tableLog <= 0 || tableLog > 15 {
		return fseTable{}, zTableLog
	}
	tableSize := 1 << tableLog
	high := tableSize - 1
	nextSym := make([]uint16, maxSym+1)
	cell := make([]fseSym, tableSize)
	fast := true
	large := int16(1 << (tableLog - 1))
	for s := 0; s <= maxSym; s++ {
		c := norm[s]
		if c == -1 {
			cell[high].base = uint32(s)
			high--
			nextSym[s] = 1
			continue
		}
		if c < 0 {
			return fseTable{}, zCorruption
		}
		if c >= large {
			fast = false
		}
		nextSym[s] = uint16(c)
	}
	step := (tableSize >> 1) + (tableSize >> 3) + 3
	mask := tableSize - 1
	if high == tableSize-1 {
		spread := make([]byte, tableSize)
		pos := 0
		for s := 0; s <= maxSym; s++ {
			n := int(norm[s])
			for i := 0; i < n; i++ {
				if pos >= tableSize {
					return fseTable{}, zCorruption
				}
				spread[pos] = byte(s)
				pos++
			}
		}
		if pos != tableSize {
			return fseTable{}, zCorruption
		}
		position := 0
		for s := 0; s < tableSize; s++ {
			cell[position].base = uint32(spread[s])
			position = (position + step) & mask
		}
	} else {
		position := 0
		for s := 0; s <= maxSym; s++ {
			n := int(norm[s])
			for i := 0; i < n; i++ {
				cell[position].base = uint32(s)
				position = (position + step) & mask
				spins := 0
				for position > high {
					position = (position + step) & mask
					spins++
					if spins > tableSize {
						return fseTable{}, zCorruption
					}
				}
			}
		}
	}
	for u := 0; u < tableSize; u++ {
		sym := int(cell[u].base)
		if sym < 0 || sym > maxSym {
			return fseTable{}, zCorruption
		}
		ns := nextSym[sym]
		nextSym[sym] = ns + 1
		if ns == 0 {
			return fseTable{}, zCorruption
		}
		nb := tableLog - int(highbit32(uint32(ns)))
		if nb < 0 {
			return fseTable{}, zCorruption
		}
		cell[u].nb = uint8(nb)
		cell[u].next = uint16((uint32(ns) << uint(nb)) - uint32(tableSize))
		if base != nil {
			cell[u].add = add[sym]
			cell[u].base = base[sym]
		}
	}
	return fseTable{log: uint32(tableLog), fast: fast, cell: cell}, nil
}

func readNCount(norm []int16, maxSV int, src []byte) (nRead, newMax, tableLog int, err error) {
	if len(src) < 8 {
		buf := make([]byte, 8)
		copy(buf, src)
		n, m, tl, e := readNCount(norm, maxSV, buf)
		if e != nil {
			return 0, 0, 0, e
		}
		if n > len(src) {
			return 0, 0, 0, zCorruption
		}
		return n, m, tl, nil
	}
	for i := 0; i <= maxSV; i++ {
		norm[i] = 0
	}
	ip := 0
	if ip+4 > len(src) {
		return 0, 0, 0, zCorruption
	}
	bitStream := binary.LittleEndian.Uint32(src[ip:])
	nbBits := int(bitStream&0xF) + 5
	if nbBits > 15 {
		return 0, 0, 0, zTableLog
	}
	bitStream >>= 4
	bitCount := 4
	tableLog = nbBits
	remaining := (1 << nbBits) + 1
	threshold := 1 << nbBits
	nbBits++
	charnum := 0
	maxSV1 := maxSV + 1
	previous0 := false
	reloadStream := func() error {
		if bitCount >= 32 || bitCount < 0 {
			return zCorruption
		}
		if ip < 0 || ip+4 > len(src) {
			return zCorruption
		}
		bitStream = binary.LittleEndian.Uint32(src[ip:]) >> uint(bitCount)
		return nil
	}
	for {
		if previous0 {
			repeats := bits.TrailingZeros32(^bitStream|0x80000000) >> 1
			for repeats >= 12 {
				charnum += 36
				if ip <= len(src)-7 {
					ip += 3
				} else {
					bitCount -= 8 * (len(src) - 7 - ip)
					bitCount &= 31
					ip = len(src) - 4
				}
				if err = reloadStream(); err != nil {
					return 0, 0, 0, err
				}
				repeats = bits.TrailingZeros32(^bitStream|0x80000000) >> 1
			}
			charnum += 3 * repeats
			shift := uint(2 * repeats)
			if shift >= 32 {
				bitStream = 0
			} else {
				bitStream >>= shift
			}
			bitCount += 2 * repeats
			charnum += int(bitStream & 3)
			bitCount += 2
			if charnum >= maxSV1 {
				break
			}
			if ip <= len(src)-7 || ip+(bitCount>>3) <= len(src)-4 {
				ip += bitCount >> 3
				bitCount &= 7
			} else {
				bitCount -= 8 * (len(src) - 4 - ip)
				bitCount &= 31
				ip = len(src) - 4
			}
			if err = reloadStream(); err != nil {
				return 0, 0, 0, err
			}
		}
		maxv := (2*threshold - 1) - remaining
		var count int
		thrMask := uint32(threshold - 1)
		if bitStream&thrMask < uint32(maxv) {
			count = int(bitStream & thrMask)
			bitCount += nbBits - 1
		} else {
			count = int(bitStream & uint32(2*threshold-1))
			if count >= threshold {
				count -= maxv
			}
			bitCount += nbBits
		}
		count--
		if count >= 0 {
			remaining -= count
		} else {
			remaining += count
		}
		if charnum > maxSV {
			return 0, 0, 0, zMaxSymSmall
		}
		norm[charnum] = int16(count)
		charnum++
		previous0 = count == 0
		if remaining < threshold {
			if remaining <= 1 {
				break
			}
			nbBits = bits.Len32(uint32(remaining))
			threshold = 1 << (nbBits - 1)
		}
		if charnum >= maxSV1 {
			break
		}
		if ip <= len(src)-7 || ip+(bitCount>>3) <= len(src)-4 {
			ip += bitCount >> 3
			bitCount &= 7
		} else {
			bitCount -= 8 * (len(src) - 4 - ip)
			bitCount &= 31
			ip = len(src) - 4
		}
		if err = reloadStream(); err != nil {
			return 0, 0, 0, err
		}
	}
	if remaining != 1 {
		return 0, 0, 0, zCorruption
	}
	if charnum > maxSV1 {
		return 0, 0, 0, zMaxSymSmall
	}
	if bitCount > 32 {
		return 0, 0, 0, zCorruption
	}
	if charnum == 0 {
		return 0, 0, 0, zCorruption
	}
	ip += (bitCount + 7) >> 3
	return ip, charnum - 1, tableLog, nil
}

func (b *bitReader) decWeight(st *uint32, t *fseTable) (byte, error) {
	if int(*st) >= len(t.cell) {
		return 0, zCorruption
	}
	c := t.cell[*st]
	var low uint32
	if c.nb == 0 {
		low = 0
	} else if t.fast {
		low = uint32(b.readBitsFast(uint32(c.nb)))
	} else {
		low = uint32(b.readBits(uint32(c.nb)))
	}
	*st = uint32(c.next) + low
	return byte(c.base), nil
}

func fseDecodeWeights(dst []byte, src []byte, maxLog int) (int, error) {
	if len(src) == 0 {
		return 0, zSrcSize
	}
	norm := make([]int16, 256)
	maxSV := 255
	n, maxSym, tableLog, err := readNCount(norm, maxSV, src)
	if err != nil {
		return 0, err
	}
	if tableLog > maxLog {
		return 0, zTableLog
	}
	if n > len(src) {
		return 0, zCorruption
	}
	table, err := buildFSE(norm, maxSym, tableLog, nil, nil)
	if err != nil {
		return 0, err
	}
	var br bitReader
	if err = br.init(src[n:]); err != nil {
		return 0, err
	}
	st1, err := initFSE(&br, &table)
	if err != nil {
		return 0, err
	}
	st2, err := initFSE(&br, &table)
	if err != nil {
		return 0, err
	}
	omax := len(dst)
	olimit := omax - 3
	op := 0
	if olimit < 0 {
		olimit = 0
	}
	for {
		st := br.reload()
		if st != bitUnfinished || op >= olimit {
			break
		}
		s, e := br.decWeight(&st1, &table)
		if e != nil {
			return 0, e
		}
		dst[op] = s
		s, e = br.decWeight(&st2, &table)
		if e != nil {
			return 0, e
		}
		dst[op+1] = s
		s, e = br.decWeight(&st1, &table)
		if e != nil {
			return 0, e
		}
		dst[op+2] = s
		s, e = br.decWeight(&st2, &table)
		if e != nil {
			return 0, e
		}
		dst[op+3] = s
		op += 4
	}
	for {
		if op > omax-2 {
			return 0, zDstSmall
		}
		s, e := br.decWeight(&st1, &table)
		if e != nil {
			return 0, e
		}
		dst[op] = s
		op++
		if br.reload() == bitOverflow {
			if op > omax-1 {
				return 0, zDstSmall
			}
			s, e = br.decWeight(&st2, &table)
			if e != nil {
				return 0, e
			}
			dst[op] = s
			op++
			break
		}
		if op > omax-2 {
			return 0, zDstSmall
		}
		s, e = br.decWeight(&st2, &table)
		if e != nil {
			return 0, e
		}
		dst[op] = s
		op++
		if br.reload() == bitOverflow {
			if op > omax-1 {
				return 0, zDstSmall
			}
			s, e = br.decWeight(&st1, &table)
			if e != nil {
				return 0, e
			}
			dst[op] = s
			op++
			break
		}
	}
	return op, nil
}

func initFSE(br *bitReader, t *fseTable) (uint32, error) {
	st := uint32(br.readBits(t.log))
	br.reload()
	if t.log > 0 && int(st) >= len(t.cell) {
		return 0, zCorruption
	}
	return st, nil
}

type hufDec struct {
	sym byte
	nb  uint8
}

func (d *zdec) readHufTable(src []byte) (int, error) {
	if len(src) == 0 {
		return 0, zSrcSize
	}
	iSize := int(src[0])
	var weights []byte
	var nRead int
	if iSize >= 128 {
		oSize := iSize - 127
		iSz := (oSize + 1) / 2
		if iSz+1 > len(src) || oSize >= 256 || oSize < 0 {
			if iSz+1 > len(src) {
				return 0, zSrcSize
			}
			return 0, zCorruption
		}
		weights = make([]byte, oSize+1)
		ip := src[1:]
		for n := 0; n < oSize; n += 2 {
			weights[n] = ip[n/2] >> 4
			weights[n+1] = ip[n/2] & 15
		}
		weights = weights[:oSize]
		nRead = iSz + 1
	} else {
		if iSize+1 > len(src) {
			return 0, zSrcSize
		}
		buf := make([]byte, 255)
		oSize, err := fseDecodeWeights(buf, src[1:1+iSize], 6)
		if err != nil {
			return 0, err
		}
		if oSize < 0 || oSize > 255 {
			return 0, zCorruption
		}
		weights = buf[:oSize]
		nRead = iSize + 1
	}
	rank := make([]int, 13)
	total := 0
	for _, w := range weights {
		if int(w) > 12 {
			return 0, zCorruption
		}
		rank[w]++
		total += (1 << w) >> 1
	}
	if total == 0 {
		return 0, zCorruption
	}
	tableLog := int(highbit32(uint32(total))) + 1
	if tableLog > 12 {
		return 0, zCorruption
	}
	rest := (1 << tableLog) - total
	if rest <= 0 {
		return 0, zCorruption
	}
	hb := int(highbit32(uint32(rest)))
	if 1<<hb != rest {
		return 0, zCorruption
	}
	last := hb + 1
	if last > 12 {
		return 0, zCorruption
	}
	weights = append(weights, byte(last))
	rank[last]++
	nbSym := len(weights)
	const target = 11
	if tableLog < target {
		scale := target - tableLog
		for i := range weights {
			if weights[i] != 0 {
				weights[i] += byte(scale)
			}
		}
		nr := make([]int, target+1)
		for s := target; s > scale; s-- {
			nr[s] = rank[s-scale]
		}
		nr[0] = rank[0]
		rank = nr
		tableLog = target
	}
	pos := make([]int, tableLog+1)
	sum := 0
	for n := 0; n <= tableLog; n++ {
		if n >= len(rank) {
			return 0, zCorruption
		}
		pos[n] = sum
		sum += rank[n]
	}
	if sum != nbSym {
		return 0, zCorruption
	}
	syms := make([]byte, sum)
	for s, w := range weights {
		ww := int(w)
		if ww > tableLog {
			return 0, zCorruption
		}
		syms[pos[ww]] = byte(s)
		pos[ww]++
	}
	dt := make([]hufDec, 1<<tableLog)
	idx := 0
	symi := 0
	if len(rank) > 0 {
		symi = rank[0]
	}
	for weight := 1; weight <= tableLog; weight++ {
		count := rank[weight]
		length := 1 << (weight - 1)
		nb := byte(tableLog + 1 - weight)
		for j := 0; j < count; j++ {
			if symi >= len(syms) {
				return 0, zCorruption
			}
			sym := syms[symi]
			symi++
			for k := 0; k < length; k++ {
				if idx >= len(dt) {
					return 0, zCorruption
				}
				dt[idx] = hufDec{sym: sym, nb: nb}
				idx++
			}
		}
	}
	if idx != len(dt) {
		return 0, zCorruption
	}
	d.huf = dt
	d.hufLog = uint32(tableLog)
	return nRead, nil
}

func hufSymbol(br *bitReader, dt []hufDec, dtLog uint32) byte {
	val := br.lookBitsFast(dtLog)
	e := dt[val]
	br.bitsConsumed += uint32(e.nb)
	return e.sym
}

func hufStream(br *bitReader, dst []byte, dt []hufDec, dtLog uint32) {
	p := 0
	end := len(dst)
	if end-p > 3 {
		for {
			st := br.reload()
			if st != bitUnfinished || p >= end-3 {
				break
			}
			dst[p] = hufSymbol(br, dt, dtLog)
			dst[p+1] = hufSymbol(br, dt, dtLog)
			dst[p+2] = hufSymbol(br, dt, dtLog)
			dst[p+3] = hufSymbol(br, dt, dtLog)
			p += 4
		}
	} else {
		br.reload()
	}
	for p < end {
		dst[p] = hufSymbol(br, dt, dtLog)
		p++
	}
}

func hufDecode1(dst, src []byte, dt []hufDec, dtLog uint32) error {
	var br bitReader
	if err := br.init(src); err != nil {
		return err
	}
	hufStream(&br, dst, dt, dtLog)
	if !br.ended() {
		return zCorruption
	}
	return nil
}

func hufDecode4(dst, src []byte, dt []hufDec, dtLog uint32) error {
	if len(src) < 10 || len(dst) < 6 {
		return zCorruption
	}
	l1 := int(binary.LittleEndian.Uint16(src))
	l2 := int(binary.LittleEndian.Uint16(src[2:]))
	l3 := int(binary.LittleEndian.Uint16(src[4:]))
	if l1 < 0 || l2 < 0 || l3 < 0 || l1+l2+l3+6 > len(src) {
		return zCorruption
	}
	l4 := len(src) - (l1 + l2 + l3 + 6)
	off := 6
	s1 := src[off : off+l1]
	off += l1
	s2 := src[off : off+l2]
	off += l2
	s3 := src[off : off+l3]
	off += l3
	s4 := src[off : off+l4]
	seg := (len(dst) + 3) / 4
	b1 := seg
	b2 := seg * 2
	b3 := seg * 3
	if b3 > len(dst) {
		return zCorruption
	}
	var r1, r2, r3, r4 bitReader
	if err := r1.init(s1); err != nil {
		return err
	}
	if err := r2.init(s2); err != nil {
		return err
	}
	if err := r3.init(s3); err != nil {
		return err
	}
	if err := r4.init(s4); err != nil {
		return err
	}
	o1, o2, o3, o4 := 0, b1, b2, b3
	olimit := len(dst) - 3
	if len(dst)-o4 >= 8 {
		endSignal := true
		for endSignal && o4 < olimit {
			for k := 0; k < 4; k++ {
				dst[o1] = hufSymbol(&r1, dt, dtLog)
				o1++
				dst[o2] = hufSymbol(&r2, dt, dtLog)
				o2++
				dst[o3] = hufSymbol(&r3, dt, dtLog)
				o3++
				dst[o4] = hufSymbol(&r4, dt, dtLog)
				o4++
			}
			s1 := r1.reloadFast() == bitUnfinished
			s2 := r2.reloadFast() == bitUnfinished
			s3 := r3.reloadFast() == bitUnfinished
			s4 := r4.reloadFast() == bitUnfinished
			endSignal = s1 && s2 && s3 && s4
		}
	}
	if o1 > b1 || o2 > b2 || o3 > b3 || o4 > len(dst) {
		return zCorruption
	}
	hufStream(&r1, dst[o1:b1], dt, dtLog)
	hufStream(&r2, dst[o2:b2], dt, dtLog)
	hufStream(&r3, dst[o3:b3], dt, dtLog)
	hufStream(&r4, dst[o4:], dt, dtLog)
	if !r1.ended() || !r2.ended() || !r3.ended() || !r4.ended() {
		return zCorruption
	}
	return nil
}

var (
	llDefault fseTable
	ofDefault fseTable
	mlDefault fseTable
	llBase    = []uint32{
		0, 1, 2, 3, 4, 5, 6, 7,
		8, 9, 10, 11, 12, 13, 14, 15,
		16, 18, 20, 22, 24, 28, 32, 40,
		48, 64, 0x80, 0x100, 0x200, 0x400, 0x800, 0x1000,
		0x2000, 0x4000, 0x8000, 0x10000,
	}
	llBits = []uint8{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		1, 1, 1, 1, 2, 2, 3, 3,
		4, 6, 7, 8, 9, 10, 11, 12,
		13, 14, 15, 16,
	}
	llNorm = []int16{
		4, 3, 2, 2, 2, 2, 2, 2,
		2, 2, 2, 2, 2, 1, 1, 1,
		2, 2, 2, 2, 2, 2, 2, 2,
		2, 3, 2, 1, 1, 1, 1, 1,
		-1, -1, -1, -1,
	}
	ofBase = []uint32{
		0, 1, 1, 5, 0xD, 0x1D, 0x3D, 0x7D,
		0xFD, 0x1FD, 0x3FD, 0x7FD, 0xFFD, 0x1FFD, 0x3FFD, 0x7FFD,
		0xFFFD, 0x1FFFD, 0x3FFFD, 0x7FFFD, 0xFFFFD, 0x1FFFFD, 0x3FFFFD, 0x7FFFFD,
		0xFFFFFD, 0x1FFFFFD, 0x3FFFFFD, 0x7FFFFFD, 0xFFFFFFD, 0x1FFFFFFD, 0x3FFFFFFD, 0x7FFFFFFD,
	}
	ofBits = []uint8{
		0, 1, 2, 3, 4, 5, 6, 7,
		8, 9, 10, 11, 12, 13, 14, 15,
		16, 17, 18, 19, 20, 21, 22, 23,
		24, 25, 26, 27, 28, 29, 30, 31,
	}
	ofNorm = []int16{
		1, 1, 1, 1, 1, 1, 2, 2,
		2, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1,
		-1, -1, -1, -1, -1,
	}
	mlBase = []uint32{
		3, 4, 5, 6, 7, 8, 9, 10,
		11, 12, 13, 14, 15, 16, 17, 18,
		19, 20, 21, 22, 23, 24, 25, 26,
		27, 28, 29, 30, 31, 32, 33, 34,
		35, 37, 39, 41, 43, 47, 51, 59,
		67, 83, 99, 0x83, 0x103, 0x203, 0x403, 0x803,
		0x1003, 0x2003, 0x4003, 0x8003, 0x10003,
	}
	mlBits = []uint8{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		1, 1, 1, 1, 2, 2, 3, 3,
		4, 4, 5, 7, 8, 9, 10, 11,
		12, 13, 14, 15, 16,
	}
	mlNorm = []int16{
		1, 4, 3, 2, 2, 2, 2, 2,
		2, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, -1, -1,
		-1, -1, -1, -1, -1,
	}
)

func init() {
	var err error
	llDefault, err = buildFSE(llNorm, len(llNorm)-1, 6, llBase, llBits)
	if err != nil {
		panic(err)
	}
	ofDefault, err = buildFSE(ofNorm, len(ofNorm)-1, 5, ofBase, ofBits)
	if err != nil {
		panic(err)
	}
	mlDefault, err = buildFSE(mlNorm, len(mlNorm)-1, 6, mlBase, mlBits)
	if err != nil {
		panic(err)
	}
}

type zdec struct {
	ll, of, ml fseTable
	huf        []hufDec
	hufLog     uint32
	rep        [3]uint32
	litEntropy bool
	fseEntropy bool
	prefix     []byte
}

func newZDec(prefix []byte) *zdec {
	return &zdec{rep: [3]uint32{1, 4, 8}, prefix: prefix}
}

func (d *zdec) buildOne(typ, maxSym, maxLog int, src []byte, base []uint32, add []uint8, def fseTable) (fseTable, int, error) {
	switch typ {
	case setRLE:
		if len(src) < 1 {
			return fseTable{}, 0, zSrcSize
		}
		sym := int(src[0])
		if sym > maxSym {
			return fseTable{}, 0, zCorruption
		}
		cell := []fseSym{{base: base[sym], add: add[sym]}}
		return fseTable{log: 0, fast: false, cell: cell}, 1, nil
	case setBasic:
		return def, 0, nil
	case setRepeat:
		if !d.fseEntropy {
			return fseTable{}, 0, zCorruption
		}
		return fseTable{}, 0, nil
	case setCompressed:
		norm := make([]int16, maxSym+1)
		n, maxOut, tl, err := readNCount(norm, maxSym, src)
		if err != nil {
			return fseTable{}, 0, zCorruption
		}
		if tl > maxLog {
			return fseTable{}, 0, zCorruption
		}
		t, err := buildFSE(norm, maxOut, tl, base, add)
		if err != nil {
			return fseTable{}, 0, zCorruption
		}
		return t, n, nil
	default:
		return fseTable{}, 0, zCorruption
	}
}

func (d *zdec) seqHeaders(src []byte) (nbSeq, nRead int, err error) {
	if len(src) < 1 {
		return 0, 0, zSrcSize
	}
	nbSeq = int(src[0])
	ip := 1
	if nbSeq == 0 {
		if len(src) != 1 {
			return 0, 0, zSrcSize
		}
		return 0, 1, nil
	}
	if nbSeq > 0x7F {
		if nbSeq == 0xFF {
			if ip+2 > len(src) {
				return 0, 0, zSrcSize
			}
			nbSeq = int(binary.LittleEndian.Uint16(src[ip:])) + 0x7F00
			ip += 2
		} else {
			if ip >= len(src) {
				return 0, 0, zSrcSize
			}
			nbSeq = ((nbSeq - 0x80) << 8) + int(src[ip])
			ip++
		}
	}
	if ip >= len(src) {
		return 0, 0, zSrcSize
	}
	desc := src[ip]
	ip++
	llT := int(desc >> 6)
	ofT := int((desc >> 4) & 3)
	mlT := int((desc >> 2) & 3)
	var n int
	var t fseTable
	if llT == setRepeat {
		if !d.fseEntropy {
			return 0, 0, zCorruption
		}
	} else {
		t, n, err = d.buildOne(llT, 35, 9, src[ip:], llBase, llBits, llDefault)
		if err != nil {
			return 0, 0, err
		}
		d.ll = t
		ip += n
	}
	if ofT == setRepeat {
		if !d.fseEntropy {
			return 0, 0, zCorruption
		}
	} else {
		t, n, err = d.buildOne(ofT, 31, 8, src[ip:], ofBase, ofBits, ofDefault)
		if err != nil {
			return 0, 0, err
		}
		d.of = t
		ip += n
	}
	if mlT == setRepeat {
		if !d.fseEntropy {
			return 0, 0, zCorruption
		}
	} else {
		t, n, err = d.buildOne(mlT, 52, 9, src[ip:], mlBase, mlBits, mlDefault)
		if err != nil {
			return 0, 0, err
		}
		d.ml = t
		ip += n
	}
	if ip > len(src) {
		return 0, 0, zCorruption
	}
	return nbSeq, ip, nil
}

func (d *zdec) decodeLits(src []byte, dstRoom int) (lit []byte, nRead int, err error) {
	if len(src) < 2 {
		return nil, 0, zCorruption
	}
	enc := int(src[0] & 3)
	room := dstRoom
	if room > zBlockSizeMax {
		room = zBlockSizeMax
	}
	switch enc {
	case setRLE, setBasic:
		lhl := (src[0] >> 2) & 3
		var lh, litSize int
		switch lhl {
		case 0, 2:
			lh = 1
			litSize = int(src[0] >> 3)
		case 1:
			need := 2
			if enc == setRLE {
				need = 3
			}
			if len(src) < need {
				return nil, 0, zCorruption
			}
			lh = 2
			litSize = int(binary.LittleEndian.Uint16(src) >> 4)
		default:
			need := 3
			if enc == setRLE {
				need = 4
			}
			if len(src) < need {
				return nil, 0, zCorruption
			}
			lh = 3
			v := uint32(src[0]) | uint32(src[1])<<8 | uint32(src[2])<<16
			litSize = int(v >> 4)
		}
		if litSize > 0 && dstRoom < 0 {
			return nil, 0, zDstSmall
		}
		if enc == setRLE && litSize > zBlockSizeMax {
			return nil, 0, zCorruption
		}
		if room < litSize {
			return nil, 0, zDstSmall
		}
		if enc == setBasic {
			if lh+litSize > len(src) {
				return nil, 0, zCorruption
			}
			out := make([]byte, litSize)
			copy(out, src[lh:lh+litSize])
			return out, lh + litSize, nil
		}
		if lh >= len(src) {
			return nil, 0, zCorruption
		}
		out := make([]byte, litSize)
		for i := range out {
			out[i] = src[lh]
		}
		return out, lh + 1, nil
	case setRepeat, setCompressed:
		if len(src) < 5 {
			return nil, 0, zCorruption
		}
		if enc == setRepeat && !d.litEntropy {
			return nil, 0, zDictCorrupt
		}
		lhl := (src[0] >> 2) & 3
		lhc := binary.LittleEndian.Uint32(src)
		var lh, litSize, litCSize int
		single := false
		switch lhl {
		case 0, 1:
			single = lhl == 0
			lh = 3
			litSize = int((lhc >> 4) & 0x3FF)
			litCSize = int((lhc >> 14) & 0x3FF)
		case 2:
			lh = 4
			litSize = int((lhc >> 4) & 0x3FFF)
			litCSize = int(lhc >> 18)
		default:
			lh = 5
			litSize = int((lhc >> 4) & 0x3FFFF)
			litCSize = int(lhc>>22) + int(src[4])<<10
		}
		if litSize > zBlockSizeMax {
			return nil, 0, zCorruption
		}
		if !single && litSize < 6 {
			return nil, 0, zLitHeader
		}
		if lh+litCSize > len(src) {
			return nil, 0, zCorruption
		}
		if room < litSize {
			return nil, 0, zDstSmall
		}
		comp := src[lh : lh+litCSize]
		out := make([]byte, litSize)
		var herr error
		if enc == setCompressed {
			n, e := d.readHufTable(comp)
			if e != nil || n > len(comp) {
				return nil, 0, zCorruption
			}
			comp = comp[n:]
		}
		if len(d.huf) == 0 || d.hufLog == 0 {
			return nil, 0, zCorruption
		}
		if single {
			herr = hufDecode1(out, comp, d.huf, d.hufLog)
		} else {
			herr = hufDecode4(out, comp, d.huf, d.hufLog)
		}
		if herr != nil {
			return nil, 0, zCorruption
		}
		d.litEntropy = true
		return out, lh + litCSize, nil
	default:
		return nil, 0, zCorruption
	}
}

func copyMatch(buf []byte, pos, ml, offset int, prefix []byte) error {
	if ml == 0 {
		return nil
	}
	if offset <= 0 || pos < 0 || pos+ml > len(buf) {
		return zCorruption
	}
	if offset > len(prefix)+pos {
		return zCorruption
	}
	hp := len(prefix) + pos - offset
	if hp >= len(prefix) {
		src := hp - len(prefix)
		if src+ml <= pos {
			copy(buf[pos:pos+ml], buf[src:src+ml])
			return nil
		}
		for i := 0; i < ml; i++ {
			buf[pos+i] = buf[src+i]
		}
		return nil
	}
	if hp+ml <= len(prefix) {
		copy(buf[pos:pos+ml], prefix[hp:hp+ml])
		return nil
	}
	n := len(prefix) - hp
	copy(buf[pos:pos+n], prefix[hp:])
	for i := 0; i < ml-n; i++ {
		buf[pos+n+i] = buf[i]
	}
	return nil
}

func (d *zdec) execSeq(buf []byte, pos int, lit []byte, litPos, ll, ml int, offset uint64) (int, int, error) {
	if ll < 0 || ml < 0 || litPos+ll > len(lit) {
		return 0, 0, zCorruption
	}
	if pos < 0 || ll+ml > len(buf)-pos {
		return 0, 0, zDstSmall
	}
	copy(buf[pos:pos+ll], lit[litPos:litPos+ll])
	pos += ll
	litPos += ll
	if ml == 0 {
		return pos, litPos, nil
	}
	if offset > uint64(^uint(0)>>1) {
		return 0, 0, zCorruption
	}
	if err := copyMatch(buf, pos, ml, int(offset), d.prefix); err != nil {
		if err == zCorruption && pos+ml <= len(buf) && int(offset) > len(d.prefix)+pos {
			return 0, 0, zCorruption
		}
		if c, ok := err.(zcode); ok && c == zCorruption {
			return 0, 0, zCorruption
		}
		return 0, 0, err
	}
	return pos + ml, litPos, nil
}

func decodeOneSeq(br *bitReader, llT, ofT, mlT *fseTable, stLL, stOF, stML *uint32, rep *[3]uint64) (ll, ml int, offset uint64, err error) {
	if int(*stLL) >= len(llT.cell) || int(*stOF) >= len(ofT.cell) || int(*stML) >= len(mlT.cell) {
		return 0, 0, 0, zCorruption
	}
	llC := llT.cell[*stLL]
	ofC := ofT.cell[*stOF]
	mlC := mlT.cell[*stML]
	ll = int(llC.base)
	ml = int(mlC.base)
	ofAdd := uint32(ofC.add)
	if ofAdd > 1 {
		offset = uint64(ofC.base) + br.readBitsFast(ofAdd)
		rep[2] = rep[1]
		rep[1] = rep[0]
		rep[0] = offset
	} else {
		ll0 := uint64(0)
		if llC.base == 0 {
			ll0 = 1
		}
		if ofAdd == 0 {
			offset = rep[ll0]
			rep[1] = rep[1-ll0]
			rep[0] = offset
		} else {
			offset = uint64(ofC.base) + ll0 + br.readBitsFast(1)
			var temp uint64
			if offset == 3 {
				temp = rep[0] - 1
			} else if offset < 3 {
				temp = rep[offset]
			} else {
				return 0, 0, 0, zCorruption
			}
			if temp == 0 {
				temp = 1
			}
			if offset != 1 {
				rep[2] = rep[1]
			}
			rep[1] = rep[0]
			rep[0] = temp
			offset = temp
		}
	}
	if mlC.add > 0 {
		ml += int(br.readBitsFast(uint32(mlC.add)))
	}
	total := int(llC.add) + int(mlC.add) + int(ofC.add)
	if total >= accReloadAt {
		br.reload()
	}
	if llC.add > 0 {
		ll += int(br.readBitsFast(uint32(llC.add)))
	}
	*stLL = uint32(llC.next) + uint32(br.readBits(uint32(llC.nb)))
	*stML = uint32(mlC.next) + uint32(br.readBits(uint32(mlC.nb)))
	*stOF = uint32(ofC.next) + uint32(br.readBits(uint32(ofC.nb)))
	return ll, ml, offset, nil
}

func (d *zdec) decodeSequences(buf []byte, op int, lit, seq []byte, nbSeq int) (int, error) {
	if nbSeq == 0 {
		if len(lit) > len(buf)-op {
			return 0, zDstSmall
		}
		copy(buf[op:], lit)
		return len(lit), nil
	}
	d.fseEntropy = true
	var br bitReader
	if err := br.init(seq); err != nil {
		return 0, zCorruption
	}
	stLL, err := initFSE(&br, &d.ll)
	if err != nil {
		return 0, zCorruption
	}
	stOF, err := initFSE(&br, &d.of)
	if err != nil {
		return 0, zCorruption
	}
	stML, err := initFSE(&br, &d.ml)
	if err != nil {
		return 0, zCorruption
	}
	rep := [3]uint64{uint64(d.rep[0]), uint64(d.rep[1]), uint64(d.rep[2])}
	litPos := 0
	pos := op
	for {
		ll, ml, off, e := decodeOneSeq(&br, &d.ll, &d.of, &d.ml, &stLL, &stOF, &stML, &rep)
		if e != nil {
			return 0, e
		}
		var e2 error
		pos, litPos, e2 = d.execSeq(buf, pos, lit, litPos, ll, ml, off)
		if e2 != nil {
			return 0, e2
		}
		nbSeq--
		if nbSeq == 0 {
			break
		}
		br.reload()
	}
	if br.reload() < bitCompleted {
		return 0, zCorruption
	}
	for i := 0; i < 3; i++ {
		d.rep[i] = uint32(rep[i])
	}
	rest := lit[litPos:]
	if len(rest) > len(buf)-pos {
		return 0, zDstSmall
	}
	copy(buf[pos:], rest)
	return pos + len(rest) - op, nil
}

func (d *zdec) decodeCompressed(buf []byte, op int, src []byte) (int, error) {
	if len(src) > zBlockSizeMax {
		return 0, zSrcSize
	}
	lit, nLit, err := d.decodeLits(src, len(buf)-op)
	if err != nil {
		return 0, err
	}
	src = src[nLit:]
	nbSeq, nSeq, err := d.seqHeaders(src)
	if err != nil {
		return 0, err
	}
	src = src[nSeq:]
	return d.decodeSequences(buf, op, lit, src, nbSeq)
}

type zheader struct {
	size      int
	window    uint64
	blockMax  int
	content   uint64
	known     bool
	dictID    uint32
	checksum  bool
	skippable bool
}

func partialMagicOK(src []byte) bool {
	var h [4]byte
	binary.LittleEndian.PutUint32(h[:], zMagic)
	copy(h[:], src)
	if binary.LittleEndian.Uint32(h[:]) == zMagic {
		return true
	}
	binary.LittleEndian.PutUint32(h[:], zSkippableMagic)
	copy(h[:], src)
	return binary.LittleEndian.Uint32(h[:])&zSkippableMask == zSkippableMagic
}

func headerSizeOf(src []byte) (int, error) {
	if len(src) < 5 {
		return 0, zSrcSize
	}
	fhd := src[4]
	dictID := int(fhd & 3)
	single := (fhd >> 5) & 1
	fcsID := int(fhd >> 6)
	sz := 5
	if single == 0 {
		sz++
	}
	sz += []int{0, 1, 2, 4}[dictID]
	sz += []int{0, 2, 4, 8}[fcsID]
	if single == 1 && fcsID == 0 {
		sz++
	}
	return sz, nil
}

func parseZFrame(src []byte) (zheader, error) {
	if len(src) < 5 {
		return zheader{}, zSrcSize
	}
	magic := binary.LittleEndian.Uint32(src)
	if magic != zMagic {
		if magic&zSkippableMask == zSkippableMagic {
			if len(src) < 8 {
				return zheader{}, zSrcSize
			}
			return zheader{size: 8, skippable: true, content: uint64(binary.LittleEndian.Uint32(src[4:]))}, nil
		}
		return zheader{}, zPrefix
	}
	hs, err := headerSizeOf(src)
	if err != nil {
		return zheader{}, err
	}
	if len(src) < hs {
		return zheader{}, zSrcSize
	}
	fhd := src[4]
	if fhd&0x08 != 0 {
		return zheader{}, zUnsupported
	}
	dictCode := int(fhd & 3)
	checksum := (fhd>>2)&1 == 1
	single := (fhd>>5)&1 == 1
	fcsID := int(fhd >> 6)
	pos := 5
	var window uint64
	if !single {
		wl := src[pos]
		pos++
		wlog := int(wl>>3) + zWindowLogMin
		if wlog > zWindowLogMax {
			return zheader{}, zWindow
		}
		window = uint64(1) << uint(wlog)
		window += (window >> 3) * uint64(wl&7)
	}
	var dictID uint32
	switch dictCode {
	case 1:
		dictID = uint32(src[pos])
		pos++
	case 2:
		dictID = uint32(binary.LittleEndian.Uint16(src[pos:]))
		pos += 2
	case 3:
		dictID = binary.LittleEndian.Uint32(src[pos:])
		pos += 4
	}
	known := true
	var content uint64
	switch fcsID {
	case 0:
		if single {
			content = uint64(src[pos])
		} else {
			known = false
		}
	case 1:
		content = uint64(binary.LittleEndian.Uint16(src[pos:])) + 256
	case 2:
		content = uint64(binary.LittleEndian.Uint32(src[pos:]))
	case 3:
		content = binary.LittleEndian.Uint64(src[pos:])
	}
	if single {
		window = content
	}
	bmax := int(window)
	if window > zBlockSizeMax {
		bmax = zBlockSizeMax
	}
	return zheader{
		size: hs, window: window, blockMax: bmax,
		content: content, known: known, dictID: dictID, checksum: checksum,
	}, nil
}

// peekHeader reports the header when the buffer holds it. need > 0 means the
// caller must treat the frame as incomplete (guest output 0) unless err != nil.
func peekHeader(src []byte) (h zheader, need int, err error) {
	if len(src) < 5 {
		if len(src) > 0 && !partialMagicOK(src) {
			return zheader{}, 0, zPrefix
		}
		return zheader{}, 5, nil
	}
	magic := binary.LittleEndian.Uint32(src)
	if magic != zMagic {
		if magic&zSkippableMask == zSkippableMagic {
			if len(src) < 8 {
				return zheader{}, 8, nil
			}
			h, err = parseZFrame(src)
			return h, 0, err
		}
		return zheader{}, 0, zPrefix
	}
	hs, err := headerSizeOf(src)
	if err != nil {
		return zheader{}, 0, err
	}
	if len(src) < hs {
		return zheader{}, hs, nil
	}
	h, err = parseZFrame(src)
	return h, 0, err
}

func frameCompressedSize(src []byte) (int, error) {
	if len(src) >= 8 && binary.LittleEndian.Uint32(src)&zSkippableMask == zSkippableMagic {
		sz := int(binary.LittleEndian.Uint32(src[4:]))
		if sz < 0 || sz+8 < sz || sz+8 > len(src) {
			return 0, zSrcSize
		}
		return sz + 8, nil
	}
	h, err := parseZFrame(src)
	if err != nil {
		return 0, err
	}
	if len(src) < h.size {
		return 0, zSrcSize
	}
	ip := h.size
	for {
		if ip+3 > len(src) {
			return 0, zSrcSize
		}
		_, _, cSize, _, err := readBlock(src[ip:])
		if err != nil {
			return 0, err
		}
		if 3+cSize > len(src)-ip {
			return 0, zSrcSize
		}
		last := src[ip]&1 == 1
		ip += 3 + cSize
		if last {
			break
		}
	}
	if h.checksum {
		if ip+4 > len(src) {
			return 0, zSrcSize
		}
		ip += 4
	}
	return ip, nil
}

func readBlock(src []byte) (last bool, typ, cSize, orig int, err error) {
	if len(src) < 3 {
		return false, 0, 0, 0, zSrcSize
	}
	h := uint32(src[0]) | uint32(src[1])<<8 | uint32(src[2])<<16
	last = h&1 == 1
	typ = int((h >> 1) & 3)
	orig = int(h >> 3)
	cSize = orig
	if typ == btReserved {
		return false, 0, 0, 0, zCorruption
	}
	if typ == btRLE {
		cSize = 1
	}
	return last, typ, cSize, orig, nil
}

func decodeFrame(src, prefix []byte, dstCap int, h zheader, strict bool) ([]byte, error) {
	if h.dictID != 0 {
		return nil, zDictWrong
	}
	d := newZDec(prefix)
	// strict: output must fit in dstCap. loose: each block may use blockMax bytes of scratch.
	var storage []byte
	op := 0
	if strict {
		storage = make([]byte, dstCap)
	}
	pos := h.size
	for {
		if pos+3 > len(src) {
			if strict {
				return nil, zSrcSize
			}
			if op == 0 {
				return nil, nil
			}
			return storage[:op], nil
		}
		last, typ, cSize, orig, err := readBlock(src[pos:])
		if err != nil {
			return nil, err
		}
		if !strict && cSize > h.blockMax {
			return nil, zCorruption
		}
		pos += 3
		if cSize > len(src)-pos {
			if strict {
				return nil, zSrcSize
			}
			if typ == btRaw {
				n := len(src) - pos
				if op+n > dstCap {
					return nil, nil
				}
				if strict {
					copy(storage[op:], src[pos:pos+n])
				} else {
					storage = append(storage, src[pos:pos+n]...)
				}
				op += n
			}
			if op == 0 {
				return nil, nil
			}
			return storage[:op], nil
		}
		payload := src[pos : pos+cSize]
		pos += cSize
		var n int
		switch typ {
		case btRaw:
			n = len(payload)
			if strict {
				if op+n > len(storage) {
					return nil, zDstSmall
				}
				copy(storage[op:], payload)
			} else if n > h.blockMax {
				return nil, zCorruption
			}
		case btRLE:
			if len(payload) != 1 {
				return nil, zCorruption
			}
			n = orig
			if n < 0 {
				return nil, zCorruption
			}
			if !strict && n > h.blockMax {
				return nil, zCorruption
			}
			if strict {
				if op+n > len(storage) {
					return nil, zDstSmall
				}
				for i := 0; i < n; i++ {
					storage[op+i] = payload[0]
				}
			}
		case btCompressed:
			room := dstCap - op
			if !strict {
				room = h.blockMax
			}
			if room < 0 {
				return nil, nil
			}
			tmpBase := storage
			tmpOp := op
			if !strict {
				tmpBase = make([]byte, op+room)
				copy(tmpBase, storage)
			} else if op+room > len(storage) {
				return nil, zDstSmall
			}
			n, err = d.decodeCompressed(tmpBase, tmpOp, payload)
			if err != nil {
				if !strict && err == zDstSmall {
					return nil, zCorruption
				}
				return nil, err
			}
			if !strict {
				if n > h.blockMax {
					return nil, zCorruption
				}
				storage = tmpBase
			}
		default:
			return nil, zCorruption
		}
		if !strict && typ != btCompressed {
			if op+n > dstCap {
				return nil, nil
			}
			if typ == btRaw {
				storage = append(storage, payload...)
			} else if typ == btRLE {
				b := payload[0]
				for i := 0; i < n; i++ {
					storage = append(storage, b)
				}
			}
		}
		if strict && typ != btCompressed && typ != btRaw && typ != btRLE {
			return nil, zCorruption
		}
		if !strict && op+n > dstCap {
			return nil, nil
		}
		if strict && typ == btCompressed {
			// n already written into storage
		}
		op += n
		if last {
			if h.known && uint64(op) != h.content {
				return nil, zCorruption
			}
			if h.checksum {
				if pos+4 > len(src) {
					if strict {
						return nil, zChecksum
					}
					return storage[:op], nil
				}
				got := binary.LittleEndian.Uint32(src[pos:])
				if uint32(xxh64(storage[:op])) != got {
					return nil, zChecksum
				}
			}
			return storage[:op], nil
		}
	}
}

// decodePrefix is ZSTD_decompressStream after ZSTD_DCtx_refPrefix(raw).
// A non-positive guest code is fmt.Errorf("%w: zstd %d", errGuest, code).
func decodePrefix(patch, prefix []byte, dstCap, windowLog int) ([]byte, error) {
	if dstCap <= 0 {
		return nil, errGuest
	}
	maxWindow := 1 << zWindowDefault
	if windowLog > 0 {
		if windowLog < zWindowLogMin || windowLog > zWindowLogMax {
			return nil, zstdGuest(-int(zParamBound) - 1)
		}
		maxWindow = 1 << uint(windowLog)
	}
	out, err := decompressPrefix(patch, prefix, dstCap, maxWindow)
	if err != nil {
		return nil, guestFrom(err)
	}
	if len(out) == 0 {
		return nil, zstdGuest(0)
	}
	return out, nil
}

func decompressPrefix(src, prefix []byte, dstCap, maxWindow int) ([]byte, error) {
	h, need, err := peekHeader(src)
	if err != nil {
		return nil, err
	}
	if need > 0 || h.skippable {
		return nil, nil
	}
	if h.known && uint64(dstCap) >= h.content {
		n, ferr := frameCompressedSize(src)
		if ferr == nil && n <= len(src) {
			return decodeFrame(src[:n], prefix, dstCap, h, true)
		}
	}
	ws := h.window
	if ws < 1<<zWindowLogMin {
		ws = 1 << zWindowLogMin
	}
	if ws > uint64(maxWindow) {
		return nil, zWindow
	}
	return decodeFrame(src, prefix, dstCap, h, false)
}

func xxh64(b []byte) uint64 {
	var (
		p1 uint64 = 0x9E3779B185EBCA87
		p2 uint64 = 0xC2B2AE3D27D4EB4F
		p3 uint64 = 0x165667B19E3779F9
		p4 uint64 = 0x85EBCA77C2B2AE63
		p5 uint64 = 0x27D4EB2F165667C5
	)
	n := len(b)
	var h uint64
	i := 0
	if n >= 32 {
		v1 := p1 + p2
		v2 := p2
		v3 := uint64(0)
		v4 := -p1
		for i+32 <= n {
			v1 = xxhRound(v1, binary.LittleEndian.Uint64(b[i:]))
			v2 = xxhRound(v2, binary.LittleEndian.Uint64(b[i+8:]))
			v3 = xxhRound(v3, binary.LittleEndian.Uint64(b[i+16:]))
			v4 = xxhRound(v4, binary.LittleEndian.Uint64(b[i+24:]))
			i += 32
		}
		h = bits.RotateLeft64(v1, 1) + bits.RotateLeft64(v2, 7) + bits.RotateLeft64(v3, 12) + bits.RotateLeft64(v4, 18)
		h = xxhMerge(h, v1)
		h = xxhMerge(h, v2)
		h = xxhMerge(h, v3)
		h = xxhMerge(h, v4)
	} else {
		h = p5
	}
	h += uint64(n)
	p := b[i:]
	for len(p) >= 8 {
		h ^= xxhRound(0, binary.LittleEndian.Uint64(p))
		h = bits.RotateLeft64(h, 27)*p1 + p4
		p = p[8:]
	}
	if len(p) >= 4 {
		h ^= uint64(binary.LittleEndian.Uint32(p)) * p1
		h = bits.RotateLeft64(h, 23)*p2 + p3
		p = p[4:]
	}
	for len(p) > 0 {
		h ^= uint64(p[0]) * p5
		h = bits.RotateLeft64(h, 11) * p1
		p = p[1:]
	}
	h ^= h >> 33
	h *= p2
	h ^= h >> 29
	h *= p3
	h ^= h >> 32
	return h
}

func xxhRound(acc, in uint64) uint64 {
	acc += in * 0xC2B2AE3D27D4EB4F
	acc = bits.RotateLeft64(acc, 31)
	acc *= 0x9E3779B185EBCA87
	return acc
}

func xxhMerge(acc, val uint64) uint64 {
	val = xxhRound(0, val)
	acc ^= val
	acc = acc*0x9E3779B185EBCA87 + 0x85EBCA77C2B2AE63
	return acc
}
