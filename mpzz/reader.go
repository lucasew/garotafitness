// Package mpzz decodes the FitGirl mpzz atom (ProFrager OGGRE).
package mpzz

import (
	"errors"
	"io"
)

// oggre is the inner tag on fg-01 after SREP (SREP keeps unique literals).
const oggre = "OGGRE"

// NewReader wraps an OGGRE stream as compress/gzip does.
// Official decode is PE: arc.ini unpackcmd oggre_dec.exe, installer
// file cls-mpzz.dll (export name CLS-OGGRE.dll). That image is packed
// (VirtualAlloc stub only). INV-03 forbids running it. No non-PE
// source exists to wrap or compile.
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
	errPEOnly = errors.New("mpzz: oggre decoder is PE-only; no non-PE source")
)
