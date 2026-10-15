package tor

// byteIn is the Tornado InputByteStream.
// compress_all_at_once is 0, so the read chunk is LARGE_BUFFER_SIZE.
// readAt is buf+bufsize (not buf+MAXELEM): the last MAXELEM bytes of a chunk
// stay addressable for get32/get64 and are copied to the front on refill.
type byteIn struct {
	src    []byte
	off    int
	buf    []byte
	pos    int
	readAt int
	size   int
}

func newByteIn(src []byte) *byteIn {
	b := &byteIn{
		src:    src,
		buf:    make([]byte, maxElem+largeBuf),
		pos:    maxElem,
		readAt: largeBuf,
		size:   largeBuf,
	}
	b.pull(maxElem, largeBuf)
	return b
}

func (b *byteIn) pull(dst, n int) {
	avail := len(b.src) - b.off
	if avail < n {
		n = avail
	}
	if n > 0 {
		copy(b.buf[dst:dst+n], b.src[b.off:b.off+n])
		b.off += n
	}
}

func (b *byteIn) fill() {
	if b.pos < b.readAt {
		return
	}
	copy(b.buf[:maxElem], b.buf[b.size:b.size+maxElem])
	// Short read leaves the previous tail in place, matching the C callback.
	b.pull(maxElem, b.size)
	b.pos -= b.size
}

func (b *byteIn) get8() uint32 {
	b.fill()
	v := b.buf[b.pos]
	b.pos++
	return uint32(v)
}

func (b *byteIn) get16() uint32 {
	b.fill()
	v := uint32(b.buf[b.pos]) | uint32(b.buf[b.pos+1])<<8
	b.pos += 2
	return v
}

func (b *byteIn) get24() uint32 {
	b.fill()
	v := uint32(b.buf[b.pos]) | uint32(b.buf[b.pos+1])<<8 | uint32(b.buf[b.pos+2])<<16
	b.pos += 3
	return v
}

func (b *byteIn) get32() uint32 {
	b.fill()
	v := uint32(b.buf[b.pos]) | uint32(b.buf[b.pos+1])<<8 | uint32(b.buf[b.pos+2])<<16 | uint32(b.buf[b.pos+3])<<24
	b.pos += 4
	return v
}

// bitIn is InputBitStream: LSB-first, refilled 32 bits at a time.
type bitIn struct {
	byteIn
	bitbuf   uint64
	bitcount int
}

func newBitIn(src []byte) *bitIn {
	return &bitIn{byteIn: *newByteIn(src)}
}

func maskLow(x uint64, n int) uint32 {
	if n <= 0 {
		return 0
	}
	return uint32(x & ((uint64(1) << uint(n)) - 1))
}

func (b *bitIn) needbits(n int) uint32 {
	if b.bitcount <= 32 {
		b.bitbuf |= uint64(b.get32()) << uint(b.bitcount)
		b.bitcount += 32
	}
	return maskLow(b.bitbuf, n)
}

func (b *bitIn) dumpbits(n int) {
	if n <= 0 {
		return
	}
	b.bitbuf >>= uint(n)
	b.bitcount -= n
}

func (b *bitIn) getbits(n int) uint32 {
	x := b.needbits(n)
	b.dumpbits(n)
	return x
}
