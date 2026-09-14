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
// The native decoder reconstructs Ogg packets from separate command,
// header, and audio ranges. Solid streams share adaptive codebook models.
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
	src     io.Reader
	hdr     header
	buf     []byte
	off     int
	err     error
	decoder *oggreDecoder
}

func (r *reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.err != nil && r.off >= len(r.buf) {
		return 0, r.err
	}
	if r.off >= len(r.buf) {
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
	r.decoder = nil
	return nil
}

func (r *reader) fill() error {
	if r.decoder == nil {
		d, err := newOGGREDecoder(r.src, r.hdr)
		if err != nil {
			return err
		}
		r.decoder = d
	}
	out, err := r.decoder.next()
	if err != nil {
		return err
	}
	r.buf = out
	r.off = 0
	return nil
}

var (
	errNil     = errors.New("mpzz: nil reader")
	errMagic   = errors.New("mpzz: bad magic")
	errVersion = errors.New("mpzz: bad version")
	errFlags   = errors.New("mpzz: bad flags")
	errClosed  = errors.New("mpzz: closed")
)
