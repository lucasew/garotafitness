// Package mpz decodes the FitGirl mpz atom (Sound Slimmer / MPZAPI).
package mpz

import (
	"errors"
	"io"
)

// NewReader wraps a Sound Slimmer stream as compress/gzip does.
// Official decode is PE: arc.ini [External compressor:mpz]
// unpackcmd `mpz.exe d packed.mpz out.mp3`, installer files mpz.exe /
// mpzapi.exe / MpzSlimmer.dll (Eugene Shelwien wrapper around the
// closed Sound Slimmer engine). INV-03 forbids running them. No
// non-PE source exists to wrap or compile (engine is NDA). Distinct
// from mpzz (OGGRE) and from packMP3 (.pm3).
//
// RimWorld optional OST is srep+4x4:b16mb:mpz. 4x4 calls this as the
// inner method. On-disk mpz magic is unpublished; this reader rejects
// known foreign prefixes and otherwise returns errPEOnly.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	var head [8]byte
	n, err := io.ReadFull(r, head[:])
	if n == 0 && (err == io.EOF || err == io.ErrUnexpectedEOF) {
		return nil, io.EOF
	}
	if foreign(head[:n]) {
		return nil, errMagic
	}
	if err != nil {
		return nil, err
	}
	return nil, errPEOnly
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
	errPEOnly = errors.New("mpz: sound slimmer decoder is PE-only; no non-PE source")
)
