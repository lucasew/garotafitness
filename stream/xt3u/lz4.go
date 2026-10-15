package xt3u

import "encoding/binary"

const (
	lz4MinMatch     = 4
	lz4MFLimit      = 12
	lz4LastLiterals = 5
	lz4MinLength    = lz4MFLimit + 1
	lz4MaxInput     = 0x7E000000
	lz4HashLog      = 12
	lz4SkipTrigger  = 6
	lz4AccDefault   = 1
	lz464KLimit     = 64<<10 + (lz4MFLimit - 1) // 65547
	lz4RunMask      = 15
	lz4MLMask       = 15
)

func lz4CompressBound(srcSize int) int {
	if srcSize < 0 || uint32(srcSize) > lz4MaxInput {
		return 0
	}
	return srcSize + srcSize/255 + 16
}

func u32le(b []byte, i int) uint32 {
	return uint32(b[i]) | uint32(b[i+1])<<8 | uint32(b[i+2])<<16 | uint32(b[i+3])<<24
}

func u16le(b []byte, i int) uint16 {
	return uint16(b[i]) | uint16(b[i+1])<<8
}

func lz4Hash4(seq uint32) uint32 {
	return (seq * 2654435761) >> ((lz4MinMatch * 8) - (lz4HashLog + 1))
}

func lz4Hash5(src []byte, i int) uint32 {
	seq := binary.LittleEndian.Uint64(src[i:])
	const prime5 = 889523592379
	return uint32(((seq << 24) * prime5) >> (64 - lz4HashLog))
}

// matchLen is LZ4_count: identical bytes from a and b, stopping at aLimit.
func matchLen(src []byte, a, b, aLimit int) int {
	start := a
	for a+8 <= aLimit && b+8 <= len(src) {
		if binary.LittleEndian.Uint64(src[a:]) != binary.LittleEndian.Uint64(src[b:]) {
			break
		}
		a += 8
		b += 8
	}
	for a < aLimit && b < len(src) && src[a] == src[b] {
		a++
		b++
	}
	return a - start
}

func writeExt(dst []byte, op, n int) int {
	for n >= 255 {
		dst[op] = 255
		op++
		n -= 255
	}
	dst[op] = byte(n)
	op++
	return op
}

func emitSeq(src, dst []byte, anchor, ip, matchPos, matchLen, op int) (int, int, int) {
	tokenAt := op
	op++
	lit := ip - anchor
	if lit >= lz4RunMask {
		dst[tokenAt] = lz4RunMask << 4
		op = writeExt(dst, op, lit-lz4RunMask)
	} else {
		dst[tokenAt] = byte(lit << 4)
	}
	copy(dst[op:op+lit], src[anchor:ip])
	op += lit
	off := ip - matchPos
	dst[op] = byte(off)
	dst[op+1] = byte(off >> 8)
	op += 2
	ml := matchLen - lz4MinMatch
	if ml >= lz4MLMask {
		dst[tokenAt] += lz4MLMask
		op = writeExt(dst, op, ml-lz4MLMask)
	} else {
		dst[tokenAt] += byte(ml)
	}
	ip += matchLen
	return ip, ip, op
}

func emitLast(src, dst []byte, anchor, iend, op int) int {
	last := iend - anchor
	if last >= lz4RunMask {
		dst[op] = lz4RunMask << 4
		op++
		op = writeExt(dst, op, last-lz4RunMask)
	} else {
		dst[op] = byte(last << 4)
		op++
	}
	copy(dst[op:op+last], src[anchor:iend])
	return op + last
}

// lz4CompressFast is LZ4_compress_fast on a 64-bit little-endian host:
// byU16+hash4 below 64KiB+11, byU32+hash5 otherwise. dst must hold
// lz4CompressBound(len(src)) bytes (notLimited).
func lz4CompressFast(src, dst []byte, accel int) int {
	if accel < 1 {
		accel = lz4AccDefault
	}
	n := len(src)
	if uint32(n) > lz4MaxInput {
		return 0
	}
	byU16 := n < lz464KLimit
	var table16 []uint16
	var table32 []uint32
	if byU16 {
		table16 = make([]uint16, 1<<(lz4HashLog+1))
	} else {
		table32 = make([]uint32, 1<<lz4HashLog)
	}
	hashAt := func(i int) uint32 {
		if byU16 {
			return lz4Hash4(u32le(src, i))
		}
		return lz4Hash5(src, i)
	}
	putAt := func(i int, h uint32) {
		if byU16 {
			table16[h] = uint16(i)
		} else {
			table32[h] = uint32(i)
		}
	}
	getAt := func(h uint32) uint32 {
		if byU16 {
			return uint32(table16[h])
		}
		return table32[h]
	}
	ip := 0
	anchor := 0
	op := 0
	iend := n
	mflimitPlusOne := iend - lz4MFLimit + 1
	matchLimit := iend - lz4LastLiterals
	if n < lz4MinLength {
		return emitLast(src, dst, anchor, iend, op)
	}
	putAt(ip, hashAt(ip))
	ip++
	forwardH := hashAt(ip)
	for {
		forwardIP := ip
		step := 1
		searchMatchNb := accel << lz4SkipTrigger
		var match int
		for {
			h := forwardH
			current := uint32(forwardIP)
			matchIndex := getAt(h)
			ip = forwardIP
			forwardIP += step
			step = searchMatchNb >> lz4SkipTrigger
			searchMatchNb++
			if forwardIP > mflimitPlusOne {
				return emitLast(src, dst, anchor, iend, op)
			}
			match = int(matchIndex)
			forwardH = hashAt(forwardIP)
			putAt(int(current), h)
			if !byU16 && matchIndex+65535 < current {
				continue
			}
			if u32le(src, match) == u32le(src, ip) {
				break
			}
		}
		for ip > anchor && match > 0 && src[ip-1] == src[match-1] {
			ip--
			match--
		}
		for {
			mlen := lz4MinMatch + matchLen(src, ip+lz4MinMatch, match+lz4MinMatch, matchLimit)
			ip, anchor, op = emitSeq(src, dst, anchor, ip, match, mlen, op)
			if ip >= mflimitPlusOne {
				return emitLast(src, dst, anchor, iend, op)
			}
			putAt(ip-2, hashAt(ip-2))
			h := hashAt(ip)
			current := uint32(ip)
			matchIndex := getAt(h)
			putAt(ip, h)
			if byU16 || matchIndex+65535 >= current {
				if u32le(src, int(matchIndex)) == u32le(src, ip) {
					match = int(matchIndex)
					continue
				}
			}
			break
		}
		ip++
		forwardH = hashAt(ip)
	}
}

func compressFast(src []byte, accel, dstCap int) ([]byte, error) {
	if accel <= 0 {
		accel = lz4AccDefault
	}
	bound := lz4CompressBound(len(src))
	if bound <= 0 {
		return nil, errGuest
	}
	buf := make([]byte, bound)
	n := lz4CompressFast(src, buf, accel)
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
