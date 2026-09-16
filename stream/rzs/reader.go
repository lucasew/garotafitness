// Package rzs unwraps the RAZOR stdio filter used as FreeArc method rzs.
//
// unpackcmd is `rz-1.03.7-stdio.exe e -y $stdio$`. Christian Martelock's
// RAZOR has no public C/C++. stream/rzw is a reverse-engineering of rz
// 1.00. This package recovers the 1.03.7 header from official rz.exe
// (encode.su rz_1.03.7.zip) as data and strips the $stdio$ wrapper.
package rzs

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

const (
	headerSize = 16
	razor      = "CM("
	version    = 5
	// officialCM is the 17-byte raw header: magic plus one 7+6 frame.
	// rz 1.03.7, 0x41efcf: body length is indexOffset-0x11.
	officialCM = 17
	maxPacked  = 4<<30 - 1
	maxPlain   = 4<<30 - 1
)

// NewReader wraps an rzs solid as compress/gzip does.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	var hdr [headerSize]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("rzs: header: %w", err)
	}
	plain := binary.LittleEndian.Uint64(hdr[0:8])
	packed := binary.LittleEndian.Uint64(hdr[8:16])
	if packed == 0 || packed > maxPacked || plain == 0 || plain > maxPlain {
		return nil, errTooLarge
	}
	indexOffset, consumed, err := parseCM(r)
	if err != nil {
		return nil, err
	}
	if consumed < officialCM {
		return nil, errMagic
	}
	if indexOffset < officialCM || packed < officialCM || packed < indexOffset {
		return nil, errTooLarge
	}
	return &reader{src: r, indexOffset: indexOffset, packed: packed, plainN: plain}, nil
}

type reader struct {
	src         io.Reader
	indexOffset uint64
	packed      uint64
	buf         []byte
	off         int
	plainN      uint64
	err         error
	done        bool
}

func (r *reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.err != nil && r.off >= len(r.buf) {
		return 0, r.err
	}
	if !r.done {
		plain, err := decodeSolid(r.src, r.indexOffset, r.packed)
		r.done = true
		if err != nil {
			r.err = err
			return 0, err
		}
		r.buf = plain
	}
	if r.off >= len(r.buf) {
		if r.err != nil {
			return 0, r.err
		}
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.off:])
	r.off += n
	return n, nil
}

func (r *reader) Close() error {
	r.err = errClosed
	r.buf = nil
	return nil
}

// parseCM recovers the 1.03.7 header from official rz.exe.
//
// 0x41ef5d: cmpl $0x05284d43, (%rdi,%rax,1)  — "CM(\x05"
// 0x41ef66: cmpl $0x00000605, 3(%rdi,%rax,1) — version plus 3-byte length 6
// 0x41ef70: lea 0x4(%rax), %edx then 0x417cd0 readFrame
// 0x417a60 / 0x42cb70: IEEE CRC32 over 3-byte length + payload, compared
// to the stored dword after a final not (0x417da4)
// 0x41ef99: 6-byte little-endian index offset
// 0x42b140 / 0x42b160: classA/classB match rz 1.00 0x42a140 / 0x42a160
//
// The $stdio$ encoder used by FitGirl inserts a second copy of the
// 3-byte length between that official prefix and the frame CRC. Official
// readFrame at +4 then fails; the same frame at +7 matches. Both SoC
// solids do this. After the extra field, framing matches rz 1.00.
func parseCM(r io.Reader) (indexOffset, consumed uint64, err error) {
	var prefix [7]byte
	if _, err := io.ReadFull(r, prefix[:]); err != nil {
		return 0, 0, err
	}
	consumed = 7
	if string(prefix[:3]) != razor {
		return 0, consumed, errMagic
	}
	if prefix[3] != version {
		return 0, consumed, errVersion
	}
	len3 := prefix[4:7]
	if len3[0] != 6 || len3[1] != 0 || len3[2] != 0 {
		return 0, consumed, errMagic
	}
	var next [3]byte
	if _, err := io.ReadFull(r, next[:]); err != nil {
		return 0, consumed, err
	}
	consumed += 3
	var crc [4]byte
	var payload [6]byte
	if next[0] == 6 && next[1] == 0 && next[2] == 0 {
		// $stdio$ length echo. Official 0x417cd0 frame starts here.
		if _, err := io.ReadFull(r, crc[:]); err != nil {
			return 0, consumed, err
		}
		consumed += 4
		if _, err := io.ReadFull(r, payload[:]); err != nil {
			return 0, consumed, err
		}
		consumed += 6
		if !frameCRC(next[:], crc[:], payload[:]) {
			return 0, consumed, errCodec
		}
	} else {
		crc[0], crc[1], crc[2] = next[0], next[1], next[2]
		if _, err := io.ReadFull(r, crc[3:]); err != nil {
			return 0, consumed, err
		}
		consumed++
		if _, err := io.ReadFull(r, payload[:]); err != nil {
			return 0, consumed, err
		}
		consumed += 6
		if !frameCRC(len3, crc[:], payload[:]) {
			return 0, consumed, errCodec
		}
	}
	for i, b := range payload {
		indexOffset |= uint64(b) << uint(8*i)
	}
	if indexOffset < officialCM || indexOffset > maxPacked {
		return 0, consumed, errTooLarge
	}
	return indexOffset, consumed, nil
}

func frameCRC(len3, crc, payload []byte) bool {
	got := crc32.Update(crc32.ChecksumIEEE(len3), crc32.IEEETable, payload)
	return got == binary.LittleEndian.Uint32(crc)
}

var (
	errNil      = errors.New("rzs: nil reader")
	errMagic    = errors.New("rzs: bad magic")
	errVersion  = errors.New("rzs: bad version")
	errTooLarge = errors.New("rzs: packed too large")
	errCodec    = errors.New("rzs: invalid compressed data")
	errClosed   = errors.New("rzs: closed")
)
