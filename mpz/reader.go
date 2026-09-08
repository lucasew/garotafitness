// Package mpz decodes the FitGirl mpz atom (Sound Slimmer / MPZAPI).
package mpz

import (
	"errors"
	"fmt"
	"io"
)

// NewReader wraps a Sound Slimmer stream as compress/gzip does.
//
// On-disk tag is LE u32 0x01050405 (v5.4.5.1) then orig, frame
// count, and a zero dword. RimWorld optional OST uses that as the
// 4x4 inner method (srep:m3f:mem228mb+4x4:b16mb:mpz).
//
// Official decode is PE: arc.ini unpackcmd `mpz.exe d packed.mpz
// out.mp3`. mpzapi (nishi mpzapi_v1b) only LoadLibrary's
// MpzSlimmer.dll and calls GetModule()->process. INV-03 forbids
// running that image. The guest is a mechanical RetDec
// transcription of MP3Model (FULL.c) compiled to wasm.
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
	src, err := io.ReadAll(r.src)
	if err != nil {
		return err
	}
	if len(src) == 0 {
		return fmt.Errorf("mpz: %s: %w", r.hdr, errGuest)
	}
	out, err := decodeWASM(src, r.hdr.Orig, r.hdr.Frames)
	if err != nil {
		return fmt.Errorf("mpz: %s: %w", r.hdr, err)
	}
	r.buf = out
	r.off = 0
	r.eof = true
	return nil
}

func foreign(b []byte) bool {
	if hasPrefix(b, []byte("ArC\x01")) {
		return true
	}
	if hasPrefix(b, []byte("SREP")) {
		return true
	}
	if hasPrefix(b, []byte{0x17, 0x18, 0x35, 0x26}) {
		return true
	}
	if hasPrefix(b, []byte("OGGRE")) {
		return true
	}
	if hasPrefix(b, []byte("CM(")) {
		return true
	}
	if hasPrefix(b, []byte("DH(n")) {
		return true
	}
	if hasPrefix(b, []byte("ID3")) {
		return true
	}
	if len(b) >= 2 && b[0] == 0xff && b[1]&0xe0 == 0xe0 {
		return true
	}
	return false
}

func hasPrefix(b, pfx []byte) bool {
	if len(b) < len(pfx) {
		return false
	}
	for i := range pfx {
		if b[i] != pfx[i] {
			return false
		}
	}
	return true
}

var (
	errNil    = errors.New("mpz: nil reader")
	errMagic  = errors.New("mpz: bad magic")
	errClosed = errors.New("mpz: closed")
	errGuest  = errors.New("mpz: guest")
)
