package rzw

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

// A CM archive multiplexes up to eight entropy streams. Its index describes
// runs of framed packets in physical order, with lengths delta-coded backwards
// within each stream. References: rz 1.00, 0x41fdc0 and 0x418200.
type archive struct {
	streams [8][][]byte
}

// readFrame checks the CRC over the three length bytes followed by the payload.
// The checksum is not a checksum of the uncompressed output (0x4180d0).
func readFrame(r io.Reader, limit int) ([]byte, error) {
	var h [7]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return nil, err
	}
	n := int(h[0]) | int(h[1])<<8 | int(h[2])<<16
	if n == 0 || n > limit {
		return nil, fmt.Errorf("rzw: frame length %d exceeds limit %d", n, limit)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	got := crc32.Update(crc32.ChecksumIEEE(h[:3]), crc32.IEEETable, b)
	want := binary.LittleEndian.Uint32(h[3:])
	if got != want {
		return nil, fmt.Errorf("rzw: frame CRC %08x, want %08x", got, want)
	}
	return b, nil
}

// Index integers use the low bit as the continuation flag (0x41ab40).
type indexReader struct {
	src io.Reader
	buf []byte
}

func (r *indexReader) uint() (uint64, error) {
	var v uint64
	for shift := uint(0); shift < 64; shift += 7 {
		if len(r.buf) == 0 {
			b, err := readFrame(r.src, 0x1000)
			if err != nil {
				return 0, err
			}
			r.buf = b
		}
		b := r.buf[0]
		r.buf = r.buf[1:]
		if shift == 63 && b > 2 {
			return 0, fmt.Errorf("rzw: index integer overflow")
		}
		v |= uint64(b>>1) << shift
		if b&1 == 0 {
			return v, nil
		}
	}
	return 0, fmt.Errorf("rzw: index integer overflow")
}

// readArchive starts immediately after the 17-byte CM header. indexOffset is
// relative to the magic, so the bytes before the index span indexOffset-17.
func readArchive(r io.Reader, indexOffset uint64) (*archive, error) {
	if indexOffset < cmLen || indexOffset > maxPacked {
		return nil, fmt.Errorf("rzw: index offset %d: %w", indexOffset, errTooLarge)
	}
	body := make([]byte, int(indexOffset)-cmLen)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	idx := indexReader{src: r}
	n, err := idx.uint()
	if err != nil {
		return nil, fmt.Errorf("rzw: index count: %w", err)
	}
	if n == 0 || n > uint64(len(body)/8) {
		return nil, fmt.Errorf("rzw: invalid index count %d", n)
	}
	type run struct {
		stream int
		size   int64
	}
	runs := make([]run, int(n))
	var sizes [8]int64
	for i := len(runs) - 1; i >= 0; i-- {
		v, err := idx.uint()
		if err != nil {
			return nil, fmt.Errorf("rzw: index entry %d: %w", i, err)
		}
		stream := int(v & 7)
		delta := int64(v >> 4)
		if v&8 != 0 {
			delta = ^delta
		}
		size := sizes[stream] + delta
		if size < 8 || size > int64(len(body)) {
			return nil, fmt.Errorf("rzw: invalid stream %d run length %d", stream, size)
		}
		sizes[stream] = size
		runs[i] = run{stream, size}
	}
	if len(idx.buf) != 0 {
		return nil, fmt.Errorf("rzw: trailing index bytes")
	}
	a := new(archive)
	for i, run := range runs {
		if run.size > int64(len(body)) {
			return nil, fmt.Errorf("rzw: run %d crosses index", i)
		}
		packets := bytes.NewReader(body[:run.size])
		body = body[run.size:]
		for packets.Len() != 0 {
			p, err := readFrame(packets, packets.Len()-7)
			if err != nil {
				return nil, fmt.Errorf("rzw: stream %d run %d: %w", run.stream, i, err)
			}
			if len(p) < 8 || len(p)%2 != 0 {
				return nil, fmt.Errorf("rzw: invalid entropy packet length %d", len(p))
			}
			a.streams[run.stream] = append(a.streams[run.stream], p)
		}
	}
	if len(body) != 0 {
		return nil, fmt.Errorf("rzw: index leaves %d bytes unassigned", len(body))
	}
	return a, nil
}
