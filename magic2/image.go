package magic2

import "math/bits"

var imageBits = [...]uint{0, 0, 0, 1, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6}

func signedContext(v int32) int {
	sign := v >> 31
	return int(min(bits.Len32(uint32(v^sign)), 7)) + int(sign&8)
}
func (d *decoder) history(at int) int32 {
	if at < 0 {
		return 0
	}
	if at >= len(d.out) {
		return 0
	}
	return int32(d.out[at])
}

// The grayscale predictor (VA 0x14001e920) learns the differences along
// the preceding row and the two previous pixels. Coefficients wrap as i16.
func (d *decoder) imageByte(r *entropy, width int) byte {
	pos := len(d.out)
	at := func(delta int) int32 { return d.history(pos + delta) }
	var horizontal, vertical [8]int32
	horizontal[0] = at(-2) - at(-3)
	horizontal[1] = at(-1) - at(-2)
	if width != 0 {
		for i := 0; i < 4; i++ {
			vertical[i] = at(-width-2+i) - at(-width-1+i)
		}
		vertical[5] = at(-width-1) - at(-1)
		vertical[6] = at(-width) - at(-2*width)
		vertical[7] = at(-width+1) - at(-2*width+1)
	}
	var sum int32
	for i := 0; i < 8; i++ {
		sum += horizontal[i]*int32(d.imageWeights[0][i]) + vertical[i]*int32(d.imageWeights[1][i])
	}
	prediction := (sum + 2048) >> 12
	context := signedContext(prediction)*256 + signedContext(horizontal[1])*16 + signedContext(vertical[5])
	s := d.symbol(r, 0x2000000+context*34, 16, 7)
	v := slotBase(imageBits[:], s) + r.raw(imageBits[s])
	residual := int32(int8((v >> 1) ^ -(v & 1)))
	if residual != 0 {
		for k, taps := range [2][8]int32{horizontal, vertical} {
			for i, tap := range taps {
				if tap != 0 {
					if (tap ^ residual) < 0 {
						d.imageWeights[k][i]--
					} else {
						d.imageWeights[k][i]++
					}
				}
			}
		}
	}
	return byte(at(-1) + prediction + residual)
}

// Pixels (VAs 0x1400205e0, 0x140023670, 0x140026620) share spatial differences across channels.
// Each decoded channel adds its signed delta as an input to the next channel.
func (d *decoder) imagePixel(r *entropy, width, channels int) [4]byte {
	pos := len(d.out)
	at := func(delta int) int32 { return d.history(pos + delta) }
	var taps [3][8]int32
	contextOffset := 0
	if channels == 2 {
		contextOffset = 2
		for i := 0; i < 4; i++ {
			taps[0][i] = at(-4+i) - at(-6+i)
			if width != 0 {
				taps[1][i] = at(-width-4+i) - at(-4+i)
				taps[1][i+4] = at(-width+i) - at(-2*width+i)
			}
		}
		if width != 0 {
			for i := 0; i < 8; i++ {
				taps[2][i] = at(-width-4+i) - at(-width-2+i)
			}
		}
	} else {
		for i := 0; i < channels; i++ {
			taps[0][i] = at(-channels+i) - at(-2*channels+i)
			if width != 0 {
				taps[1][i] = at(-width-channels+i) - at(-channels+i)
				taps[1][i+channels] = at(-width+i) - at(-2*width+i)
				taps[2][i] = at(-width-channels+i) - at(-width+i)
				taps[2][i+channels] = at(-width+i) - at(-width+channels+i)
			}
		}
	}
	var pixel [4]byte
	for ch := 0; ch < channels; ch++ {
		weights := &d.pixelWeights[channels-2][ch]
		var sum int32
		for k := range taps {
			for i, tap := range taps[k] {
				sum += tap * int32(weights[k][i])
			}
		}
		prediction := (sum + 2048) >> 12
		context := signedContext(prediction)*256 + signedContext(taps[0][ch+contextOffset])*16 + signedContext(taps[1][ch+contextOffset])
		s := d.symbol(r, 0x3000000+(channels-2)*0x100000+ch*0x22030+context*34, 16, 7)
		v := slotBase(imageBits[:], s) + r.raw(imageBits[s])
		residual := int32(int8((v >> 1) ^ -(v & 1)))
		if residual != 0 {
			for k := range taps {
				for i, tap := range taps[k] {
					if tap != 0 {
						if (tap ^ residual) < 0 {
							weights[k][i]--
						} else {
							weights[k][i]++
						}
					}
				}
			}
		}
		delta := int32(int8(prediction + residual))
		taps[0][4+ch] = delta
		pixel[ch] = byte(at(-channels+ch) + delta)
	}
	return pixel
}
