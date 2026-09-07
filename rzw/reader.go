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
// No official C exists. encode.su 3281's "source" attachment is not
// source. nishi rzwrap_v0 only shells rz.exe after the 4-byte size.
// The payload after the CM( header is unpublished ROLZ + interleaved
// rANS (encode.su 2829). NewReader is compress/gzip shaped: a valid
// tag returns a reader; Read reports that missing decoder.
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
	// CM( + u32 version + u32 crc + u32 packed + u16 zero.
	cmLen = 3 + 4 + 4 + 4 + 2
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
	return h, nil
}

type reader struct {
	src io.Reader
	hdr header
	err error
}

func (r *reader) Read([]byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	r.err = fmt.Errorf("rzw: packed %d crc %#x: %w", r.hdr.packed, r.hdr.crc, errCodec)
	return 0, r.err
}

func (r *reader) Close() error {
	r.err = errClosed
	return nil
}

var (
	errNil     = errors.New("rzw: nil reader")
	errMagic   = errors.New("rzw: bad magic")
	errVersion = errors.New("rzw: bad version")
	errClosed  = errors.New("rzw: closed")
	errCodec   = errors.New("rzw: unpublished ROLZ+rANS payload; no non-PE decoder")
)
