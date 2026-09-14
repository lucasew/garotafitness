package mpzz

import (
	"fmt"
	"math"
	"math/bits"
)

type cachedBook struct {
	lengths []byte
	quant   int
	delta   float32
	uses    uint32
	used    bool
}

type bookCache struct {
	capacity int
	books    []cachedBook
}

func vorbisFloat(v uint32) float32 {
	m := int(v & 0x1fffff)
	if v>>31 != 0 {
		m = -m
	}
	return float32(math.Ldexp(float64(m), int(v>>21&0x3ff)-788))
}

// 0x1000f550 shares residue probabilities between codebooks with similar
// lengths, the same entry count, quantizer size, and delta.
func (c *bookCache) selectBook(b *codebook, probs []uint16) int {
	lengths := append([]byte(nil), b.lengths...)
	longest := byte(0)
	used := 0
	for _, n := range lengths {
		longest = max(longest, n)
		if n != 0 {
			used++
		}
	}
	// 0x10005b21 keeps a compact representation only below one quarter
	// occupancy. Dense books retain the unused-entry sentinel in the cache.
	missing := byte(0xff)
	if used < len(lengths)/4 {
		missing = longest
	}
	for i, n := range lengths {
		if n == 0 {
			lengths[i] = missing
		}
	}
	delta := vorbisFloat(b.delta)
	best, distance := 0, uint32(0xffffffff)
	for i, old := range c.books {
		if old.quant != len(b.quant) || old.delta != delta || len(old.lengths) != len(lengths) {
			continue
		}
		d := uint32(0)
		for j, n := range lengths {
			x := int(n) - int(old.lengths[j])
			if x < 0 {
				x = -x
			}
			d += uint32(x)
		}
		if d < distance {
			best, distance = i, d
			if d == 0 {
				break
			}
		}
	}
	score := (distance << 8) / uint32(len(lengths))
	if score <= 256 && len(c.books) > 0 {
		c.books[best].uses++
		c.books[best].used = true
		return best
	}
	if len(c.books) < c.capacity {
		best = len(c.books)
		c.books = append(c.books, cachedBook{})
	} else {
		if score >= 4095 {
			best = 0
			for i := range c.books {
				if c.books[i].uses < c.books[best].uses {
					best = i
				}
			}
			base := 0x9225c + best*0x45400
			for i := base; i < base+0x45400; i++ {
				probs[i] = 0x8000
			}
		}
		uses := uint32(0)
		for i, old := range c.books {
			if i != best {
				uses += old.uses
			}
		}
		c.books[best].uses = uses >> uint(bits.TrailingZeros(uint(c.capacity)))
	}
	c.books[best].lengths, c.books[best].quant, c.books[best].delta = lengths, len(b.quant), delta
	c.books[best].used = true
	return best
}

// 0x10007ceb builds the inverse quantizer map. Component order in the
// arithmetic stream is opposite to the radix digits of a Vorbis entry.
func (b *codebook) buildLookup() error {
	if b.lookup != 1 {
		return fmt.Errorf("mpzz: lookup type %d is not reconstructed", b.lookup)
	}
	b.symbols = make(map[int]int, len(b.lengths))
	for i, n := range b.lengths {
		if n == 0 {
			continue
		}
		v, key := i, 0
		for range b.dim {
			key = key*b.radix + b.quant[v%len(b.quant)]
			v /= len(b.quant)
		}
		b.symbols[key] = i
	}
	return nil
}

// 0x1001b610. Each signed residue component has a zero flag, sign, unit
// flag, and optional magnitude. Contexts retain the preceding pass at
// this spectral position as well as recent component histories.
func (a *audioDecoder) component(residue, context, pass, pos int) int {
	return a.channelComponent(residue, context, pass, pos, 0)
}

// 0x1001c660 extends the component model with channel-specific sign and
// spectral history. Its magnitude contexts still use channel zero's history.
func (a *audioDecoder) channelComponent(residue, context, pass, pos, channel int) int {
	if pos < 0 || pos >= 8192 {
		a.body.err = fmt.Errorf("mpzz: residue position %d exceeds model capacity", pos)
		return 0
	}
	channel = min(channel, 3)
	base := 0x9225c + context*0x45400
	histPos := residue<<18 | max(pass-1, 0)<<15 | pos*4 | channel
	old := a.residueHistory[histPos]
	ctx := residue + (pos&^3)*8 + channel*4 + int(a.componentPresent[0])*16
	if old&7 == 0 {
		ctx += 1 << 16
	}
	if a.lastContext == context {
		ctx += 1 << 17
	}
	present := a.body.bit(&a.probs[base+ctx])
	outPos := residue<<18 | pass<<15 | pos*4 | channel
	a.lastContext = context
	if present == 0 {
		a.componentPresent[channel], a.componentLength[channel] = 0, 0
		a.residueHistory[outPos] = 0
		return 0
	}
	zeroCtx := 0
	if old == 0 {
		zeroCtx = 1 << 12
	}
	logpos := bits.Len(uint(pos))
	ctx = residue + channel*4 + logpos*128 + zeroCtx + int(a.componentPresent[0])*64 + (int(a.componentSign[channel])+int(old&0x80))*16
	sign := a.body.bit(&a.probs[base+0x40000+ctx])
	a.componentPresent[channel] = 1
	a.componentSign[channel] = (2*a.componentSign[channel] + sign) & 3
	ctx = residue + channel*4 + logpos*256 + zeroCtx + int(a.componentLength[0])*32 + int(sign)*16
	unit := a.body.bit(&a.probs[base+0x42000+ctx])
	v, length := 1, 1
	if unit == 0 {
		n := int(a.tree(a.body, base+0x44000+int(a.componentLength[0])*128+residue*8+channel*32, 3))
		v = 1
		for range n + 1 {
			ctx := base + 0x44400 + int(a.componentLength[0])*64 + residue*4 + channel*16 + n*512 + (v & 3)
			v = 2*v + int(a.body.bit(&a.probs[ctx]))
		}
		length = n + 2
	}
	a.componentLength[channel] = uint32(length)
	a.residueHistory[outPos] = byte(length + int(sign)*128)
	if sign != 0 {
		v = -v
	}
	return v
}

func (a *audioDecoder) residues(mapping mappingConfig, silent []bool) error {
	for _, pair := range mapping.coupling {
		if !silent[pair[0]] || !silent[pair[1]] {
			silent[pair[0]], silent[pair[1]] = false, false
		}
	}
	for sub, idx := range mapping.residues {
		r := a.setup.residues[idx]
		if r.kind != 1 && r.kind != 2 {
			return fmt.Errorf("mpzz: residue type %d is not reconstructed", r.kind)
		}
		var channels []int
		for ch := range a.channels {
			if mapping.mux[ch] == sub && !silent[ch] {
				channels = append(channels, ch)
			}
		}
		if len(channels) == 0 {
			continue
		}
		interleaved := 0
		if r.kind == 2 {
			for ch := range a.channels {
				if mapping.mux[ch] == sub {
					interleaved++
				}
			}
			channels = channels[:1]
		}
		book := &a.setup.books[r.classbook]
		parts := (r.end - r.begin) / r.size
		classes := make([][]int, len(channels))
		for ch := range classes {
			classes[ch] = make([]int, parts+book.dim)
		}
		for pass := 0; pass < 8; pass++ {
			for part := 0; part < parts; part += book.dim {
				if pass == 0 {
					for ch := range channels {
						v := 0
						for j := 0; j < book.dim; j++ {
							cl := int(a.tree(a.body, 0x6225c+(part+j)*32+(idx&3)*16, 4))
							if cl >= len(r.books) {
								return fmt.Errorf("mpzz: residue class %d exceeds %d", cl, len(r.books))
							}
							classes[ch][part+j] = cl
							v = v*len(r.books) + cl
						}
						if err := book.write(&a.w, v); err != nil {
							return err
						}
					}
				}
				for j := 0; j < book.dim && part+j < parts; j++ {
					for ch := range channels {
						bi := r.books[classes[ch][part+j]][pass]
						if bi < 0 {
							continue
						}
						b := &a.setup.books[bi]
						if b.lookup == 0 {
							return fmt.Errorf("mpzz: residue book %d has no lookup table", bi)
						}
						for off := 0; off < r.size; off += b.dim {
							key := 0
							for k := 0; k < min(b.dim, r.size-off); k++ {
								pos := r.begin + (part+j)*r.size + off + k
								channel := 0
								if interleaved != 0 {
									channel, pos = pos%interleaved, pos/interleaved
								}
								v := a.channelComponent(idx&3, b.context, pass, pos, channel) + b.radix/2
								if v < 0 || v >= b.radix {
									return fmt.Errorf("mpzz: residue component %d out of radix %d", v, b.radix)
								}
								key = key*b.radix + v
							}
							sym, ok := b.symbols[key]
							if !ok {
								return fmt.Errorf("mpzz: residue vector %d absent from book %d", key, bi)
							}
							if err := b.write(&a.w, sym); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}
	return a.body.err
}
