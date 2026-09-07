// Package magic2 decodes the FitGirl magic2 atom (ProFrager LOLZ v22c4b).
//
// On-disk tag is DH(n, then 0x1f on RimWorld fg-02 and fg-06. The rest
// is unpublished adaptive rANS+LZ (optional ldmf, DXT/raw models).
//
// Official decode is PE: lolz22c4b.7z ships lolz_x64.exe and
// cls-lolz_{x86,x64}.exe + cls-lolz.dll (PE strings: "lolz (ldmf)",
// "v22c4b [Dec 30 2018]"). FitGirl names the same image cls-magic2.
// INV-03 forbids running them. No non-PE source exists to wrap
// (GitHub, encode.su, krinkels.org, nishi.dreamhosters.com
// lolz19j.7z / lolz22c4b.7z are PE-only).
//
// fg-06 steam_appid.txt is method magic2 with no outer srep, but it
// shares a 93116-byte solid (unpacked 430889) with four other members.
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
	if _, err := ParseHeader(r); err != nil {
		return nil, err
	}
	return nil, errPEOnly
}

var (
	errNil    = errors.New("magic2: nil reader")
	errMagic  = errors.New("magic2: bad magic")
	errPEOnly = errors.New("magic2: lolz v22c4b decoder is PE-only; no non-PE source")
)
