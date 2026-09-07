// Package magic2 decodes the FitGirl magic2 atom (ProFrager LOLZ v22c4b).
//
// On-disk tag is DH(n, then 0x1f on RimWorld fg-02 and fg-06. After
// that: adaptive rANS (L=1<<23) plus LZ (rep0, nibble/binary FCM
// models, optional ldmf / DXT / raw). FitGirl method magic2 is the
// non-ldmf image; magic2l is ldmf.
//
// Official images are PE (cls-lolz / cls-magic2, "v22c4b [Dec 30
// 2018]"). INV-03 forbids running them. This package reconstructs
// the bitstream; it does not load those PEs.
package magic2

import (
	"errors"
	"io"
)

// NewReader wraps a lolz v22c4b stream as compress/gzip does.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	h, err := ParseHeader(r)
	if err != nil {
		return nil, err
	}
	return &reader{src: r, hdr: h}, nil
}

type reader struct {
	src io.Reader
	hdr Header
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
	out, err := decodeStream(r.src)
	if err != nil {
		return err
	}
	r.buf = out
	r.off = 0
	r.eof = true
	return nil
}

var (
	errNil       = errors.New("magic2: nil reader")
	errMagic     = errors.New("magic2: bad magic")
	errClosed    = errors.New("magic2: closed")
	errBitstream = errors.New("magic2: rANS+LZ bitstream not reconstructed (adaptive FCM nibble/binary models, L=1<<23)")
)
