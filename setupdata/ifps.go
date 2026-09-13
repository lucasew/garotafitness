package setupdata

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf16"
)

// The container and operand layouts follow TPSExec.LoadData and ReadVariable in
// RemObjects PascalScript's Source/uPSRuntime.pas. This reads bytecode as data;
// it does not instantiate a script runtime or resolve DLL imports.
type psCursor struct {
	data []byte
	pos  int
	err  error
}

func (r *psCursor) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || n > len(r.data)-r.pos {
		r.err = io.ErrUnexpectedEOF
		return nil
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}
func (r *psCursor) byte() byte {
	b := r.take(1)
	if len(b) == 0 {
		return 0
	}
	return b[0]
}
func (r *psCursor) u32() uint32 {
	b := r.take(4)
	if len(b) == 0 {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}
func (r *psCursor) text() string { return string(r.take(int(r.u32()))) }

type psType struct {
	base byte
	size int
}
type psProc struct {
	name, decl string
	external   bool
	code       []byte
}
type psProgram struct {
	types   []psType
	procs   []psProc
	globals int
}
type psValue struct {
	text   string
	number int64
	kind   byte
}
type psOperand struct {
	tag     byte
	address uint32
	value   psValue
}
type psInstruction struct {
	offset, end int
	op, sub     byte
	arg         uint32
	operands    []psOperand
}

func (p *psProgram) value(r *psCursor, index uint32) psValue {
	if int(index) >= len(p.types) {
		r.err = fmt.Errorf("IFPS: invalid type %d", index)
		return psValue{}
	}
	t := p.types[index]
	switch t.base {
	case 10, 14:
		return psValue{text: r.text(), kind: 's'}
	case 19, 28:
		n := r.u32()
		if n > 1<<20 {
			r.err = fmt.Errorf("IFPS: oversized string")
			return psValue{}
		}
		b := r.take(int(n) * 2)
		u := make([]uint16, len(b)/2)
		for i := range u {
			u[i] = binary.LittleEndian.Uint16(b[2*i:])
		}
		return psValue{text: string(utf16.Decode(u)), kind: 's'}
	}
	size := t.size
	switch t.base {
	case 1, 2, 18:
		size = 1
	case 3, 4, 20:
		size = 2
	case 5, 6, 7, 21:
		size = 4
	case 8, 17, 24, 29:
		size = 8
	case 9:
		size = 10
	case 23:
	default:
		r.err = fmt.Errorf("IFPS: unsupported constant type %d", t.base)
		return psValue{}
	}
	b := r.take(size)
	if t.base == 7 || t.base == 8 || t.base == 9 || t.base == 23 || t.base == 24 {
		return psValue{}
	}
	var v uint64
	for i, c := range b {
		v |= uint64(c) << (8 * i)
	}
	if (t.base == 2 || t.base == 4 || t.base == 6) && size < 8 {
		v = uint64(int64(v<<(64-size*8)) >> (64 - size*8))
	}
	return psValue{number: int64(v), kind: 'n'}
}

func (p *psProgram) attributes(r *psCursor) {
	n := r.u32()
	if n > 4096 {
		r.err = fmt.Errorf("IFPS: excessive attributes")
		return
	}
	for range int(n) {
		r.text()
		fields := r.u32()
		if fields > 4096 {
			r.err = fmt.Errorf("IFPS: excessive attribute fields")
			return
		}
		for range int(fields) {
			p.value(r, r.u32())
		}
	}
}

func parseIFPS(data []byte) (*psProgram, error) {
	r := psCursor{data: data}
	if !bytes.Equal(r.take(4), []byte("IFPS")) {
		return nil, fmt.Errorf("IFPS: bad magic")
	}
	version, nt, nf, nv := r.u32(), r.u32(), r.u32(), r.u32()
	r.take(8) // entry point and import-table size
	if version < 21 || version > 23 || nt > 65536 || nf > 65536 || nv > 65536 {
		return nil, fmt.Errorf("IFPS: unsupported header")
	}
	p := &psProgram{globals: int(nv)}
	for range int(nt) {
		flags := r.byte()
		t := psType{base: flags & 127}
		switch t.base {
		case 21, 25:
			r.text()
		case 26:
			r.take(16)
		case 23:
			t.size = int((r.u32() + 7) / 8)
			if t.size > 32 {
				r.err = fmt.Errorf("IFPS: invalid set")
			}
		case 22:
			r.take(8)
			if version > 22 {
				r.take(4)
			}
		case 12:
			r.take(4)
		case 11:
			n := r.u32()
			if n > 65536 {
				return nil, fmt.Errorf("IFPS: excessive fields")
			}
			r.take(int(n) * 4)
		case 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 13, 14, 16, 17, 18, 19, 20, 24, 28, 29:
		default:
			return nil, fmt.Errorf("IFPS: unsupported type %d", t.base)
		}
		p.types = append(p.types, t)
		if flags&128 != 0 {
			r.text()
		}
		p.attributes(&r)
		if r.err != nil {
			return nil, r.err
		}
	}
	for range int(nf) {
		flags := r.byte()
		f := psProc{external: flags&1 != 0}
		if f.external {
			f.name = string(r.take(int(r.byte())))
			if flags&2 != 0 {
				f.decl = r.text()
			}
		} else {
			off, n := uint64(r.u32()), uint64(r.u32())
			if n == 0 || off > uint64(len(data)) || n > uint64(len(data))-off {
				return nil, fmt.Errorf("IFPS: invalid procedure span")
			}
			f.code = data[off : off+n]
			if flags&2 != 0 {
				f.name = r.text()
				f.decl = r.text()
			}
		}
		if flags&4 != 0 {
			p.attributes(&r)
		}
		if r.err != nil {
			return nil, r.err
		}
		p.procs = append(p.procs, f)
	}
	for range int(nv) {
		if r.u32() >= nt {
			return nil, fmt.Errorf("IFPS: invalid global type")
		}
		if r.byte()&1 != 0 {
			r.text()
		}
	}
	if r.err != nil {
		return nil, r.err
	}
	return p, nil
}

func (p *psProgram) operand(r *psCursor) psOperand {
	o := psOperand{tag: r.byte()}
	switch o.tag {
	case 0:
		o.address = r.u32()
	case 1:
		o.value = p.value(r, r.u32())
	case 2, 3:
		r.take(8) // aggregate members cannot supply a known scalar here
	default:
		r.err = fmt.Errorf("IFPS: invalid operand %d", o.tag)
	}
	return o
}

func (p *psProgram) instructions(code []byte) ([]psInstruction, error) {
	r := psCursor{data: code}
	var out []psInstruction
	for r.pos < len(code) && r.err == nil {
		start := r.pos
		x := psInstruction{offset: start, op: r.byte()}
		n := 0
		if x.op == 1 || x.op == 12 {
			x.sub = r.byte()
		}
		switch x.op {
		case 0, 1, 14, 22:
			n = 2
		case 12:
			n = 3
		case 2, 3, 13, 15, 16, 21, 23, 24:
			n = 1
		case 5, 6, 7, 8, 11, 18, 25, 26:
			x.arg = r.u32()
			if x.op == 7 || x.op == 8 {
				n = 1
			}
		case 10:
			r.take(8)
		case 17:
			x.operands = append(x.operands, p.operand(&r))
			x.sub = r.byte()
		case 19:
			r.take(16)
		case 20:
			r.byte()
		case 4, 9, 255:
		default:
			return nil, fmt.Errorf("IFPS: unsupported opcode %d at %x", x.op, x.offset)
		}
		for range n {
			x.operands = append(x.operands, p.operand(&r))
		}
		x.end = r.pos
		out = append(out, x)
	}
	return out, r.err
}
