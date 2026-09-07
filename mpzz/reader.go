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
// forbids running it. No public C/C++ of OGGRE v0.1.1 exists.
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
	err error
}

func (r *reader) Read([]byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	r.err = fmt.Errorf("mpzz: v%d flags %#x: %w", r.hdr.ver, r.hdr.flags, errCodec)
	return 0, r.err
}

func (r *reader) Close() error {
	r.err = errClosed
	return nil
}

var (
	errNil     = errors.New("mpzz: nil reader")
	errMagic   = errors.New("mpzz: bad magic")
	errVersion = errors.New("mpzz: bad version")
	errFlags   = errors.New("mpzz: bad flags")
	errClosed  = errors.New("mpzz: closed")
	errCodec   = errors.New("mpzz: unpublished ogg/codebook entropy; no non-PE decoder")
)
