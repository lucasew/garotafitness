package mpzz

import (
	"fmt"
	"math/bits"
)

type bitWriter struct {
	data []byte
	n    uint
}

func (w *bitWriter) write(v uint32, n uint) {
	for i := uint(0); i < n; i++ {
		if w.n&7 == 0 {
			w.data = append(w.data, 0)
		}
		w.data[len(w.data)-1] |= byte(v&1) << (w.n & 7)
		v >>= 1
		w.n++
	}
}

type setupDecoder struct {
	r      *rangeDecoder
	models []integerModel
	probs  []uint16
	w      bitWriter
	cache  *bookCache
}

func (s *setupDecoder) integer(slot int, n, sign, low uint32, grouped bool, order int, outputBits uint) uint32 {
	m := &s.models[slot]
	v := m.predict(m.integer(s.r, n, sign, low, grouped), order)
	s.w.write(v, outputBits)
	return v
}

func (s *setupDecoder) tree(base, n, outputBits uint) uint32 {
	v := uint32(1)
	for range n {
		v = 2*v + s.r.bit(&s.probs[base+uint(v)])
	}
	v &= (1 << n) - 1
	s.w.write(v, outputBits)
	return v
}

// 0x1000e0b0 stores the sign, exponent delta, and mantissa delta of the
// Vorbis float representation. Preserve its bits without float conversion.
func (s *setupDecoder) floatBits(which int) uint32 {
	sign := s.r.bit(&s.probs[0x2250+which])
	m := &s.models[50+which]
	exponent := m.predict(m.integer(s.r, 4, 1, 2, false), 1)
	m = &s.models[48]
	mantissa := m.predict(m.integer(s.r, 5, 1, 6, false), 1)
	v := sign<<31 + exponent<<21 + mantissa
	s.w.write(v, 32)
	return v
}

type codebook struct {
	dim            int
	lengths        []byte
	lookup         int
	codes          []uint32
	quant          []int
	delta          uint32
	context, radix int
	symbols        map[int]int
}

type floorClass struct {
	dim, sub, master int
	books            []int
}
type floorConfig struct {
	partitions []int
	classes    []floorClass
	mult       int
	x          []int
}
type residueConfig struct {
	kind, begin, end, size, classbook int
	books                             [][8]int
}
type mappingConfig struct {
	mux, floors, residues []int
	coupling              [][2]int
}
type modeConfig struct {
	large   bool
	mapping int
}
type vorbisSetup struct {
	books    []codebook
	floors   []floorConfig
	residues []residueConfig
	mappings []mappingConfig
	modes    []modeConfig
}

func (s *setupDecoder) config(channels int, books []codebook) (vorbisSetup, error) {
	c := vorbisSetup{books: books}
	count := func(slot int) int { return int(s.integer(slot, 3, 1, 2, false, 1, 6)) + 1 }
	fail := func(what string) (vorbisSetup, error) {
		return c, fmt.Errorf("mpzz: invalid setup %s at compressed byte %d", what, s.r.pos)
	}
	nt := count(24)
	if nt < 1 || nt > 64 {
		return fail("time count")
	}
	for range nt {
		s.w.write(0, 16)
	}
	nf := int(s.integer(25, 3, 1, 4, false, 1, 6)) + 1
	if nf < 1 || nf > 64 {
		return fail("floor count")
	}
	for range nf {
		if s.tree(0x2226, 1, 16) != 1 {
			return fail("floor type")
		}
		f := floorConfig{}
		np := int(s.integer(26, 3, 1, 4, false, 1, 5))
		if np < 0 || np > 31 {
			return fail("floor partitions")
		}
		maxClass := -1
		for range np {
			cl := int(s.integer(27, 2, 1, 2, false, 1, 4))
			if cl < 0 || cl > 15 {
				return fail("floor class")
			}
			f.partitions = append(f.partitions, cl)
			maxClass = max(maxClass, cl)
		}
		for range maxClass + 1 {
			cl := floorClass{dim: int(s.tree(0x2228, 3, 3)) + 1, sub: int(s.tree(0x2230, 2, 2)), master: -1}
			if cl.sub > 0 {
				cl.master = int(s.integer(28, 3, 1, 4, false, 1, 8))
				if cl.master < 0 || cl.master >= len(books) {
					return fail("floor masterbook")
				}
			}
			for range 1 << cl.sub {
				book := int(s.integer(29, 3, 1, 4, true, 1, 8)) - 1
				if book < -1 || book >= len(books) {
					return fail("floor subbook")
				}
				cl.books = append(cl.books, book)
			}
			f.classes = append(f.classes, cl)
		}
		f.mult = int(s.tree(0x2234, 2, 2)) + 1
		xbits := s.tree(0x2238, 4, 4)
		f.x = []int{0, 1 << xbits}
		for _, cl := range f.partitions {
			for range f.classes[cl].dim {
				x := int(s.integer(30, 4, 1, 4, false, 1, uint(xbits)))
				if x < 0 || x >= 1<<xbits {
					return fail("floor coordinate")
				}
				for _, prev := range f.x {
					if x == prev {
						return fail("duplicate floor coordinate")
					}
				}
				f.x = append(f.x, x)
			}
		}
		c.floors = append(c.floors, f)
	}
	nr := count(31)
	if nr < 1 || nr > 64 {
		return fail("residue count")
	}
	for range nr {
		r := residueConfig{kind: int(s.tree(0x2248, 2, 16))}
		r.begin = int(s.integer(32, 4, 1, 4, false, 1, 24))
		r.end = int(s.integer(33, 4, 1, 4, false, 1, 24))
		r.size = int(s.integer(34, 4, 1, 1, true, 1, 24)) + 1
		nc := int(s.integer(35, 3, 1, 4, false, 1, 6)) + 1
		s.models[37].value = s.models[29].value
		s.models[37].delta = 0
		r.classbook = int(s.integer(37, 3, 1, 4, true, 2, 8))
		if r.kind > 2 || r.begin < 0 || r.end < r.begin || r.size < 1 || nc < 1 || nc > 64 || r.classbook < 0 || r.classbook >= len(books) {
			return fail("residue")
		}
		cascade := make([]uint32, nc)
		for j := range cascade {
			m := &s.models[36]
			v := m.predict(m.integer(s.r, 3, 1, 2, false), 1)
			if v > 255 {
				return fail("residue cascade")
			}
			cascade[j] = v
			s.w.write(v&7, 3)
			if v > 7 {
				s.w.write(1, 1)
				s.w.write(v>>3, 5)
			} else {
				s.w.write(0, 1)
			}
		}
		for _, v := range cascade {
			row := [8]int{-1, -1, -1, -1, -1, -1, -1, -1}
			for k := range row {
				if v>>k&1 != 0 {
					row[k] = int(s.integer(37, 3, 1, 4, true, 2, 8))
					if row[k] < 0 || row[k] >= len(books) {
						return fail("residue book")
					}
				}
			}
			r.books = append(r.books, row)
		}
		c.residues = append(c.residues, r)
	}
	nm := count(38)
	if nm < 1 || nm > 64 {
		return fail("mapping count")
	}
	for range nm {
		s.w.write(0, 16)
		m := mappingConfig{mux: make([]int, channels)}
		model := &s.models[39]
		sub := int(model.predict(model.integer(s.r, 3, 1, 2, false), 1))
		if sub < 0 || sub > 16 {
			return fail("mapping submaps")
		}
		if sub > 0 {
			s.w.write(1, 1)
			s.w.write(uint32(sub-1), 4)
		} else {
			s.w.write(0, 1)
			sub = 1
		}
		model = &s.models[40]
		nc := int(model.predict(model.integer(s.r, 3, 1, 2, false), 1))
		if nc < 0 || nc > 256 {
			return fail("mapping coupling")
		}
		if nc > 0 {
			s.w.write(1, 1)
			s.w.write(uint32(nc-1), 8)
		} else {
			s.w.write(0, 1)
		}
		for range nc {
			n := uint(bits.Len(uint(channels - 1)))
			mag := int(s.integer(41, 2, 1, 1, false, 1, n))
			angle := int(s.integer(41, 2, 1, 1, false, 1, n))
			if mag < 0 || mag >= channels || angle < 0 || angle >= channels || mag == angle {
				return fail("coupling channels")
			}
			m.coupling = append(m.coupling, [2]int{mag, angle})
		}
		s.w.write(0, 2)
		if sub > 1 {
			for j := range m.mux {
				m.mux[j] = int(s.integer(42, 2, 1, 2, false, 1, 4))
				if m.mux[j] < 0 || m.mux[j] >= sub {
					return fail("mapping mux")
				}
			}
		}
		for range sub {
			if s.integer(43, 3, 1, 2, false, 1, 8) != 0 {
				return fail("mapping time")
			}
			f := int(s.integer(44, 3, 1, 2, false, 1, 8))
			r := int(s.integer(45, 3, 1, 2, false, 1, 8))
			if f < 0 || f >= len(c.floors) || r < 0 || r >= len(c.residues) {
				return fail("mapping floor or residue")
			}
			m.floors = append(m.floors, f)
			m.residues = append(m.residues, r)
		}
		c.mappings = append(c.mappings, m)
	}
	nmode := count(46)
	if nmode < 1 || nmode > 64 {
		return fail("mode count")
	}
	for range nmode {
		m := modeConfig{large: s.tree(0x224c, 1, 1) != 0}
		s.w.write(0, 32)
		m.mapping = int(s.integer(47, 3, 1, 2, false, 1, 8))
		if m.mapping < 0 || m.mapping >= len(c.mappings) {
			return fail("mode mapping")
		}
		c.modes = append(c.modes, m)
	}
	s.w.write(1, 1)
	return c, s.r.err
}

// lookupCount is floor(entries^(1/dim)), computed with integer arithmetic.
func lookupCount(entries, dim int) int {
	lo, hi := 1, entries
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		v := 1
		for i := 0; i < dim && v <= entries; i++ {
			if v > entries/mid {
				v = entries + 1
				break
			}
			v *= mid
		}
		if v <= entries {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// 0x100053cf: reconstruct setup codebooks, retaining the code lengths used
// later to write the original Huffman codewords into audio packets.
func (s *setupDecoder) codebooks() ([]codebook, error) {
	s.w.write(5, 8)
	for _, b := range []byte("vorbis") {
		s.w.write(uint32(b), 8)
	}
	n := s.integer(16, 3, 1, 4, true, 1, 8) + 1
	if n > 256 {
		return nil, fmt.Errorf("mpzz: %d codebooks", n)
	}
	books := make([]codebook, n)
	for i := range books {
		s.w.write(0x564342, 24)
		dim := s.integer(17, 4, 2, 4, false, 1, 16)
		entries := s.integer(18, 5, 0, 6, true, 0, 24)
		if dim == 0 || dim > 65535 || entries == 0 || entries > 1<<20 {
			return nil, fmt.Errorf("mpzz: codebook %d dimensions %d entries %d", i, dim, entries)
		}
		book := &books[i]
		book.dim = int(dim)
		book.lengths = make([]byte, entries)
		ordered := s.tree(0x2218, 1, 1)
		sparse := uint32(0)
		if ordered == 0 {
			sparse = s.tree(0x221a, 1, 1)
		}
		if ordered != 0 {
			length := s.integer(19, 3, 0, 4, true, 0, 5) + 1
			for filled := 0; filled < int(entries); length++ {
				if length > 32 {
					return nil, fmt.Errorf("mpzz: codebook %d length exceeds 32", i)
				}
				count := int(s.integer(20, 4, 0, 4, true, 0, uint(bits.Len32(entries-uint32(filled)))))
				if count > int(entries)-filled {
					return nil, fmt.Errorf("mpzz: codebook %d length run exceeds entries", i)
				}
				for j := 0; j < count; j++ {
					book.lengths[filled+j] = byte(length)
				}
				filled += count
			}
		} else {
			prev := uint(2)
			for j := range book.lengths {
				if sparse != 0 {
					prev = uint(s.tree(0x221c+prev, 1, 1))
				}
				if prev == 0 {
					continue
				}
				length := s.integer(21, 3, 1, 4, true, 1, 5) + 1
				if length > 32 {
					return nil, fmt.Errorf("mpzz: codebook %d symbol %d length %d", i, j, length)
				}
				book.lengths[j] = byte(length)
			}
		}
		book.lookup = int(s.tree(0x2220, 2, 4))
		if book.lookup > 2 {
			return nil, fmt.Errorf("mpzz: codebook %d lookup %d", i, book.lookup)
		}
		if book.lookup != 0 {
			s.floatBits(0)
			book.delta = s.floatBits(1)
			count := int(dim) * int(entries)
			if book.lookup == 1 {
				count = lookupCount(int(entries), int(dim))
			}
			if count > 1<<20 {
				return nil, fmt.Errorf("mpzz: oversized codebook lookup")
			}
			valueBits := s.integer(22, 2, 1, 2, true, 1, 4) + 1
			if valueBits > 16 {
				return nil, fmt.Errorf("mpzz: codebook %d value width %d", i, valueBits)
			}
			s.tree(0x2224, 1, 1)
			center := count / 2
			book.quant = make([]int, count)
			for j := 0; j < count; j++ {
				pred := center
				if j&1 != 0 {
					pred = count - center - 2
				}
				v := uint32(pred) - s.models[23].integer(s.r, 4, 1, 4, false)
				s.w.write(v, uint(valueBits))
				book.quant[j] = int(v)
				book.radix = max(book.radix, int(v)+1)
				if j&1 != 0 {
					center++
				}
			}
		}
		if s.r.err != nil {
			return nil, s.r.err
		}
		if err := book.buildCodes(); err != nil {
			return nil, fmt.Errorf("mpzz: codebook %d: %w", i, err)
		}
		if book.lookup != 0 {
			if s.cache == nil {
				s.cache = &bookCache{capacity: 32}
			}
			book.context = s.cache.selectBook(book, s.probs)
			if err := book.buildLookup(); err != nil {
				return nil, err
			}
		}
	}
	return books, nil
}
