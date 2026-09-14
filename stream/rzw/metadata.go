package rzw

import (
	"encoding/binary"
	"fmt"
)

type fileRecord struct {
	name       string
	size       uint64
	time       uint64
	crc        uint32
	attributes uint32
	parent     int32
}

type metadata struct {
	window uint32
	files  []fileRecord
}

// The metadata is logical stream zero, decoded with a 1 MiB LZ window.
// Fields are columnar: names, sizes, times, CRCs, attributes, parent indices.
// Reference: rz 1.00, 0x41e280.
func readMetadata(frames [][]byte) (metadata, error) {
	var m metadata
	d := newDec(nil, 1<<20)
	d.r.frames = frames
	off := 0
	take := func(n int) ([]byte, error) {
		if n < 0 || n > cap(d.out)-off {
			return nil, fmt.Errorf("rzw: metadata too large")
		}
		for d.pos < off+n {
			before := d.pos
			if !d.decodeByte() || d.pos <= before {
				return nil, fmt.Errorf("rzw: metadata at byte %d: %w", before, errCodec)
			}
		}
		p := d.out[off : off+n]
		off += n
		return p, nil
	}
	p, err := take(8)
	if err != nil {
		return m, err
	}
	m.window = binary.LittleEndian.Uint32(p)
	count := binary.LittleEndian.Uint32(p[4:])
	if m.window == 0 || m.window > 1<<30 || count > 1<<15 {
		return m, fmt.Errorf("rzw: invalid metadata limits")
	}
	m.files = make([]fileRecord, count)
	for i := range m.files {
		name := make([]byte, 0, 32)
		for {
			p, err = take(1)
			if err != nil {
				return m, err
			}
			if p[0] == 0 {
				break
			}
			if len(name) == 511 {
				return m, fmt.Errorf("rzw: file name too long")
			}
			name = append(name, p[0])
		}
		m.files[i].name = string(name)
	}
	for column, width := range []int{8, 8, 4, 4, 4} {
		for i := range m.files {
			p, err = take(width)
			if err != nil {
				return m, err
			}
			f := &m.files[i]
			switch column {
			case 0:
				f.size = binary.LittleEndian.Uint64(p)
			case 1:
				f.time = binary.LittleEndian.Uint64(p)
			case 2:
				f.crc = binary.LittleEndian.Uint32(p)
			case 3:
				f.attributes = binary.LittleEndian.Uint32(p)
			case 4:
				f.parent = int32(binary.LittleEndian.Uint32(p))
			}
		}
	}
	if off != d.pos || !d.r.finished() {
		return m, fmt.Errorf("rzw: trailing metadata symbols")
	}
	for i, f := range m.files {
		if f.parent < -1 || int64(f.parent) >= int64(i) {
			return m, fmt.Errorf("rzw: invalid parent index")
		}
		if f.parent >= 0 && m.files[f.parent].attributes&0x10 == 0 {
			return m, fmt.Errorf("rzw: parent is not a directory")
		}
	}
	return m, nil
}

func (r *rans) finished() bool {
	return r.ok() && r.end == 0 && len(r.frames) == 0 && r.s0 == ransLimit && r.s1 == ransLimit
}
