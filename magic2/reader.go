// Package magic2 decodes the FitGirl magic2 atom (ProFrager LOLZ v22c4b).
package magic2

import (
	"errors"
	"io"
)

// lolzTag is the 4-byte tag on RimWorld fg-02 and fg-06 solids.
const lolzTag = "DH(n"

// NewReader wraps a lolz v22c4b stream as compress/gzip does.
// Official decode is PE: CLS.ini [magic2], installer files
// cls-magic2.dll and cls-magic2_{x86,x64}.exe (PE strings:
// "lolz (ldmf)", "v22c4b [Dec 30 2018]"). INV-03 forbids
// running them. No non-PE source exists to wrap or compile.
// The rest of the bitstream is unpublished adaptive rANS+LZ.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errNil
	}
	var tag [4]byte
	if _, err := io.ReadFull(r, tag[:]); err != nil {
		return nil, err
	}
	if string(tag[:]) != lolzTag {
		return nil, errMagic
	}
	return nil, errPEOnly
}

var (
	errNil    = errors.New("magic2: nil reader")
	errMagic  = errors.New("magic2: bad magic")
	errPEOnly = errors.New("magic2: lolz v22c4b decoder is PE-only; no non-PE source")
)
