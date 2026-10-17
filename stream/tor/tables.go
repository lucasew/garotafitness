package tor

// Data tables are stored diffed (DataTables.cpp) and undiffed on each flush
// so the bytes written out are original. The circular buffer keeps a 512-byte
// pad on both sides; a table that crosses a wrap is copied into the front pad.

type tableEntry struct {
	typ   int
	start int
	rows  int
}

type dataTables struct {
	list     []tableEntry
	base     []byte
	original []byte
	baseN    int
}

func newDataTables() *dataTables {
	return &dataTables{
		list:     make([]tableEntry, 0, tableEntries),
		base:     make([]byte, maxTableRow),
		original: make([]byte, maxTableRow),
	}
}

func (t *dataTables) add(typ, start, rows int) error {
	if typ <= 0 || typ > maxTableRow {
		return errFormat
	}
	// Grow only once the current list is full. filled() observes len==cap
	// before that next add, which is what triggers a flush.
	if len(t.list) == cap(t.list) {
		ncap := cap(t.list) * 2
		next := make([]tableEntry, len(t.list), ncap)
		copy(next, t.list)
		t.list = next
	}
	t.list = append(t.list, tableEntry{typ: typ, start: start, rows: rows})
	return nil
}

func (t *dataTables) filled() bool {
	return len(t.list) == cap(t.list)
}

func (t *dataTables) process(buf []byte, diff bool, writeEnd int) error {
	for i := range t.list {
		p := &t.list[i]
		if p.typ <= 0 {
			return errFormat
		}
		// 1 + (writeEnd-start)/type, division toward zero, matching C++.
		extra := 1 + (writeEnd-p.start)/p.typ
		n := p.rows
		if extra < n {
			n = extra
		}
		if n <= 0 {
			continue
		}
		end := int64(p.start) + int64(p.typ)*int64(n)
		if p.start < 0 || end > int64(len(buf)) {
			return errFormat
		}
		if diff {
			diffTable(buf, p.typ, p.start, n)
		} else {
			undiffTable(buf, p.typ, p.start, n)
		}
	}
	return nil
}

func (t *dataTables) undiff(buf []byte, writeStart, writeEnd int) error {
	if len(t.list) > 0 && t.list[0].start < writeStart {
		typ := t.list[0].typ
		n := typ
		if d := writeStart - t.list[0].start; d < n {
			n = d
		}
		if n < 0 || t.list[0].start < 0 || n > len(t.original) || t.list[0].start+n > len(buf) {
			return errFormat
		}
		t.baseN = n
		copy(t.original[:n], buf[t.list[0].start:t.list[0].start+n])
		copy(buf[t.list[0].start:t.list[0].start+n], t.base[:n])
	}
	return t.process(buf, false, writeEnd)
}

func (t *dataTables) diff(buf []byte, writeStart, writeEnd int) error {
	if len(t.list) > 0 {
		p := t.list[len(t.list)-1]
		if p.typ <= 0 {
			return errFormat
		}
		processed := (writeEnd - p.start) / p.typ
		if processed < p.rows {
			processed -= 2
			if processed < 0 {
				processed = 0
			}
			p.start += processed * p.typ
			p.rows -= processed
			if p.typ > len(t.base) || p.start < 0 || p.start+p.typ > len(buf) {
				return errFormat
			}
			copy(t.base[:p.typ], buf[p.start:p.start+p.typ])
			if err := t.process(buf, true, writeEnd); err != nil {
				return err
			}
			t.restore(buf, writeStart)
			t.list = t.list[:0]
			t.list = append(t.list, p)
			return nil
		}
	}
	if err := t.process(buf, true, writeEnd); err != nil {
		return err
	}
	t.restore(buf, writeStart)
	t.list = t.list[:0]
	return nil
}

func (t *dataTables) restore(buf []byte, writeStart int) {
	if len(t.list) == 0 || t.list[0].start >= writeStart || t.baseN <= 0 {
		return
	}
	n := t.baseN
	start := t.list[0].start
	copy(buf[start:start+n], t.original[:n])
}

// shift moves table pointers back by oldPos-newPos and copies the prefix that
// landed before newPos into the pad so undiff can walk a contiguous row.
func (t *dataTables) shift(buf []byte, oldPos, newPos int) error {
	if len(t.list) > 1 || oldPos <= newPos {
		return errFormat
	}
	delta := oldPos - newPos
	for i := range t.list {
		old := t.list[i].start
		t.list[i].start -= delta
		ncopy := newPos - t.list[i].start
		if ncopy < 0 || t.list[i].start < 0 || old < 0 || old+ncopy > len(buf) || t.list[i].start+ncopy > len(buf) {
			return errFormat
		}
		copy(buf[t.list[i].start:t.list[i].start+ncopy], buf[old:old+ncopy])
	}
	return nil
}

func undiffTable(buf []byte, n, start, rows int) {
	if rows <= 1 {
		return
	}
	end := start + n*rows
	switch n {
	case 2:
		v := uint16(buf[start]) | uint16(buf[start+1])<<8
		for r := start + 2; r < end; r += 2 {
			v += uint16(buf[r]) | uint16(buf[r+1])<<8
			buf[r] = byte(v)
			buf[r+1] = byte(v >> 8)
		}
	case 4:
		v := uint32(buf[start]) | uint32(buf[start+1])<<8 | uint32(buf[start+2])<<16 | uint32(buf[start+3])<<24
		for r := start + 4; r < end; r += 4 {
			v += uint32(buf[r]) | uint32(buf[r+1])<<8 | uint32(buf[r+2])<<16 | uint32(buf[r+3])<<24
			buf[r] = byte(v)
			buf[r+1] = byte(v >> 8)
			buf[r+2] = byte(v >> 16)
			buf[r+3] = byte(v >> 24)
		}
	default:
		r := start
		for {
			r += n
			if r >= end {
				break
			}
			var carry int
			for i := 0; i < n; i++ {
				temp := int(buf[r+i]) + int(buf[r+i-n]) + carry
				buf[r+i] = byte(temp)
				carry = temp >> 8
			}
		}
	}
}

func diffTable(buf []byte, n, start, rows int) {
	if rows <= 1 {
		return
	}
	end := start + n*rows
	switch n {
	case 2:
		prev := uint16(buf[start]) | uint16(buf[start+1])<<8
		for r := start + 2; r < end; r += 2 {
			v := uint16(buf[r]) | uint16(buf[r+1])<<8
			d := v - prev
			buf[r] = byte(d)
			buf[r+1] = byte(d >> 8)
			prev = v
		}
	case 4:
		prev := uint32(buf[start]) | uint32(buf[start+1])<<8 | uint32(buf[start+2])<<16 | uint32(buf[start+3])<<24
		for r := start + 4; r < end; r += 4 {
			v := uint32(buf[r]) | uint32(buf[r+1])<<8 | uint32(buf[r+2])<<16 | uint32(buf[r+3])<<24
			d := v - prev
			buf[r] = byte(d)
			buf[r+1] = byte(d >> 8)
			buf[r+2] = byte(d >> 16)
			buf[r+3] = byte(d >> 24)
			prev = v
		}
	default:
		r := end
		for {
			r -= n
			if r <= start {
				break
			}
			var carry int
			for i := 0; i < n; i++ {
				sum := int(buf[r+i-n]) + carry
				borrow := 0
				if int(buf[r+i]) < sum {
					borrow = 1
				}
				buf[r+i] = byte(int(buf[r+i]) - sum)
				carry = borrow
			}
		}
	}
}
