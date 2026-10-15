package tor

// Sizes and codes from Tornado.cpp / EntropyCoder.cpp / LZ77_Coder.cpp.
// compress_all_at_once is 0, so input chunks are LARGE_BUFFER_SIZE and the
// output window flushes in HUGE_BUFFER_SIZE steps.
const (
	byteCoder = 1
	bitCoder  = 2
	hufCoder  = 3
	ariCoder  = 4

	largeBuf  = 256 * 1024
	hugeBuf   = 8 * 1024 * 1024
	maxElem   = 8
	padTables = 512

	// IMPOSSIBLE_LEN / IMPOSSIBLE_DIST = INT_MAX/2.
	impossible = uint32(1<<30 - 1)

	fastBits = 11
	maxHuf   = 2048
	// 256 + LEN_CODES*DIST_CODES, DIST_CODES = 32+REPDIST_CODES.
	eobCode      = 832
	repchar      = 833
	codes        = 842
	repdistCodes = 4
	lenCodes     = 16
	vleSize      = 1024 + 16384 + 1

	tableEntries = 10000
	maxTableRow  = 256
)
