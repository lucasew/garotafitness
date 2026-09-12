package magic2

import (
	"encoding/binary"
	"fmt"
	"io"
)

const headerLen = 9

// Header contains the packed LOLZ options. The apparent "DH(n" prefix
// is an options word, not a signature (cls-magic2_x64, VA 0x140013e7d).
type Header struct {
	DictionarySize                                                uint32
	Workers                                                       int
	Mixed                                                         bool
	Independent                                                   bool
	LongDistance                                                  bool
	ROLZ                                                          bool
	LiteralMode                                                   byte
	ColorMode, AlphaMode, ImageMode                               byte
	ClassShift, HighShift, LowShift, PredictionShift, WeightShift uint
}

func ParseHeader(r io.Reader) (Header, error) {
	var b [headerLen]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return Header{}, err
	}
	flags := binary.LittleEndian.Uint32(b[:4])
	opts := binary.LittleEndian.Uint32(b[4:8])
	h := Header{DictionarySize: ((opts >> 13) & 127) << 24, Workers: int((opts>>20)&15) + 1,
		Mixed: flags&(1<<30) != 0, Independent: flags>>31 != 0, LongDistance: b[8]&1 != 0,
		ROLZ: (opts>>11)&3 != 0, LiteralMode: byte((opts >> 8) & 7),
		ColorMode: byte((flags >> 26) & 7), AlphaMode: byte((flags >> 23) & 7), ImageMode: (b[8] >> 1) & 7}
	shifts := []*uint{&h.ClassShift, &h.WeightShift, &h.LowShift, &h.PredictionShift, &h.HighShift}
	for i, p := range shifts {
		n := (flags >> uint(i*4)) & 15
		if n > 8 {
			return Header{}, fmt.Errorf("magic2: invalid context width %d", n)
		}
		*p = uint(8 - n)
	}
	if h.DictionarySize == 0 || b[8]&0xf0 != 0 {
		return Header{}, fmt.Errorf("magic2: invalid options")
	}
	return h, nil
}
