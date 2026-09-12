package magic2

import (
	"encoding/binary"
	"io"
)

const ansLower = 1 << 23

type entropy struct {
	data []byte
	off  int
	x    uint32
	err  error
}

func newEntropy(b []byte) *entropy {
	r := &entropy{data: b, off: 4}
	if len(b) < 4 {
		r.err = io.ErrUnexpectedEOF
		return r
	}
	r.x = binary.LittleEndian.Uint32(b)
	return r
}
func (r *entropy) normalize() {
	for r.x < ansLower && r.err == nil {
		if r.off >= len(r.data) {
			r.err = io.ErrUnexpectedEOF
			return
		}
		r.x = r.x<<8 | uint32(r.data[r.off])
		r.off++
	}
}
func (r *entropy) bit(p *uint16, scale, shift uint) int {
	v := uint32(*p)
	lo := r.x & ((1 << scale) - 1)
	hi := r.x >> scale
	b := 0
	if lo < v {
		r.x = hi*v + lo
		*p += uint16(((1 << scale) - v) >> shift)
	} else {
		r.x -= v * (hi + 1)
		*p -= uint16(v >> shift)
		b = 1
	}
	r.normalize()
	return b
}
func (r *entropy) raw(n uint) int {
	v := r.x & ((1 << n) - 1)
	r.x >>= n
	r.normalize()
	return int(v)
}
func uniform(n int) []uint16 {
	c := make([]uint16, n+1)
	for i := range c {
		c[i] = uint16(i * 32768 / n)
	}
	return c
}
func adaptCDF(c []uint16, s int, shift uint) {
	n := len(c) - 1
	for i := 0; i < n; i++ {
		target := int((1<<shift)/n) * i
		if i > s {
			target += 32767
		}
		c[i] += uint16(int16(target-int(c[i])) >> shift)
	}
}
func (r *entropy) symbol(c []uint16, shift uint) int {
	s := r.selectSymbol(c)
	adaptCDF(c, s, shift)
	r.normalize()
	return s
}
func (r *entropy) selectSymbol(c []uint16) int {
	lo := r.x & 32767
	s := 0
	for s < len(c)-2 && uint32(c[s+1]) <= lo {
		s++
	}
	r.x = (r.x>>15)*uint32(c[s+1]-c[s]) + lo - uint32(c[s])
	return s
}
func (r *entropy) finish() error {
	if r.err != nil {
		return r.err
	}
	if r.x != ansLower || r.off != len(r.data) {
		return errBitstream
	}
	return nil
}
