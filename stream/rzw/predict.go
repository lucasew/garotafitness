package rzw

func (m *nibModel) initWeights(weights []byte) {
	total := 0
	for _, w := range weights {
		total += int(w)
	}
	sum := 0
	for i, w := range weights {
		m[i] = uint16(sum * 16384 / total)
		sum += int(w)
	}
}

// Typed delta tokens always emit eight bytes. Residuals wrap at the sample
// width; the preceding wrapped difference selects one of two CDFs.
// rz 1.00: 0x406e33 (four u16 samples), 0x403e72 (two u32 samples).
func (d *dec) wordBefore(width, back int) uint32 {
	start := d.pos - back*width
	var v uint32
	for i := 0; i < width; i++ {
		if start+i >= 0 {
			v |= uint32(d.out[start+i]) << uint(8*i)
		}
	}
	return v
}

func (d *dec) delta16() bool {
	widths := [...]byte{0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 0}
	for i := 0; i < 4; i++ {
		prev := uint16(d.wordBefore(2, 1))
		sign := (prev - uint16(d.wordBefore(2, 2))) >> 15
		sym := d.u16[sign].sym(&d.r)
		var v uint32
		for j := 0; j < sym; j++ {
			v += 1 << widths[j]
		}
		if widths[sym] != 0 {
			v += d.r.bits(int(widths[sym]))
		}
		w := prev + uint16(v)
		d.emit(byte(w))
		d.emit(byte(w >> 8))
	}
	return d.r.ok()
}

func (d *dec) delta32() bool {
	widths := [...]byte{31, 28, 26, 24, 22, 20, 18, 16, 14, 12, 10, 8, 6, 4, 2, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 31}
	var bias uint32
	for _, w := range widths[:15] {
		bias += 1 << w
	}
	for i := 0; i < 2; i++ {
		prev := d.wordBefore(4, 1)
		sign := (prev - d.wordBefore(4, 2)) >> 31
		m := &d.u32[sign]
		d.r.swap()
		if !d.r.ok() {
			return false
		}
		slot := d.r.s0 & nibMask
		sym := 0
		for sym < 30 && uint32(m[sym+1]) <= slot {
			sym++
		}
		start := uint32(m[sym])
		freq := uint32(m[sym+1]) - start
		d.r.s0 = (d.r.s0>>nibScale)*freq + slot - start
		for j := 0; j < 31; j++ {
			target := j
			if j > sym {
				target += 0x407f - 31
			}
			m[j] = uint16(int(m[j]) + ((target - int(m[j])) >> adaptSh))
		}
		var v uint32
		for j := 0; j < sym; j++ {
			v += 1 << widths[j]
		}
		if widths[sym] != 0 {
			v += d.r.bits(int(widths[sym]))
		}
		v += prev - bias
		for j := 0; j < 4; j++ {
			d.emit(byte(v >> uint(8*j)))
		}
	}
	return d.r.ok()
}

// Token 5 predicts the second difference of two interleaved u16 channels.
func (d *dec) delta16x2() bool {
	widths := [...]byte{0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 0}
	for i := 0; i < 4; i++ {
		prev := uint16(d.wordBefore(2, 2))
		prev2 := uint16(d.wordBefore(2, 4))
		prev3 := uint16(d.wordBefore(2, 6))
		sign := (prev - 2*prev2 + prev3) >> 15
		sym := d.u16x2[2*((3-i)%2)+int(sign)].sym(&d.r)
		var v uint32
		for j := 0; j < sym; j++ {
			v += 1 << widths[j]
		}
		if widths[sym] != 0 {
			v += d.r.bits(int(widths[sym]))
		}
		w := 2*prev - prev2 + uint16(v)
		d.emit(byte(w))
		d.emit(byte(w >> 8))
	}
	return d.r.ok()
}
