package magic2

import (
	"math"
	"math/bits"
)

// Cost table at RVA 0x5e60: round(4096*log2(8192/frequency)).
var entropyCosts = func() [8193]int {
	var t [8193]int
	t[0] = 65535
	for i := 1; i < len(t); i++ {
		t[i] = int(math.Round(4096 * math.Log2(8192/float64(i))))
	}
	return t
}()

// An inactive endpoint model observes the decoded symbols and adapts exactly
// like the active model, but does not consume entropy. Both report coding cost.
type endpointCoder struct {
	d    *decoder
	r    *entropy
	cost int
}

func (c *endpointCoder) symbol(a, n int, shift uint, value int) int {
	m := c.d.model(0x5000000+a, n)
	if c.r != nil {
		value = c.r.selectSymbol(m)
	}
	c.cost += entropyCosts[int(m[value+1]-m[value])>>2]
	adaptCDF(m, value, shift)
	if c.r != nil {
		c.r.normalize()
	}
	return value
}
func (c *endpointCoder) bit(a int, scale, shift uint, value int) int {
	a += 0x5000000
	p, ok := c.d.probabilities[a]
	if !ok {
		p = 1 << (scale - 1)
	}
	freq := int(p)
	if c.r != nil {
		value = c.r.bit(&p, scale, shift)
	} else if value == 0 {
		p += uint16(((1 << scale) - int(p)) >> shift)
	} else {
		p -= p >> shift
	}
	if value != 0 {
		freq = (1 << scale) - freq
	}
	c.cost += entropyCosts[freq>>(scale-13)]
	c.d.probabilities[a] = p
	return value
}
func (c *endpointCoder) byte(hi, lo, stride, value int) int {
	h := c.symbol(hi, 16, 7, value>>4)
	l := c.symbol(lo+h*stride, 16, 7, value&15)
	return h*16 + l
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func bitLength(v int) int { return bits.Len(uint(v)) }
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

type alphaContext struct{ left, top, pred [2]int }

func (c *endpointCoder) endpoints(mode int, ctx alphaContext, value [2]int) [2]int {
	a, b := value[0], value[1]
	l, t, p := ctx.top, ctx.left, ctx.pred
	variation := abs(l[0]-l[1]) + abs(t[0]-t[1])
	switch mode {
	case 0: // Joint bit planes of signed prediction residuals, VA 0x14001adb0.
		var z [2]int
		for i := range z {
			v := int(int8(value[i] - p[i]))
			z[i] = (v << 1) ^ (v >> 31)
		}
		group := bitLength(abs(p[0] - p[1]))
		n := c.symbol(0xb8c00+bitLength(variation>>1)*288+group*32, 9, 6, bitLength(z[0]|z[1]))
		base := 0xb9620 + group*2048 + (n-1)*256
		v0, v1, previous := 0, 0, 0
		for i := n - 1; i >= 0; i-- {
			b0 := c.bit(base+previous*8, 14, 4, (z[0]>>i)&1)
			b1 := c.bit(base+previous*8+2+b0*2, 14, 4, (z[1]>>i)&1)
			v0 = v0*2 + b0
			v1 = v1*2 + b1
			previous = b0*2 + b1
			base += 32
		}
		return [2]int{int(byte(p[0] + ((v0 >> 1) ^ -(v0 & 1)))), int(byte(p[1] + ((v1 >> 1) ^ -(v1 & 1))))}
	case 1: // Mean first, then signed endpoint separation, VA 0x14001b940.
		base := (variation>>7)*34 + ((p[0]+p[1])>>3)*136
		mean := c.byte(0xdfe20+base, 0xbde20+base, 8704, (a+b)>>1)
		sign := c.bit(0xe2020+bitLength(variation>>2)*4+boolInt(t[0] <= t[1])*2, 15, 6, boolInt(a <= b))
		base = mean*68 + sign*34
		diff := c.byte(0x126040+base, 0xe2040+base, 17408, abs(a-b))
		if sign != 0 {
			diff = -diff
		}
		a = mean + (diff >> 1) + (diff & 1)
		b = a - diff
	case 2: // Separation first, then mean, VA 0x14001c980.
		group := bitLength(abs(t[0] - t[1]))
		sign := c.bit(0x12a440+group*4+boolInt(t[0] <= t[1])*2, 15, 6, boolInt(a <= b))
		base := group*544 + bitLength((l[0]+l[1])>>2)*68 + sign*34
		diff := c.byte(0x14c480+base, 0x12a480+base, 8704, abs(a-b))
		base = ((p[0]+p[1])>>5)*272 + min(bitLength(diff>>4), 7)*34
		mean := c.byte(0x15f680+base, 0x14e680+base, 4352, (a+b)>>1)
		if sign != 0 {
			diff = -diff
		}
		a = mean + (diff >> 1) + (diff & 1)
		b = a - diff
	case 3: // Independent endpoint nibbles, VA 0x14001d910.
		base := (p[0]>>2)*272 + min(bitLength(p[1]), 7)*34
		a = c.byte(0x1a4780+base, 0x160780+base, 17408, a)
		base = (p[1]>>2)*272 + min(bitLength(a), 7)*34
		b = c.byte(0x1ecb80+base, 0x1a8b80+base, 17408, b)
	}
	return [2]int{a, b}
}

// BC3 endpoint dispatch and model competition, VA 0x1400353a0.
func (d *decoder) alphaEndpoints(r *entropy, width int) [2]byte {
	pos := len(d.out)
	at := func(delta int) int { return int(d.history(pos + delta)) }
	ctx := alphaContext{left: [2]int{at(-16), at(-15)}}
	ctx.top, ctx.pred = ctx.left, ctx.left
	if width != 0 {
		ctx.top = [2]int{at(-width), at(-width + 1)}
		lp, tp := alphaPalette(ctx.left[0], ctx.left[1]), alphaPalette(ctx.top[0], ctx.top[1])
		pack := func(delta int) uint64 {
			var v uint64
			for i := 0; i < 6; i++ {
				v |= uint64(at(delta+i)) << uint(8*i)
			}
			return v
		}
		lb, tb := pack(-14), pack(-width+2)
		ctx.pred = [2]int{0, 255}
		for i := 0; i < 4; i++ {
			l, t := lp[(lb>>uint((i*4+3)*3))&7], tp[(tb>>uint((12+i)*3))&7]
			ctx.pred[0] = max(ctx.pred[0], l, t)
			ctx.pred[1] = min(ctx.pred[1], l, t)
		}
	}
	selected := int(d.alphaSelector >> 13)
	if d.header.AlphaMode < 4 {
		selected = int(d.header.AlphaMode)
	}
	c := endpointCoder{d: d, r: r}
	value := c.endpoints(selected, ctx, [2]int{})
	if value[0] < 0 || value[0] > 255 || value[1] < 0 || value[1] > 255 {
		r.err = errBitstream
		return [2]byte{}
	}
	if d.header.AlphaMode < 4 {
		return [2]byte{byte(value[0]), byte(value[1])}
	}
	var costs [4]int
	costs[selected] = c.cost
	for i := range costs {
		if i != selected {
			observer := endpointCoder{d: d}
			observer.endpoints(i, ctx, value)
			costs[i] = observer.cost
		}
	}
	updateSelector(&d.alphaSelector, costs[:], 4000)
	return [2]byte{byte(value[0]), byte(value[1])}
}
