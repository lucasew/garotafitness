package xt2png

import "math/bits"

// ArithmeticDecoder from support/arithmetic_coder.cpp. Encoder is unused
// on the reencode path.

var normLUT = [8]uint8{0x33, 0x77, 0xff, 0xff, 0x33, 0x77, 0xff, 0xff}

type arithDec struct {
	br    *bitReader
	low   uint32
	high  uint32
	value uint32
}

func newArith(br *bitReader) *arithDec {
	d := &arithDec{br: br, high: 0x7fffffff}
	d.value = d.br.getReverse(16) << 15
	d.value |= d.br.getReverse(15)
	return d
}

func (d *arithDec) needsNorm() bool {
	return normLUT[d.low>>29]&(1<<(d.high>>29)) != 0
}

func (d *arithDec) normalize() {
	lh := (^d.low) & d.high
	if lh&0x40000000 == 0 {
		if lh&0x20000000 == 0 {
			l := uint32(bitsLeading(shl32(lh, 2) + 3))
			d.low = shl32(d.low, l+1) & 0x7fffffff
			d.high = (shl32(d.high+1, l+1) - 1) & 0x7fffffff
			if l <= 15 {
				d.value = (shl32(d.value, l+1) + d.br.getReverse(l+1)) & 0x7fffffff
			} else {
				d.value = (shl32(d.value, 16) + d.br.getReverse(16)) & 0x7fffffff
				d.value = (shl32(d.value, l-15) + d.br.getReverse(l-15)) & 0x7fffffff
			}
		} else {
			d.low = shl32(d.low, 1) & 0x7fffffff
			d.high = (shl32(d.high, 1) + 1) & 0x7fffffff
			d.value = (shl32(d.value, 1) + d.br.get(1)) & 0x7fffffff
		}
	}
	lh = (^d.low) | d.high
	if lh&0x20000000 == 0 {
		l := uint32(bitsLeading(shl32(lh, 2) + 3))
		d.low = shl32(d.low, l) & 0x3fffffff
		d.high = (shl32(d.high+1, l)-1)&0x3fffffff | 0x40000000
		if l <= 16 {
			d.value = (shl32(d.value, l) + d.br.getReverse(l) - 0x40000000) & 0x7fffffff
		} else {
			d.value = shl32(d.value, 16) + d.br.getReverse(16)
			d.value = (shl32(d.value, l-16) + d.br.getReverse(l-16) - 0x40000000) & 0x7fffffff
		}
	}
}

func bitsLeading(v uint32) int { return bits.LeadingZeros32(v) }

func (d *arithDec) decodeBits(n uint32) uint32 {
	step := shr32((d.high-d.low)+1, n)
	result := (d.value - d.low) / step
	d.low += step * result
	d.high = d.low + step - 1
	d.normalize()
	return result
}

func findIndex(bounds []uint32, n int, val uint32) uint32 {
	for i := n; i > 1; i-- {
		if val >= bounds[i-1] {
			return uint32(i - 1)
		}
	}
	return 0
}

func (d *arithDec) decodeShiftScale(shift uint32, bounds []uint32, n int) uint32 {
	step := shr32((d.high-d.low)+1, shift)
	val := (d.value - d.low) / step
	result := findIndex(bounds, n, val)
	d.high = d.low + step*bounds[result+1] - 1
	d.low += step * bounds[result]
	if d.needsNorm() {
		d.normalize()
	}
	return result
}
