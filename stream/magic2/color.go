package magic2

// RGB565 channels occupy separate bytes while the endpoint models operate.
func expand565(v int) int { return (v&0xf800)<<5 | (v&0x7e0)<<3 | v&31 }
func pack565(v int) int   { return (v>>5)&0xf800 | (v>>3)&0x7e0 | v&31 }
func colorPalette(a, b int) [4]int {
	p := [4]int{a, b}
	if a > b {
		for ch := 0; ch < 3; ch++ {
			x, y := (a>>uint(ch*8))&255, (b>>uint(ch*8))&255
			p[2] |= ((2*x + y) * 86 >> 8) << uint(ch*8)
			p[3] |= ((x + 2*y) * 86 >> 8) << uint(ch*8)
		}
	} else {
		p[2] = ((a + b) >> 1) & 0x1f3f1f
	}
	return p
}
func colorSignedMagnitude(v int) int { v = int(int8(v)); return v ^ (v >> 31) }
func colorBitLength(v int) int       { return bitLength((v | v>>8 | v>>16) & 255) }

func (c *endpointCoder) colors(mode int, ctx alphaContext, value [2]int) [2]int {
	t, p := ctx.left, ctx.pred
	if mode == 0 { // Three-channel residual bit planes, VA 0x140017410.
		var difference int
		for ch := 0; ch < 3; ch++ {
			difference |= colorSignedMagnitude((t[0] >> uint(ch*8)) - (t[1] >> uint(ch*8)))
		}
		base := bitLength(difference)*128 + bitLength(colorSignedMagnitude((p[0]>>8)-(p[1]>>8)))*16 + bitLength(((p[0]+p[1])>>9)&63)*1024
		var result [2]int
		firstLength := 0
		for k := 0; k < 2; k++ {
			z := 0
			for ch, mask := range [3]int{31, 63, 31} {
				v := ((value[k] >> uint(ch*8)) - (p[k] >> uint(ch*8))) & mask
				if v > mask/2 {
					v -= mask + 1
				}
				z |= ((v << 1) ^ (v >> 31)) << uint(ch*8)
			}
			lengthAt, planesAt := base, 0x1c00
			if k == 1 {
				lengthAt += 0x3700 + firstLength*8192
				planesAt = 0x11700
			}
			n := c.symbol(lengthAt, 7, 6, colorBitLength(z))
			if k == 0 {
				firstLength = n
			}
			planeAt := planesAt + (n-1)*1152
			previous, decoded := 0, 0
			for i := n - 1; i >= 0; i-- {
				s := ((z>>uint(i))&1)*4 + ((z>>uint(i+8))&1)*2 + ((z >> uint(i+16)) & 1)
				s = c.symbol(planeAt+previous*18, 8, 6, s)
				decoded = decoded*2 + (s >> 2) + ((s>>1)&1)*256 + (s&1)*65536
				previous = s
				planeAt += 144
			}
			for ch, mask := range [3]int{31, 63, 31} {
				v := (decoded >> uint(ch*8)) & 255
				result[k] |= (((p[k] >> uint(ch*8)) + ((v >> 1) ^ -(v & 1))) & mask) << uint(ch*8)
			}
		}
		return result
	}
	var result [2]int
	for k := 0; k < 2; k++ {
		var firstAt, hiAt, loAt, lastAt int
		firstShift, lastShift := uint(0), uint(16)
		if mode == 1 {
			firstAt = 0x13200 + bitLength((p[0]+p[1])>>17)*66 + bitLength(t[0]&255)*528
			hiAt, loAt, lastAt = 0x15480, 0x14280, 0x156c0
			if k == 1 {
				firstAt = 0x366c0 + (t[1]&255)*66
				hiAt, loAt, lastAt = 0x38100, 0x36f00, 0x38340
			}
		} else {
			firstShift, lastShift = 16, 0
			firstAt = 0x59340 + (t[0]>>16)*1056 + ((p[0]+p[1])>>18)*66
			hiAt, loAt, lastAt = 0x62940, 0x61740, 0x62b80
			if k == 1 {
				firstAt = 0x83b80 + (t[1]>>16)*66
				hiAt, loAt, lastAt = 0x855c0, 0x843c0, 0x85800
			}
		}
		first := c.symbol(firstAt, 32, 7, (value[k]>>firstShift)&31)
		hi := c.symbol(hiAt+first*18, 8, 6, (value[k]>>11)&7)
		lo := c.symbol(loAt+first*18+hi*576, 8, 6, (value[k]>>8)&7)
		green := hi*8 + lo
		last := c.symbol(lastAt+first*4224+green*66, 32, 7, (value[k]>>lastShift)&31)
		result[k] = first<<firstShift | green<<8 | last<<lastShift
	}
	return result
}

func updateSelector(selector *uint16, costs []int, threshold int) {
	selected, score := int(*selector>>13), int(*selector&8191)
	best := 0
	for i := 1; i < len(costs); i++ {
		if costs[i] < costs[best] {
			best = i
		}
	}
	if best == selected {
		score >>= 2
	} else {
		score = min(8191, score+(costs[selected]-costs[best])/2)
		if score >= threshold {
			selected = best
			score >>= 3
		}
	}
	*selector = uint16(selected<<13 | score)
}

func (d *decoder) colorEndpoints(r *entropy, width, blockSize int) [4]byte {
	pos := len(d.out)
	at := func(delta int) int { return int(d.history(pos + delta)) }
	u16 := func(delta int) int { return at(delta) | at(delta+1)<<8 }
	u32 := func(delta int) uint32 { return uint32(at(delta) | at(delta+1)<<8 | at(delta+2)<<16 | at(delta+3)<<24) }
	ctx := alphaContext{left: [2]int{expand565(u16(-blockSize)), expand565(u16(-blockSize + 2))}}
	ctx.top, ctx.pred = ctx.left, ctx.left
	if width != 0 {
		ctx.top = [2]int{expand565(u16(-width)), expand565(u16(-width + 2))}
		lp, tp := colorPalette(ctx.left[0], ctx.left[1]), colorPalette(ctx.top[0], ctx.top[1])
		lb, tb := u32(-blockSize+4), u32(-width+4)
		ctx.pred = [2]int{lp[(lb>>6)&3], lp[(lb>>6)&3]}
		for k := 0; k < 8; k++ {
			v := 0
			if k < 4 {
				v = lp[(lb>>uint((k*4+3)*2))&3]
			} else {
				v = tp[(tb>>uint((12+k-4)*2))&3]
			}
			if v&65535 > ctx.pred[0]&65535 {
				ctx.pred[0] = v
			}
			if v&65535 < ctx.pred[1]&65535 {
				ctx.pred[1] = v
			}
		}
	}
	selected := int(d.colorSelector >> 13)
	if d.header.ColorMode < 3 {
		selected = int(d.header.ColorMode)
	}
	c := endpointCoder{d: d, r: r}
	value := c.colors(selected, ctx, [2]int{})
	if d.header.ColorMode < 3 {
		a, b := pack565(value[0]), pack565(value[1])
		return [4]byte{byte(a), byte(a >> 8), byte(b), byte(b >> 8)}
	}
	var costs [3]int
	costs[selected] = c.cost
	for i := range costs {
		if i != selected {
			observer := endpointCoder{d: d}
			observer.colors(i, ctx, value)
			costs[i] = observer.cost
		}
	}
	updateSelector(&d.colorSelector, costs[:], 3500)
	a, b := pack565(value[0]), pack565(value[1])
	return [4]byte{byte(a), byte(a >> 8), byte(b), byte(b >> 8)}
}
