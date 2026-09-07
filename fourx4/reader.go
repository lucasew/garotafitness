// Package fourx4 decodes FreeArc 4x4 framing.
//
// Format and param syntax are from
// third_party/freearc/Compression/4x4/C_4x4.cpp
// (_4x4MTCompressor::ReaderThread / Process, parse_4x4).
// WASM wrap is blocked: repo mise.toml has no wasi-sdk (ADR-0004).
package fourx4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

const version = 0

const storedOut = ^uint32(0)

// Inner opens one inner-method stream. name and params are the remainder
// after 4x4's own options (block size, threads, …).
type Inner func(r io.Reader, name, params string) (io.ReadCloser, error)

// NewReader unwraps a 4x4 stream and feeds each compressed block to inner.
// params is the on-disk token after "4x4:", e.g. "b128mb:rzw".
func NewReader(r io.Reader, params string, inner Inner) (io.ReadCloser, error) {
	if r == nil {
		return nil, fmt.Errorf("fourx4: nil reader")
	}
	if inner == nil {
		return nil, fmt.Errorf("fourx4: nil inner")
	}
	name, iparams, err := parseInner(params)
	if err != nil {
		return nil, err
	}
	rd := &reader{src: r, inner: inner, name: name, iparams: iparams}
	if err := rd.readVersion(); err != nil {
		return nil, err
	}
	return rd, nil
}

type reader struct {
	src     io.Reader
	inner   Inner
	name    string
	iparams string
	buf     []byte
	off     int
	err     error
	eof     bool
}

func (r *reader) readVersion() error {
	var b [4]byte
	n, err := io.ReadFull(r.src, b[:])
	if n == 0 && (err == io.EOF || err == io.ErrUnexpectedEOF) {
		r.eof = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("fourx4: version: %w", err)
	}
	if v := binary.LittleEndian.Uint32(b[:]); v != version {
		return fmt.Errorf("fourx4: version %d", v)
	}
	return nil
}

func (r *reader) Read(p []byte) (int, error) {
	if r.err != nil && r.off >= len(r.buf) {
		return 0, r.err
	}
	for r.off >= len(r.buf) {
		if r.eof {
			return 0, io.EOF
		}
		block, err := r.next()
		if err == io.EOF {
			r.eof = true
			return 0, io.EOF
		}
		if err != nil {
			r.err = err
			return 0, err
		}
		r.buf = block
		r.off = 0
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

func (r *reader) next() ([]byte, error) {
	var hdr [8]byte
	n, err := io.ReadFull(r.src, hdr[:])
	if n == 0 && (err == io.EOF || err == io.ErrUnexpectedEOF) {
		return nil, io.EOF
	}
	if err != nil {
		return nil, fmt.Errorf("fourx4: block header: %w", err)
	}
	outSize := binary.LittleEndian.Uint32(hdr[0:4])
	inSize := binary.LittleEndian.Uint32(hdr[4:8])
	if inSize > 0x7fffffff || (outSize != storedOut && outSize > 0x7fffffff) {
		return nil, fmt.Errorf("fourx4: block too large")
	}
	in := make([]byte, inSize)
	if _, err := io.ReadFull(r.src, in); err != nil {
		return nil, fmt.Errorf("fourx4: block: %w", err)
	}
	if outSize == storedOut {
		return in, nil
	}
	ir, err := r.inner(bytes.NewReader(in), r.name, r.iparams)
	if err != nil {
		return nil, err
	}
	defer ir.Close()
	out := make([]byte, outSize)
	if _, err := io.ReadFull(ir, out); err != nil {
		return nil, fmt.Errorf("fourx4: inner: %w", err)
	}
	return out, nil
}

// parseInner splits 4x4 options from the inner method. parse_4x4 in C_4x4.cpp:
// a token is a 4x4 option when the first or second byte is a digit.
func parseInner(params string) (name, iparams string, err error) {
	if params == "" {
		return "", "", fmt.Errorf("fourx4: missing inner method")
	}
	parts := strings.Split(params, ":")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if !is4x4Opt(p) {
			inner := strings.Join(parts[i:], ":")
			name, iparams, _ = strings.Cut(inner, ":")
			if name == "" {
				return "", "", fmt.Errorf("fourx4: missing inner method")
			}
			return name, iparams, nil
		}
		if err := skip4x4Opt(p); err != nil {
			return "", "", err
		}
	}
	return "", "", fmt.Errorf("fourx4: missing inner method")
}

func is4x4Opt(p string) bool {
	if p == "" {
		return false
	}
	if p[0] >= '0' && p[0] <= '9' {
		return true
	}
	return len(p) > 1 && p[1] >= '0' && p[1] <= '9'
}

func skip4x4Opt(p string) error {
	switch p[0] {
	case 'b':
		_, err := parseMem(p[1:])
		return err
	case 't', 'i':
		_, err := parseInt(p[1:])
		return err
	case 'r':
		return nil
	}
	if _, err := parseInt(p); err == nil {
		return nil
	}
	_, err := parseMem(p)
	return err
}

// parseMem is Common.cpp parseMem64 with default spec '^'.
func parseMem(s string) (uint64, error) {
	s = strings.TrimPrefix(s, "=")
	if s == "" || s[0] < '0' || s[0] > '9' {
		return 0, fmt.Errorf("fourx4: bad mem %q", s)
	}
	var n uint64
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		n = n*10 + uint64(s[i]-'0')
		i++
	}
	suf := byte('^')
	if i < len(s) {
		suf = s[i]
	}
	switch suf {
	case 'b':
		return n, nil
	case 'k':
		return n * 1024, nil
	case 'm':
		return n * 1024 * 1024, nil
	case 'g':
		return n * 1024 * 1024 * 1024, nil
	case '^':
		if n >= 64 {
			return 0, fmt.Errorf("fourx4: bad mem %q", s)
		}
		return 1 << n, nil
	}
	return 0, fmt.Errorf("fourx4: bad mem %q", s)
}

func parseInt(s string) (uint64, error) {
	s = strings.TrimPrefix(s, "=")
	if s == "" {
		return 0, fmt.Errorf("fourx4: bad int %q", s)
	}
	var n uint64
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("fourx4: bad int %q", s)
		}
		n = n*10 + uint64(s[i]-'0')
	}
	return n, nil
}

type errString string

func (e errString) Error() string { return string(e) }

const errClosed = errString("fourx4: closed")
