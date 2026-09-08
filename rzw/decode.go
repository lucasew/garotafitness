package rzw

import (
	"fmt"
	"hash/crc32"
)

// Token names from encode.su 3281 (Shelwien). rz 1.00 encode
// switch at 0x40288c / decode switch at 0x40a106 are 16-way.
const (
	tokDelta1xU8 = iota
	tokDelta2xU8
	tokDelta3xU8
	tokDelta4xU8
	tokDelta1xU16
	tokDelta2xU16
	tokDelta1xU32
	tokRGB
	tokRGBA
	tokImagePred
	tokMono8
	tokStereo8
	tokMono16
	tokStereo16
	tokLiterals32
	tokRawBytes
	numTok
)

// rz 1.00 encode uses dual u16 rANS, scale 2^14, renorm at <=0xffff
// (cmp [ctx+0x40], 0xffff then swap with [ctx+0x3c]). Decode 0x409050
// is the RetDec guest (rzwdec.wasm). Header crc is IEEE of the plain.
func decompress(packed []byte, h header) ([]byte, error) {
	if len(packed) != int(h.packed) {
		return nil, fmt.Errorf("rzw: packed %d want %d: %w", len(packed), h.packed, errCodec)
	}
	dcap := uint32(len(packed)) * 64
	if dcap < 1<<20 {
		dcap = 1 << 20
	}
	if dcap > 512<<20 {
		dcap = 512 << 20
	}
	plain, err := decodeWASM(packed, dcap)
	if err != nil {
		return nil, fmt.Errorf("rzw: packed %d crc %#x extra %d: %w", h.packed, h.crc, h.extra, err)
	}
	if crc32.ChecksumIEEE(plain) != h.crc {
		return nil, fmt.Errorf("rzw: crc %#x: %w", h.crc, errCodec)
	}
	return plain, nil
}
