// Package mpzz decodes the FitGirl mpzz atom (ProFrager OGGRE).
package mpzz

import (
	"errors"
	"fmt"
	"io"
)

// oggre is the inner tag on fg-01 after SREP (SREP keeps unique literals).
const oggre = "OGGRE"

const (
	ver0         = 0x00
	statMask     = 0x07 // books-stat -sX, X in [0,3]
	flagSolidBit = 0x08 // set when solid mode is on (default; -ds clears)
)

// NewReader wraps an OGGRE stream as compress/gzip does.
//
// On-disk: "OGGRE" + u8 version + u8 flags, then a 16-bit range-coded
// payload that rebuilds Ogg pages (writes "OggS") and setup frames.
// flags bits 0-2 are books-stat (0..3); bit 3 is solid. fg-01 is
// version 0, flags 0x09 (-s1, solid). Not LZMA/zstd/gzip.
//
// Official decode is PE: arc.ini unpackcmd oggre_dec.exe, installer
// file cls-mpzz.dll (export name CLS-OGGRE.dll). That image is a
// VirtualAlloc LZMA stub (lc=3,lp=0,pb=2) over the real CLS. INV-03
// forbids running it. The guest is a mechanical RetDec transcription
// of ClsMain/decode (CLS-OGGRE.c) compiled to wasm.
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
	ver   uint8
	flags uint8
}

func parseHeader(r io.Reader) (header, error) {
	var b [7]byte
	n, err := io.ReadFull(r, b[:])
	if n >= 5 && string(b[:5]) != oggre {
		return header{}, errMagic
	}
	if err != nil {
		return header{}, err
	}
	h := header{ver: b[5], flags: b[6]}
	if h.ver != ver0 {
		return header{}, fmt.Errorf("mpzz: version %d: %w", h.ver, errVersion)
	}
	if h.flags&statMask > 3 {
		return header{}, fmt.Errorf("mpzz: books-stat %d: %w", h.flags&statMask, errFlags)
	}
	return h, nil
}

type reader struct {
	src io.Reader
	hdr header
	buf []byte
	off int
	err error
	eof bool
}

func (r *reader) Read(p []byte) (int, error) {
	if r.err != nil && r.off >= len(r.buf) {
		return 0, r.err
	}
	if r.off >= len(r.buf) {
		if r.eof {
			return 0, io.EOF
		}
		if err := r.fill(); err != nil {
			r.err = err
			return 0, err
		}
	}
	n := copy(p, r.buf[r.off:])
	r.off += n
	return n, nil
}

func (r *reader) Close() error {
	r.err = errClosed
	r.buf = nil
	r.src = nil
	return nil
}

func (r *reader) fill() error {
	rest, err := io.ReadAll(r.src)
	if err != nil {
		return fmt.Errorf("mpzz: v%d flags %#x: %w", r.hdr.ver, r.hdr.flags, err)
	}
	src := make([]byte, 7+len(rest))
	copy(src, oggre)
	src[5] = r.hdr.ver
	src[6] = r.hdr.flags
	copy(src[7:], rest)
	out, err := decodeWASM(src)
	if err != nil {
		return fmt.Errorf("mpzz: v%d flags %#x: %w", r.hdr.ver, r.hdr.flags, err)
	}
	r.buf = out
	r.off = 0
	r.eof = true
	return nil
}

var (
	errNil     = errors.New("mpzz: nil reader")
	errMagic   = errors.New("mpzz: bad magic")
	errVersion = errors.New("mpzz: bad version")
	errFlags   = errors.New("mpzz: bad flags")
	errClosed  = errors.New("mpzz: closed")
	errGuest   = errors.New("mpzz: guest")
)
