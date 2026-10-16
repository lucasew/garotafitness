package jpgrec

// WriteJpeg translated from jpeg_data_writer.cc and jpeg_bit_writer.h.

type jpgCodeTab struct {
	depth [256]int
	code  [256]int
}

func buildJPGCodes(h *jpegHuff) (jpgCodeTab, bool) {
	var tab jpgCodeTab
	var huffCode [256]int
	var huffSize [257]int
	p := 0
	for l := 1; l <= 16; l++ {
		i := h.counts[l]
		if p+i > 257 {
			return tab, false
		}
		for i > 0 {
			huffSize[p] = l
			p++
			i--
		}
	}
	if p == 0 {
		return tab, true
	}
	last := p - 1
	huffSize[last] = 0
	c := 0
	si := huffSize[0]
	p = 0
	for huffSize[p] != 0 {
		for huffSize[p] == si {
			huffCode[p] = c
			p++
			c++
		}
		c <<= 1
		si++
	}
	for p = 0; p < last; p++ {
		i := h.values[p]
		if i < 0 || i > 255 {
			return tab, false
		}
		tab.depth[i] = huffSize[p]
		tab.code[i] = huffCode[p]
	}
	return tab, true
}

func hasZeroByte(x uint64) uint64 {
	return (x - 0x0101010101010101) &^ x & 0x8080808080808080
}

type jpgBits struct {
	data    []byte
	pos     int
	put     uint64
	putBits int
	over    bool
	bad     bool
}

func newJpgBits() *jpgBits {
	return &jpgBits{data: make([]byte, 1<<17), putBits: 64}
}

func (w *jpgBits) emit(b int) {
	if w.pos < len(w.data) {
		w.data[w.pos] = byte(b)
		w.pos++
	} else {
		w.over = true
	}
	if b == 0xFF {
		w.emit(0)
	}
}

func (w *jpgBits) write(nbits int, bits uint64) {
	if nbits == 0 {
		w.bad = true
		return
	}
	w.putBits -= nbits
	w.put |= bits << uint(w.putBits)
	if w.putBits <= 16 {
		if hasZeroByte(^w.put|0xFFFF) != 0 {
			w.emit(int((w.put >> 56) & 0xFF))
			w.emit(int((w.put >> 48) & 0xFF))
			w.emit(int((w.put >> 40) & 0xFF))
			w.emit(int((w.put >> 32) & 0xFF))
			w.emit(int((w.put >> 24) & 0xFF))
			w.emit(int((w.put >> 16) & 0xFF))
		} else if w.pos+6 < len(w.data) {
			w.data[w.pos] = byte((w.put >> 56) & 0xFF)
			w.data[w.pos+1] = byte((w.put >> 48) & 0xFF)
			w.data[w.pos+2] = byte((w.put >> 40) & 0xFF)
			w.data[w.pos+3] = byte((w.put >> 32) & 0xFF)
			w.data[w.pos+4] = byte((w.put >> 24) & 0xFF)
			w.data[w.pos+5] = byte((w.put >> 16) & 0xFF)
			w.pos += 6
		} else {
			w.over = true
		}
		w.put <<= 48
		w.putBits += 48
	}
}

func (w *jpgBits) marker(m int) {
	if w.pos+1 < len(w.data) {
		w.data[w.pos] = 0xFF
		w.data[w.pos+1] = byte(m)
		w.pos += 2
	} else {
		w.over = true
	}
}

func (w *jpgBits) jump(pad byte) {
	for w.putBits <= 56 {
		w.emit(int((w.put >> 56) & 0xFF))
		w.put <<= 8
		w.putBits += 8
	}
	if w.putBits < 64 {
		padMask := uint64(0xFF) >> uint(64-w.putBits)
		c := ((w.put >> 56) &^ padMask) | uint64(pad)
		w.emit(int(c))
	}
	w.put = 0
	w.putBits = 64
}

type dctState struct {
	eob int
	ac  *jpgCodeTab
	ref []int
}

func (s *dctState) flush(w *jpgBits) {
	if s.eob > 0 {
		nbits := log2FloorNonZero(uint32(s.eob))
		symbol := nbits << 4
		w.write(s.ac.depth[symbol], uint64(s.ac.code[symbol]))
		if nbits > 0 {
			w.write(nbits, uint64(s.eob)&((1<<uint(nbits))-1))
		}
		s.eob = 0
	}
	for _, b := range s.ref {
		w.write(1, uint64(b))
	}
	s.ref = s.ref[:0]
}

func (s *dctState) bufferEOB(ac *jpgCodeTab, bits []int, w *jpgBits) {
	if s.eob == 0 {
		s.ac = ac
	}
	s.eob++
	if bits != nil {
		s.ref = append(s.ref, bits...)
	}
	if s.eob == 0x7FFF || len(s.ref) > (1<<16)-64+1 {
		s.flush(w)
	}
}

func encodeSeq(coeffs []int16, dc, ac *jpgCodeTab, zeroRuns int, last *int16, w *jpgBits) {
	temp2 := int32(coeffs[0])
	temp := int16(temp2 - int32(*last))
	*last = int16(temp2)
	t2 := temp
	mag := temp
	if temp < 0 {
		mag = -temp
		t2--
	}
	nbits := 0
	if mag != 0 {
		nbits = log2FloorNonZero(uint32(int32(mag))) + 1
	}
	w.write(dc.depth[nbits], uint64(dc.code[nbits]))
	if nbits > 0 {
		w.write(nbits, uint64(uint32(int32(t2)))&((1<<uint(nbits))-1))
	}
	r := 0
	for k := 1; k < 64; k++ {
		t := coeffs[kJPEGNaturalOrder[k]]
		if t == 0 {
			r++
			continue
		}
		var t2a int16
		if t < 0 {
			t = -t
			t2a = ^t
		} else {
			t2a = t
		}
		for r > 15 {
			w.write(ac.depth[0xF0], uint64(ac.code[0xF0]))
			r -= 16
		}
		nb := log2FloorNonZero(uint32(int32(t))) + 1
		sym := (r << 4) + nb
		w.write(ac.depth[sym], uint64(ac.code[sym]))
		w.write(nb, uint64(uint32(int32(t2a)))&((1<<uint(nb))-1))
		r = 0
	}
	for i := 0; i < zeroRuns; i++ {
		w.write(ac.depth[0xF0], uint64(ac.code[0xF0]))
		r -= 16
	}
	if r > 0 {
		w.write(ac.depth[0], uint64(ac.code[0]))
	}
}

func encodeProg(coeffs []int16, dc, ac *jpgCodeTab, ss, se, al, zeroRuns int, st *dctState, last *int16, w *jpgBits) {
	eobOK := ss > 0
	if ss == 0 {
		temp2 := int32(coeffs[0]) >> uint(al)
		temp := int16(temp2 - int32(*last))
		*last = int16(temp2)
		t2 := temp
		mag := temp
		if temp < 0 {
			mag = -temp
			t2--
		}
		nbits := 0
		if mag != 0 {
			nbits = log2FloorNonZero(uint32(int32(mag))) + 1
		}
		w.write(dc.depth[nbits], uint64(dc.code[nbits]))
		if nbits > 0 {
			w.write(nbits, uint64(uint32(int32(t2)))&((1<<uint(nbits))-1))
		}
		ss++
	}
	if ss > se {
		return
	}
	r := 0
	for k := ss; k <= se; k++ {
		t := coeffs[kJPEGNaturalOrder[k]]
		var t2 int16
		if t == 0 {
			r++
			continue
		}
		if t < 0 {
			t = -t
			t >>= uint(al)
			t2 = ^t
		} else {
			t >>= uint(al)
			t2 = t
		}
		if t == 0 {
			r++
			continue
		}
		st.flush(w)
		for r > 15 {
			w.write(ac.depth[0xF0], uint64(ac.code[0xF0]))
			r -= 16
		}
		nb := log2FloorNonZero(uint32(int32(t))) + 1
		sym := (r << 4) + nb
		w.write(ac.depth[sym], uint64(ac.code[sym]))
		w.write(nb, uint64(uint32(int32(t2)))&((1<<uint(nb))-1))
		r = 0
	}
	if zeroRuns > 0 {
		st.flush(w)
		for i := 0; i < zeroRuns; i++ {
			w.write(ac.depth[0xF0], uint64(ac.code[0xF0]))
			r -= 16
		}
	}
	if r > 0 {
		st.bufferEOB(ac, nil, w)
		if !eobOK {
			st.flush(w)
		}
	}
}

func encodeRefine(coeffs []int16, ac *jpgCodeTab, ss, se, al int, st *dctState, w *jpgBits) {
	eobOK := ss > 0
	if ss == 0 {
		w.write(1, uint64((int32(coeffs[0])>>uint(al))&1))
		ss++
	}
	if ss > se {
		return
	}
	var absV [64]int
	eob := 0
	for k := ss; k <= se; k++ {
		av := coeffs[kJPEGNaturalOrder[k]]
		if av < 0 {
			av = -av
		}
		absV[k] = int(av) >> al
		if absV[k] == 1 {
			eob = k
		}
	}
	r := 0
	var ref []int
	for k := ss; k <= se; k++ {
		if absV[k] == 0 {
			r++
			continue
		}
		for r > 15 && k <= eob {
			st.flush(w)
			w.write(ac.depth[0xF0], uint64(ac.code[0xF0]))
			r -= 16
			for _, b := range ref {
				w.write(1, uint64(b))
			}
			ref = ref[:0]
		}
		if absV[k] > 1 {
			ref = append(ref, absV[k]&1)
			continue
		}
		st.flush(w)
		sym := (r << 4) + 1
		bit := 0
		if coeffs[kJPEGNaturalOrder[k]] >= 0 {
			bit = 1
		}
		w.write(ac.depth[sym], uint64(ac.code[sym]))
		w.write(1, uint64(bit))
		for _, b := range ref {
			w.write(1, uint64(b))
		}
		ref = ref[:0]
		r = 0
	}
	if r > 0 || len(ref) > 0 {
		st.bufferEOB(ac, ref, w)
		if !eobOK {
			st.flush(w)
		}
	}
}

type padCur struct {
	bits []int
	i    int
	on   bool
}

func (p *padCur) next(n int) (byte, bool) {
	if !p.on {
		if n >= 32 {
			return 0, false
		}
		return byte((1 << uint(n)) - 1), true
	}
	var v byte
	for n > 0 {
		n--
		v <<= 1
		if p.i >= len(p.bits) {
			return 0, false
		}
		v |= byte(p.bits[p.i])
		p.i++
	}
	return v, true
}

func encodeSOF(jpg *jpegPic, marker byte, out *[]byte) bool {
	n := len(jpg.comps)
	markerLen := 8 + 3*n
	data := make([]byte, markerLen+2)
	data[0], data[1] = 0xFF, marker
	data[2] = byte(markerLen >> 8)
	data[3] = byte(markerLen)
	data[4] = 8
	data[5] = byte(jpg.h >> 8)
	data[6] = byte(jpg.h)
	data[7] = byte(jpg.w >> 8)
	data[8] = byte(jpg.w)
	data[9] = byte(n)
	pos := 10
	for i := 0; i < n; i++ {
		c := &jpg.comps[i]
		if c.quant < 0 || c.quant >= len(jpg.quant) {
			return false
		}
		data[pos] = byte(c.id)
		data[pos+1] = byte((c.h << 4) | c.v)
		data[pos+2] = byte(jpg.quant[c.quant].index)
		pos += 3
	}
	*out = append(*out, data[:pos]...)
	return true
}

func encodeSOS(jpg *jpegPic, si *jpegScan, out *[]byte) bool {
	n := len(si.comps)
	markerLen := 6 + 2*n
	data := make([]byte, markerLen+2)
	data[0], data[1] = 0xFF, 0xDA
	data[2] = byte(markerLen >> 8)
	data[3] = byte(markerLen)
	data[4] = byte(n)
	pos := 5
	for i := 0; i < n; i++ {
		sc := si.comps[i]
		if sc.comp < 0 || sc.comp >= len(jpg.comps) {
			return false
		}
		data[pos] = byte(jpg.comps[sc.comp].id)
		data[pos+1] = byte((sc.dc << 4) + sc.ac)
		pos += 2
	}
	data[pos] = byte(si.ss)
	data[pos+1] = byte(si.se)
	data[pos+2] = byte((si.ah << 4) | si.al)
	*out = append(*out, data[:pos+3]...)
	return true
}

func encodeDHT(huffs []jpegHuff, idx *int, out *[]byte, dc, ac []jpgCodeTab) bool {
	markerLen := 2
	for i := *idx; i < len(huffs); i++ {
		h := &huffs[i]
		markerLen += 16
		for _, c := range h.counts {
			markerLen += c
		}
		if h.isLast {
			break
		}
	}
	data := make([]byte, 0, markerLen+2)
	data = append(data, 0xFF, 0xC4, byte(markerLen>>8), byte(markerLen))
	for {
		hi := *idx
		*idx = hi + 1
		if hi >= len(huffs) {
			return false
		}
		h := &huffs[hi]
		index := h.slot
		var tab *jpgCodeTab
		if index&0x10 != 0 {
			index -= 0x10
			if index < 0 || index >= len(ac) {
				return false
			}
			tab = &ac[index]
		} else {
			if index < 0 || index >= len(dc) {
				return false
			}
			tab = &dc[index]
		}
		built, ok := buildJPGCodes(h)
		if !ok {
			return false
		}
		*tab = built
		total := 0
		maxLen := 0
		for i, c := range h.counts {
			if c != 0 {
				maxLen = i
			}
			total += c
		}
		total--
		data = append(data, byte(h.slot))
		for i := 1; i <= 16; i++ {
			c := h.counts[i]
			if i == maxLen {
				c--
			}
			data = append(data, byte(c))
		}
		for i := 0; i < total; i++ {
			data = append(data, byte(h.values[i]))
		}
		if h.isLast {
			break
		}
	}
	*out = append(*out, data...)
	return true
}

func encodeDQT(jpg *jpegPic, idx *int, out *[]byte) bool {
	markerLen := 2
	for i := *idx; i < len(jpg.quant); i++ {
		q := &jpg.quant[i]
		n := 1
		if q.precision != 0 {
			n = 2
		}
		markerLen += 1 + n*64
		if q.isLast {
			break
		}
	}
	data := make([]byte, 0, markerLen+2)
	data = append(data, 0xFF, 0xDB, byte(markerLen>>8), byte(markerLen))
	for {
		qi := *idx
		*idx = qi + 1
		if qi >= len(jpg.quant) {
			return false
		}
		q := &jpg.quant[qi]
		data = append(data, byte((q.precision<<4)+q.index))
		for i := 0; i < 64; i++ {
			val := int(q.values[kJPEGNaturalOrder[i]])
			if q.precision != 0 {
				data = append(data, byte(val>>8))
			}
			data = append(data, byte(val))
		}
		if q.isLast {
			break
		}
	}
	*out = append(*out, data...)
	return true
}

func encodeScan(jpg *jpegPic, si *jpegScan, progressive bool, dc, ac []jpgCodeTab, restart int, pad *padCur, out *[]byte) bool {
	if !encodeSOS(jpg, si, out) {
		return false
	}
	interleaved := len(si.comps) > 1
	var mcusPerRow, mcuRows int
	if interleaved {
		mcusPerRow = divCeil(jpg.w, jpg.maxH*8)
		mcuRows = divCeil(jpg.h, jpg.maxV*8)
	} else {
		c := &jpg.comps[si.comps[0].comp]
		mcusPerRow = divCeil(jpg.w*c.h, 8*jpg.maxH)
		mcuRows = divCeil(jpg.h*c.v, 8*jpg.maxV)
	}
	var lastDC [4]int16
	bw := newJpgBits()
	restarts := restart
	nextRST := 0
	blockScan := 0
	extraPos := 0
	nextExtra := -1
	if len(si.extra) > 0 {
		nextExtra = si.extra[0].block
	}
	var coding dctState
	al, ah, ss, se := 0, 0, 0, 63
	if progressive {
		al, ah, ss, se = si.al, si.ah, si.ss, si.se
	}
	sequential := !progressive || (ah == 0 && al == 0 && ss == 0 && se == 63)
	for mcuY := 0; mcuY < mcuRows; mcuY++ {
		for mcuX := 0; mcuX < mcusPerRow; mcuX++ {
			if restart > 0 && restarts == 0 {
				coding.flush(bw)
				nPad := bw.putBits & 7
				pat, ok := pad.next(nPad)
				if !ok {
					return false
				}
				bw.jump(pat)
				bw.marker(0xD0 + nextRST)
				nextRST = (nextRST + 1) & 7
				restarts = restart
				lastDC = [4]int16{}
			}
			for i := range si.comps {
				sc := si.comps[i]
				if sc.comp < 0 || sc.comp >= len(jpg.comps) || sc.dc < 0 || sc.dc >= len(dc) || sc.ac < 0 || sc.ac >= len(ac) {
					return false
				}
				c := &jpg.comps[sc.comp]
				ny, nx := 1, 1
				if interleaved {
					ny, nx = c.v, c.h
				}
				for iy := 0; iy < ny; iy++ {
					for ix := 0; ix < nx; ix++ {
						by := mcuY*ny + iy
						bx := mcuX*nx + ix
						block := by*c.wBlocks + bx
						if _, ok := si.reset[blockScan]; ok {
							coding.flush(bw)
						}
						zeroRuns := 0
						if blockScan == nextExtra {
							zeroRuns = si.extra[extraPos].n
							extraPos++
							if extraPos < len(si.extra) {
								nextExtra = si.extra[extraPos].block
							} else {
								nextExtra = -1
							}
						}
						if block < 0 || (block+1)*64 > len(c.coeffs) {
							return false
						}
						coeffs := c.coeffs[block*64:]
						if sequential {
							encodeSeq(coeffs, &dc[sc.dc], &ac[sc.ac], zeroRuns, &lastDC[sc.comp], bw)
						} else if ah == 0 {
							encodeProg(coeffs, &dc[sc.dc], &ac[sc.ac], ss, se, al, zeroRuns, &coding, &lastDC[sc.comp], bw)
						} else {
							encodeRefine(coeffs, &ac[sc.ac], ss, se, al, &coding, bw)
						}
						blockScan++
					}
				}
			}
			restarts--
			if bw.pos > 1<<16 {
				*out = append(*out, bw.data[:bw.pos]...)
				bw.pos = 0
			}
		}
	}
	coding.flush(bw)
	nPad := bw.putBits & 7
	pat, ok := pad.next(nPad)
	if !ok {
		return false
	}
	bw.jump(pat)
	if bw.over || bw.bad {
		return false
	}
	*out = append(*out, bw.data[:bw.pos]...)
	return true
}

func writeJPEG(jpg *jpegPic) ([]byte, error) {
	if jpg.version == 1 {
		if jpg.orig == nil {
			return nil, errBr
		}
		return append([]byte(nil), jpg.orig...), nil
	}
	if len(jpg.order) == 0 {
		return nil, errBr
	}
	out := []byte{0xFF, 0xD8}
	dht, dqt, appI, comI, dataI, scanI := 0, 0, 0, 0, 0, 0
	var dc [4]jpgCodeTab
	var ac [4]jpgCodeTab
	pad := padCur{on: jpg.hasPad, bits: jpg.pad}
	seenDRI := false
	progressive := false
	for _, marker := range jpg.order {
		ok := true
		switch marker {
		case 0xC0, 0xC1, 0xC2:
			progressive = marker == 0xC2
			ok = encodeSOF(jpg, marker, &out)
		case 0xC9, 0xCA:
			ok = encodeSOF(jpg, marker, &out)
		case 0xC4:
			ok = encodeDHT(jpg.huff, &dht, &out, dc[:], ac[:])
		case 0xD0, 0xD1, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7:
			out = append(out, 0xFF, marker)
		case 0xD9:
		case 0xDA:
			if scanI >= len(jpg.scans) {
				return nil, errBr
			}
			ri := 0
			if seenDRI {
				ri = jpg.restart
			}
			ok = encodeScan(jpg, &jpg.scans[scanI], progressive, dc[:], ac[:], ri, &pad, &out)
			scanI++
		case 0xDB:
			ok = encodeDQT(jpg, &dqt, &out)
		case 0xDD:
			seenDRI = true
			out = append(out, 0xFF, 0xDD, 0x00, 0x04, byte(jpg.restart>>8), byte(jpg.restart))
		case 0xE0, 0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7, 0xE8, 0xE9, 0xEA, 0xEB, 0xEC, 0xED, 0xEE, 0xEF:
			if appI >= len(jpg.app) {
				return nil, errBr
			}
			out = append(out, 0xFF)
			out = append(out, jpg.app[appI]...)
			appI++
		case 0xFE:
			if comI >= len(jpg.com) {
				return nil, errBr
			}
			out = append(out, 0xFF, 0xFE)
			out = append(out, jpg.com[comI]...)
			comI++
		case 0xFF:
			if dataI >= len(jpg.inter) {
				return nil, errBr
			}
			out = append(out, jpg.inter[dataI]...)
			dataI++
		default:
			ok = false
		}
		if !ok {
			return nil, errBr
		}
	}
	out = append(out, 0xFF, 0xD9)
	out = append(out, jpg.tail...)
	return out, nil
}
