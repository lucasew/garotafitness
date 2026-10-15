package mpz

import (
	"encoding/binary"
	"fmt"
	"math"
)

const memorySize = 64 << 20

// state is the lifted MPZ 5.4.5.1 machine from stream/mpz/guest/mpz-native.h.
type state struct {
	eax, ebx, ecx, edx, esi, edi, ebp, esp uint32
	cf, zf, sf, of, fs0                    uint32
	flushing                               uint32
	memory                                 []byte
	data                                   []byte
	input                                  []byte
	output                                 []byte
	inlen, inpos                           uint32
	outcap, outpos                         uint32
	heap                                   uint32
}

type decodeFail struct {
	why    string
	detail uint32
}

func (e decodeFail) Error() string {
	return fmt.Sprintf("mpz: %s %08x", e.why, e.detail)
}

// flag is a setcc result: 1 when b is true.
func flag(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

func (s *state) fail(why string, detail uint32) {
	panic(decodeFail{why: why, detail: detail})
}

func (s *state) mem(a, n uint32) []byte {
	if a >= 0x10024000 && a <= 0x10039000 && n <= 0x10039000-a {
		off := a - 0x10024000
		return s.data[off : off+n]
	}
	if a >= 0x1000 && a <= memorySize && n <= memorySize-a {
		return s.memory[a : a+n]
	}
	s.fail("address", a)
	return nil
}

func (s *state) rd8(a uint32) uint32 { return uint32(s.mem(a, 1)[0]) }

func (s *state) rd16(a uint32) uint32 {
	b := s.mem(a, 2)
	return uint32(binary.LittleEndian.Uint16(b))
}

func (s *state) rd32(a uint32) uint32 {
	b := s.mem(a, 4)
	return binary.LittleEndian.Uint32(b)
}

func (s *state) wr8(a, v uint32) { s.mem(a, 1)[0] = byte(v) }

func (s *state) wr16(a, v uint32) {
	binary.LittleEndian.PutUint16(s.mem(a, 2), uint16(v))
}

func (s *state) wr32(a, v uint32) {
	binary.LittleEndian.PutUint32(s.mem(a, 4), v)
}

func (s *state) push(v uint32) {
	s.esp -= 4
	s.wr32(s.esp, v)
}

func (s *state) pop() uint32 {
	v := s.rd32(s.esp)
	s.esp += 4
	return v
}

func (s *state) alloc(size uint32) uint32 {
	a := s.heap
	if size > memorySize/2 || a > memorySize/2-size {
		s.fail("allocation", size)
	}
	s.heap = (a + size + 15) &^ 15
	return a
}

func (s *state) alu(a, b uint32, width, op int) uint32 {
	var mask, sign uint32
	if width == 32 {
		mask = ^uint32(0)
		sign = 1 << 31
	} else {
		mask = (1 << width) - 1
		sign = 1 << (width - 1)
	}
	a &= mask
	b &= mask
	var v uint32
	switch op {
	case 0, 5:
		extra := uint64(0)
		if op == 5 {
			extra = uint64(s.cf)
		}
		q := uint64(a) + uint64(b) + extra
		v = uint32(q) & mask
		if q>>uint(width) != 0 {
			s.cf = 1
		} else {
			s.cf = 0
		}
		if (^(a^b)&(a^v))&sign != 0 {
			s.of = 1
		} else {
			s.of = 0
		}
	case 1:
		v = (a - b) & mask
		if a < b {
			s.cf = 1
		} else {
			s.cf = 0
		}
		if ((a^b)&(a^v))&sign != 0 {
			s.of = 1
		} else {
			s.of = 0
		}
	case 2:
		v = a ^ b
		s.cf, s.of = 0, 0
	case 3:
		v = a | b
		s.cf, s.of = 0, 0
	default:
		v = a & b
		s.cf, s.of = 0, 0
	}
	if v == 0 {
		s.zf = 1
	} else {
		s.zf = 0
	}
	if v&sign != 0 {
		s.sf = 1
	} else {
		s.sf = 0
	}
	return v
}

func (s *state) shift(a, n uint32, op int) uint32 {
	n &= 31
	if n == 0 {
		return a
	}
	var v uint32
	if op == 2 {
		v = a << n
		s.cf = (a >> (32 - n)) & 1
		s.of = ((v >> 31) ^ s.cf) & 1
	} else {
		if op == 0 {
			v = uint32(int32(a) >> n)
		} else {
			v = a >> n
		}
		s.cf = (a >> (n - 1)) & 1
		if op == 1 {
			s.of = a >> 31
		} else {
			s.of = 0
		}
	}
	if v == 0 {
		s.zf = 1
	} else {
		s.zf = 0
	}
	s.sf = v >> 31
	return v
}

func (s *state) shrd(a, b, n uint32) uint32 {
	n &= 31
	if n == 0 {
		return a
	}
	v := (a >> n) | (b << (32 - n))
	s.cf = (a >> (n - 1)) & 1
	s.of = (a ^ v) >> 31
	if v == 0 {
		s.zf = 1
	} else {
		s.zf = 0
	}
	s.sf = v >> 31
	return v
}

func (s *state) divide(b uint32, sign int) {
	if b == 0 {
		s.fail("zero divisor", 0)
	}
	a := uint64(s.edx)<<32 | uint64(s.eax)
	if sign != 0 {
		x := int64(a)
		y := int64(int32(b))
		if x == math.MinInt64 && y == -1 {
			s.fail("division overflow", 0)
		}
		z := x / y
		if z < math.MinInt32 || z > math.MaxInt32 {
			s.fail("division overflow", 0)
		}
		s.eax = uint32(int32(z))
		s.edx = uint32(int32(x % y))
		return
	}
	q := a / uint64(b)
	r := a % uint64(b)
	if q > math.MaxUint32 {
		s.fail("division overflow", 0)
	}
	s.eax = uint32(q)
	s.edx = uint32(r)
}

func (s *state) callback(target uint32) {
	switch target {
	case 0x10018e24:
		f10018e24(s)
	case 0x10018e10:
		f10018e10(s)
	default:
		s.hostcall(target)
	}
}

func (s *state) hostcall(f uint32) {
	a := s.rd32(s.esp + 4)
	b := s.rd32(s.esp + 8)
	c := s.rd32(s.esp + 12)
	switch f {
	case 0x1001938c:
	case 0x1001961e:
		s.eax = s.alloc(a)
	case 0x10019d6c:
		if s.inpos >= s.inlen {
			if s.flushing == 0 {
				s.fail("truncated input", s.inpos)
			}
			s.eax = ^uint32(0)
		} else {
			s.eax = uint32(s.input[s.inpos])
			s.inpos++
		}
	case 0x1001960e:
		if s.outpos >= s.outcap {
			s.fail("output overflow", s.outpos)
		}
		s.output[s.outpos] = byte(a)
		s.outpos++
		s.eax = a & 255
	case 0x1001975e:
		if b == 0 {
			s.eax = 0
			break
		}
		want := uint64(b) * uint64(c)
		remain := uint64(s.inlen - s.inpos)
		n := remain
		if want < n {
			n = want
		}
		dst := s.mem(a, uint32(n))
		copy(dst, s.input[s.inpos:s.inpos+uint32(n)])
		s.inpos += uint32(n)
		s.eax = uint32(n / uint64(b))
	case 0x10019d80:
		q := (uint64(b)<<32 | uint64(a)) * (uint64(s.rd32(s.esp+16))<<32 | uint64(c))
		s.eax = uint32(q)
		s.edx = uint32(q >> 32)
		s.esp += 16
	case 0x10019dc0:
		x := int64(uint64(b)<<32 | uint64(a))
		y := int64(uint64(s.rd32(s.esp+16))<<32 | uint64(c))
		if y == 0 || (x == math.MinInt64 && y == -1) {
			s.fail("division overflow", 0)
		}
		quot := x / y
		uq := uint64(quot)
		s.eax = uint32(uq)
		s.edx = uint32(uq >> 32)
		s.esp += 16
	case 0x10019e70:
		s.esp -= s.eax
	case 0x10021724:
		buf := s.mem(a, c)
		clear(buf)
		s.eax = a
	case 0x10021850:
		buf := s.mem(a, c)
		for i := range buf {
			buf[i] = byte(b)
		}
		s.eax = a
	default:
		s.fail("unknown callback", f)
	}
	s.esp += 4
}

// decodeLift runs the lifted 5.4.5.1 decoder. input is the 16-byte header
// followed by the payload. capacity is the header's original size.
func decodeLift(input []byte, capacity uint32) (out []byte, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			if fe, ok := rec.(decodeFail); ok {
				err = fe
				return
			}
			panic(rec)
		}
	}()
	if capacity == 0 || len(input) > math.MaxUint32 {
		return nil, errGuest
	}
	s := &state{
		memory: make([]byte, memorySize),
		data:   append([]byte(nil), tables[:]...),
		input:  input,
		inlen:  uint32(len(input)),
		output: make([]byte, capacity),
		outcap: capacity,
		heap:   0x10000,
		esp:    memorySize - 0x1000,
	}
	s.push(0)
	f100158cc(s)
	st := s.alloc(64)
	s.push(2)
	s.push(1)
	s.push(0)
	s.ecx = st
	f10016dd0(s)
	if s.eax != 0 {
		s.fail("initialization", s.eax)
	}
	var steps uint32
	for {
		s.push(0)
		s.ecx = st
		f100133d0(s)
		steps++
		if steps > capacity+1 {
			s.fail("step limit", steps)
		}
		if s.eax == 0 {
			break
		}
	}
	s.flushing = 1
	s.push(0)
	s.ecx = st
	f10016c7c(s)
	if s.eax != 0 {
		s.fail("flush", s.eax)
	}
	if s.outpos != capacity {
		s.fail("output size mismatch", s.outpos)
	}
	return append([]byte(nil), s.output[:s.outpos]...), nil
}
