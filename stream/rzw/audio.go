package rzw

// Each audio predictor has sixteen signed sample/difference taps. Coefficients
// move one unit in the direction of the wrapped residual's sign correlation.
type audioFilter struct {
	history  [16]int16
	coeff    [2][16]int16
	channel  int
	channels int
}

func (f *audioFilter) predict(shift uint) int32 {
	sum := int32(1) << (shift - 1)
	for i, h := range f.history {
		sum += int32(h) * int32(f.coeff[f.channel][i])
	}
	return sum >> shift
}

func (f *audioFilter) update(sample, residual int16) {
	if residual != 0 {
		for i, h := range f.history {
			if h != 0 {
				if (h ^ residual) < 0 {
					f.coeff[f.channel][i]--
				} else {
					f.coeff[f.channel][i]++
				}
			}
		}
	}
	copy(f.history[:], f.history[1:])
	f.history[15-max(f.channels, 1)] -= sample
	f.history[15] = sample
	if f.channels == 2 {
		f.channel ^= 1
	}
}

type audioResidual struct {
	models [16][24]uint16
	k      int
	sum    uint32
}

func (m *audioResidual) init() {
	for i := range m.models {
		for j := range m.models[i] {
			m.models[i][j] = uint16(min(j*16384/20, 16384))
		}
	}
}

func (m *audioResidual) decode(r *rans) int {
	if m.k < 0 || m.k >= len(m.models) {
		r.off = len(r.src) + 1
		return 0
	}
	s := decodeCDF(r, m.models[m.k][:], 20)
	v := uint32(imgBase20[s])
	if imgExtra20[s] != 0 {
		v += r.bits(int(imgExtra20[s]))
	}
	if m.k != 0 {
		v = v<<m.k | r.bits(m.k)
	}
	m.sum += v - (m.sum >> 3)
	q := m.sum >> m.k
	if q > 16 {
		m.k++
	} else if q <= 7 {
		m.k -= (m.k + 31) >> 5
	}
	return unzig(int(v))
}

type audio8 struct {
	filters  [4]audioFilter
	residual [2]audioResidual
	end      int
}

func (a *audio8) init() {
	for i := range a.residual {
		a.residual[i].init()
	}
}

// rz 1.00, 0x4050c3. A nonconsecutive mono token first trains both
// predictors on the preceding sixteen bytes, then emits exactly 32 bytes.
func (d *dec) audioPCM8(ch int) bool {
	a := &d.mono8
	if ch == 2 {
		a = &d.stereo8
	}
	for i := 0; i < 2; i++ {
		a.filters[i].channels = ch
	}
	if a.end != d.pos {
		for i := d.pos - 16; i < d.pos; i++ {
			b := byte(0)
			if i >= 0 {
				b = d.out[i]
			}
			x := int16(int8(b - 128))
			e := int16(int8(int32(x) - a.filters[0].predict(12)))
			a.filters[0].update(x, e)
			e2 := int16(int8(int32(e) - a.filters[1].predict(10)))
			a.filters[1].update(e, e2)
		}
	}
	for i := 0; i < 32; i++ {
		e := int16(int8(a.residual[i%ch].decode(&d.r)))
		x2 := int16(int8(int32(e) + a.filters[1].predict(10)))
		a.filters[1].update(x2, e)
		x := int16(int8(int32(x2) + a.filters[0].predict(12)))
		a.filters[0].update(x, x2)
		d.emit(byte(x) + 128)
	}
	a.end = d.pos
	return d.r.ok()
}

// The 16-bit tokens use a third filter independently for each channel.
// Reference: 0x40744d (mono), 0x4057db (stereo).
func (d *dec) audioPCM16(ch int) bool {
	a := &d.mono16
	if ch == 2 {
		a = &d.stereo16
	}
	for i := 0; i < 2; i++ {
		a.filters[i].channels = ch
	}
	if a.end != d.pos {
		for i := 0; i < 16; i++ {
			start := d.pos - 32 + 2*i
			var x int16
			if start >= 0 {
				x = int16(uint16(d.out[start]) | uint16(d.out[start+1])<<8)
			}
			for stage := 0; stage < 3; stage++ {
				f := &a.filters[stage]
				if stage == 2 {
					f = &a.filters[2+i%ch]
				}
				shift := uint(10)
				if stage == 0 {
					shift = 12
				}
				e := x - int16(f.predict(shift))
				f.update(x, e)
				x = e
			}
		}
	}
	for i := 0; i < 16; i++ {
		x := int16(a.residual[i%ch].decode(&d.r))
		for stage := 2; stage >= 0; stage-- {
			f := &a.filters[stage]
			if stage == 2 {
				f = &a.filters[2+i%ch]
			}
			shift := uint(10)
			if stage == 0 {
				shift = 12
			}
			v := x + int16(f.predict(shift))
			f.update(v, x)
			x = v
		}
		d.emit(byte(x))
		d.emit(byte(x >> 8))
	}
	a.end = d.pos
	return d.r.ok()
}
