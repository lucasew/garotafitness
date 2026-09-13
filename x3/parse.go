// Package x3 reads and applies RTPatch 9.00 update records without installer code.
package x3

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

type fileVersion struct {
	name   string
	size   uint32
	w1, w2 uint32
}

// Record is one file replacement, including its source and target checksums.
type Record struct {
	Source, Target string
	old, new       fileVersion
	code           []byte
}

// Parse supports the single-source, uncompressed MODIFY records used by this
// installer. Other record types or compression modes return an error.
func Parse(data []byte) ([]Record, error) {
	r := cursor{data: data}
	h := r.take(40)
	if r.err != nil {
		return nil, r.err
	}
	if string(h[:2]) != "K*" || binary.LittleEndian.Uint16(h[2:]) != 0x384 ||
		binary.LittleEndian.Uint16(h[4:]) != 0x82e4 || binary.LittleEndian.Uint32(h[6:]) != 0x10000 {
		return nil, fmt.Errorf("x3: unsupported RTPatch header")
	}
	options := binary.LittleEndian.Uint16(h[10:])
	count := binary.LittleEndian.Uint32(h[32:])
	if count > 1<<16 {
		return nil, fmt.Errorf("x3: too many records")
	}
	n := r.u16()
	if n > 4096 {
		return nil, fmt.Errorf("x3: too many directories")
	}
	dirs := make([]string, n)
	for i := range dirs {
		dirs[i] = r.text()
	}
	var records []Record
	for r.err == nil {
		header := r.u16()
		if header == 0x1000 {
			if r.err != nil {
				return nil, r.err
			}
			if len(r.data) != 0 || len(records) != int(count) {
				return nil, fmt.Errorf("x3: record count or trailing data mismatch")
			}
			return records, nil
		}
		if header != 0x4441 && header != 0x4443 {
			return nil, fmt.Errorf("x3: unsupported record %04x", header)
		}
		opts := options
		if header&2 != 0 {
			opts = r.u16()
		}
		if opts != 0x208 && opts != 0x288 {
			return nil, fmt.Errorf("x3: unsupported record options %04x", opts)
		}
		dir := ""
		if opts&0xc0 != 0 {
			short, long := r.vli(), r.vli()
			if short < 0 || long < 0 || short >= int64(n) || long >= int64(n) {
				return nil, fmt.Errorf("x3: invalid directory index")
			}
			dir = dirs[long] + "/"
		}
		r.take(10) // record metadata
		mode := r.u16()
		srcCount, dstCount := r.vli(), r.vli()
		original, stored := r.u32(), r.u32()
		if mode != 9 || original != stored || srcCount != 1 || dstCount != 1 {
			return nil, fmt.Errorf("x3: unsupported delta mode")
		}
		old, target := r.version(), r.version()
		code := r.take(int(stored))
		if r.err != nil {
			return nil, r.err
		}
		sourceName, err := filePath(dir + old.name)
		if err != nil {
			return nil, err
		}
		targetName, err := filePath(dir + target.name)
		if err != nil {
			return nil, err
		}
		records = append(records, Record{sourceName, targetName, old, target, code})
		if len(records) > int(count) {
			return nil, fmt.Errorf("x3: excess records")
		}
	}
	return nil, r.err
}

func filePath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if !fs.ValidPath(name) || strings.ContainsAny(name, ":\x00") {
		return "", fmt.Errorf("x3: invalid member path %q", name)
	}
	return name, nil
}

type cursor struct {
	data []byte
	err  error
}

func (r *cursor) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || n > len(r.data) {
		r.err = io.ErrUnexpectedEOF
		return nil
	}
	b := r.data[:n]
	r.data = r.data[n:]
	return b
}

func (r *cursor) byte() byte {
	b := r.take(1)
	if len(b) == 0 {
		return 0
	}
	return b[0]
}
func (r *cursor) u16() uint16 {
	b := r.take(2)
	if len(b) == 0 {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}
func (r *cursor) u32() uint32 {
	b := r.take(4)
	if len(b) == 0 {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

func (r *cursor) vli() int64 {
	b := r.byte()
	count, mask := 0, byte(64)
	for mask != 0 && b&mask != 0 {
		count++
		mask >>= 1
	}
	if count > 4 {
		r.err = fmt.Errorf("x3: oversized integer")
		return 0
	}
	n := int64(b&(mask-1)) << (8 * count)
	for i := 0; i < count; i++ {
		n |= int64(r.byte()) << (8 * i)
	}
	if b&128 != 0 {
		n = -n
	}
	return n
}

func (r *cursor) text() string {
	n := int(r.byte())
	if n == 255 {
		n = int(r.u16())
	}
	if n == 0 {
		return ""
	}
	b := r.take(n)
	if len(b) == 0 {
		return ""
	}
	if b[n-1] != 0 {
		r.err = fmt.Errorf("x3: unterminated name")
		return ""
	}
	var out strings.Builder
	for _, c := range b[:n-1] {
		if c < 128 {
			out.WriteByte(c)
		} else {
			out.WriteRune(cp866[c-128])
		}
	}
	return out.String()
}

func (r *cursor) version() fileVersion {
	b := r.take(24)
	if r.err != nil {
		return fileVersion{}
	}
	v := fileVersion{size: binary.LittleEndian.Uint32(b[16:])}
	r.take(2) // weak length accumulators
	v.w1, v.w2 = r.u32()&0x7fffffff, r.u32()&0x3fffffff
	r.take(8) // timestamps
	v.name = r.text()
	return v
}
