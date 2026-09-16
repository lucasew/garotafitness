package magic2

import "math/bits"

// Color-index prediction uses only the six-bit green components of RGB565.
func greenPalette(a, b int) [4]int {
	x, y := (a>>5)&63, (b>>5)&63
	p := [4]int{x, y}
	if a > b {
		p[2] = (2*x + y) * 86 >> 8
		p[3] = (x + 2*y) * 86 >> 8
	} else {
		p[2] = (x + y) >> 1
	}
	return p
}

func (d *decoder) colorIndices(r *entropy, width, blockSize int) [4]byte {
	pos := len(d.out)
	at := func(delta int) int { return int(d.history(pos + delta)) }
	u16 := func(delta int) int { return at(delta) | at(delta+1)<<8 }
	u32 := func(delta int) uint32 { return uint32(at(delta) | at(delta+1)<<8 | at(delta+2)<<16 | at(delta+3)<<24) }
	a, b := u16(-4), u16(-2)
	p := greenPalette(a, b)
	rank := [4]int{3, 1, 2, 0}
	if a > b {
		rank = [4]int{3, 0, 2, 1}
		if p[0] < p[1] {
			rank = [4]int{0, 3, 1, 2}
		}
	} else if p[0] < p[1] {
		rank = [4]int{1, 3, 2, 0}
	}
	lp := greenPalette(u16(-blockSize-4), u16(-blockSize-2))
	lb := u32(-blockSize)
	var left, top [4]int
	for i := range left {
		left[i] = lp[(lb>>uint((i*4+3)*2))&3]
	}
	if width != 0 {
		tp := greenPalette(u16(-width-4), u16(-width-2))
		tb := u32(-width)
		for i := range top {
			top[i] = tp[(tb>>uint((12+i)*2))&3]
		}
	}
	nearest := func(v int) int {
		best := 0
		for i := 1; i < 4; i++ {
			if abs(p[i]-v) < abs(p[best]-v) {
				best = i
			}
		}
		return best
	}
	base := 0x6000000 + bitLength(abs(p[0]-p[1]))*0x2480 + boolInt(a > b)*0x1240
	var raw [16]int
	var out uint32
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			i := y*4 + x
			var ctx int
			switch {
			case i == 0:
				ctx = 0x1000 + nearest(left[0])*16
			case y == 0:
				ctx = 0x1000 + (raw[i-1]+1)*64 + nearest(medianEdge(p[raw[i-1]], top[x], top[x-1]))*16
			case x == 0:
				ctx = 0x1000 + (raw[i-4]+5)*64 + nearest(medianEdge(left[y], p[raw[i-4]], left[y-1]))*16
			default:
				ctx = medianEdge(rank[raw[i-1]], rank[raw[i-4]], rank[raw[i-5]])*1024 + (raw[i-5]*16+raw[i-4]*4+raw[i-1])*16
			}
			raw[i] = d.symbol(r, base+ctx, 4, 7)
			out |= uint32(raw[i]) << uint(i*2)
		}
	}
	return [4]byte{byte(out), byte(out >> 8), byte(out >> 16), byte(out >> 24)}
}

// BC2 explicit alpha, VA 0x14002bde0 (mixer 4 in jmp table 0xa880).
// Sixteen 4-bit samples; the CDF at 0x285c00 is indexed by (left, above, diag).
func explicitAlphaContext(left, above, diag int) int {
	return (left*256 + above*16 + diag) * 34
}

func (d *decoder) explicitAlpha(r *entropy, width int) [8]byte {
	pos := len(d.out)
	at := func(delta int) int { return int(d.history(pos + delta)) }
	u32 := func(delta int) uint32 {
		return uint32(at(delta) | at(delta+1)<<8 | at(delta+2)<<16 | at(delta+3)<<24)
	}
	l0, l1 := u32(-16), u32(-12)
	left := [4]int{int(l0>>12) & 15, int(l0>>28) & 15, int(l1>>12) & 15, int(l1>>28) & 15}
	var top [4]int
	if width != 0 {
		t := u32(-width + 4)
		top = [4]int{int(t>>16) & 15, int(t>>20) & 15, int(t>>24) & 15, int(t>>28) & 15}
	}
	var raw [16]int
	var bits uint64
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			i := y*4 + x
			l, a := left[y], top[x]
			if x != 0 {
				l = raw[i-1]
			}
			if y != 0 {
				a = raw[i-4]
			}
			diag := l
			switch {
			case x == 0 && y == 0:
				diag = left[0]
			case x == 0:
				diag = left[y-1]
			case y == 0:
				diag = top[x-1]
			default:
				diag = raw[i-5]
			}
			raw[i] = d.symbol(r, 0x285c00+explicitAlphaContext(l, a, diag), 16, 6)
			bits |= uint64(raw[i]) << uint(i*4)
		}
	}
	var out [8]byte
	for i := range out {
		out[i] = byte(bits >> uint(i*8))
	}
	return out
}

func medianEdge(a, b, c int) int {
	if c >= max(a, b) {
		return min(a, b)
	}
	if c <= min(a, b) {
		return max(a, b)
	}
	return a + b - c
}

// The texture predictor uses the integer interpolation in VA 0x140034f80,
// including its fixed-point approximations of division by five and seven.
func alphaPalette(a, b int) [8]int {
	p := [8]int{a, b}
	if a > b {
		for i := 2; i < 8; i++ {
			p[i] = ((8-i)*a + (i-1)*b) * 36 >> 8
		}
	} else {
		for i := 2; i < 6; i++ {
			p[i] = ((6-i)*a + (i-1)*b) * 51 >> 8
		}
		p[7] = 255
	}
	return p
}

func closestAlpha(p [8]int, v int) int {
	best, distance := 0, 1<<30
	for i, x := range p {
		d := x - v
		if d < 0 {
			d = -d
		}
		if d < distance {
			best, distance = i, d
		}
	}
	return best
}

// BC3 alpha indices, VA 0x140033770 / 0x140033990. Boundary pixels use
// neighboring blocks' alpha intensities; interior pixels use palette ranks.
func (d *decoder) alphaIndices(r *entropy, width int) [6]byte {
	pos := len(d.out)
	at := func(delta int) int { return int(d.history(pos + delta)) }
	packed := func(delta int) uint64 {
		var v uint64
		for i := 0; i < 6; i++ {
			v |= uint64(at(delta+i)) << uint(i*8)
		}
		return v
	}
	a, b := at(-2), at(-1)
	p := alphaPalette(a, b)
	rank := [8]int{1, 6, 2, 3, 4, 5, 0, 7}
	if a > b {
		rank = [8]int{7, 0, 6, 5, 4, 3, 2, 1}
	}
	leftPalette := alphaPalette(at(-18), at(-17))
	leftBits := packed(-16)
	var left, top [4]int
	for y := 0; y < 4; y++ {
		left[y] = leftPalette[(leftBits>>uint((y*4+3)*3))&7]
	}
	if width != 0 {
		topPalette := alphaPalette(at(-width-2), at(-width-1))
		topBits := packed(-width)
		for x := 0; x < 4; x++ {
			top[x] = topPalette[(topBits>>uint((12+x)*3))&7]
		}
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	group := bits.Len(uint(diff))
	if group >= 5 {
		group--
	}
	base := 0x4000000 + group*0x12990
	var raw [16]int
	var out uint64
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			i := y*4 + x
			var context int
			switch {
			case i == 0:
				context = 0x12000 + closestAlpha(p, top[0])*18
			case y == 0:
				pred := medianEdge(p[raw[i-1]], top[x], top[x-1])
				context = 0x12000 + (raw[i-1]+1)*144 + closestAlpha(p, pred)*18
			case x == 0:
				pred := medianEdge(left[y], p[raw[i-4]], left[y-1])
				context = 0x12000 + (raw[i-4]+9)*144 + closestAlpha(p, pred)*18
			default:
				pred := medianEdge(rank[raw[i-1]], rank[raw[i-4]], rank[raw[i-5]])
				context = pred*9216 + (raw[i-5]*64+raw[i-4]*8+raw[i-1])*18
			}
			raw[i] = d.symbol(r, base+context, 8, 6)
			out |= uint64(raw[i]) << uint(i*3)
		}
	}
	var result [6]byte
	for i := range result {
		result[i] = byte(out >> uint(i*8))
	}
	return result
}
