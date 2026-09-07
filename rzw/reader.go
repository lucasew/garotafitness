// Package rzw decodes the FitGirl rzw/rzwb atom (Christian Martelock RAZOR).
package rzw

import (
	"errors"
	"io"
)

// razor is the on-disk tag. rzwb solids prefix 4 bytes; 4x4 inner rzw does not.
const razor = "CM("

// NewReader wraps a RAZOR stream as compress/gzip does.
// Official decode is PE: arc.ini unpackcmd `rzw d f2 f1 128 128` (rzw)
// and `rzw d f2 f1 1023 1023` (rzwb). Installer files rzw.exe, rz.exe,
// razor.dll. INV-03 forbids running them. No non-PE source exists to
// wrap or compile.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	var head [7]byte
	n, err := io.ReadFull(r, head[:])
	if tagged(head[:n]) {
		return nil, errPEOnly
	}
	if err != nil {
		return nil, err
	}
	return nil, errMagic
}

func tagged(b []byte) bool {
	if len(b) >= 3 && string(b[:3]) == razor {
		return true
	}
	return len(b) >= 7 && string(b[4:7]) == razor
}

var (
	errNil    = errors.New("rzw: nil reader")
	errMagic  = errors.New("rzw: bad magic")
	errPEOnly = errors.New("rzw: razor decoder is PE-only; no non-PE source")
)
