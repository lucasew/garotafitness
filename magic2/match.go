package magic2

// LZ match coding for lolz v22c4b (FitGirl magic2).
// Static read of cls-magic2_x64 (pe_3251605 / cls-lolz-v22c4b-x64).
// The same shapes sit in the x86 twin and in lolly v20d3 0x140014b28.

// Token / mixer (lit vs match), v20d3 0x140013680, v22 same family:
//
//	bit = binary FCM, scale 14 (state&0x3fff, >>14), adapt >>5
//	0 → literal
//	1 → match
//
//	-cm1 (default on): mixer table[ctx] at 0x140007c40 (v20) is 9
//	everywhere except a few slots. cmp edx, 9 / jge at 0x14001387f
//	skips a second binary bit when the table is 9. When the second
//	bit is decoded and is 1, al becomes 2 (special 4-byte / dxt
//	path). The mixer mixes a pair of models on this decision.
//
// Match type is not a standalone 0/1 bit. After the match token,
// decodeMatch @ 0x140039ae0 reads a 16-symbol FCM class (scale 15,
// pmovmskb|0x10000, bsf-1). Class 0 reuses rep0. The hypothesized
// polarity "binary 0 = new offset, 1 = reuse rep0" is not in this
// image: reuse is class==0 of that 16-sym, not bit==1 of a binary
// FCM. Jump table 0x14000a8c0:
//
//	0        → 0x14003a374  rep0 (load *rep0, no extra bits)
//	1        → 0x14003a1c3  new / short
//	2        → 0x14003a0ad  new; length 3+bit
//	3        → 0x14003a013  new; length base 5
//	4..10    → 0x140039cce  reps[a690[cls-4]] or long (cls==10)
//	11       → 0x140039c1d  special; length base 2
//
// The 8-sym cmp ecx,1 / add ecx,-2 at 0x14001775e (and the twin at
// 0x140017c93) is the FCM extra-class loop used by the models in
// .text 0x1656b..0x17920, not the LZ class itself: ecx==1 means no
// extra symbols, else ecx-=2 extra 8/16-sym symbols.

const (
	// matchMinLen is added to the 8-sym length symbol at 0x140039f87.
	matchMinLen = 3

	// matchLenEsc is the escape after +matchMinLen (cmp rbx, 0xa).
	// A further decode is added on top of this value (add rbx, 0xa).
	matchLenEsc = 10

	// matchClassMax is the last valid 16-sym class (cmp r15, 0xb).
	matchClassMax = 11

	// minOffset is the smallest stored distance. Class-2/3 helpers
	// reject a 0 result (test r10d / je) before the copy.
	minOffset = 1
)

// extraBitsA690 is the byte table at 0x14000a690, indexed by
// (class-4) at 0x140039cd9. Slot 6 (class 10) is special-cased
// before the lookup (cmp r15, 6 / jne).
var extraBitsA690 = [7]int{0, 1, 2, 3, 17, 18, 0}

// isRep0 reports whether match class cls reuses the last offset.
// Class 0 at 0x14003a374: ebp = *rep0, no extra offset bits.
func isRep0(cls int) bool { return cls == 0 }

// decodeMatchClass is the 16-sym FCM at 0x140039ae0.
//
//	slot = state & 0x7fff; quo = state >> 15
//	cdf row at model+0x1240 (two movdqu)
//	cls = bsf(pmovmskb(cdf>slot) | 0x10000) - 1
//
// Returns 0..matchClassMax, or -1 if the bsf overflowed.
func decodeMatchClass(st *rANS, cdf []uint16, adapt uint) (int, error) {
	bsf, err := st.bsfSym(cdf, 16, 15, adapt)
	if err != nil {
		return -1, err
	}
	cls := bsf - 1
	if cls < 0 || cls > matchClassMax {
		return -1, errBitstream
	}
	return cls, nil
}

// decodeOff writes a new distance into *rep0.
//
//	class 0:        leave *rep0 (0x14003a394)
//	class 4..9:     *rep0 = reps[extraBitsA690[cls-4]]; rotate
//	                (0x140039cce / 0x140039ddc)
//	class 10:       another 16-sym then extra bits (cmp r15,6)
//	class 1,2,3,11: new distance, min 1
//
// extra is the already-decoded extra integer for a new-offset class
// (helpers at 0x14003a0ad / 0x14003a013). For class 0 extra is ignored.
func decodeOff(cls, extra int, rep0 *int, reps []int) int {
	if cls == 0 {
		if *rep0 < minOffset {
			*rep0 = minOffset
		}
		return *rep0
	}
	if cls >= 4 && cls <= 10 {
		slot := cls - 4
		n := extraBitsA690[slot]
		if cls == 10 {
			// 0x140039ce6: slot 6 takes a second 16-sym, not a690[6].
			n = extra
		}
		if n >= 0 && n < len(reps) {
			d := reps[n]
			if n > 0 {
				copy(reps[1:n+1], reps[:n])
			}
			if d < minOffset {
				d = minOffset
			}
			reps[0] = d
			*rep0 = d
			return d
		}
	}
	d := extra
	if d < minOffset {
		d = minOffset
	}
	if len(reps) > 0 {
		copy(reps[1:], reps[:len(reps)-1])
		reps[0] = d
	}
	*rep0 = d
	return d
}

// decodeLen is the 8-sym length at 0x140039f08 (add 0x100 + bsf).
//
//	sym = bsf - 1                    // 0..7
//	n   = sym + matchMinLen          // 3..10   add rbx, 3 @ 0x140039f87
//	if n == matchLenEsc {            // cmp rbx, 0xa
//	    n = matchLenEsc + extra      // add rbx, 0xa @ 0x140039fef
//	}
//
// Class 2 uses a 1-bit form instead: n = 3 + bit (0x14003a1ba).
// Class 3 starts at 5 (add rbx, 5 @ 0x14003a0a4).
// Class 11 starts at 2 (add rbx, 2 @ 0x140039ca3).
func decodeLen(st *rANS, cdf []uint16, adapt uint) (int, error) {
	bsf, err := st.bsfSym(cdf, 8, 15, adapt)
	if err != nil {
		return 0, err
	}
	sym := bsf - 1
	if sym < 0 {
		sym = 0
	}
	n := sym + matchMinLen
	if n == matchLenEsc {
		en, err := st.symbol(cdf, 8, 15, adapt)
		if err != nil {
			return 0, err
		}
		n = matchLenEsc + en
	}
	return n, nil
}

// decodeLenBit is the 1-bit length used by class 2 at 0x14003a1ba:
// ebx = 0 or 1 from a scale-14 binary FCM, then add rbx, 3.
func decodeLenBit(bit int) int { return matchMinLen + bit&1 }

// newOffsetBits is how many raw bits a new-offset class still needs
// after the 16-sym. Classes 4–9 pull a recent offset instead.
func newOffsetBits(cls int) int {
	if cls < 4 || cls > 10 {
		return -1
	}
	if cls == 10 {
		return -1
	}
	return extraBitsA690[cls-4]
}
