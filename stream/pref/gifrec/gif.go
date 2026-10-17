// Package gifrec restores Precomp 0.4.8 D_GIF streams.
// Recompress is recompress_gif with a NULL GifCodeStruct, not a generic GIF codec.
package gifrec

import "fmt"

const (
	fileStateWrite  int32 = 0x01
	fileStateScreen int32 = 0x02
	fileStateImage  int32 = 0x04
	fileStateRead   int32 = 0x08

	lzMaxCode   int32 = 4095
	flushOutput int32 = 4096
	firstCode   int32 = 4097

	recImage = 1
	recExt   = 2
	recTerm  = 3
)

// CodeMask matches egif_lib_gcc.c. Index is BitsPerPixel.
var codeMask = [9]byte{0x00, 0x01, 0x03, 0x07, 0x0f, 0x1f, 0x3f, 0x7f, 0xff}

var (
	laceOff  = [4]int{0, 4, 2, 1}
	laceJump = [4]int{8, 8, 4, 2}
)

type gifColor struct {
	r, g, b byte
}

type colorMap struct {
	count int32
	bpp   int32
	color []gifColor
}

func bitSize(n int32) int32 {
	var i int32
	for i = 1; i <= 8; i++ {
		if int32(1)<<i >= n {
			break
		}
	}
	return i
}

func makeMap(count int32, src []gifColor) *colorMap {
	if count != int32(1)<<bitSize(count) {
		return nil
	}
	m := &colorMap{
		count: count,
		bpp:   bitSize(count),
		color: make([]gifColor, count),
	}
	copy(m.color, src)
	return m
}

type imageDesc struct {
	left, top, width, height int32
	interlace                int32
	colorMap                 *colorMap
}

type gifPriv struct {
	state       int32
	bpp         int32
	clearCode   int32
	eofCode     int32
	runningCode int32
	runningBits int32
	maxCode1    int32
	crntCode    int32
	shiftState  int32
	shiftDWord  uint64
	pixelCount  uint64
	buf         [256]byte
	codes       []int32
	in          *cursor
	out         *cursor
	mayWrite    *bool
}

func (p *gifPriv) read(n int) ([]byte, error) {
	return p.in.read(n)
}

func (p *gifPriv) writeBytes(b []byte) int {
	if p.mayWrite != nil && !*p.mayWrite {
		return len(b)
	}
	p.out.write(b)
	return len(b)
}

type gifFile struct {
	w, h      int32
	colorRes  int32
	bg        int32
	aspect    int32
	blockSize byte
	sMap      *colorMap
	image     imageDesc
	priv      *gifPriv
}

func newCodeTable() []int32 {
	t := make([]int32, 1024*1024)
	for i := range t {
		t[i] = -1
	}
	return t
}

func clearCodeTable(t []int32) {
	for i := range t {
		t[i] = -1
	}
}

// gifDiff is the restore-side view of GifDiffStruct.
// Records are a type byte, a 4-byte big-endian position, and when type > 1
// four more big-endian data bytes.
type gifDiff struct {
	index     int32
	size      int32
	codeCount int32
	data      []byte
}

func newDiff(b []byte) *gifDiff {
	if b == nil {
		b = []byte{}
	}
	return &gifDiff{size: int32(len(b)), data: b}
}

func (g *gifDiff) peekType() int32 {
	if g.index < g.size && int(g.index) < len(g.data) {
		return int32(g.data[g.index])
	}
	return -1
}

func (g *gifDiff) isPos() bool {
	next := int32(-1)
	if g.index < g.size && int(g.index)+4 < len(g.data) {
		next = int32(g.data[g.index+1])
		next = (next << 8) + int32(g.data[g.index+2])
		next = (next << 8) + int32(g.data[g.index+3])
		next = (next << 8) + int32(g.data[g.index+4])
	}
	return g.codeCount == next
}

func (g *gifDiff) dataWord() int32 {
	data := int32(-1)
	if g.index < g.size && int(g.index)+8 < len(g.data) {
		data = int32(g.data[g.index+5])
		data = (data << 8) + int32(g.data[g.index+6])
		data = (data << 8) + int32(g.data[g.index+7])
		data = (data << 8) + int32(g.data[g.index+8])
	}
	return data
}

func (g *gifDiff) next() {
	if g.peekType() > 1 {
		g.index += 9
	} else {
		g.index += 5
	}
}

type cursor struct {
	b []byte
	p int
}

func (c *cursor) tell() int { return c.p }

func (c *cursor) seek(p int) error {
	if p < 0 || p > len(c.b) {
		return fmt.Errorf("%w: seek %d", errRecompress, p)
	}
	c.p = p
	return nil
}

func (c *cursor) read(n int) ([]byte, error) {
	if n < 0 || c.p > len(c.b) || n > len(c.b)-c.p {
		return nil, fmt.Errorf("%w: short read", errRecompress)
	}
	b := c.b[c.p : c.p+n]
	c.p += n
	return b, nil
}

func (c *cursor) write(src []byte) {
	end := c.p + len(src)
	if end > len(c.b) {
		n := make([]byte, end)
		copy(n, c.b)
		c.b = n
	}
	copy(c.b[c.p:end], src)
	c.p = end
}

func fastCopy(src, dst *cursor, n int) error {
	if n == 0 {
		return nil
	}
	b, err := src.read(n)
	if err != nil {
		return err
	}
	dst.write(b)
	return nil
}
