package x3

import (
	"context"
	"fmt"
)

func checksum(data []byte) (uint32, uint32) {
	var a, b uint32
	for _, c := range data {
		x, y := a^uint32(c), b^uint32(c)
		a, b = (x<<8|x>>23)&0x7fffffff, (y<<8|y>>22)&0x3fffffff
	}
	return a, b
}

// Apply checks the source version, executes bounded delta instructions, and
// checks both rolling checksums of the result before returning it.
func (r Record) Apply(ctx context.Context, old []byte) ([]byte, error) {
	a, b := checksum(old)
	if uint64(len(old)) != uint64(r.old.size) || a != r.old.w1 || b != r.old.w2 {
		return nil, fmt.Errorf("x3: source checksum mismatch: %s", r.Source)
	}
	if r.new.size > 1<<30 {
		return nil, fmt.Errorf("x3: target exceeds memory limit")
	}
	out, err := applyCode(ctx, old, r.code, int(r.new.size))
	if err != nil {
		return nil, fmt.Errorf("x3: %s: %w", r.Target, err)
	}
	a, b = checksum(out)
	if a != r.new.w1 || b != r.new.w2 {
		return nil, fmt.Errorf("x3: target checksum mismatch: %s (%08x/%08x want %08x/%08x)", r.Target, a, b, r.new.w1, r.new.w2)
	}
	return out, nil
}

// Opcode dispatch is lifted from PATCHW32 11.00, VA 0x1001874e.
func applyCode(ctx context.Context, old, code []byte, size int) ([]byte, error) {
	r := cursor{data: code}
	out := make([]byte, size)
	pos, poke := int64(0), int64(0)
	type span struct{ off, n int64 }
	var gaps, templates []span
	selected := false
	bound := func(off, n int64, size int) bool {
		return off >= 0 && n >= 0 && off <= int64(size) && n <= int64(size)-off
	}
	gap := func(n int64) {
		if !bound(pos, n, size) {
			r.err = fmt.Errorf("gap outside target")
			return
		}
		if n > 0 {
			gaps = append(gaps, span{pos, n})
			pos += n
		}
	}
	copySource := func(s span) {
		if !bound(s.off, s.n, len(old)) || !bound(pos, s.n, size) {
			r.err = fmt.Errorf("copy outside file")
			return
		}
		copy(out[pos:pos+s.n], old[s.off:s.off+s.n])
		pos += s.n
	}
	delta := func(seek int64, width int, value uint32) {
		poke += seek
		if !bound(poke, int64(width), size) {
			r.err = fmt.Errorf("delta outside target")
			return
		}
		var v uint32
		for i := 0; i < width; i++ {
			v |= uint32(out[poke+int64(i)]) << (8 * i)
		}
		v += value
		for i := 0; i < width; i++ {
			out[poke+int64(i)] = byte(v >> (8 * i))
		}
	}
	for step := 0; r.err == nil; step++ {
		if step&4095 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		op := r.byte()
		if r.err != nil {
			break
		}
		if !selected && op != 0x15 {
			return nil, fmt.Errorf("missing source selection")
		}
		switch {
		case op == 0x16:
			if len(r.data) != 0 || len(gaps) != 0 || pos != int64(size) {
				return nil, fmt.Errorf("incomplete target or trailing instructions")
			}
			return out, nil
		case op == 0x15:
			if selected || r.vli() != 0 {
				return nil, fmt.Errorf("unsupported source selection")
			}
			selected = true
		case op >= 1 && op <= 6:
			if op <= 3 {
				gap(r.vli())
			}
			width := []int{4, 2, 1}[(op-1)%3]
			pattern := r.take(width)
			n := r.vli()
			if r.err != nil {
				break
			}
			if !bound(pos, n, size) {
				return nil, fmt.Errorf("fill outside target")
			}
			for i := int64(0); i < n; i++ {
				out[pos+i] = pattern[i%int64(width)]
			}
			pos += n
		case op >= 7 && op <= 10 || op == 0x10:
			poke = 0
			width := 1
			if op == 7 {
				width = 4
			} else if op == 8 {
				width = 2
			}
			var value uint32
			if op != 10 {
				for i := 0; i < width; i++ {
					value |= uint32(r.byte()) << (8 * i)
				}
			}
			n := r.vli()
			if n < 0 || n > int64(len(r.data)) {
				return nil, fmt.Errorf("invalid delta count")
			}
			for i := int64(0); i < n && r.err == nil; i++ {
				seek := r.vli()
				if op == 10 {
					value = uint32(r.byte())
				}
				delta(seek, width, value)
			}
		case op == 0x11:
			seek := r.vli()
			delta(seek, 1, uint32(r.byte()))
		case op == 0xb || op == 0xc:
			if op == 0xb {
				gap(r.vli())
			}
			n := r.vli()
			if !bound(pos, n, size) {
				return nil, fmt.Errorf("zero fill outside target")
			}
			clear(out[pos : pos+n])
			pos += n
		case op == 0xd || op == 0xe:
			if op == 0xd {
				gap(r.vli())
			}
			i := r.vli()
			if i < 0 || i >= int64(len(templates)) {
				return nil, fmt.Errorf("invalid copy template")
			}
			copySource(templates[i])
		case op == 0xf:
			templates = append(templates, span{r.vli(), r.vli()})
		case op == 0x12:
			gap(int64(size) - pos)
			for _, s := range gaps {
				b := r.take(int(s.n))
				if r.err != nil {
					break
				}
				copy(out[s.off:s.off+s.n], b)
			}
			gaps = nil
		case op == 0x13 || op == 0x14:
			if op == 0x13 {
				gap(r.vli())
			}
			copySource(span{r.vli(), r.vli()})
		default:
			return nil, fmt.Errorf("unknown opcode %02x", op)
		}
	}
	return nil, r.err
}
