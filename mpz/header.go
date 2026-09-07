package mpz

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Sound Slimmer on-disk versions (LE). MpzSlimmer.dll accepts both.
const (
	version5450 = 0x00050405
	version5451 = 0x01050405
)

const headerLen = 16

// Header is the 16-byte tag on a raw mpz block (4x4 inner payload).
type Header struct {
	Version uint32
	Orig    uint32
	Frames  uint32
	Extra   uint32
}

// ParseHeader reads the official 16-byte mpz tag.
func ParseHeader(r io.Reader) (Header, error) {
	var b [headerLen]byte
	n, err := io.ReadFull(r, b[:4])
	if n == 0 && (err == io.EOF || err == io.ErrUnexpectedEOF) {
		return Header{}, io.EOF
	}
	if err != nil {
		return Header{}, err
	}
	ver := binary.LittleEndian.Uint32(b[0:4])
	if ver != version5450 && ver != version5451 {
		m, _ := io.ReadFull(r, b[4:8])
		if foreign(b[:4+m]) {
			return Header{}, errMagic
		}
		return Header{}, errMagic
	}
	if _, err := io.ReadFull(r, b[4:]); err != nil {
		if err == io.EOF {
			return Header{}, io.ErrUnexpectedEOF
		}
		return Header{}, err
	}
	return Header{
		Version: ver,
		Orig:    binary.LittleEndian.Uint32(b[4:8]),
		Frames:  binary.LittleEndian.Uint32(b[8:12]),
		Extra:   binary.LittleEndian.Uint32(b[12:16]),
	}, nil
}

func (h Header) String() string {
	return fmt.Sprintf("v%x orig %d frames %d", h.Version, h.Orig, h.Frames)
}
