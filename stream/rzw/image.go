package rzw

// tokImagePred is jump-table slot 9.
// 1.00 table 0x42a000 → 0x404752; 1.03.7 table 0x42b000 → 0x404762.
// Width bit at +0x10fb0 (adapt >>5). Bit 1 reads bits(16) into +0x10fb4 (0x4080ab).
// Residual is the 20-sym imgExtra20 alphabet plus per-channel rice
// (0xffd0 G, 0x102e0 R, 0x105f0 B) and unzig at 0x404b05.
// One token writes 12 bytes (0x404836 lea 0xc; 0x408316 addl $0xc).
// Predictor is a 16-tap FIR on neighbor differences, +512 >>10, not a 2D filter.

// 16 rice slots: ctor 0x40d46a fills 0x300 bytes, 0x30 per (k,bank) pair.
const imgRiceN = 16

type imgPred struct {
	bit   binModel
	width int
	rice  [3]int
	cdf   [3][imgRiceN][2][20]uint16
	coeff [3][2][8]int16
}

func (p *imgPred) init() {
	p.bit = 0x8000
	for ch := range p.cdf {
		for r := range p.cdf[ch] {
			for ctx := range p.cdf[ch][r] {
				for s := 0; s < imgSyms; s++ {
					p.cdf[ch][r][ctx][s] = uint16(s * 16384 / imgSyms)
				}
			}
		}
	}
}

func (d *dec) at(i int) byte {
	if i < 0 || i >= d.pos {
		return 0
	}
	return d.out[i]
}

func imgDot(feat [16]int16, c0, c1 [8]int16) int {
	// pmaddwd of fc50/fc70 and fc60/fc80, paddd, +0x200, sar $10 (0x40496a).
	sum := 512
	for i := 0; i < 8; i++ {
		sum += int(feat[i])*int(c0[i]) + int(feat[8+i])*int(c1[i])
	}
	return sum >> 10
}

func imgAdapt(feat [16]int16, c0, c1 *[8]int16, res int16) {
	if res == 0 {
		return
	}
	for i := 0; i < 8; i++ {
		if feat[i] != 0 {
			if feat[i]^res < 0 {
				c0[i]--
			} else {
				c0[i]++
			}
		}
		if feat[8+i] != 0 {
			if feat[8+i]^res < 0 {
				c1[i]--
			} else {
				c1[i]++
			}
		}
	}
}

func (d *dec) imgRes(ch, pred int, feat [16]int16) int {
	rice := d.img.rice[ch]
	if rice < 0 {
		rice = 0
	}
	k := rice
	if k >= imgRiceN {
		k = imgRiceN - 1
	}
	ctx := 0
	// 0x4049a4: lea 4(pred); cmp $8; seta — unsigned, so |pred| > 4.
	if uint32(int32(pred)+4) > 8 {
		ctx = 1
	}
	s := decodeCDF(&d.r, d.img.cdf[ch][k][ctx][:], imgSyms)
	v := uint32(imgBase20[s])
	if e := imgExtra20[s]; e != 0 {
		v += d.r.bits(int(e))
	}
	if rice != 0 {
		v = v<<uint(rice) | d.r.bits(rice)
	}
	// 0x404ae9 / 0x408170: ++ if v > 2<<rice, -- if v < 1<<rice.
	if int(v) > 2<<rice {
		d.img.rice[ch] = rice + 1
	} else if int(v) < 1<<rice {
		d.img.rice[ch] = rice - (rice+31)>>5
	}
	res := unzig(int(v))
	imgAdapt(feat, &d.img.coeff[ch][0], &d.img.coeff[ch][1], int16(res))
	return res
}

func (d *dec) tokImagePred() bool {
	if d.img.bit.bitSh(&d.r, 5) == 1 {
		d.img.width = int(d.r.bits(16))
	}
	w := d.img.width
	for i := 0; i < 12; i += 3 {
		p := d.pos
		n, nn := p-w, p-2*w
		var feat [16]int16
		// 0x40483f..0x404953. rsi=north, rbx=nn, rdi=dest.
		feat[0] = int16(int(d.at(n)) - int(d.at(nn)))
		feat[1] = int16(int(d.at(n+1)) - int(d.at(nn+1)))
		feat[2] = int16(int(d.at(n+2)) - int(d.at(nn+2)))
		feat[3] = int16(int(d.at(n)) - int(d.at(n-3)))
		feat[4] = int16(int(d.at(n+1)) - int(d.at(n-2)))
		feat[5] = int16(int(d.at(n+2)) - int(d.at(n-1)))
		feat[6] = int16(int(d.at(n)) - int(d.at(n+3)))
		feat[7] = int16(int(d.at(n+1)) - int(d.at(n+4)))
		feat[8] = int16(int(d.at(n+2)) - int(d.at(n+5)))
		feat[9] = int16(int(d.at(n-3)) - int(d.at(p-3)))
		feat[10] = int16(int(d.at(n-2)) - int(d.at(p-2)))
		feat[11] = int16(int(d.at(n-1)) - int(d.at(p-1)))
		// 0x40491e: WW - W (eax=WW, ecx=W).
		feat[12] = int16(int(d.at(p-6)) - int(d.at(p-3)))
		feat[13] = int16(int(d.at(p-5)) - int(d.at(p-2)))
		feat[14] = int16(int(d.at(p-4)) - int(d.at(p-1)))

		westG := d.at(p - 2)
		predG := imgDot(feat, d.img.coeff[0][0], d.img.coeff[0][1])
		g := byte(int(westG) + predG + d.imgRes(0, predG, feat))
		feat[15] = int16(int(g) - int(westG))

		westR := d.at(p - 3)
		predR := imgDot(feat, d.img.coeff[1][0], d.img.coeff[1][1])
		r := byte(int(westR) + predR + d.imgRes(1, predR, feat))
		feat[0] = int16(int(r) - int(westR))

		westB := d.at(p - 1)
		predB := imgDot(feat, d.img.coeff[2][0], d.img.coeff[2][1])
		b := byte(int(westB) + predB + d.imgRes(2, predB, feat))

		d.emit(r)
		d.emit(g)
		d.emit(b)
	}
	return d.r.ok() && !d.overflow
}
