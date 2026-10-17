package bz2

import "unsafe"

const BZ_LESSER_ICOST = 0
const BZ_GREATER_ICOST = 15

func BZ2_bsInitWrite(s *EState) {
	s.BsLive = 0
	s.BsBuff = 0
}
func bsFinishWrite(s *EState) {
	for s.BsLive > 0 {
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Zbits), s.NumZ)) = UChar(s.BsBuff >> 24)
		s.NumZ++
		s.BsBuff <<= 8
		s.BsLive -= 8
	}
}
func bsW(s *EState, n Int32, v UInt32) {
	{
		for s.BsLive >= 8 {
			*(*UChar)(unsafe.Add(unsafe.Pointer(s.Zbits), s.NumZ)) = UChar(s.BsBuff >> 24)
			s.NumZ++
			s.BsBuff <<= 8
			s.BsLive -= 8
		}
	}
	s.BsBuff |= v << UInt32(32-s.BsLive-n)
	s.BsLive += n
}
func bsPutUInt32(s *EState, u UInt32) {
	bsW(s, 8, (u>>24)&0xFF)
	bsW(s, 8, (u>>16)&0xFF)
	bsW(s, 8, (u>>8)&0xFF)
	bsW(s, 8, u&0xFF)
}
func bsPutUChar(s *EState, c UChar) {
	bsW(s, 8, UInt32(c))
}
func makeMaps_e(s *EState) {
	var i Int32
	s.NInUse = 0
	for i = 0; i < 256; i++ {
		if s.InUse[i] != 0 {
			s.UnseqToSeq[i] = UChar(int8(s.NInUse))
			s.NInUse++
		}
	}
}
func generateMTFValues(s *EState) {
	var (
		yy    [256]UChar
		i     Int32
		j     Int32
		zPend Int32
		wr    Int32
		EOB   Int32
		ptr   *UInt32 = s.Ptr
		block *UChar  = s.Block
		mtfv  *UInt16 = s.Mtfv
	)
	makeMaps_e(s)
	EOB = s.NInUse + 1
	for i = 0; i <= EOB; i++ {
		s.MtfFreq[i] = 0
	}
	wr = 0
	zPend = 0
	for i = 0; i < s.NInUse; i++ {
		yy[i] = UChar(int8(i))
	}
	for i = 0; i < s.Nblock; i++ {
		var ll_i UChar
		for {
			if true {
				break
			}
		}
		j = Int32(*(*UInt32)(unsafe.Add(unsafe.Pointer(ptr), unsafe.Sizeof(UInt32(0))*uintptr(i))) - 1)
		if j < 0 {
			j += s.Nblock
		}
		ll_i = s.UnseqToSeq[*(*UChar)(unsafe.Add(unsafe.Pointer(block), j))]
		for {
			if true {
				break
			}
		}
		if yy[0] == ll_i {
			zPend++
		} else {
			if zPend > 0 {
				zPend--
				for {
					if zPend&1 != 0 {
						*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(wr))) = BZ_RUNB
						wr++
						s.MtfFreq[BZ_RUNB]++
					} else {
						*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(wr))) = BZ_RUNA
						wr++
						s.MtfFreq[BZ_RUNA]++
					}
					if zPend < 2 {
						break
					}
					zPend = (zPend - 2) / 2
				}
				zPend = 0
			}
			{
				var (
					rtmp  UChar
					ryy_j *UChar
					rll_i UChar
				)
				rtmp = yy[1]
				yy[1] = yy[0]
				ryy_j = &(yy[1])
				rll_i = ll_i
				for rll_i != rtmp {
					var rtmp2 UChar
					ryy_j = (*UChar)(unsafe.Add(unsafe.Pointer(ryy_j), 1))
					rtmp2 = rtmp
					rtmp = *ryy_j
					*ryy_j = rtmp2
				}
				yy[0] = rtmp
				j = Int32(int64(uintptr(unsafe.Pointer(ryy_j)) - uintptr(unsafe.Pointer(&(yy[0])))))
				*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(wr))) = UInt16(int16(j + 1))
				wr++
				s.MtfFreq[j+1]++
			}
		}
	}
	if zPend > 0 {
		zPend--
		for {
			if zPend&1 != 0 {
				*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(wr))) = BZ_RUNB
				wr++
				s.MtfFreq[BZ_RUNB]++
			} else {
				*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(wr))) = BZ_RUNA
				wr++
				s.MtfFreq[BZ_RUNA]++
			}
			if zPend < 2 {
				break
			}
			zPend = (zPend - 2) / 2
		}
		zPend = 0
	}
	*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(wr))) = UInt16(int16(EOB))
	wr++
	s.MtfFreq[EOB]++
	s.NMTF = wr
}
func sendMTFValues(s *EState) {
	var (
		v    Int32
		t    Int32
		i    Int32
		j    Int32
		gs   Int32
		ge   Int32
		totc Int32
	)
	_ = totc
	var bt Int32
	var bc Int32
	var iter Int32
	var nSelectors Int32
	var alphaSize Int32
	var minLen Int32
	var maxLen Int32
	var selCtr Int32
	var nGroups Int32
	var nBytes Int32
	_ = nBytes
	var cost [6]UInt16
	var fave [6]Int32
	var mtfv *UInt16 = s.Mtfv
	if s.Verbosity >= 3 {
		for {
			if true {
				break
			}
		}
	}
	alphaSize = s.NInUse + 2
	for t = 0; t < BZ_N_GROUPS; t++ {
		for v = 0; v < alphaSize; v++ {
			s.Len[t][v] = BZ_GREATER_ICOST
		}
	}
	{
		if s.NMTF <= 0 {
			bz_internal_error(3001)
		}
	}
	if s.NMTF < 200 {
		nGroups = 2
	} else if s.NMTF < 600 {
		nGroups = 3
	} else if s.NMTF < 1200 {
		nGroups = 4
	} else if s.NMTF < 2400 {
		nGroups = 5
	} else {
		nGroups = 6
	}
	{
		var (
			nPart Int32
			remF  Int32
			tFreq Int32
			aFreq Int32
		)
		nPart = nGroups
		remF = s.NMTF
		gs = 0
		for nPart > 0 {
			tFreq = remF / nPart
			ge = gs - 1
			aFreq = 0
			for aFreq < tFreq && ge < alphaSize-1 {
				ge++
				aFreq += s.MtfFreq[ge]
			}
			if ge > gs && nPart != nGroups && nPart != 1 && (nGroups-nPart)%2 == 1 {
				aFreq -= s.MtfFreq[ge]
				ge--
			}
			if s.Verbosity >= 3 {
				for {
					if true {
						break
					}
				}
			}
			for v = 0; v < alphaSize; v++ {
				if v >= gs && v <= ge {
					s.Len[nPart-1][v] = BZ_LESSER_ICOST
				} else {
					s.Len[nPart-1][v] = BZ_GREATER_ICOST
				}
			}
			nPart--
			gs = ge + 1
			remF -= aFreq
		}
	}
	for iter = 0; iter < BZ_N_ITERS; iter++ {
		for t = 0; t < nGroups; t++ {
			fave[t] = 0
		}
		for t = 0; t < nGroups; t++ {
			for v = 0; v < alphaSize; v++ {
				s.Rfreq[t][v] = 0
			}
		}
		if nGroups == 6 {
			for v = 0; v < alphaSize; v++ {
				s.Len_pack[v][0] = (UInt32(s.Len[1][v]) << 16) | UInt32(s.Len[0][v])
				s.Len_pack[v][1] = (UInt32(s.Len[3][v]) << 16) | UInt32(s.Len[2][v])
				s.Len_pack[v][2] = (UInt32(s.Len[5][v]) << 16) | UInt32(s.Len[4][v])
			}
		}
		nSelectors = 0
		totc = 0
		gs = 0
		for {
			if gs >= s.NMTF {
				break
			}
			ge = gs + BZ_G_SIZE - 1
			if ge >= s.NMTF {
				ge = s.NMTF - 1
			}
			for t = 0; t < nGroups; t++ {
				cost[t] = 0
			}
			if nGroups == 6 && ge-gs+1 == 50 {
				var (
					cost01 UInt32
					cost23 UInt32
					cost45 UInt32
					icv    UInt16
				)
				cost01 = func() UInt32 {
					cost23 = func() UInt32 {
						cost45 = 0
						return cost45
					}()
					return cost23
				}()
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+0)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+1)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+2)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+3)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+4)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+5)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+6)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+7)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+8)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+9)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+10)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+11)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+12)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+13)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+14)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+15)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+16)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+17)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+18)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+19)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+20)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+21)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+22)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+23)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+24)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+25)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+26)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+27)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+28)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+29)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+30)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+31)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+32)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+33)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+34)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+35)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+36)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+37)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+38)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+39)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+40)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+41)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+42)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+43)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+44)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+45)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+46)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+47)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+48)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				icv = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+49)))
				cost01 += s.Len_pack[icv][0]
				cost23 += s.Len_pack[icv][1]
				cost45 += s.Len_pack[icv][2]
				cost[0] = UInt16(cost01 & 0xFFFF)
				cost[1] = UInt16(cost01 >> 16)
				cost[2] = UInt16(cost23 & 0xFFFF)
				cost[3] = UInt16(cost23 >> 16)
				cost[4] = UInt16(cost45 & 0xFFFF)
				cost[5] = UInt16(cost45 >> 16)
			} else {
				for i = gs; i <= ge; i++ {
					var icv UInt16 = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(i)))
					for t = 0; t < nGroups; t++ {
						cost[t] += UInt16(s.Len[t][icv])
					}
				}
			}
			bc = 999999999
			bt = -1
			for t = 0; t < nGroups; t++ {
				if Int32(cost[t]) < bc {
					bc = Int32(cost[t])
					bt = t
				}
			}
			totc += bc
			fave[bt]++
			s.Selector[nSelectors] = UChar(int8(bt))
			nSelectors++
			if nGroups == 6 && ge-gs+1 == 50 {
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+0)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+1)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+2)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+3)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+4)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+5)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+6)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+7)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+8)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+9)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+10)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+11)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+12)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+13)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+14)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+15)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+16)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+17)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+18)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+19)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+20)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+21)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+22)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+23)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+24)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+25)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+26)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+27)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+28)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+29)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+30)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+31)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+32)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+33)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+34)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+35)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+36)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+37)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+38)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+39)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+40)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+41)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+42)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+43)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+44)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+45)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+46)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+47)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+48)))]++
				s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+49)))]++
			} else {
				for i = gs; i <= ge; i++ {
					s.Rfreq[bt][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(i)))]++
				}
			}
			gs = ge + 1
		}
		if s.Verbosity >= 3 {
			for {
				if true {
					break
				}
			}
			for t = 0; t < nGroups; t++ {
				for {
					if true {
						break
					}
				}
			}
			for {
				if true {
					break
				}
			}
		}
		for t = 0; t < nGroups; t++ {
			BZ2_hbMakeCodeLengths(&(s.Len[t][0]), &(s.Rfreq[t][0]), alphaSize, 17)
		}
	}
	{
		if nGroups >= 8 {
			bz_internal_error(3002)
		}
	}
	{
		if nSelectors >= 32768 || nSelectors > Int32(int32(900000/BZ_G_SIZE)+2) {
			bz_internal_error(3003)
		}
	}
	{
		var (
			pos  [6]UChar
			ll_i UChar
			tmp2 UChar
			tmp  UChar
		)
		for i = 0; i < nGroups; i++ {
			pos[i] = UChar(int8(i))
		}
		for i = 0; i < nSelectors; i++ {
			ll_i = s.Selector[i]
			j = 0
			tmp = pos[j]
			for ll_i != tmp {
				j++
				tmp2 = tmp
				tmp = pos[j]
				pos[j] = tmp2
			}
			pos[0] = tmp
			s.SelectorMtf[i] = UChar(int8(j))
		}
	}
	for t = 0; t < nGroups; t++ {
		minLen = 32
		maxLen = 0
		for i = 0; i < alphaSize; i++ {
			if Int32(s.Len[t][i]) > maxLen {
				maxLen = Int32(s.Len[t][i])
			}
			if Int32(s.Len[t][i]) < minLen {
				minLen = Int32(s.Len[t][i])
			}
		}
		{
			if maxLen > 17 {
				bz_internal_error(3004)
			}
		}
		{
			if minLen < 1 {
				bz_internal_error(3005)
			}
		}
		BZ2_hbAssignCodes(&(s.Code[t][0]), &(s.Len[t][0]), minLen, maxLen, alphaSize)
	}
	{
		var inUse16 [16]Bool
		for i = 0; i < 16; i++ {
			inUse16[i] = 0
			for j = 0; j < 16; j++ {
				if s.InUse[i*16+j] != 0 {
					inUse16[i] = 1
				}
			}
		}
		nBytes = s.NumZ
		for i = 0; i < 16; i++ {
			if inUse16[i] != 0 {
				bsW(s, 1, 1)
			} else {
				bsW(s, 1, 0)
			}
		}
		for i = 0; i < 16; i++ {
			if inUse16[i] != 0 {
				for j = 0; j < 16; j++ {
					if s.InUse[i*16+j] != 0 {
						bsW(s, 1, 1)
					} else {
						bsW(s, 1, 0)
					}
				}
			}
		}
		if s.Verbosity >= 3 {
			for {
				if true {
					break
				}
			}
		}
	}
	nBytes = s.NumZ
	bsW(s, 3, UInt32(nGroups))
	bsW(s, 15, UInt32(nSelectors))
	for i = 0; i < nSelectors; i++ {
		for j = 0; j < Int32(s.SelectorMtf[i]); j++ {
			bsW(s, 1, 1)
		}
		bsW(s, 1, 0)
	}
	if s.Verbosity >= 3 {
		for {
			if true {
				break
			}
		}
	}
	nBytes = s.NumZ
	for t = 0; t < nGroups; t++ {
		var curr Int32 = Int32(s.Len[t][0])
		bsW(s, 5, UInt32(curr))
		for i = 0; i < alphaSize; i++ {
			for curr < Int32(s.Len[t][i]) {
				bsW(s, 2, 2)
				curr++
			}
			for curr > Int32(s.Len[t][i]) {
				bsW(s, 2, 3)
				curr--
			}
			bsW(s, 1, 0)
		}
	}
	if s.Verbosity >= 3 {
		for {
			if true {
				break
			}
		}
	}
	nBytes = s.NumZ
	selCtr = 0
	gs = 0
	for {
		if gs >= s.NMTF {
			break
		}
		ge = gs + BZ_G_SIZE - 1
		if ge >= s.NMTF {
			ge = s.NMTF - 1
		}
		{
			if Int32(s.Selector[selCtr]) >= nGroups {
				bz_internal_error(3006)
			}
		}
		if nGroups == 6 && ge-gs+1 == 50 {
			var (
				mtfv_i            UInt16
				s_len_sel_selCtr  *UChar = &(s.Len[s.Selector[selCtr]][0])
				s_code_sel_selCtr *Int32 = &(s.Code[s.Selector[selCtr]][0])
			)
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+0)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+1)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+2)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+3)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+4)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+5)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+6)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+7)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+8)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+9)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+10)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+11)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+12)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+13)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+14)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+15)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+16)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+17)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+18)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+19)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+20)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+21)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+22)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+23)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+24)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+25)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+26)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+27)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+28)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+29)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+30)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+31)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+32)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+33)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+34)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+35)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+36)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+37)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+38)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+39)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+40)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+41)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+42)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+43)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+44)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+45)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+46)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+47)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+48)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
			mtfv_i = *(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(gs+49)))
			bsW(s, Int32(*(*UChar)(unsafe.Add(unsafe.Pointer(s_len_sel_selCtr), mtfv_i))), UInt32(*(*Int32)(unsafe.Add(unsafe.Pointer(s_code_sel_selCtr), unsafe.Sizeof(Int32(0))*uintptr(mtfv_i)))))
		} else {
			for i = gs; i <= ge; i++ {
				bsW(s, Int32(s.Len[s.Selector[selCtr]][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(i)))]), UInt32(s.Code[s.Selector[selCtr]][*(*UInt16)(unsafe.Add(unsafe.Pointer(mtfv), unsafe.Sizeof(UInt16(0))*uintptr(i)))]))
			}
		}
		gs = ge + 1
		selCtr++
	}
	{
		if selCtr != nSelectors {
			bz_internal_error(3007)
		}
	}
	if s.Verbosity >= 3 {
		for {
			if true {
				break
			}
		}
	}
}
func BZ2_compressBlock(s *EState, is_last_block Bool) {
	if s.Nblock > 0 {
		s.BlockCRC = ^s.BlockCRC
		s.CombinedCRC = (s.CombinedCRC << 1) | s.CombinedCRC>>31
		s.CombinedCRC ^= s.BlockCRC
		if s.BlockNo > 1 {
			s.NumZ = 0
		}
		if s.Verbosity >= 2 {
			for {
				if true {
					break
				}
			}
		}
		BZ2_blockSort(s)
	}
	s.Zbits = (*UChar)(unsafe.Add(unsafe.Pointer((*UChar)(unsafe.Pointer(s.Arr2))), s.Nblock))
	if s.BlockNo == 1 {
		BZ2_bsInitWrite(s)
		bsPutUChar(s, BZ_HDR_B)
		bsPutUChar(s, BZ_HDR_Z)
		bsPutUChar(s, BZ_HDR_h)
		bsPutUChar(s, UChar(int8(BZ_HDR_0+s.BlockSize100k)))
	}
	if s.Nblock > 0 {
		bsPutUChar(s, 0x31)
		bsPutUChar(s, 0x41)
		bsPutUChar(s, 0x59)
		bsPutUChar(s, 0x26)
		bsPutUChar(s, 0x53)
		bsPutUChar(s, 0x59)
		bsPutUInt32(s, s.BlockCRC)
		bsW(s, 1, 0)
		bsW(s, 24, UInt32(s.OrigPtr))
		generateMTFValues(s)
		sendMTFValues(s)
	}
	if is_last_block != 0 {
		bsPutUChar(s, 0x17)
		bsPutUChar(s, 0x72)
		bsPutUChar(s, 0x45)
		bsPutUChar(s, 0x38)
		bsPutUChar(s, 0x50)
		bsPutUChar(s, 0x90)
		bsPutUInt32(s, s.CombinedCRC)
		if s.Verbosity >= 2 {
			for {
				if true {
					break
				}
			}
		}
		bsFinishWrite(s)
	}
}
