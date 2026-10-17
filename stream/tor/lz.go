package tor

// byteDec is LZ77_ByteDecoder. Flags are 16 symbol kinds, 2 bits each, LSB first.
type byteDec struct {
	in      *byteIn
	flags   uint32
	flagpos int
	dist    uint32
}

func newByteDec(src []byte) *byteDec {
	return &byteDec{in: newByteIn(src), flagpos: 1}
}

func (d *byteDec) next() (bool, error) {
	d.flagpos--
	if d.flagpos == 0 {
		d.flagpos = 16
		d.flags = d.in.get32()
	} else {
		d.flags >>= 2
	}
	return d.flags&3 == 0, nil
}

func (d *byteDec) char() byte { return byte(d.in.get8()) }

func (d *byteDec) length(minlen uint32) uint32 {
	var n uint32
	switch d.flags & 3 {
	case 1:
		x := d.in.get16()
		n = x >> 12
		d.dist = x & ((1 << 12) - 1)
	case 2:
		x := d.in.get24()
		n = x >> 18
		d.dist = x & ((1 << 18) - 1)
	case 3:
		n = d.in.get8()
		if n == 255 {
			d.dist = d.in.get8() << 24
			n = d.in.get8()
		} else {
			d.dist = 0
		}
		if n == 254 {
			n = d.in.get24() << 8
			n += d.in.get8()
		}
		d.dist += d.in.get24()
	}
	return minlen + n
}

func (d *byteDec) distance() uint32 { return d.dist }

func (d *byteDec) failed() error { return nil }

// bitDec is LZ77_BitDecoder. A symbol is 9 bits: <256 literal, else len/dist slots.
type bitDec struct {
	in *bitIn
	x  uint32
}

func newBitDec(src []byte) *bitDec { return &bitDec{in: newBitIn(src)} }

func (d *bitDec) next() (bool, error) {
	d.x = d.in.getbits(9)
	return d.x < 256, nil
}

func (d *bitDec) char() byte { return byte(d.x) }

func (d *bitDec) length(minlen uint32) uint32 {
	lcode := (d.x >> 5) - 8
	return minlen + lcBase[lcode] + d.in.getbits(int(lcExtra[lcode]))
}

func (d *bitDec) distance() uint32 {
	dcode := d.x & 31
	return dcBase[dcode] + d.in.getbits(int(dcExtra[dcode]))
}

func (d *bitDec) failed() error { return nil }

// symDec is the entropy coder behind LZ77_Decoder (huffman or arithmetic).
type symDec interface {
	decode() uint32
	getbits(n int) uint32
	failed() error
}

// lz77Dec is LZ77_Decoder. prevdists[0:4] start at 0 and hp starts at 4,
// matching the decoder (the encoder's -1 sentinels are a different history).
type lz77Dec struct {
	src  symDec
	x    uint32
	hist [128]uint32
	hp   int
	bad  error
}

func newLZ77(src symDec) *lz77Dec {
	return &lz77Dec{src: src, hp: repdistCodes}
}

func (d *lz77Dec) fail(err error) error {
	if d.bad == nil {
		d.bad = err
	}
	return d.bad
}

func (d *lz77Dec) next() (bool, error) {
	if d.bad != nil {
		return false, d.bad
	}
	d.x = d.src.decode()
	if err := d.src.failed(); err != nil {
		return false, d.fail(err)
	}
	return d.x < 256, nil
}

func (d *lz77Dec) char() byte { return byte(d.x) }

func (d *lz77Dec) length(minlen uint32) uint32 {
	if d.x == repchar {
		return 1
	}
	lcode := d.x % lenCodes
	n := lc2Base[lcode] + d.src.getbits(int(lc2Extra[lcode]))
	if err := d.src.failed(); err != nil {
		d.fail(err)
		return 0
	}
	switch {
	case n > 104:
		return n - 4 + minlen
	case n > 100:
		return n - 100 + impossible
	default:
		return n + minlen
	}
}

func (d *lz77Dec) distance() uint32 {
	if d.bad != nil {
		return 0
	}
	if d.x == repchar {
		return d.hist[d.hp-1]
	}
	dcode := int(d.x-256)/lenCodes - repdistCodes
	if dcode < 0 {
		return d.repdist(dcode)
	}
	if dcode >= len(dcExtra) {
		d.fail(errFormat)
		return 0
	}
	dist := dcBase[dcode] + d.src.getbits(int(dcExtra[dcode])) + 1
	if err := d.src.failed(); err != nil {
		d.fail(err)
		return 0
	}
	// When the history buffer is full, keep the last 3 distances at the front.
	if d.hp == len(d.hist) {
		d.hist[0] = d.hist[125]
		d.hist[1] = d.hist[126]
		d.hist[2] = d.hist[127]
		d.hp = repdistCodes - 1
	}
	d.hist[d.hp] = dist
	d.hp++
	return dist
}

func (d *lz77Dec) repdist(dcode int) uint32 {
	switch dcode {
	case -4:
		return d.hist[d.hp-1]
	case -3:
		dist := d.hist[d.hp-2]
		d.hist[d.hp-2] = d.hist[d.hp-1]
		d.hist[d.hp-1] = dist
		return dist
	case -2:
		dist := d.hist[d.hp-3]
		d.hist[d.hp-3] = d.hist[d.hp-2]
		d.hist[d.hp-2] = d.hist[d.hp-1]
		d.hist[d.hp-1] = dist
		return dist
	default:
		dist := d.hist[d.hp-4]
		d.hist[d.hp-4] = d.hist[d.hp-3]
		d.hist[d.hp-3] = d.hist[d.hp-2]
		d.hist[d.hp-2] = d.hist[d.hp-1]
		d.hist[d.hp-1] = dist
		return dist
	}
}

func (d *lz77Dec) failed() error { return d.bad }
