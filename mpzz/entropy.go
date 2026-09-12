package mpzz

import (
	"encoding/binary"
	"fmt"
	"io"
)

// readSize reconstructs OGGRE's byte count at 0x10019d10. The low two
// bits give the number of following bytes, not part of the count.
func readSize(r io.Reader) (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:1]); err != nil {
		return 0, err
	}
	n := int(b[0] & 3)
	if _, err := io.ReadFull(r, b[1:1+n]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b[:]) >> 2, nil
}

func readFrame(r io.Reader) ([]byte, error) {
	n, err := readSize(r)
	if err != nil {
		return nil, err
	}
	if n > 512<<20 {
		return nil, fmt.Errorf("mpzz: frame exceeds 512 MiB: %d", n)
	}
	b := make([]byte, int(n))
	_, err = io.ReadFull(r, b)
	return b, err
}

// rangeDecoder follows 0x1001e540 and 0x100038c0 in the fully unpacked
// OGGRE image. Separate frames have separate arithmetic states; models
// can be shared between those states.
type rangeDecoder struct {
	data        []byte
	pos         int
	tail        int
	code, width uint32
	err         error
}

func newRange(data []byte) *rangeDecoder {
	r := &rangeDecoder{data: data, width: 0xffffffff}
	if len(data) == 0 {
		r.err = io.ErrUnexpectedEOF
		return r
	}
	for i := 0; i < min(4, len(data)); i++ {
		r.code |= uint32(data[i]) << (24 - 8*i)
	}
	r.pos = min(4, len(data))
	return r
}

func (r *rangeDecoder) bit(p *uint16) uint32 {
	if r.err != nil {
		return 0
	}
	for r.width < 1<<24 {
		var next byte
		if r.pos >= len(r.data) {
			// 0x1000390a leaves the input cursor on the last byte at
			// frame exhaustion. Arithmetic lookahead repeats that byte.
			// A 32-bit code register needs at most four lookahead bytes.
			if r.tail == 4 {
				r.err = io.ErrUnexpectedEOF
				return 0
			}
			r.tail++
			next = r.data[len(r.data)-1]
		} else {
			next = r.data[r.pos]
			r.pos++
		}
		r.code = r.code<<8 | uint32(next)
		r.width <<= 8
	}
	f := uint32(*p)
	bound := (r.width >> 16) * f
	if r.code >= bound {
		r.code -= bound
		r.width -= bound
		*p = uint16(f - (f >> 6))
		return 1
	}
	r.width = bound
	*p = uint16(f + ((65535 - f) >> 6))
	return 0
}

// Each model occupies 0x2000 bytes in the original image: two predictors,
// three history bytes, and probabilities starting at byte 12.
type integerModel struct {
	value, delta          uint32
	nonzero, sign, length uint32
	prob                  [4090]uint16
}

func (m *integerModel) reset() {
	m.value, m.delta, m.nonzero, m.sign, m.length = 0, 0, 0, 0, 0
	for i := range m.prob {
		m.prob[i] = 0x8000
	}
}

func (m *integerModel) bit(r *rangeDecoder, offset uint32) uint32 {
	if offset < 12 || offset&1 != 0 || offset >= 0x2000 {
		r.err = fmt.Errorf("mpzz: invalid model offset %#x", offset)
		return 0
	}
	return r.bit(&m.prob[(offset-12)/2])
}

// integer decodes a nonzero flag, an optional sign, the magnitude's bit
// length, and its low bits. Once the low-bit tree reaches its final level,
// subsequent bits reuse the same probability. grouped selects a separate
// low-bit tree for each magnitude length.
func (m *integerModel) integer(r *rangeDecoder, lengthBits, signBits, lowBits uint32, grouped bool) uint32 {
	b := m.bit(r, 12+2*m.nonzero)
	m.nonzero = (2*m.nonzero + b) & 3
	if b == 0 {
		m.length = 0
		return 0
	}
	sign := uint32(0)
	if signBits != 0 {
		sign = m.bit(r, 20+2*m.sign)
		m.sign = (2*m.sign + sign) & ((1 << signBits) - 1)
	}
	base := uint32(0x34) + m.length<<(lengthBits+1)
	v := uint32(1)
	for range lengthBits {
		v = 2*v + m.bit(r, base+2*v)
	}
	n := v & ((1 << lengthBits) - 1)
	m.length = n + 1
	base = 0x834
	if grouped {
		base += n << (lowBits + 1)
	}
	v, ctx := uint32(1), uint32(1)
	for range n {
		b = m.bit(r, base+2*ctx)
		v = 2*v + b
		if ctx < 1<<(lowBits-1) {
			ctx = (2*ctx + b) & ((1 << lowBits) - 1)
		}
	}
	return (v ^ -sign) + sign
}

func (m *integerModel) predict(v uint32, order int) uint32 {
	if order >= 2 {
		m.delta += v
		v = m.delta
	}
	if order >= 1 {
		m.value += v
		v = m.value
	}
	return v
}
