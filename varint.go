package garotafitness

import "fmt"

// FreeArc packed integer (ArcStructure.h readInteger).
func readPacked(b []byte, i int) (v uint64, n int, err error) {
	if i >= len(b) {
		return 0, i, fmt.Errorf("packed: eof")
	}
	need := func(k int) error {
		if i+k > len(b) {
			return fmt.Errorf("packed: eof")
		}
		return nil
	}
	x := uint32(b[i])
	if i+1 < len(b) {
		x |= uint32(b[i+1]) << 8
	}
	if i+2 < len(b) {
		x |= uint32(b[i+2]) << 16
	}
	if i+3 < len(b) {
		x |= uint32(b[i+3]) << 24
	}
	switch {
	case x&1 == 0:
		return uint64(x&0xff) >> 1, i + 1, nil
	case x&3 == 1:
		if err := need(2); err != nil {
			return 0, i, err
		}
		return uint64(x&0xffff) >> 2, i + 2, nil
	case x&7 == 3:
		if err := need(3); err != nil {
			return 0, i, err
		}
		return uint64(x&0xffffff) >> 3, i + 3, nil
	case x&15 == 7:
		if err := need(4); err != nil {
			return 0, i, err
		}
		return uint64(x) >> 4, i + 4, nil
	}
	if err := need(8); err != nil {
		return 0, i, err
	}
	y := uint64(b[i]) | uint64(b[i+1])<<8 | uint64(b[i+2])<<16 | uint64(b[i+3])<<24 |
		uint64(b[i+4])<<32 | uint64(b[i+5])<<40 | uint64(b[i+6])<<48 | uint64(b[i+7])<<56
	switch {
	case x&31 == 15:
		return (y & ((1 << 40) - 1)) >> 5, i + 5, nil
	case x&63 == 31:
		return (y & ((1 << 48) - 1)) >> 6, i + 6, nil
	case x&127 == 63:
		return (y & ((1 << 56) - 1)) >> 7, i + 7, nil
	case x&255 == 127:
		return y >> 8, i + 8, nil
	}
	if err := need(9); err != nil {
		return 0, i, err
	}
	y = uint64(b[i+1]) | uint64(b[i+2])<<8 | uint64(b[i+3])<<16 | uint64(b[i+4])<<24 |
		uint64(b[i+5])<<32 | uint64(b[i+6])<<40 | uint64(b[i+7])<<48 | uint64(b[i+8])<<56
	return y, i + 9, nil
}

func readCString(b []byte, i int) (s string, n int, err error) {
	if i >= len(b) {
		return "", i, fmt.Errorf("string: eof")
	}
	for j := i; j < len(b); j++ {
		if b[j] == 0 {
			return string(b[i:j]), j + 1, nil
		}
	}
	return "", i, fmt.Errorf("string: no nul")
}

func readU32(b []byte, i int) (uint32, int, error) {
	if i+4 > len(b) {
		return 0, i, fmt.Errorf("u32: eof")
	}
	v := uint32(b[i]) | uint32(b[i+1])<<8 | uint32(b[i+2])<<16 | uint32(b[i+3])<<24
	return v, i + 4, nil
}
