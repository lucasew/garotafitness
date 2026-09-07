// Package rzw decodes the FitGirl rzw/rzwb atom (Christian Martelock RAZOR).
//
// On-disk tag is CM( (version 0x00000605). rzwb solids and 4x4-inner rzw
// packets prefix a little-endian u32 size (rzwrap_v0: size includes those
// 4 bytes). A bare CM( stream is also accepted.
//
// Official decode is PE: arc.ini unpackcmd `rzw d f2 f1 128 128` (rzw)
// and `rzw d f2 f1 1023 1023` (rzwb); installer files rzw.exe, rz.exe,
// razor.dll. INV-03 forbids running them.
//
// Stream after CM(: u32 crc (IEEE of plain), u32 packed, u16 extra,
// then packed bytes of ROLZ + 16 token types + interleaved adaptive
// nibble rANS (encode.su 2829/3281). rz 1.00 vtable pairs 0x409050
// (decode) with 0x4022b0 (encode). Read loads the packed payload and
// runs decode; an unfinished kernel is errCodec, not a guessed CRC.
package rzw

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	razor   = "CM("
	version = 0x00000605
	// cmLen is bytes from CM( through the trailing u16.
	// CM( + u32 version + u32 crc + u32 packed + u16 extra.
	cmLen = 3 + 4 + 4 + 4 + 2
	// maxPacked covers fg-03 rzwb (~17 MiB packed) and 4x4 packets.
	maxPacked = 64 << 20
)

// NewReader wraps a RAZOR stream as compress/gzip does.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	h, err := parseHeader(r)
	if err != nil {
		return nil, err
	}
	return &reader{src: r, hdr: h}, nil
}

type header struct {
	prefix uint32
	crc    uint32
	packed uint32
	extra  uint16
}

func parseHeader(r io.Reader) (header, error) {
	var peek [7]byte
	n, err := io.ReadFull(r, peek[:])
	if n == 0 && (err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF)) {
		return header{}, io.EOF
	}
	if err != nil {
		return header{}, err
	}
	var h header
	var cm [cmLen]byte
	switch {
	case string(peek[:3]) == razor:
		copy(cm[:7], peek[:])
		if _, err := io.ReadFull(r, cm[7:]); err != nil {
			if errors.Is(err, io.EOF) {
				return header{}, io.ErrUnexpectedEOF
			}
			return header{}, err
		}
	case string(peek[4:7]) == razor:
		h.prefix = binary.LittleEndian.Uint32(peek[:4])
		copy(cm[:3], peek[4:])
		if _, err := io.ReadFull(r, cm[3:]); err != nil {
			if errors.Is(err, io.EOF) {
				return header{}, io.ErrUnexpectedEOF
			}
			return header{}, err
		}
	default:
		return header{}, errMagic
	}
	ver := binary.LittleEndian.Uint32(cm[3:7])
	if ver != version {
		return header{}, fmt.Errorf("rzw: version %#x: %w", ver, errVersion)
	}
	h.crc = binary.LittleEndian.Uint32(cm[7:11])
	h.packed = binary.LittleEndian.Uint32(cm[11:15])
	h.extra = binary.LittleEndian.Uint16(cm[15:17])
	if h.packed == 0 || h.packed > maxPacked {
		return header{}, fmt.Errorf("rzw: packed %d: %w", h.packed, errTooLarge)
	}
	return h, nil
}

type reader struct {
	src  io.Reader
	hdr  header
	buf  []byte
	off  int
	err  error
	done bool
}

func (r *reader) Read(p []byte) (int, error) {
	if r.err != nil && r.off >= len(r.buf) {
		return 0, r.err
	}
	if !r.done {
		plain, err := r.decode()
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

func (r *reader) decode() ([]byte, error) {
	packed := make([]byte, r.hdr.packed)
	if _, err := io.ReadFull(r.src, packed); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, fmt.Errorf("rzw: packed: %w", err)
	}
	plain, err := decompress(packed, r.hdr)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

var (
	errNil      = errors.New("rzw: nil reader")
	errMagic    = errors.New("rzw: bad magic")
	errVersion  = errors.New("rzw: bad version")
	errClosed   = errors.New("rzw: closed")
	errTooLarge = errors.New("rzw: packed too large")
	errCodec    = errors.New("rzw: ROLZ+rANS kernel not lifted")
)
