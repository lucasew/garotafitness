package magic2

import (
	"errors"
	"io"
)

// lolzTag is the 4-byte tag on RimWorld fg-02 and fg-06 solids.
const lolzTag = "DH(n"

// verV22c4b is the byte after the tag on both RimWorld solids
// (FitGirl magic2 = lolz v22c4b).
const verV22c4b = 0x1f

// Header is the identified prefix of a lolz v22c4b stream.
type Header struct {
	Ver byte
}

// ParseHeader reads the 5-byte tag+version prefix.
func ParseHeader(r io.Reader) (Header, error) {
	var b [5]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return Header{}, err
	}
	if string(b[:4]) != lolzTag {
		return Header{}, errMagic
	}
	if b[4] != verV22c4b {
		return Header{}, errVersion
	}
	return Header{Ver: b[4]}, nil
}

var errVersion = errors.New("magic2: unsupported version")
