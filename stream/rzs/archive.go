package rzs

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"

	"github.com/lucasew/garotafitness/stream/rzw"
)

// decodeSolid is rz 1.03.7 extract under the InstDecode "e"+$stdio$ hooks.
//
// After DllMain's 16-byte prefix and parseCM, official 0x41efcf seeks to
// the index. Decode SetFilePointer 0x54ff00 is virtual; ReadFile 0x54fa00
// is sequential stdin, so the next packed-indexOffset bytes are the index
// (one n≤0x1000 frame). Integers are rz 1.00 0x41ab40 / 1.03.7 0x41a820.
//
// 0x417e00 then SetFilePointer to each run's field_8 and readFrame until
// run.size bytes of frames are consumed. On this solid the prefix-sum
// field_8 values land mid-frame (stdio body order is not the 1.00
// huge-first layout). run.size still matches unique framed packets for
// streams 0–3; the leftover frames are stream 4. DecodeStreams is rzw 1.00.
func decodeSolid(r io.Reader, indexOffset, packed uint64) ([]byte, error) {
	if packed < indexOffset || indexOffset < officialCM {
		return nil, errTooLarge
	}
	idxN := packed - indexOffset
	bodyN := indexOffset - officialCM
	idx := make([]byte, idxN)
	if _, err := io.ReadFull(r, idx); err != nil {
		return nil, fmt.Errorf("rzs: index: %w", err)
	}
	body := make([]byte, bodyN)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("rzs: body: %w", err)
	}
	runs, err := parseIndexRuns(idx, int64(bodyN))
	if err != nil {
		return nil, err
	}
	frames, err := splitFrames(body)
	if err != nil {
		return nil, err
	}
	streams, err := assignRuns(frames, runs)
	if err != nil {
		return nil, err
	}
	return rzw.DecodeStreams(streams)
}

type run struct {
	stream int
	size   int64
}

func parseIndexRuns(idx []byte, bodyN int64) ([]run, error) {
	var ir indexReader
	ir.buf = idx
	// skip the 7-byte frame header; payload follows after CRC check
	if len(idx) < 7 {
		return nil, errCodec
	}
	n := int(idx[0]) | int(idx[1])<<8 | int(idx[2])<<16
	if n == 0 || n > 0x1000 || 7+n != len(idx) {
		return nil, fmt.Errorf("rzs: index frame %d", n)
	}
	payload := idx[7:]
	got := crc32.Update(crc32.ChecksumIEEE(idx[:3]), crc32.IEEETable, payload)
	if got != binary.LittleEndian.Uint32(idx[3:7]) {
		return nil, errCodec
	}
	ir.buf = payload
	count, err := ir.uint()
	if err != nil {
		return nil, fmt.Errorf("rzs: index count: %w", err)
	}
	if count == 0 || count > uint64(bodyN/8) {
		return nil, fmt.Errorf("rzs: invalid index count %d", count)
	}
	var sizes [8]int64
	runs := make([]run, count)
	for i := 0; i < int(count); i++ {
		v, err := ir.uint()
		if err != nil {
			return nil, fmt.Errorf("rzs: index entry %d: %w", i, err)
		}
		stream := int(v & 7)
		delta := int64(v >> 4)
		if v&8 != 0 {
			delta = ^delta
		}
		size := sizes[stream] + delta
		if size < 8 || size > bodyN {
			return nil, fmt.Errorf("rzs: invalid stream %d run length %d", stream, size)
		}
		sizes[stream] = size
		runs[i] = run{stream, size}
	}
	if len(ir.buf) != 0 {
		return nil, fmt.Errorf("rzs: trailing index bytes")
	}
	var sum int64
	for _, r := range runs {
		sum += r.size
	}
	if sum != bodyN {
		return nil, fmt.Errorf("rzs: run sizes %d != body %d", sum, bodyN)
	}
	return runs, nil
}

type indexReader struct{ buf []byte }

func (r *indexReader) uint() (uint64, error) {
	var v uint64
	for shift := uint(0); shift < 64; shift += 7 {
		if len(r.buf) == 0 {
			return 0, io.ErrUnexpectedEOF
		}
		b := r.buf[0]
		r.buf = r.buf[1:]
		if shift == 63 && b > 2 {
			return 0, errTooLarge
		}
		v |= uint64(b>>1) << shift
		if b&1 == 0 {
			return v, nil
		}
	}
	return 0, errTooLarge
}

func splitFrames(body []byte) ([][]byte, error) {
	var frames [][]byte
	for len(body) > 0 {
		if len(body) < 7 {
			return nil, errCodec
		}
		n := int(body[0]) | int(body[1])<<8 | int(body[2])<<16
		if n == 0 || n > len(body)-7 {
			return nil, fmt.Errorf("rzs: frame length %d", n)
		}
		payload := body[7 : 7+n]
		got := crc32.Update(crc32.ChecksumIEEE(body[:3]), crc32.IEEETable, payload)
		if got != binary.LittleEndian.Uint32(body[3:7]) {
			return nil, errCodec
		}
		if n < 8 || n%2 != 0 {
			return nil, fmt.Errorf("rzs: invalid entropy packet length %d", n)
		}
		frames = append(frames, payload)
		body = body[7+n:]
	}
	return frames, nil
}

func assignRuns(frames [][]byte, runs []run) ([8][][]byte, error) {
	var streams [8][][]byte
	used := make([]bool, len(frames))
	var multi *run
	for i := range runs {
		r := &runs[i]
		matches := 0
		var at int
		for j, f := range frames {
			if 7+len(f) == int(r.size) {
				matches++
				at = j
			}
		}
		switch matches {
		case 1:
			if used[at] {
				return streams, fmt.Errorf("rzs: run stream %d reuses frame", r.stream)
			}
			used[at] = true
			streams[r.stream] = append(streams[r.stream], frames[at])
		case 0:
			if multi != nil {
				return streams, fmt.Errorf("rzs: extra unframed run stream %d size %d", r.stream, r.size)
			}
			multi = r
		default:
			return streams, fmt.Errorf("rzs: run stream %d size %d matches %d frames", r.stream, r.size, matches)
		}
	}
	if multi == nil {
		for _, u := range used {
			if !u {
				return streams, fmt.Errorf("rzs: leftover frames")
			}
		}
		return streams, nil
	}
	for j, f := range frames {
		if used[j] {
			continue
		}
		streams[multi.stream] = append(streams[multi.stream], f)
	}
	return streams, nil
}
