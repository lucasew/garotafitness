package magic2

// Segment metadata uses its own rANS state and three adaptive integer models.
// Reconstructed from cls-magic2_x64 VAs 0x14003afc0 and 0x14003b430.
type integerModel struct {
	history, previous int
	presence          [4]uint16
	slots, escape     [32][]uint16
	bits              [2048]uint16
}

func newIntegerModel() integerModel {
	var m integerModel
	for i := range m.presence {
		m.presence[i] = 16384
	}
	for i := range m.slots {
		m.slots[i] = uniform(16)
		m.escape[i] = uniform(16)
	}
	for i := range m.bits {
		m.bits[i] = 16384
	}
	return m
}
func (m *integerModel) read(r *entropy) uint32 {
	bit := r.bit(&m.presence[m.history], 15, 4)
	m.history = (m.history*2 + bit) & 3
	if bit == 0 {
		m.previous = 0
		return 0
	}
	s := r.symbol(m.slots[m.previous], 5)
	if s == 15 {
		s += r.symbol(m.escape[m.previous], 5)
	}
	m.previous = s + 1
	v := uint32(1)
	ctx := 1
	for i := 0; i < s; i++ {
		b := r.bit(&m.bits[s*64+ctx], 15, 4)
		v = v*2 + uint32(b)
		ctx = (ctx*2 + b) & 63
	}
	return v
}

type segment struct {
	option            int
	size, packed, aux uint32
}
type metadata struct {
	r                 *entropy
	presence          [8]uint16
	history, previous int
	options, escape   []uint16
	size, packed, aux integerModel
}

func newMetadata(b []byte) *metadata {
	m := &metadata{r: newEntropy(b), options: uniform(16), escape: uniform(16), size: newIntegerModel(), packed: newIntegerModel(), aux: newIntegerModel()}
	for i := range m.presence {
		m.presence[i] = 16384
	}
	return m
}
func (m *metadata) next() (segment, error) {
	b := m.r.bit(&m.presence[m.history], 15, 5)
	m.history = b
	if b == 0 {
		s := m.r.symbol(m.options, 5)
		m.previous = [16]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 12, 13, 14, 15, 63, 62, 0}[s]
		if s == 15 {
			m.previous = [16]int{16, 17, 18, 19, 32, 33, 34, 35, 36}[m.r.symbol(m.escape, 5)]
		}
	}
	s := segment{option: m.previous, size: m.size.read(m.r)}
	if s.option == 63 {
		s.packed = s.size
	} else {
		s.packed = m.packed.read(m.r)
	}
	if s.option >= 1 && s.option <= 11 {
		s.aux = m.aux.read(m.r)
	}
	return s, m.r.err
}
