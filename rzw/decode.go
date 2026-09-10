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

// rz 1.00 uses dual u32 rANS, 16-bit renormalization at <=0xffff,
// binary models at scale 2^12 and nibble models at scale 2^14.
// The decode vtable slot is 0x4022b0 (0x409050 is the encoder).
// Binary rANS: PE 0x4023ef `cmp freq,slot; jbe match` — literal iff slot < freq.
// Header crc is IEEE of the plain.
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
	plain, err := decodeNative(packed, int(dcap))
	if err != nil || crc32.ChecksumIEEE(plain) != h.crc {
		// WASM guest is the RetDec transcription; keep it as a second try.
		if wplain, werr := decodeWASM(packed, dcap); werr == nil && crc32.ChecksumIEEE(wplain) == h.crc {
			return wplain, nil
		}
		if err != nil {
			return nil, fmt.Errorf("rzw: packed %d crc %#x extra %d: %w", h.packed, h.crc, h.extra, err)
		}
		return nil, fmt.Errorf("rzw: crc %#x got %#x: %w", h.crc, crc32.ChecksumIEEE(plain), errCodec)
	}
	return plain, nil
}
