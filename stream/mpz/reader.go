// Package mpz decodes the FitGirl mpz atom (Sound Slimmer / MPZAPI).
package mpz

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// NewReader decodes one bounded MPZ stream. Version 5.4.5.1 carries a
// 16-byte header followed by range-coded MP3 frames and literal runs.
// Version 5.4.5.0 carries complemented literal bytes after its four-byte tag.
// The reconstructed decoder is compiled to WASM; no installer code is loaded.
func NewReader(ctx context.Context, r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	h, err := ParseHeader(r)
	if err != nil {
		return nil, err
	}
	if h.Version == version5451 && (h.Orig == 0 || h.Orig > maxBlock) {
		return nil, fmt.Errorf("mpz: invalid output size %d", h.Orig)
	}
	ctx, cancel := context.WithCancel(ctx)
	return &reader{src: r, hdr: h, ctx: ctx, cancel: cancel}, nil
}

const maxBlock = 64 << 20

type reader struct {
	ctx    context.Context
	cancel context.CancelFunc
	src    io.Reader
	hdr    Header
	buf    []byte
	off    int
	err    error
	eof    bool
}

func (r *reader) Read(p []byte) (int, error) {
	if r.err != nil && r.off >= len(r.buf) {
		return 0, r.err
	}
	if len(p) == 0 {
		return 0, nil
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
	if r.off == len(r.buf) && r.eof {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.off:])
	r.off += n
	return n, nil
}

func (r *reader) Close() error {
	r.cancel()
	r.err = errClosed
	r.buf = nil
	r.src = nil
	return nil
}

func (r *reader) fill() error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	src, err := io.ReadAll(io.LimitReader(r.src, maxBlock+1))
	if err != nil {
		return err
	}
	if len(src) > maxBlock {
		return fmt.Errorf("mpz: compressed block exceeds memory limit")
	}
	if r.hdr.Version == version5450 {
		for i := range src {
			src[i] ^= 255
		}
		r.buf = src
		r.eof = true
		return nil
	}
	if len(src) == 0 {
		return fmt.Errorf("mpz: %s: %w", r.hdr, errGuest)
	}
	out, err := decodeWASM(r.ctx, src, r.hdr)
	if err != nil {
		return fmt.Errorf("mpz: %s: %w", r.hdr, err)
	}
	r.buf = out
	r.off = 0
	r.eof = true
	return nil
}

var (
	errNil    = errors.New("mpz: nil reader")
	errMagic  = errors.New("mpz: bad magic")
	errClosed = errors.New("mpz: closed")
	errGuest  = errors.New("mpz: guest")
)
