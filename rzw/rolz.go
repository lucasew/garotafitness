package rzw

import (
	"math/bits"
	"strconv"
)

type matchRecord struct {
	pos          int
	length, last byte
}

func initCDF(m []uint16, weights []byte) {
	total, sum := 0, 0
	for _, w := range weights {
		total += int(w)
	}
	for i, w := range weights {
		m[i] = uint16(sum * 16384 / total)
		sum += int(w)
	}
	for i := len(weights); i < len(m); i++ {
		m[i] = 16384
	}
}

func decodeCDF(r *rans, m []uint16, n int) int {
	r.swap()
	if !r.ok() {
		return 0
	}
	slot := r.s0 & nibMask
	s := 0
	for s+1 < n && uint32(m[s+1]) <= slot {
		s++
	}
	start := uint32(m[s])
	end := uint32(16384)
	if s+1 < n {
		end = uint32(m[s+1])
	}
	r.s0 = (r.s0>>nibScale)*(end-start) + slot - start
	for i := 0; i < n; i++ {
		target := i
		if i > s {
			target += 0x407f - n
		}
		m[i] = uint16(int(m[i]) + ((target - int(m[i])) >> 7))
	}
	return s
}

func (d *dec) recordMatch(start, length int) {
	key := byte(0)
	if start > 0 {
		key = d.out[start-1]
	}
	d.matches[key] = append(d.matches[key], matchRecord{pos: start, length: byte(length)})
}

// ROLZ references prior far/ROLZ match starts, not all literal positions.
// The selected record's original and last-used lengths predict its length.
// Reference: rz 1.00, 0x4031c0..0x40365c.
func (d *dec) rolzMatch() bool {
	key := d.prev()
	slot := decodeCDF(&d.r, d.rolz32[key][:], 27)
	off := rolzBase[slot]
	if slot != 0 {
		off += int(d.r.bits(slot))
	}
	list := d.matches[key]
	if !d.r.ok() || off >= len(list) {
		d.why = "invalid ROLZ index " + strconv.Itoa(off) + " records " + strconv.Itoa(len(list)) + " key " + strconv.Itoa(int(key))
		return false
	}
	rec := &list[len(list)-1-off]
	pred := int(rec.length)
	ctx := (bits.Len(uint(off+1))-1)&^7 | (bits.Len(uint(pred)) - 1)
	if rec.last != 0 {
		last := int(rec.last)
		if last >= pred {
			ctx = 40 | (bits.Len(uint(last-pred+1)) - 1)
			pred = last
		} else {
			ctx = 32 | (bits.Len(uint(pred-last)) - 1)
		}
	}
	if ctx < 0 || ctx >= len(d.matchLen) {
		d.why = "invalid ROLZ context"
		return false
	}
	sym := decodeCDF(&d.r, d.matchLen[ctx][:], 22)
	widths := [...]byte{6, 5, 4, 3, 2, 2, 2, 1, 0, 0, 0, 0, 0, 0, 1, 1, 2, 2, 3, 4, 5, 6}
	n := pred - 136
	for i := 0; i < sym; i++ {
		n += 1 << widths[i]
	}
	if widths[sym] != 0 {
		n += int(d.r.bits(int(widths[sym])))
	}
	rec.last = byte(n + 1)
	dist := d.pos - rec.pos
	d.reps = [4]int{dist, d.reps[0], d.reps[1], d.reps[2]}
	start := d.pos

	if !d.r.ok() || !d.copyHistory(dist, n) {
		return false
	}
	d.recordMatch(start, n)
	d.stepA8(kindRolz)
	return true
}
