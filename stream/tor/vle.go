package tor

// Length and distance slot tables from LZ77_Coder.cpp (extra_lbits, extra_lbits2, extra_dbits).
var (
	lcExtra  []uint32
	lcBase   []uint32
	lc2Extra []uint32
	lc2Base  []uint32
	dcExtra  []uint32
	dcBase   []uint32
)

func init() {
	lcExtra, lcBase = buildVLE([]uint32{0, 0, 0, 1, 2, 4, 8, 30})
	lc2Extra, lc2Base = buildVLE([]uint32{0, 0, 0, 0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 8, 30})
	dcExtra, dcBase = buildDist([]uint32{
		4, 4, 5, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11,
		11, 12, 12, 13, 13, 14, 14, 15, 15, 16, 17, 18, 19, 21, 23, 30,
	})
}

func buildVLE(bits []uint32) (extra, base []uint32) {
	extra = make([]uint32, len(bits))
	base = make([]uint32, len(bits))
	value := 0
	for code, eb := range bits {
		extra[code] = eb
		base[code] = uint32(value)
		n := 1 << eb
		for i := 0; i < n; i++ {
			if value >= vleSize {
				break
			}
			value++
		}
	}
	return extra, base
}

// buildDist follows DistanceCoder::DistanceCoder, including the xcode-size break
// that stops the last slot early.
func buildDist(bits []uint32) (extra, base []uint32) {
	extra = make([]uint32, len(bits))
	base = make([]uint32, len(bits))
	dist := 0
	code := 0
	for dist < 512 {
		extra[code] = bits[code]
		base[code] = uint32(dist)
		dist += 1 << bits[code]
		code++
	}
	dist >>= 8
	for dist < 512 {
		extra[code] = bits[code]
		base[code] = uint32(dist << 8)
		dist += 1 << (bits[code] - 8)
		code++
	}
	dist >>= 8
	for code < len(bits) {
		extra[code] = bits[code]
		base[code] = uint32(dist << 16)
		n := 1 << (bits[code] - 16)
		for i := 0; i < n; i++ {
			if 1024+dist >= vleSize {
				break
			}
			dist++
		}
		code++
	}
	return extra, base
}
