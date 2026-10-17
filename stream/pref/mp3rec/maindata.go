package mp3rec

// Main-data restore from packmp3.cpp pmp_decode_main_data and pmp_predict_lame_anc.
// Coefficients are re-encoded with the packMP3 Huffman encode tables.

type lamePred struct {
	pred    [2048]byte
	lameStr [20]byte
	lameLen int
	lameBit int
	alt     bool
}

func (p *lamePred) reset() {
	p.lameBit = 0
	p.lameLen = 4
	copy(p.lameStr[:4], []byte("LAME"))
	p.alt = false
}

func predShr(b byte, offset int) byte {
	n := 8 - offset
	if n >= 8 || n < 0 {
		return 0
	}
	return b >> uint(n)
}

// predict builds the LAME ancillary guess. matched is set when ref is present and equals the guess.
func (p *lamePred) predict(nbits int, ref []byte) (buf []byte, matched bool) {
	if nbits < 0 {
		p.reset()
		return nil, false
	}
	offset := nbits % 8
	nbytes := nbits / 8
	if !p.alt {
		i := 0
		for nbits >= 8 && i < 4 {
			p.pred[i] = p.lameStr[i]
			nbits -= 8
			i++
		}
		if nbits >= 32 {
			for nbits >= 8 && i < p.lameLen {
				p.pred[i] = p.lameStr[i]
				nbits -= 8
				i++
			}
		}
		for nbits > 0 {
			if p.lameBit == 0 {
				p.pred[i] = 0x55
			} else {
				p.pred[i] = 0xAA
			}
			nbits -= 8
			i++
		}
		p.lameBit ^= offset % 2
	}
	if ref != nil {
		same := true
		if nbytes > len(ref) || nbytes >= len(p.pred) {
			same = false
		} else {
			for i := 0; i < nbytes; i++ {
				if p.pred[i] != ref[i] {
					same = false
					break
				}
			}
			if same && (nbytes >= len(ref) || nbytes >= len(p.pred) || ref[nbytes] != predShr(p.pred[nbytes], offset)) {
				same = false
			}
		}
		if same {
			return nil, true
		}
		if nbytes >= 8 && ref[0] == p.lameStr[0] && ref[1] == p.lameStr[1] && ref[2] == p.lameStr[2] && ref[3] == p.lameStr[3] {
			i := 4
			for i < 20 && i < nbytes {
				if ref[i] == 0x55 {
					p.lameBit = 0
					break
				} else if ref[i] == 0xAA {
					p.lameBit = 1
					break
				}
				p.lameStr[i] = ref[i]
				i++
			}
			p.lameLen = i
		} else if nbytes == 0 && offset > 1 {
			if (ref[0] >> uint(8-offset)) == 0 {
				for i := range p.pred {
					p.pred[i] = 0
				}
				p.alt = true
			} else if ((0xFF ^ ref[0]) >> uint(8-offset)) == 0 {
				for i := range p.pred {
					p.pred[i] = 0xFF
				}
				p.alt = true
			} else {
				p.alt = false
			}
		} else if nbytes >= 1 {
			if ref[0] == 0 {
				for i := range p.pred {
					p.pred[i] = 0
				}
				p.alt = true
			} else if ref[0] == 0xFF {
				for i := range p.pred {
					p.pred[i] = 0xFF
				}
				p.alt = true
			} else {
				p.alt = false
			}
		}
	}
	return p.pred[:], false
}

func writePred(h *huffW, pred []byte, n int) {
	i := 0
	for n >= 8 {
		h.writeBits(uint32(pred[i]), 8)
		i++
		n -= 8
	}
	if n > 0 {
		h.writeBits(uint32(pred[i]>>uint(8-n)), int32(n))
	}
}

func (s *session) decodeMain(dec *arith) ([]byte, error) {
	jCoding := s.iChannels == mp3Stereo || s.iChannels == mp3Joint
	h := newHuffW()
	s.pred.predict(-1, nil)

	modSvb := newModelS(576/4+2, 576/4+2, 1)
	modBvf := newModelS(576/2+1, 0, 0)
	modNst := newModelS(stuffingStep+1, stuffingStep+1, 1)
	modBst := newModelB(16, 1)
	modPad := newModelB(256, 1)
	modPap := newModelB(0, 0)
	modAux := newModelS(auxDataStep+1, auxDataStep+1, 1)
	modBtr := newModelS(16, 16, 1)
	modRes := newModelB(14, 2)

	nch := int(s.gNchannels)
	var modScf [2][4][10]*modelS
	var modAbv [2][8][32]*modelS
	var modAsv [2][8][2]*modelB
	var modSgn [2][8]*modelB
	var modLen [2][14]*modelS
	for ch := 0; ch < nch; ch++ {
		for g := 0; g < 10; g++ {
			modScf[ch][0][g] = newModelS(2, 16, 2)
			modScf[ch][1][g] = newModelS(4, 16, 2)
			modScf[ch][2][g] = newModelS(8, 16, 2)
			modScf[ch][3][g] = newModelS(16, 16, 2)
		}
		nflags := 2
		if jCoding && ch != 0 {
			nflags = 8
		}
		for flags := 0; flags < nflags; flags++ {
			modAbv[ch][flags][1] = newModelS(2, 16, 2)
			modAbv[ch][flags][2] = newModelS(3, 16, 2)
			modAbv[ch][flags][3] = newModelS(3, 16, 2)
			modAbv[ch][flags][5] = newModelS(4, 16, 2)
			modAbv[ch][flags][6] = newModelS(4, 16, 2)
			modAbv[ch][flags][7] = newModelS(6, 16, 2)
			modAbv[ch][flags][8] = newModelS(6, 16, 2)
			modAbv[ch][flags][9] = newModelS(6, 16, 2)
			modAbv[ch][flags][10] = newModelS(8, 16, 2)
			modAbv[ch][flags][11] = newModelS(8, 16, 2)
			modAbv[ch][flags][12] = newModelS(8, 16, 2)
			modAbv[ch][flags][13] = newModelS(16, 16, 2)
			modAbv[ch][flags][15] = newModelS(16, 16, 2)
			m16 := newModelS(16, 16, 2)
			modAbv[ch][flags][16] = m16
			for t := 17; t <= 23; t++ {
				modAbv[ch][flags][t] = m16
			}
			m24 := newModelS(16, 16, 2)
			modAbv[ch][flags][24] = m24
			for t := 25; t <= 31; t++ {
				modAbv[ch][flags][t] = m24
			}
			modAsv[ch][flags][0] = newModelB(16, 2)
			modAsv[ch][flags][1] = newModelB(16, 2)
			modSgn[ch][flags] = newModelB(16, 2)
		}
		for i := 1; i <= 13; i++ {
			modLen[ch][i] = newModelS(int32(i+1), 14, 1)
		}
	}

	scfC := [2][]byte{}
	scfLong := [2][]byte{}
	scfShort := [2][]byte{}
	absS := [2][]byte{}
	sgnS := [2][]byte{}
	lenS := [2][]byte{}
	lbtS := [2][]uint16{}
	absl := [2][]byte{}
	abss := [2][]byte{}
	sgnl := [2][]byte{}
	sgns := [2][]byte{}
	lenl := [2][]byte{}
	lens := [2][]byte{}
	for ch := 0; ch < nch; ch++ {
		scfC[ch] = make([]byte, 21)
		scfLong[ch] = make([]byte, 21)
		scfShort[ch] = make([]byte, 21)
		absS[ch] = make([]byte, 580)
		sgnS[ch] = make([]byte, 580)
		lenS[ch] = make([]byte, 580)
		lbtS[ch] = make([]uint16, 576)
		absl[ch] = make([]byte, 580)
		abss[ch] = make([]byte, 580)
		sgnl[ch] = make([]byte, 580)
		sgns[ch] = make([]byte, 580)
		lenl[ch] = make([]byte, 580)
		lens[ch] = make([]byte, 580)
	}

	var ctxSvb [2]byte
	var ctxNst, ctxBst, ctxAux byte
	var shared [2][4]int8
	flags := 0
	bitres := 0
	var pad [2048]byte

	sr := int(s.iSamplerate)
	if sr < 0 || sr >= len(mp3BitratePred) || sr >= len(mp3FrameSize) {
		return nil, errf("samplerate %d", sr)
	}

	for fr := s.first; fr != nil; fr = fr.next {
		bitp := 0
		for gr := 0; gr < 2; gr++ {
			for ch := 0; ch < nch; ch++ {
				g := fr.granules[ch][gr]
				sbl := 0
				if g.blockType == shortBlock {
					sbl = 1
				}
				if g.slength < 0 || int(g.slength) >= len(slengthTable) {
					return nil, errf("slength %d", g.slength)
				}
				slen := slengthTable[g.slength]
				abs := absS[ch]
				sgn := sgnS[ch]
				lenb := lenS[ch]
				lbt := lbtS[ch]
				if !jCoding || ch == 0 {
					flags = sbl
				} else {
					flags = sbl | (int(fr.stereoMS) << 1) | ((flags ^ sbl) << 2)
				}
				if flags < 0 || flags > 7 {
					return nil, errf("channel flags %d", flags)
				}
				if gr == 0 {
					sh := int32(g.share)
					shared[ch][0] = int8((sh >> 3) & 1)
					shared[ch][1] = int8((sh >> 2) & 1)
					shared[ch][2] = int8((sh >> 1) & 1)
					shared[ch][3] = int8(sh & 1)
				}
				h.reset()
				fr.bitReservoir = int16(bitres)

				if sbl == 0 {
					scf := scfC[ch]
					prev := scfLong[ch]
					ctxScf := byte(0)
					p := 0
					for grp := 0; grp < 4; grp++ {
						sl := int(slen[0])
						if grp >= 2 {
							sl = int(slen[1])
						}
						if gr != 0 && shared[ch][grp] != 0 {
							copy(scf[p:p+scfWidth[grp]], prev[p:p+scfWidth[grp]])
							p = scfBounds[grp]
							ctxScf = scf[p-1]
						} else if sl == 0 {
							for i := 0; i < scfWidth[grp]; i++ {
								scf[p+i] = 0
							}
							p = scfBounds[grp]
							ctxScf = 0
						} else {
							mi := grp
							if shared[ch][grp] != 0 {
								mi = grp | 4
							}
							mod := modScf[ch][sl-1][mi]
							for p < scfBounds[grp] {
								if err := shiftS(mod, int32(ctxScf), int32(prev[p])); err != nil {
									return nil, err
								}
								c, err := dec.decodeS(mod)
								if err != nil {
									return nil, err
								}
								scf[p] = byte(c)
								h.writeBits(uint32(scf[p]), int32(sl))
								ctxScf = scf[p]
								p++
							}
						}
					}
					scfC[ch], scfLong[ch] = scfLong[ch], scfC[ch]
				} else {
					for sub := 0; sub < 3; sub++ {
						scf := scfC[ch]
						prev := scfShort[ch]
						ctxScf := byte(0)
						p := 0
						for grp := 0; grp < 2; grp++ {
							sl := int(slen[grp])
							if sl == 0 {
								for i := 0; i < scfLHWidthShort[grp]; i++ {
									scf[p+i] = 0
								}
								p = scfLHBoundsShort[grp]
							} else {
								mod := modScf[ch][sl-1][grp|8]
								for p < scfLHBoundsShort[grp] {
									if err := shiftS(mod, int32(ctxScf), int32(prev[p])); err != nil {
										return nil, err
									}
									c, err := dec.decodeS(mod)
									if err != nil {
										return nil, err
									}
									scf[p] = byte(c)
									h.writeBits(uint32(scf[p]), int32(sl))
									ctxScf = scf[p]
									p++
								}
							}
						}
						scfC[ch], scfShort[ch] = scfShort[ch], scfC[ch]
					}
				}

				if sbl == 0 {
					if err := modSvb.shift(int32(ctxSvb[ch])); err != nil {
						return nil, err
					}
				} else if err := modSvb.shift(145); err != nil {
					return nil, err
				}
				cBound, err := dec.decodeS(modSvb)
				if err != nil {
					return nil, err
				}
				if cBound <= 144 {
					g.svBound = int16(cBound << 2)
					if int(g.regionBound[2])%4 == 2 {
						g.svBound += 2
					}
					if sbl == 0 {
						ctxSvb[ch] = byte(cBound)
					}
				} else {
					cBound, err = dec.decodeS(modBvf)
					if err != nil {
						return nil, err
					}
					g.svBound = int16(int32(g.regionBound[2]) - (cBound << 1))
					g.regionBound[2] = g.svBound
					for i := 1; i >= 0; i-- {
						if g.svBound < g.regionBound[i] {
							g.regionBound[i] = g.svBound
						} else {
							break
						}
					}
				}

				var ctxAbsB, ctxSgnB, ctxLenB []byte
				if g.blockType != shortBlock {
					ctxAbsB = absl[ch]
					ctxSgnB = sgnl[ch]
					ctxLenB = lenl[ch]
				} else {
					ctxAbsB = abss[ch]
					ctxSgnB = sgns[ch]
					ctxLenB = lens[ch]
				}
				ctxAbs := byte(0)
				ctxPat := byte(0)
				rlb := int(g.regionBound[2])
				modSgc := modSgn[ch][flags]
				ht := int(g.selectHtabB)
				if ht != 0 && ht != 1 {
					return nil, errf("small value table %d", ht)
				}
				modAsc := modAsv[ch][flags][ht]
				p := int(g.svBound) - 1
				for p >= rlb {
					if p < 0 || p+1 >= len(abs) {
						return nil, errf("small value index %d", p)
					}
					if err := shiftB(modAsc, int32(ctxPat), int32(ctxAbsB[p+1])); err != nil {
						return nil, err
					}
					av, err := dec.decodeB(modAsc)
					if err != nil {
						return nil, err
					}
					abs[p+1] = byte(av)
					ctxPat = ((ctxPat << 1) | abs[p+1]) & 0xF
					ctxAbs = byte((2*int(abs[p+1]) + int(ctxAbs) + 2) / 3)
					if abs[p+1] == 1 {
						if err := shiftB(modSgc, int32(ctxAbsB[p+1]), int32(ctxSgnB[p+1])); err != nil {
							return nil, err
						}
						sg, err := dec.decodeB(modSgc)
						if err != nil {
							return nil, err
						}
						sgn[p+1] = byte(sg)
					}
					p--
				}

				for r := 2; r >= 0; r-- {
					if r == 0 {
						rlb = 0
					} else {
						rlb = int(g.regionBound[r-1])
					}
					if p < rlb {
						continue
					}
					if g.regionTable[r] == 0 {
						for i := rlb; i < p; i++ {
							abs[i+1] = 0
						}
						p = rlb
						ctxAbs = 0
						continue
					}
					tab := int(g.regionTable[r])
					if tab < 0 || tab >= len(bvEnc) || bvEnc[tab].rows == nil {
						return nil, errf("huffman table %d", tab)
					}
					linbits := int(bvEnc[tab].linbits)
					modAbc := modAbv[ch][flags][tab]
					if modAbc == nil {
						return nil, errf("huffman model %d", tab)
					}
					var modLnc *modelS
					if linbits > 0 {
						if linbits >= len(modLen[ch]) || modLen[ch][linbits] == nil {
							return nil, errf("linbits %d", linbits)
						}
						modLnc = modLen[ch][linbits]
					}
					for p >= rlb {
						if p < 0 || p >= len(lbt) || p+1 >= len(abs) {
							return nil, errf("big value index %d", p)
						}
						if err := shiftS(modAbc, int32(ctxAbs), int32(ctxAbsB[p+1])); err != nil {
							return nil, err
						}
						av, err := dec.decodeS(modAbc)
						if err != nil {
							return nil, err
						}
						abs[p+1] = byte(av)
						ctxAbs = byte((2*int(abs[p+1]) + int(ctxAbs) + 2) / 3)
						if abs[p+1] > 0 {
							if err := shiftB(modSgc, int32(ctxAbsB[p+1]), int32(ctxSgnB[p+1])); err != nil {
								return nil, err
							}
							sg, err := dec.decodeB(modSgc)
							if err != nil {
								return nil, err
							}
							sgn[p+1] = byte(sg)
							if linbits > 0 && abs[p+1] == 15 {
								if err := modLnc.shift(int32(ctxLenB[p+1])); err != nil {
									return nil, err
								}
								ln, err := dec.decodeS(modLnc)
								if err != nil {
									return nil, err
								}
								lenb[p+1] = byte(ln)
								if ln > 0 {
									lbt[p] = 1
									for i := int(ln) - 2; i >= 0; i-- {
										if err := shiftB(modRes, ln, int32(i)); err != nil {
											return nil, err
										}
										bit, err := dec.decodeB(modRes)
										if err != nil {
											return nil, err
										}
										lbt[p] = (lbt[p] << 1) | uint16(bit)
									}
								} else {
									lbt[p] = 0
								}
							}
						}
						p--
					}
				}

				p = 0
				for r := 0; r < 3; r++ {
					if g.regionTable[r] == 0 {
						p = int(g.regionBound[r])
						continue
					}
					tab := int(g.regionTable[r])
					rows := bvEnc[tab].rows
					linbits := int(bvEnc[tab].linbits)
					for p < int(g.regionBound[r]) {
						if p < 0 || p+2 >= len(abs) {
							return nil, errf("encode index %d", p)
						}
						if err := h.encodePair(rows, abs[p+1:p+3]); err != nil {
							return nil, err
						}
						for i := 0; i < 2; i++ {
							if abs[p+1] > 0 {
								if linbits > 0 && abs[p+1] == 15 {
									if p < 0 || p >= len(lbt) {
										return nil, errf("linbit index %d", p)
									}
									h.writeBits(uint32(lbt[p]), int32(linbits))
								}
								h.writeBit(sgn[p+1])
							}
							p++
						}
					}
				}
				quad := &htabAEnc
				if g.selectHtabB != 0 {
					quad = &htabBEnc
				}
				for p < int(g.svBound) {
					if p < 0 || p+4 >= len(abs) {
						return nil, errf("quad index %d", p)
					}
					h.encodeQuad(quad, abs[p+1:p+5])
					for i := 0; i < 4; i++ {
						if abs[p+1] != 0 {
							h.writeBit(sgn[p+1])
						}
						p++
					}
				}
				if p < 578 {
					for i := p; i < 578; i++ {
						abs[i+1] = 0
					}
				}
				if !jCoding || ch == 0 {
					for i := 577; i >= 0; i-- {
						sum := 3*int(abs[i+1]) + int(abs[i]) + int(abs[i+2]) + 2*int(ctxAbsB[i+1]) + 4
						ctxAbsB[i+1] = byte(sum / 7)
						if abs[i+1] > 0 {
							ctxSgnB[i+1] = ((ctxSgnB[i+1] << 1) | (sgn[i+1] & 1)) & 0xF
							if abs[i+1] == 15 {
								ctxLenB[i+1] = byte((2*int(lenb[i+1]) + int(ctxLenB[i+1]) + 2) / 3)
							} else {
								ctxLenB[i+1] >>= 1
							}
						} else {
							ctxSgnB[i+1] = (ctxSgnB[i+1] << 1) & 0xF
							ctxLenB[i+1] >>= 1
						}
					}
					if jCoding {
						if sbl == 0 {
							ctxAbsB = absl[1]
							ctxSgnB = sgnl[1]
							ctxLenB = lenl[1]
						} else {
							ctxAbsB = abss[1]
							ctxSgnB = sgns[1]
							ctxLenB = lens[1]
						}
						if p < 578 {
							for i := p; i < 578; i++ {
								ctxAbsB[i+1] = 0
								ctxSgnB[i+1] = 2
								ctxLenB[i+1] = 0
							}
						}
						for i := p - 1; i >= 0; i-- {
							ctxAbsB[i+1] = abs[i+1]
							if abs[i+1] > 0 {
								ctxSgnB[i+1] = sgn[i+1]
								if abs[i+1] == 15 {
									ctxLenB[i+1] = lenb[i+1]
								} else {
									ctxLenB[i+1] = 0
								}
							} else {
								ctxSgnB[i+1] = 2
								ctxLenB[i+1] = 0
							}
						}
					}
				}

				n := 0
				c := int32(stuffingStep)
				for c == stuffingStep {
					if err := modNst.shift(int32(ctxNst)); err != nil {
						return nil, err
					}
					c, err = dec.decodeS(modNst)
					if err != nil {
						return nil, err
					}
					ctxNst = byte(c)
					n += int(c)
				}
				for n > 0 {
					if err := modBst.shift(int32(ctxBst)); err != nil {
						return nil, err
					}
					bit, err := dec.decodeB(modBst)
					if err != nil {
						return nil, err
					}
					h.writeBit(byte(bit))
					ctxBst = ((ctxBst << 1) | byte(bit)) & 0xF
					n--
				}
				g.mainDataBit = int16(h.count)
				bitp += int(g.mainDataBit)
			}
		}
		fr.mainBits = int32(bitp)
		fr.mainSize = int32((bitp + 7) / 8)
		if s.iBitrate == -1 {
			ms := int(fr.mainSize)
			if ms < 0 || ms >= len(mp3BitratePred[sr]) {
				return nil, errf("bitrate context %d", ms)
			}
			if err := modBtr.shift(mp3BitratePred[sr][ms]); err != nil {
				return nil, err
			}
			bits, err := dec.decodeS(modBtr)
			if err != nil {
				return nil, err
			}
			fr.bits = int8(bits)
		}
		bi := int(fr.bits)
		if bi < 0 || bi >= 16 {
			return nil, errf("frame bits %d", bi)
		}
		fr.frameSize = mp3FrameSize[sr][bi]
		if fr.frameSize < 0 {
			return nil, errf("frame size %d", fr.frameSize)
		}
		if fr.padding != 0 {
			fr.frameSize++
		}
		if s.iBitRes != 0 {
			bitres = int(fr.frameSize) - int(fr.fixedSize) + int(fr.bitReservoir) - int(fr.mainSize)
			if fr != s.last {
				n := 0
				if bitres >= 511 {
					n = bitres - 511
				}
				c := int32(auxDataStep)
				for c == auxDataStep {
					if err := modAux.shift(int32(ctxAux)); err != nil {
						return nil, err
					}
					var err error
					c, err = dec.decodeS(modAux)
					if err != nil {
						return nil, err
					}
					ctxAux = byte(c)
					n += int(c)
				}
				fr.auxSize = int32(n)
			} else {
				fr.auxSize = int32(bitres)
			}
			bitres -= int(fr.auxSize)
		} else {
			bitres = 0
			fr.auxSize = fr.frameSize - fr.fixedSize + int32(fr.bitReservoir) - fr.mainSize
		}
		nbits := (int(fr.mainSize)+int(fr.auxSize))*8 - bitp
		if nbits < 0 || nbits > len(pad)*8 {
			return nil, errf("ancillary bits %d", nbits)
		}
		pap, err := dec.decodeB(modPap)
		if err != nil {
			return nil, err
		}
		if pap == 1 {
			buf, _ := s.pred.predict(nbits, nil)
			writePred(h, buf, nbits)
		} else {
			pna := 0
			pad[0] = 0
			ctxPad := byte(0xFF)
			for i := 1; i <= nbits; i++ {
				if err := modPad.shift(int32(ctxPad)); err != nil {
					return nil, err
				}
				bit, err := dec.decodeB(modPad)
				if err != nil {
					return nil, err
				}
				h.writeBit(byte(bit))
				ctxPad = ((ctxPad << 1) | byte(bit)) & 0xFF
				pad[pna] = (pad[pna] << 1) | byte(bit)
				if i%8 == 0 {
					pna++
					pad[pna] = 0
				}
			}
			if _, matched := s.pred.predict(nbits, pad[:]); matched {
				return nil, errf("ancilary prediction error, please report")
			}
		}
	}
	return h.w.bytes(), nil
}
