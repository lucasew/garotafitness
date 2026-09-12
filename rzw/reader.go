// Package rzw decodes CM version 5 RAZOR archives and their rzw/rzwb wrappers.
// All framing, entropy decoding, and transforms run in Go.
package rzw

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	razor   = "CM("
	version = 5
	cmLen   = 17 // four magic bytes plus a checksummed six-byte index offset
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
	prefix      uint32 // wrapper length, including its four-byte length field
	indexOffset uint64 // relative to the CM magic
}

func parseHeader(r io.Reader) (header, error) {
	var h header
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return h, err
	}
	if string(magic[:3]) != razor {
		h.prefix = binary.LittleEndian.Uint32(magic[:])
		if _, err := io.ReadFull(r, magic[:]); err != nil {
			return h, err
		}
		if string(magic[:3]) != razor {
			return h, errMagic
		}
		if h.prefix < 4+cmLen || h.prefix > maxPacked {
			return h, errTooLarge
		}
	}
	if magic[3] != version {
		return h, errVersion
	}
	payload, err := readFrame(r, 6)
	if err != nil {
		return h, err
	}
	if len(payload) != 6 {
		return h, fmt.Errorf("rzw: invalid header frame")
	}
	for i, b := range payload {
		h.indexOffset |= uint64(b) << uint(8*i)
	}
	if h.indexOffset < cmLen || h.indexOffset > maxPacked {
		return h, errTooLarge
	}
	if h.prefix != 0 && h.indexOffset >= uint64(h.prefix-4) {
		return h, fmt.Errorf("rzw: index outside wrapper")
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
	if len(p) == 0 {
		return 0, nil
	}
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
	src := r.src
	var limited *io.LimitedReader
	if r.hdr.prefix != 0 {
		limited = &io.LimitedReader{R: src, N: int64(r.hdr.prefix) - 4 - cmLen}
		src = limited
	}
	a, err := readArchive(src, r.hdr.indexOffset)
	if err != nil {
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if limited != nil && limited.N != 0 {
		return nil, fmt.Errorf("rzw: trailing wrapper bytes")
	}
	return a.decode()
}

var (
	errNil      = errors.New("rzw: nil reader")
	errMagic    = errors.New("rzw: bad magic")
	errVersion  = errors.New("rzw: bad version")
	errClosed   = errors.New("rzw: closed")
	errTooLarge = errors.New("rzw: packed too large")
	errCodec    = errors.New("rzw: invalid compressed data")
)
