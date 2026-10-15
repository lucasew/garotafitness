package tor

// Schindler range coder and semi-adaptive arithmetic model (EntropyCoder.cpp).
// TRangeDecoder::buffer is not initialized in C++; wasm memory is zero, so the
// five startup bytes are shifted into a zero register.

const (
	rangeBits = 14
	rangeTop  = 1 << rangeBits
	ariIndex  = 2048
)

type rangeDec struct {
	in     *byteIn
	rng    uint32
	buffer uint32
	bad    error
}

func newRangeDec(src []byte) *rangeDec {
	r := &rangeDec{in: newByteIn(src), rng: 0xffffffff}
	for i := 0; i < 5; i++ {
		r.buffer = (r.buffer << 8) + r.in.get8()
	}
	return r
}

func (r *rangeDec) getCount(bits uint32) (uint32, error) {
	if r.bad != nil {
		return 0, r.bad
	}
	if bits >= 32 {
		r.bad = errFormat
		return 0, r.bad
	}
	r.rng >>= bits
	if r.rng == 0 {
		r.bad = errFormat
		return 0, r.bad
	}
	count := r.buffer / r.rng
	if count >= uint32(1)<<bits {
		r.bad = errFormat
		return 0, r.bad
	}
	return count, nil
}

func (r *rangeDec) update(cum, cnt uint32) {
	r.buffer -= cum * r.rng
	r.rng *= cnt
	for r.rng < 1<<24 {
		r.rng <<= 8
		r.buffer = (r.buffer << 8) + r.in.get8()
	}
}

// counter is TCounter<Decoder>.
type counter struct {
	n         int
	remainder uint32
	cnt       []uint32
	cum       []uint32
	live      []uint32
	index     []uint32
}

func newCounter(n int) *counter {
	c := &counter{
		n:     n,
		cnt:   make([]uint32, n),
		cum:   make([]uint32, n),
		live:  make([]uint32, n),
		index: make([]uint32, ariIndex),
	}
	base := rangeTop / n
	extra := rangeTop - base*n
	for s := 0; s < n; s++ {
		c.live[s] = uint32(base)
		if s < extra {
			c.live[s]++
		}
	}
	c.rescale()
	return c
}

func (c *counter) rescale() {
	var total uint32
	c.remainder = rangeTop
	ind := 0
	step := uint32(rangeTop / ariIndex)
	for s := 0; s < c.n; s++ {
		c.cnt[s] = c.live[s]
		c.cum[s] = total
		total += c.cnt[s]
		sub := c.live[s] / 6
		if c.live[s] > 1 && c.live[s] < 6 {
			sub = 1
		}
		c.live[s] -= sub
		c.remainder -= c.live[s]
		// Each index slot is the first symbol whose range covers that bin.
		for ind < ariIndex && c.cum[s]+c.cnt[s]-1 >= step*uint32(ind) {
			c.index[ind] = uint32(s)
			ind++
		}
	}
}

func (c *counter) inc(s uint32) {
	c.live[s]++
	c.remainder--
	if c.remainder == 0 {
		c.rescale()
	}
}

func (c *counter) decode(count uint32) (uint32, bool) {
	if count >= rangeTop {
		return 0, false
	}
	s := c.index[count/(rangeTop/ariIndex)]
	for int(s) < c.n && c.cum[s]+c.cnt[s]-1 < count {
		s++
	}
	if int(s) >= c.n {
		return 0, false
	}
	return s, true
}

type arithDec struct {
	rd  *rangeDec
	c   *counter
	bad error
}

func newArithDec(src []byte, n int) *arithDec {
	return &arithDec{rd: newRangeDec(src), c: newCounter(n)}
}

func (a *arithDec) fail(err error) {
	if a.bad == nil {
		a.bad = err
	}
}

func (a *arithDec) decode() uint32 {
	if a.bad != nil {
		return 0
	}
	count, err := a.rd.getCount(rangeBits)
	if err != nil {
		a.fail(err)
		return 0
	}
	x, ok := a.c.decode(count)
	if !ok {
		a.fail(errFormat)
		return 0
	}
	a.rd.update(a.c.cum[x], a.c.cnt[x])
	a.c.inc(x)
	return x
}

func (a *arithDec) getbits(n int) uint32 {
	if a.bad != nil {
		return 0
	}
	if n <= 24 {
		return a.rawbits(n)
	}
	x1 := a.rawbits(15)
	if a.bad != nil {
		return 0
	}
	x2 := a.rawbits(n - 15)
	return (x2 << 15) + x1
}

func (a *arithDec) rawbits(n int) uint32 {
	x, err := a.rd.getCount(uint32(n))
	if err != nil {
		a.fail(err)
		return 0
	}
	a.rd.update(x, 1)
	return x
}

func (a *arithDec) failed() error { return a.bad }
