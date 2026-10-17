package xt3u

import "encoding/binary"

const (
	lz4HCHashLog   = 15
	lz4HCMaxD      = 1 << 16
	lz4HCDict      = 64 << 10
	lz4OptNum      = 1 << 12
	lz4OptimalML   = (lz4MLMask - 1) + lz4MinMatch // 18
	lz4HCLevelMax  = 12
	lz4HCLevelDef  = 9
	lz4TrailingLit = 3
)

type lz4HCParam struct {
	opt    bool
	nb     int
	target int
}

// clTable matches lz4hc.c levels 0..12. Level 0 is unused; callers clamp.
var lz4HCTable = [lz4HCLevelMax + 1]lz4HCParam{
	{false, 2, 16},
	{false, 2, 16},
	{false, 2, 16},
	{false, 4, 16},
	{false, 8, 16},
	{false, 16, 16},
	{false, 32, 16},
	{false, 64, 16},
	{false, 128, 16},
	{false, 256, 16},
	{true, 96, 64},
	{true, 512, 128},
	{true, 16384, lz4OptNum},
}

type hcCtx struct {
	hash  [1 << lz4HCHashLog]uint32
	chain [lz4HCMaxD]uint16
	next  uint32
}

func newHCCtx() *hcCtx {
	c := &hcCtx{next: lz4HCDict}
	for i := range c.chain {
		c.chain[i] = 0xFFFF
	}
	return c
}

func hcHash(seq uint32) uint32 {
	return (seq * 2654435761) >> ((lz4MinMatch * 8) - lz4HCHashLog)
}

func (c *hcCtx) insert(src []byte, ip int) {
	target := uint32(ip) + lz4HCDict
	idx := c.next
	for idx < target {
		s := int(idx - lz4HCDict)
		h := hcHash(u32le(src, s))
		delta := idx - c.hash[h]
		if delta > 65535 {
			delta = 65535
		}
		c.chain[uint16(idx)] = uint16(delta)
		c.hash[h] = idx
		idx++
	}
	c.next = target
}

func countBack(src []byte, ip, match, iMin, mMin int) int {
	back := 0
	limit := iMin - ip
	if d := mMin - match; d > limit {
		limit = d
	}
	for back > limit && src[ip+back-1] == src[match+back-1] {
		back--
	}
	return back
}

func countPattern(src []byte, ip, iEnd int, pattern uint32) int {
	if ip >= iEnd {
		return 0
	}
	start := ip
	p := uint64(pattern) | uint64(pattern)<<32
	for ip+8 <= iEnd {
		if binary.LittleEndian.Uint64(src[ip:]) != p {
			break
		}
		ip += 8
	}
	b0 := byte(pattern)
	b1 := byte(pattern >> 8)
	b2 := byte(pattern >> 16)
	b3 := byte(pattern >> 24)
	k := 0
	for ip < iEnd {
		var pb byte
		switch k & 3 {
		case 0:
			pb = b0
		case 1:
			pb = b1
		case 2:
			pb = b2
		default:
			pb = b3
		}
		if src[ip] != pb {
			break
		}
		ip++
		k++
	}
	return ip - start
}

func reverseCountPattern(src []byte, ip, iLow int, pattern uint32) int {
	start := ip
	for ip >= iLow+4 && u32le(src, ip-4) == pattern {
		ip -= 4
	}
	tail := [3]byte{byte(pattern >> 24), byte(pattern >> 16), byte(pattern >> 8)}
	for i := 0; i < 3 && ip > iLow; i++ {
		if src[ip-1] != tail[i] {
			break
		}
		ip--
	}
	return start - ip
}

// wider is LZ4HC_InsertAndGetWiderMatch for a fresh single-block context
// (no dictionary, favorDecSpeed off). Positions are source offsets.
func (c *hcCtx) wider(src []byte, ip, iLow, iHigh, longest, attempts int, pa, chainSwap bool) (int, int, int) {
	ipIndex := uint32(ip) + lz4HCDict
	var lowest uint32
	if lz4HCDict+lz4HCDict > ipIndex {
		lowest = lz4HCDict
	} else {
		lowest = ipIndex - 65535
	}
	lookBack := ip - iLow
	c.insert(src, ip)
	pattern := u32le(src, ip)
	matchIndex := c.hash[hcHash(pattern)]
	matchChainPos := uint32(0)
	repeat := 0 // 0 untested, 1 no, 2 confirmed
	srcPattern := 0
	matchPos := 0
	startPos := ip
	nb := attempts

	for matchIndex >= lowest && nb > 0 {
		nb--
		matchLength := 0
		if matchIndex < lz4HCDict || int(matchIndex-lz4HCDict) >= len(src) {
			break
		}
		matchPtr := int(matchIndex - lz4HCDict)
		p1 := iLow + longest - 1
		p2 := matchPtr - lookBack + longest - 1
		if p1 >= 0 && p2 >= 0 && p1+1 < len(src) && p2+1 < len(src) &&
			u16le(src, p1) == u16le(src, p2) &&
			matchPtr+4 <= len(src) && u32le(src, matchPtr) == pattern {
			back := 0
			if lookBack != 0 {
				back = countBack(src, ip, matchPtr, iLow, 0)
			}
			matchLength = lz4MinMatch + matchLen(src, ip+lz4MinMatch, matchPtr+lz4MinMatch, iHigh) - back
			if matchLength > longest {
				longest = matchLength
				matchPos = matchPtr + back
				startPos = ip + back
			}
		}
		if chainSwap && matchLength == longest && matchIndex+uint32(longest) <= ipIndex {
			distanceToNext := uint32(1)
			for pos := 0; pos <= longest-lz4MinMatch; pos++ {
				cand := uint32(c.chain[uint16(matchIndex+uint32(pos))])
				if cand > distanceToNext {
					distanceToNext = cand
					matchChainPos = uint32(pos)
				}
			}
			if distanceToNext > 1 {
				if distanceToNext > matchIndex {
					break
				}
				matchIndex -= distanceToNext
				continue
			}
		}
		distNext := uint32(c.chain[uint16(matchIndex)])
		if pa && distNext == 1 && matchChainPos == 0 {
			cand := matchIndex - 1
			if repeat == 0 {
				if (pattern&0xFFFF) == (pattern>>16) && (pattern&0xFF) == (pattern>>24) {
					repeat = 2
					srcPattern = countPattern(src, ip+4, iHigh, pattern) + 4
				} else {
					repeat = 1
				}
			}
			if repeat == 2 && cand >= lz4HCDict {
				mp := int(cand - lz4HCDict)
				if mp >= 0 && mp+4 <= len(src) && u32le(src, mp) == pattern {
					forward := countPattern(src, mp+4, iHigh, pattern) + 4
					lowestPos := 0
					if ip > 65535 {
						lowestPos = ip - 65535
					}
					backLen := reverseCountPattern(src, mp, lowestPos, pattern)
					curSeg := backLen + forward
					if curSeg >= srcPattern && forward <= srcPattern {
						matchIndex = cand + uint32(forward) - uint32(srcPattern)
					} else {
						matchIndex = cand - uint32(backLen)
						if lookBack == 0 {
							maxML := curSeg
							if maxML > srcPattern {
								maxML = srcPattern
							}
							if longest < maxML {
								diff := int64(ipIndex) - int64(matchIndex)
								if diff > 65535 {
									break
								}
								if diff > 0 {
									longest = maxML
									matchPos = ip - int(diff)
									startPos = ip
								}
							}
							distToNext := uint32(c.chain[uint16(matchIndex)])
							if distToNext > matchIndex {
								break
							}
							matchIndex -= distToNext
						}
					}
					continue
				}
			}
		}
		delta := uint32(c.chain[uint16(matchIndex+matchChainPos)])
		matchIndex -= delta
	}
	return longest, matchPos, startPos
}

func (c *hcCtx) findBest(src []byte, ip, iLimit, attempts int, pa bool) (int, int) {
	ml, matchPos, _ := c.wider(src, ip, ip, iLimit, lz4MinMatch-1, attempts, pa, false)
	return ml, matchPos
}

func (c *hcCtx) compressHashChain(src, dst []byte, attempts int) int {
	pa := attempts > 128
	ip := 0
	anchor := 0
	op := 0
	iend := len(src)
	mflimit := iend - lz4MFLimit
	matchLimit := iend - lz4LastLiterals
	var ml, ref int
	var start0, ref0, ml0 int
	var start2, ref2, ml2 int
	var start3, ref3, ml3 int
	var corr int
	if iend < lz4MinLength {
		goto lastLiterals
	}
	for ip <= mflimit {
		ml, ref = c.findBest(src, ip, matchLimit, attempts, pa)
		if ml < lz4MinMatch {
			ip++
			continue
		}
		start0, ref0, ml0 = ip, ref, ml
	search2:
		if ip+ml <= mflimit {
			ml2, ref2, start2 = c.wider(src, ip+ml-2, ip, matchLimit, ml, attempts, pa, false)
		} else {
			ml2 = ml
		}
		if ml2 == ml {
			ip, anchor, op = emitSeq(src, dst, anchor, ip, ref, ml, op)
			continue
		}
		if start0 < ip && start2 < ip+ml0 {
			ip, ref, ml = start0, ref0, ml0
		}
		if start2-ip < 3 {
			ml, ip, ref = ml2, start2, ref2
			goto search2
		}
	search3:
		if start2-ip < lz4OptimalML {
			newML := ml
			if newML > lz4OptimalML {
				newML = lz4OptimalML
			}
			if ip+newML > start2+ml2-lz4MinMatch {
				newML = (start2 - ip) + ml2 - lz4MinMatch
			}
			if corr = newML - (start2 - ip); corr > 0 {
				start2 += corr
				ref2 += corr
				ml2 -= corr
			}
		}
		if start2+ml2 <= mflimit {
			ml3, ref3, start3 = c.wider(src, start2+ml2-3, start2, matchLimit, ml2, attempts, pa, false)
		} else {
			ml3 = ml2
		}
		if ml3 == ml2 {
			if start2 < ip+ml {
				ml = start2 - ip
			}
			ip, anchor, op = emitSeq(src, dst, anchor, ip, ref, ml, op)
			ip = start2
			ip, anchor, op = emitSeq(src, dst, anchor, ip, ref2, ml2, op)
			continue
		}
		if start3 < ip+ml+3 {
			if start3 >= ip+ml {
				if start2 < ip+ml {
					corr = ip + ml - start2
					start2 += corr
					ref2 += corr
					ml2 -= corr
					if ml2 < lz4MinMatch {
						start2, ref2, ml2 = start3, ref3, ml3
					}
				}
				ip, anchor, op = emitSeq(src, dst, anchor, ip, ref, ml, op)
				ip, ref, ml = start3, ref3, ml3
				start0, ref0, ml0 = start2, ref2, ml2
				goto search2
			}
			start2, ref2, ml2 = start3, ref3, ml3
			goto search3
		}
		if start2 < ip+ml {
			if start2-ip < lz4OptimalML {
				if ml > lz4OptimalML {
					ml = lz4OptimalML
				}
				if ip+ml > start2+ml2-lz4MinMatch {
					ml = (start2 - ip) + ml2 - lz4MinMatch
				}
				if corr = ml - (start2 - ip); corr > 0 {
					start2 += corr
					ref2 += corr
					ml2 -= corr
				}
			} else {
				ml = start2 - ip
			}
		}
		ip, anchor, op = emitSeq(src, dst, anchor, ip, ref, ml, op)
		ip, ref, ml = start2, ref2, ml2
		start2, ref2, ml2 = start3, ref3, ml3
		goto search3
	}
lastLiterals:
	return emitLast(src, dst, anchor, iend, op)
}

type optNode struct {
	price  int
	off    int
	mlen   int
	litlen int
}

func literalsPrice(litlen int) int {
	price := litlen
	if litlen >= lz4RunMask {
		price += 1 + (litlen-lz4RunMask)/255
	}
	return price
}

func sequencePrice(litlen, mlen int) int {
	price := 1 + 2 + literalsPrice(litlen)
	if mlen >= lz4MLMask+lz4MinMatch {
		price += 1 + (mlen-(lz4MLMask+lz4MinMatch))/255
	}
	return price
}

func (c *hcCtx) findLonger(src []byte, ip, iHigh, minLen, nb int) (int, int) {
	ml, matchPos, _ := c.wider(src, ip, ip, iHigh, minLen, nb, true, true)
	if ml <= minLen {
		return 0, 0
	}
	return ml, ip - matchPos
}

func (c *hcCtx) compressOptimal(src, dst []byte, nb int, sufficient int, fullUpdate bool) int {
	if sufficient >= lz4OptNum {
		sufficient = lz4OptNum - 1
	}
	opt := make([]optNode, lz4OptNum+lz4TrailingLit)
	ip := 0
	anchor := 0
	op := 0
	iend := len(src)
	mflimit := iend - lz4MFLimit
	matchLimit := iend - lz4LastLiterals
	for ip <= mflimit {
		llen := ip - anchor
		firstLen, firstOff := c.findLonger(src, ip, matchLimit, lz4MinMatch-1, nb)
		if firstLen == 0 {
			ip++
			continue
		}
		if firstLen > sufficient {
			ip, anchor, op = emitSeq(src, dst, anchor, ip, ip-firstOff, firstLen, op)
			continue
		}
		for rPos := 0; rPos < lz4MinMatch; rPos++ {
			opt[rPos].mlen = 1
			opt[rPos].off = 0
			opt[rPos].litlen = llen + rPos
			opt[rPos].price = literalsPrice(llen + rPos)
		}
		for mlen := lz4MinMatch; mlen <= firstLen; mlen++ {
			opt[mlen].mlen = mlen
			opt[mlen].off = firstOff
			opt[mlen].litlen = llen
			opt[mlen].price = sequencePrice(llen, mlen)
		}
		lastMatch := firstLen
		for add := 1; add <= lz4TrailingLit; add++ {
			opt[lastMatch+add].mlen = 1
			opt[lastMatch+add].off = 0
			opt[lastMatch+add].litlen = add
			opt[lastMatch+add].price = opt[lastMatch].price + literalsPrice(add)
		}
		bestMlen := 0
		bestOff := 0
		cur := 0
		for cur = 1; cur < lastMatch; cur++ {
			curPtr := ip + cur
			if curPtr > mflimit {
				break
			}
			if fullUpdate {
				if opt[cur+1].price <= opt[cur].price && opt[cur+lz4MinMatch].price < opt[cur].price+3 {
					continue
				}
			} else if opt[cur+1].price <= opt[cur].price {
				continue
			}
			var newLen, newOff int
			if fullUpdate {
				newLen, newOff = c.findLonger(src, curPtr, matchLimit, lz4MinMatch-1, nb)
			} else {
				newLen, newOff = c.findLonger(src, curPtr, matchLimit, lastMatch-cur, nb)
			}
			if newLen == 0 {
				continue
			}
			if newLen > sufficient || newLen+cur >= lz4OptNum {
				bestMlen = newLen
				bestOff = newOff
				lastMatch = cur + 1
				goto encode
			}
			baseLit := opt[cur].litlen
			for lit := 1; lit < lz4MinMatch; lit++ {
				price := opt[cur].price - literalsPrice(baseLit) + literalsPrice(baseLit+lit)
				pos := cur + lit
				if price < opt[pos].price {
					opt[pos].mlen = 1
					opt[pos].off = 0
					opt[pos].litlen = baseLit + lit
					opt[pos].price = price
				}
			}
			for ml := lz4MinMatch; ml <= newLen; ml++ {
				pos := cur + ml
				var price, ll int
				if opt[cur].mlen == 1 {
					ll = opt[cur].litlen
					if cur > ll {
						price = opt[cur-ll].price
					}
					price += sequencePrice(ll, ml)
				} else {
					ll = 0
					price = opt[cur].price + sequencePrice(0, ml)
				}
				if pos > lastMatch+lz4TrailingLit || price <= opt[pos].price {
					if ml == newLen && lastMatch < pos {
						lastMatch = pos
					}
					opt[pos].mlen = ml
					opt[pos].off = newOff
					opt[pos].litlen = ll
					opt[pos].price = price
				}
			}
			for add := 1; add <= lz4TrailingLit; add++ {
				opt[lastMatch+add].mlen = 1
				opt[lastMatch+add].off = 0
				opt[lastMatch+add].litlen = add
				opt[lastMatch+add].price = opt[lastMatch].price + literalsPrice(add)
			}
		}
		bestMlen = opt[lastMatch].mlen
		bestOff = opt[lastMatch].off
		cur = lastMatch - bestMlen
	encode:
		cand := cur
		selLen := bestMlen
		selOff := bestOff
		for {
			nextLen := opt[cand].mlen
			nextOff := opt[cand].off
			opt[cand].mlen = selLen
			opt[cand].off = selOff
			selLen = nextLen
			selOff = nextOff
			if nextLen > cand {
				break
			}
			cand -= nextLen
		}
		rPos := 0
		for rPos < lastMatch {
			ml := opt[rPos].mlen
			off := opt[rPos].off
			if ml == 1 {
				ip++
				rPos++
				continue
			}
			rPos += ml
			ip, anchor, op = emitSeq(src, dst, anchor, ip, ip-off, ml, op)
		}
	}
	return emitLast(src, dst, anchor, iend, op)
}

func compressHC(src []byte, level, dstCap int) ([]byte, error) {
	if level < 1 {
		level = lz4HCLevelDef
	}
	if level > lz4HCLevelMax {
		level = lz4HCLevelMax
	}
	bound := lz4CompressBound(len(src))
	if bound <= 0 {
		return nil, errGuest
	}
	buf := make([]byte, bound)
	c := newHCCtx()
	p := lz4HCTable[level]
	var n int
	if p.opt {
		n = c.compressOptimal(src, buf, p.nb, p.target, level == lz4HCLevelMax)
	} else {
		n = c.compressHashChain(src, buf, p.nb)
	}
	if n <= 0 {
		return nil, errGuest
	}
	if dstCap <= 0 {
		dstCap = bound
	}
	if n > dstCap {
		return nil, errGuest
	}
	return buf[:n], nil
}
