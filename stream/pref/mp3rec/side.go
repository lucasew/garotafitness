package mp3rec

import "fmt"

// PMP header, side information, and frame assembly from packmp3.cpp.

var (
	errUnexpected = errf("unexpected end of data")
)

func errf(format string, args ...any) error {
	return fmt.Errorf("mp3rec: "+format, args...)
}

func errLimit(n int) error {
	return errf("recompressed length %d exceeds MP3_MAX_MEMORY_SIZE (%d)", n, mp3MaxMemorySize)
}

type session struct {
	r            *memReader
	iSamplerate  int32
	iChannels    int32
	iBitrate     int32
	iPadding     int32
	iStereoMS    int32
	iStereoInt   int32
	iSblocks     int32
	iSbGain      int32
	iShare       int32
	iPreemphasis int32
	iCoarse      int32
	iProtection  int32
	iOriginal    int32
	iCopyright   int32
	iPrivbit     int32
	iEmphasis    int32
	dataBeforeN  int32
	dataAfterN   int32
	iBitRes      int32
	iSbDiff      int32
	nBadFirst    int32
	iMpeg        int32
	iLayer       int32
	iPadbits     int32
	iMixed       int32
	gNframes     int32
	gNchannels   int32
	first        *mp3Frame
	last         *mp3Frame
	lastGran     [2]*granule
	frameN       int32
	unmute       []byte
	dataBefore   []byte
	dataAfter    []byte
	gg           [2][]byte
	pred         lamePred
}

func (s *session) readHeader() error {
	hb, err := s.r.readExact(4)
	if err != nil {
		return errUnexpected
	}
	nb, err := s.r.readExact(4)
	if err != nil {
		return errUnexpected
	}
	s.iSamplerate = int32((hb[0] >> 6) & 0x3)
	s.iChannels = int32((hb[0] >> 4) & 0x3)
	s.iBitrate = int32(hb[0] & 0xF)
	if s.iBitrate == 0 {
		s.iBitrate = -1
	}
	s.iPadding = -int32((hb[1] >> 7) & 0x1)
	s.iStereoMS = -int32((hb[1] >> 6) & 0x1)
	s.iStereoInt = int32((hb[1] >> 5) & 0x1)
	s.iSblocks = int32((hb[1] >> 4) & 0x1)
	s.iSbGain = -int32((hb[1] >> 3) & 0x1)
	s.iShare = -int32((hb[1] >> 2) & 0x1)
	s.iPreemphasis = -int32((hb[1] >> 1) & 0x1)
	s.iCoarse = -int32(hb[1] & 0x1)
	s.iProtection = int32((hb[2] >> 7) & 0x1)
	s.iOriginal = int32((hb[2] >> 6) & 0x1)
	s.iCopyright = int32((hb[2] >> 5) & 0x1)
	s.iPrivbit = int32((hb[2] >> 4) & 0x1)
	s.iEmphasis = int32((hb[2] >> 2) & 0x3)
	s.dataBeforeN = int32((hb[2] >> 1) & 0x1)
	s.dataAfterN = int32(hb[2] & 0x1)
	s.iBitRes = -int32((hb[3] >> 7) & 0x1)
	s.iSbDiff = int32((hb[3] >> 6) & 0x1)
	s.nBadFirst = int32((hb[3] >> 5) & 0x1)
	s.iMpeg = mp3V1
	s.iLayer = layerIII
	s.iPadbits = 0
	s.iMixed = 0
	if s.iBitrate == 0xF || s.iSamplerate == 0x3 {
		return errf("not a proper PMP file")
	}
	s.gNframes = int32(uint32(nb[0])<<24 | uint32(nb[1])<<16 | uint32(nb[2])<<8 | uint32(nb[3]))
	if s.gNframes <= 0 {
		return errf("not a proper PMP file")
	}
	if s.gNframes > mp3MaxMemorySize/minFrameBytes {
		return errLimit(int(s.gNframes))
	}
	for n := int32(0); n < s.gNframes; n++ {
		fr := s.buildFrame()
		s.appendFrame(fr)
	}
	if s.iChannels == mp3Mono {
		s.gNchannels = 1
	} else {
		s.gNchannels = 2
	}
	return nil
}

func (s *session) buildFrame() *mp3Frame {
	f := &mp3Frame{
		mpeg:       int8(s.iMpeg),
		layer:      int8(s.iLayer),
		protection: int8(s.iProtection),
		bits:       int8(s.iBitrate),
		samples:    int8(s.iSamplerate),
		padding:    int8(s.iPadding),
		privbit:    int8(s.iPrivbit),
		channels:   int8(s.iChannels),
		stereoMS:   int8(s.iStereoMS),
		stereoInt:  int8(s.iStereoInt),
		copyright:  int8(s.iCopyright),
		original:   int8(s.iOriginal),
		emphasis:   int8(s.iEmphasis),
		padbits:    int8(s.iPadbits),
	}
	nch := 2
	if f.channels == mp3Mono {
		if f.protection != 0 {
			f.fixedSize = 4 + 2 + 17
		} else {
			f.fixedSize = 4 + 17
		}
		f.nchannels = 1
		nch = 1
	} else {
		if f.protection != 0 {
			f.fixedSize = 4 + 2 + 32
		} else {
			f.fixedSize = 4 + 32
		}
		f.nchannels = 2
	}
	f.granules = make([][]*granule, nch)
	for ch := 0; ch < nch; ch++ {
		f.granules[ch] = []*granule{&granule{}, &granule{}}
		for gr := 0; gr < 2; gr++ {
			g := f.granules[ch][gr]
			if gr == 0 {
				g.share = int8(s.iShare)
			}
			g.windowSwitching = int8(s.iSblocks)
			g.mixed = int8(s.iMixed)
			g.blockType = longBlock
			g.sbGain[0] = int8(s.iSbGain)
			g.sbGain[1] = int8(s.iSbGain)
			g.sbGain[2] = int8(s.iSbGain)
			g.preemphasis = int8(s.iPreemphasis)
			g.coarse = int8(s.iCoarse)
		}
	}
	return f
}

func (s *session) appendFrame(f *mp3Frame) {
	if s.last == nil {
		s.first = f
		f.prev = nil
		s.lastGran = [2]*granule{}
		s.frameN = 0
	} else {
		s.last.next = f
		f.prev = s.last
	}
	s.last = f
	nch := int(f.nchannels)
	for ch := 0; ch < nch; ch++ {
		f.granules[ch][0].n = s.frameN << 1
		f.granules[ch][1].n = f.granules[ch][0].n | 1
		f.granules[ch][0].next = f.granules[ch][1]
		f.granules[ch][1].prev = f.granules[ch][0]
		f.granules[ch][0].prev = s.lastGran[ch]
		f.granules[ch][1].next = nil
		if s.lastGran[ch] != nil {
			s.lastGran[ch].next = f.granules[ch][0]
		}
		s.lastGran[ch] = f.granules[ch][1]
	}
	f.n = s.frameN
	s.frameN++
	f.next = nil
}

func (s *session) unstoreUnmute() error {
	b, err := s.r.readExact(1)
	if err != nil {
		return errUnexpected
	}
	n := int(b[0])
	ums := 2 + int(s.gNchannels)*2*4
	rest := n * ums
	buf := make([]byte, 1+rest)
	buf[0] = b[0]
	if rest > 0 {
		rb, err := s.r.readExact(rest)
		if err != nil {
			return errUnexpected
		}
		copy(buf[1:], rb)
	}
	s.unmute = buf
	s.nBadFirst = 0
	return nil
}

func (s *session) unmuteFrames() error {
	if len(s.unmute) == 0 {
		return nil
	}
	n := int(s.unmute[0])
	ums := 2 + int(s.gNchannels)*2*4
	fr := s.first
	for i := 0; i < n; i++ {
		if fr == nil {
			return errf("unmute frame missing")
		}
		if err := unmuteFrame(fr, s.unmute[1+i*ums:], int(s.gNchannels)); err != nil {
			return err
		}
		fr = fr.next
	}
	return nil
}

func unmuteFrame(f *mp3Frame, p []byte, nch int) error {
	need := 2 + nch*2*4
	if len(p) < need {
		return errUnexpected
	}
	f.privbit = int8((p[0] >> 7) & 1)
	f.original = int8((p[0] >> 6) & 1)
	f.copyright = int8((p[0] >> 5) & 1)
	f.bitReservoir = int16(uint16(p[0])<<8 | uint16(p[1]))
	o := 2
	for ch := 0; ch < nch; ch++ {
		for gr := 0; gr < 2; gr++ {
			g := f.granules[ch][gr]
			g.mainDataBit = int16(p[o]) << 4
			g.mainDataBit |= int16(p[o+1] >> 4)
			g.slength = int8(p[o+1] & 0x0F)
			g.bigValPairs = int16(p[o+2]) << 1
			g.bigValPairs |= int16(p[o+3] >> 7)
			o += 4
		}
	}
	return nil
}

func u32signed(v int32) uint32 { return uint32(v) }

func layer3CRC(header []byte, side []byte) uint16 {
	crc := uint16(0xFFFF)
	crc = uint16(crc<<8) ^ crcTable[(crc>>8)^uint16(header[2])]
	crc = uint16(crc<<8) ^ crcTable[(crc>>8)^uint16(header[3])]
	for _, b := range side {
		crc = uint16(crc<<8) ^ crcTable[(crc>>8)^uint16(b)]
	}
	return crc
}

func buildFixed(f *mp3Frame) ([]byte, error) {
	fixed := make([]byte, 64)
	nch := int(f.nchannels)
	nsb := 17
	if nch != 1 {
		nsb = 32
	}
	sideOff := 4
	if f.protection != 0 {
		sideOff = 6
	}
	fixed[0] = 0xFF
	fixed[1] = 0xFA
	fixed[1] |= byte(uint32(int32(f.mpeg)) << 3)
	fixed[1] |= byte(uint32(int32(f.layer)) << 1)
	fixed[1] |= byte(uint32(int32(f.protection)) ^ 1)
	fixed[2] |= byte(uint32(int32(f.bits)) << 4)
	fixed[2] |= byte(uint32(int32(f.samples)) << 2)
	fixed[2] |= byte(uint32(int32(f.padding)) << 1)
	fixed[2] |= byte(uint32(int32(f.privbit)))
	fixed[3] |= byte(uint32(int32(f.channels)) << 6)
	fixed[3] |= byte(uint32(int32(f.stereoMS)) << 5)
	fixed[3] |= byte(uint32(int32(f.stereoInt)) << 4)
	fixed[3] |= byte(uint32(int32(f.copyright)) << 3)
	fixed[3] |= byte(uint32(int32(f.original)) << 2)
	fixed[3] |= byte(uint32(int32(f.emphasis)))

	w := newBitWriter(64)
	w.write(u32signed(int32(f.bitReservoir)), 9)
	if nch == 1 {
		w.write(u32signed(int32(f.padbits)), 5)
	} else {
		w.write(u32signed(int32(f.padbits)), 3)
	}
	for ch := 0; ch < nch; ch++ {
		w.write(u32signed(int32(f.granules[ch][0].share)), 4)
	}
	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < nch; ch++ {
			g := f.granules[ch][gr]
			w.write(u32signed(int32(g.mainDataBit)), 12)
			w.write(u32signed(int32(g.bigValPairs)), 9)
			w.write(u32signed(int32(g.globalGain)), 8)
			w.write(u32signed(int32(g.slength)), 4)
			w.write(u32signed(int32(g.windowSwitching)), 1)
			if g.windowSwitching == 0 {
				w.write(u32signed(int32(g.regionTable[0])), 5)
				w.write(u32signed(int32(g.regionTable[1])), 5)
				w.write(u32signed(int32(g.regionTable[2])), 5)
				w.write(u32signed(int32(g.region0)), 4)
				w.write(u32signed(int32(g.region1)), 3)
			} else {
				w.write(u32signed(int32(g.blockType)), 2)
				w.write(u32signed(int32(g.mixed)), 1)
				w.write(u32signed(int32(g.regionTable[0])), 5)
				w.write(u32signed(int32(g.regionTable[1])), 5)
				w.write(u32signed(int32(g.sbGain[0])), 3)
				w.write(u32signed(int32(g.sbGain[1])), 3)
				w.write(u32signed(int32(g.sbGain[2])), 3)
			}
			w.write(u32signed(int32(g.preemphasis)), 1)
			w.write(u32signed(int32(g.coarse)), 1)
			w.write(u32signed(int32(g.selectHtabB)), 1)
		}
	}
	side := w.bytes()
	if len(side) < nsb {
		return nil, errf("side info short")
	}
	copy(fixed[sideOff:], side[:nsb])
	if f.protection == 1 {
		crc := layer3CRC(fixed, fixed[sideOff:sideOff+nsb])
		fixed[4] = byte(crc >> 8)
		fixed[5] = byte(crc)
	}
	return fixed, nil
}

func (s *session) writeMP3(main []byte) ([]byte, error) {
	out := make([]byte, 0, len(s.dataBefore)+len(s.dataAfter)+len(main)+int(s.gNframes)*38)
	out = append(out, s.dataBefore...)
	off := 0
	for fr := s.first; fr != nil; fr = fr.next {
		fixed, err := buildFixed(fr)
		if err != nil {
			return nil, err
		}
		out = append(out, fixed[:fr.fixedSize]...)
		body := int(fr.frameSize - fr.fixedSize)
		if body < 0 || off+body > len(main) {
			return nil, errf("main data size mismatch")
		}
		out = append(out, main[off:off+body]...)
		off += body
	}
	if off != len(main) {
		return nil, errf("main data size mismatch")
	}
	out = append(out, s.dataAfter...)
	if len(out) > mp3MaxMemorySize {
		return nil, errLimit(len(out))
	}
	return out, nil
}

func (s *session) decodeID3(dec *arith) error {
	wantBefore := s.dataBeforeN > 0
	wantAfter := s.dataAfterN > 0
	model := newModelS(257, 256, 0)
	if wantBefore {
		buf, err := decodeID3Chunk(dec, model)
		if err != nil {
			return err
		}
		s.dataBefore = buf
		s.dataBeforeN = int32(len(buf))
	}
	if wantAfter {
		buf, err := decodeID3Chunk(dec, model)
		if err != nil {
			return err
		}
		s.dataAfter = buf
		s.dataAfterN = int32(len(buf))
	}
	return nil
}

func decodeID3Chunk(dec *arith, model *modelS) ([]byte, error) {
	var buf []byte
	for {
		c, err := dec.decodeS(model)
		if err != nil {
			return nil, err
		}
		if c == 256 {
			break
		}
		if c < 0 || c > 255 {
			return nil, errf("id3 symbol %d", c)
		}
		if len(buf) >= mp3MaxMemorySize {
			return nil, errLimit(len(buf) + 1)
		}
		buf = append(buf, byte(c))
	}
	return buf, nil
}

func (s *session) decodePadding(dec *arith) error {
	model := newModelS(paddingMaxRun+1, paddingMaxRun+1, 1)
	run, err := dec.decodeS(model)
	if err != nil {
		return err
	}
	if err := model.shift(run); err != nil {
		return err
	}
	bit := int8(0)
	for fr := s.first; fr != nil; fr = fr.next {
		for run == 0 {
			bit ^= 1
			run, err = dec.decodeS(model)
			if err != nil {
				return err
			}
			if err := model.shift(run); err != nil {
				return err
			}
		}
		run--
		fr.padding = bit
	}
	return nil
}

func (s *session) decodeBlockTypes(dec *arith) error {
	modSw := newModelS(switchingMaxRun+1, switchingMaxRun+1, 1)
	modBt := newModelS(4, 4, 1)
	for ch := int32(0); ch < s.gNchannels; ch++ {
		if s.iSbDiff == 0 && ch == 1 {
			break
		}
		if err := modSw.shift(0); err != nil {
			return err
		}
		run, err := dec.decodeS(modSw)
		if err != nil {
			return err
		}
		if err := modSw.shift(run); err != nil {
			return err
		}
		bit := int8(0)
		for g := s.first.granules[ch][0]; g != nil; g = g.next {
			for run == 0 {
				bit ^= 1
				run, err = dec.decodeS(modSw)
				if err != nil {
					return err
				}
				if err := modSw.shift(run); err != nil {
					return err
				}
			}
			run--
			g.windowSwitching = bit
		}
		c := int32(stopBlock)
		for g := s.first.granules[ch][0]; g != nil; g = g.next {
			if g.windowSwitching != 0 {
				var ctx int32
				if c == stopBlock {
					ctx = startBlock
				} else if g.next != nil {
					if g.next.windowSwitching != 0 {
						ctx = shortBlock
					} else {
						ctx = stopBlock
					}
				} else {
					ctx = stopBlock
				}
				if err := modBt.shift(ctx); err != nil {
					return err
				}
				c, err = dec.decodeS(modBt)
				if err != nil {
					return err
				}
				g.blockType = int8(c)
			} else {
				g.blockType = longBlock
			}
		}
	}
	if s.iSbDiff == 0 && s.gNchannels == 2 {
		g1 := s.first.granules[1][0]
		for g0 := s.first.granules[0][0]; g0 != nil; g0 = g0.next {
			if g1 == nil {
				return errf("block type channel mismatch")
			}
			g1.windowSwitching = g0.windowSwitching
			g1.blockType = g0.blockType
			g1 = g1.next
		}
	}
	return nil
}

func (s *session) decodeGlobalGain(dec *arith) error {
	model := newModelS(256, 0, 0)
	last := int32(0)
	for g := s.first.granules[0][0]; g != nil; g = g.next {
		c, err := dec.decodeS(model)
		if err != nil {
			return err
		}
		last = (c + last) & 0xFF
		g.globalGain = int16(last)
	}
	if s.gNchannels == 2 {
		g0 := s.first.granules[0][0]
		for g1 := s.first.granules[1][0]; g1 != nil; g1 = g1.next {
			c, err := dec.decodeS(model)
			if err != nil {
				return err
			}
			g1.globalGain = int16((c + int32(g0.globalGain)) & 0xFF)
			g0 = g0.next
		}
	}
	return nil
}

func (s *session) buildContext() error {
	ngr := int(s.gNframes * 2)
	s.gg[0] = make([]byte, ngr)
	s.gg[1] = make([]byte, ngr)
	var count [2][256]int32
	i := 0
	for g := s.first.granules[0][0]; g != nil; g = g.next {
		if i >= ngr {
			return errf("global gain count")
		}
		s.gg[0][i] = byte(g.globalGain)
		count[0][s.gg[0][i]]++
		i++
	}
	for i := 1; i < 256; i++ {
		count[0][i] += count[0][i-1]
	}
	if s.gNchannels == 2 {
		i = 0
		for g := s.first.granules[1][0]; g != nil; g = g.next {
			if i >= ngr {
				return errf("global gain count")
			}
			s.gg[1][i] = byte(g.globalGain)
			count[1][s.gg[1][i]]++
			i++
		}
		for i := 1; i < 256; i++ {
			count[1][i] += count[1][i-1]
		}
	}
	for ch := int32(0); ch < s.gNchannels; ch++ {
		lbound := int32(0)
		inc0 := int32(0)
		for i := int32(0); i < 256-ggContextSize; i++ {
			inc1 := count[ch][i+ggContextSize] - count[ch][i]
			if inc1 >= inc0 {
				inc0 = inc1
				lbound = i
			}
		}
		gg := s.gg[ch]
		for i := 0; i < ngr; i++ {
			if int32(gg[i]) >= lbound {
				gg[i] -= byte(lbound)
				if gg[i] >= ggContextSize {
					gg[i] = ggContextSize - 1
				}
			} else {
				gg[i] = 0
			}
		}
	}
	return nil
}

func (s *session) decodeSlength(dec *arith) error {
	ctxN := int32(16)
	if ggContextSize > ctxN {
		ctxN = ggContextSize
	}
	model := newModelS(16, ctxN, 2)
	for ch := int32(0); ch < s.gNchannels; ch++ {
		model.flush(1)
		c := int32(0)
		gi := 0
		gg := s.gg[ch]
		for g := s.first.granules[ch][0]; g != nil; g = g.next {
			if gi >= len(gg) {
				return errf("slength context")
			}
			if err := shiftS(model, int32(gg[gi]), c); err != nil {
				return err
			}
			gi++
			var err error
			c, err = dec.decodeS(model)
			if err != nil {
				return err
			}
			g.slength = int8(c)
		}
	}
	return nil
}

func (s *session) decodeRegion(dec *arith) error {
	if s.iSamplerate < 0 || int(s.iSamplerate) >= len(mp3BandwidthConv) {
		return errf("samplerate %d", s.iSamplerate)
	}
	bwConv := mp3BandwidthConv[s.iSamplerate][:]
	bounds := mp3BandwidthBounds[s.iSamplerate][:]
	boundsShort := mp3BandwidthBoundsShort[s.iSamplerate][:]
	tCtx := int32(32)
	if ggContextSize > tCtx {
		tCtx = ggContextSize
	}
	modT0 := newModelS(32, tCtx, 2)
	modT1 := newModelS(32, 32, 2)
	modT2 := newModelS(32, 32, 2)
	modTs := newModelB(16, 1)
	modS0 := newModelS(16, 22, 2)
	modS1 := newModelS(8, 22, 2)
	modBv := newModelS(576/2+1, 2, 1)
	for ch := int32(0); ch < s.gNchannels; ch++ {
		modS0.flush(1)
		modS1.flush(1)
		modBv.flush(1)
		gg := s.gg[ch]
		gi := 0
		ctxSv := int32(0)
		tR0 := int32(0)
		sR0 := int32(0)
		sR2 := int32(0)
		for g := s.first.granules[ch][0]; g != nil; g = g.next {
			if gi >= len(gg) {
				return errf("region context")
			}
			if g.windowSwitching == 0 {
				if err := modBv.shift(0); err != nil {
					return err
				}
				bv, err := dec.decodeS(modBv)
				if err != nil {
					return err
				}
				g.bigValPairs = int16(bv)
				idx := int(g.bigValPairs) << 1
				if idx < 0 || idx >= len(bwConv) {
					return errf("bandwidth conv %d", idx)
				}
				sR2 = bwConv[idx]
				if err := shiftS(modS0, sR0, sR2); err != nil {
					return err
				}
				sR0, err = dec.decodeS(modS0)
				if err != nil {
					return err
				}
				g.region0 = int8(sR0)
				if err := shiftS(modS1, sR0, sR2); err != nil {
					return err
				}
				r1, err := dec.decodeS(modS1)
				if err != nil {
					return err
				}
				g.region1 = int8(r1)
				b0, err := bandAt(bounds, sR0+1)
				if err != nil {
					return err
				}
				b1, err := bandAt(bounds, sR0+int32(g.region1)+2)
				if err != nil {
					return err
				}
				g.regionBound[0] = b0
				g.regionBound[1] = b1
				g.regionBound[2] = int16(int32(g.bigValPairs) << 1)
				if g.regionBound[0] > g.regionBound[2] {
					g.regionBound[0] = g.regionBound[2]
					g.regionBound[1] = g.regionBound[2]
				} else if g.regionBound[1] > g.regionBound[2] {
					g.regionBound[1] = g.regionBound[2]
				}
				sR0++
			} else if g.blockType != shortBlock {
				g.region0 = 8
				g.region1 = 0
				b0, err := bandAt(bounds, 8)
				if err != nil {
					return err
				}
				g.regionBound[0] = b0
				if err := modBv.shift(0); err != nil {
					return err
				}
				bv, err := dec.decodeS(modBv)
				if err != nil {
					return err
				}
				g.bigValPairs = int16(bv)
				g.regionBound[1] = int16(int32(g.bigValPairs) << 1)
				if g.regionBound[0] > g.regionBound[1] {
					g.regionBound[0] = g.regionBound[1]
				}
				g.regionBound[2] = g.regionBound[1]
				sR0 = 0
			} else {
				g.region0 = 9
				g.region1 = 0
				b0, err := bandAt(boundsShort, 9/3)
				if err != nil {
					return err
				}
				g.regionBound[0] = b0 * 3
				if err := modBv.shift(1); err != nil {
					return err
				}
				bv, err := dec.decodeS(modBv)
				if err != nil {
					return err
				}
				g.bigValPairs = int16(bv)
				g.regionBound[1] = int16(int32(g.bigValPairs) << 1)
				if g.regionBound[0] > g.regionBound[1] {
					g.regionBound[0] = g.regionBound[1]
				}
				g.regionBound[2] = g.regionBound[1]
				sR0 = 0
			}
			if err := shiftS(modT0, int32(gg[gi]), tR0); err != nil {
				return err
			}
			var err error
			tR0, err = dec.decodeS(modT0)
			if err != nil {
				return err
			}
			g.regionTable[0] = int8(tR0)
			if err := shiftS(modT1, tR0, sR0); err != nil {
				return err
			}
			tR1, err := dec.decodeS(modT1)
			if err != nil {
				return err
			}
			g.regionTable[1] = int8(tR1)
			if g.windowSwitching == 0 {
				if err := shiftS(modT2, tR0, tR1); err != nil {
					return err
				}
				tR2, err := dec.decodeS(modT2)
				if err != nil {
					return err
				}
				g.regionTable[2] = int8(tR2)
			} else {
				g.regionTable[2] = 0
			}
			if err := modTs.shift(ctxSv); err != nil {
				return err
			}
			tSv, err := dec.decodeB(modTs)
			if err != nil {
				return err
			}
			g.selectHtabB = int8(tSv)
			ctxSv = ((ctxSv << 1) | tSv) & 0xF
			gi++
		}
	}
	return nil
}

func bandAt(tab []int32, i int32) (int16, error) {
	if i < 0 || int(i) >= len(tab) {
		return 0, errf("bandwidth index %d", i)
	}
	return int16(tab[i]), nil
}

func (s *session) decodeSharing(dec *arith) error {
	model := newModelS(16, 16, 3)
	for ch := int32(0); ch < s.gNchannels; ch++ {
		model.flush(1)
		c := int32(0)
		for g := s.first.granules[ch][0]; g != nil; g = g.next.next {
			if g.next == nil {
				return errf("sharing granule")
			}
			if err := shiftS(model, c, int32(g.slength), int32(g.next.slength)); err != nil {
				return err
			}
			var err error
			c, err = dec.decodeS(model)
			if err != nil {
				return err
			}
			g.share = int8(c)
		}
	}
	return nil
}

func (s *session) decodeBitChain(dec *arith, set func(*granule, int8)) error {
	model := newModelB(16, 1)
	for ch := int32(0); ch < s.gNchannels; ch++ {
		model.flush(1)
		ctx := int32(0)
		for g := s.first.granules[ch][0]; g != nil; g = g.next {
			if err := model.shift(ctx); err != nil {
				return err
			}
			c, err := dec.decodeB(model)
			if err != nil {
				return err
			}
			set(g, int8(c))
			ctx = ((ctx << 1) | c) & 0xF
		}
	}
	return nil
}

func (s *session) decodePreemphasis(dec *arith) error {
	return s.decodeBitChain(dec, func(g *granule, c int8) { g.preemphasis = c })
}

func (s *session) decodeCoarse(dec *arith) error {
	return s.decodeBitChain(dec, func(g *granule, c int8) { g.coarse = c })
}

func (s *session) decodeSubblock(dec *arith) error {
	model := newModelS(8, 8, 1)
	if err := model.shift(0); err != nil {
		return err
	}
	for ch := int32(0); ch < s.gNchannels; ch++ {
		for g := s.first.granules[ch][0]; g != nil; g = g.next {
			if g.windowSwitching != 0 {
				for sb := 0; sb < 3; sb++ {
					c, err := dec.decodeS(model)
					if err != nil {
						return err
					}
					g.sbGain[sb] = int8(c)
					if err := model.shift(c); err != nil {
						return err
					}
				}
			} else {
				g.sbGain[0], g.sbGain[1], g.sbGain[2] = 0, 0, 0
			}
		}
	}
	return nil
}

func (s *session) decodeStereoMS(dec *arith) error {
	model := newModelB(16, 1)
	ctx := byte(0)
	for fr := s.first; fr != nil; fr = fr.next {
		if err := model.shift(int32(ctx)); err != nil {
			return err
		}
		c, err := dec.decodeB(model)
		if err != nil {
			return err
		}
		fr.stereoMS = int8(c)
		ctx = ((ctx << 1) | byte(c)) & 0xF
	}
	return nil
}
