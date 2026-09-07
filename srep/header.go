// Package srep wraps official SuperREP as a stream atom.
package srep

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	ziganshin = 0x26351817
	signature = 0x50455253 // SREP
)

// Format is the on-disk SREP version.
type Format uint8

const (
	FormatInvalid Format = iota
	FormatIOLZRounded
	FormatIOLZ
	FormatFutureLZ
	FormatIndexLZ
)

// Header is the 16-byte archive tag at the start of a raw SREP solid.
type Header struct {
	Format    Format
	HashNum   uint8
	Seed      uint8
	HashExtra uint8
	BaseLen   uint32
}

// ParseHeader reads the official 16-byte SREP tag.
func ParseHeader(r io.Reader) (Header, error) {
	var b [16]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return Header{}, fmt.Errorf("srep header: %w", err)
	}
	if binary.LittleEndian.Uint32(b[0:4]) != ziganshin {
		return Header{}, fmt.Errorf("srep header: bad ziganshin tag")
	}
	if binary.LittleEndian.Uint32(b[4:8]) != signature {
		return Header{}, fmt.Errorf("srep header: bad signature")
	}
	h2 := binary.LittleEndian.Uint32(b[8:12])
	ver := Format(h2 & 0xff)
	if ver < FormatIOLZRounded || ver > FormatIndexLZ {
		return Header{}, fmt.Errorf("srep header: version %d", ver)
	}
	return Header{
		Format:    ver,
		HashNum:   uint8(h2 >> 8),
		Seed:      uint8(h2 >> 16),
		HashExtra: uint8(h2 >> 24),
		BaseLen:   binary.LittleEndian.Uint32(b[12:16]),
	}, nil
}
