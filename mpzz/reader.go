// Package mpzz decodes the FitGirl mpzz atom (ProFrager OGGRE).
package mpzz

import (
	"errors"
	"io"
)

// oggre is the inner tag on fg-01 after SREP (SREP keeps unique literals).
const oggre = "OGGRE"

// NewReader wraps an OGGRE stream as compress/gzip does.
//
// Official decode is PE: arc.ini unpackcmd oggre_dec.exe, installer
// file cls-mpzz.dll (export name CLS-OGGRE.dll). That image is packed
// (VirtualAlloc stub only). INV-03 forbids running it.
//
// OGGRE v0.1.1 (ProFrager, 2017, closed, discontinued) is an Ogg
// Vorbis recompressor: ogg-page and setup-frame (codebook) dedup,
// optional solid mode. Encoder flags: -sX books-stat [0;3] default 1,
// -dd disable ogg dedup, -df disable setup-frame dedup, -ds disable
// solid. No public C/C++. Searched GitHub, grep.app, encode.su
// (binaries only, Cloudflare), krinkels.org (registration),
// archive.org: no non-PE source to wrap or compile.
//
// Live fg-01 (mpzz+srep:m3f:mem228mb → inner.fgpack): after SREP the
// first 16 bytes are testdata/header.hex. Bytes after 00 09 are not
// LZMA/xz/zstd/gzip/zlib/lz4/bzip2. An 8 MiB scan has no OggS,
// vorbis, ArC, or fgpack tag; every 64 KiB window uses all 256 byte
// values. A 1-byte XOR sweep of the first MiB found no repeating
// OggS. The payload is a custom entropy coder; the bitstream alone
// does not give a decoder.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	var magic [5]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return nil, err
	}
	if string(magic[:]) != oggre {
		return nil, errMagic
	}
	return nil, errPEOnly
}

var (
	errNil    = errors.New("mpzz: nil reader")
	errMagic  = errors.New("mpzz: bad magic")
	errPEOnly = errors.New("mpzz: oggre v0.1.1 decoder is PE-only; no non-PE source")
)
