package mp3rec

// Types and constants from packMP3 pmp3tbl.h / huffmp3.h.
// C int is int32, C char is int8, C short is int16.

const (
	mp3MaxMemorySize = 512 * 1024 * 1024
	appVersion       = 10

	mp3V1     = 3
	layerIII  = 1
	mp3Stereo = 0
	mp3Joint  = 1
	mp3Mono   = 3

	longBlock  = 0
	startBlock = 1
	shortBlock = 2
	stopBlock  = 3

	paddingMaxRun   = 255
	switchingMaxRun = 31
	ggContextSize   = 16
	auxDataStep     = 255
	stuffingStep    = 15

	minFrameBytes = 96 // MPEG-1 L3, 48 kHz, bitrate index 1
)

type huffCode struct {
	len  uint8
	code uint32
}

type bvTable struct {
	linbits int32
	rows    [][]huffCode
}

var scfBounds = [4]int{6, 11, 16, 21}
var scfWidth = [4]int{6, 5, 5, 5}
var scfLHBoundsShort = [2]int{6, 12}
var scfLHWidthShort = [2]int{6, 6}

type granule struct {
	n               int32
	share           int8
	mainDataBit     int16
	bigValPairs     int16
	globalGain      int16
	slength         int8
	windowSwitching int8
	preemphasis     int8
	coarse          int8
	selectHtabB     int8
	regionTable     [3]int8
	region0         int8
	region1         int8
	regionBound     [3]int16
	svBound         int16
	blockType       int8
	mixed           int8
	sbGain          [3]int8
	next            *granule
	prev            *granule
}

type mp3Frame struct {
	n            int32
	mpeg         int8
	layer        int8
	protection   int8
	bits         int8
	samples      int8
	padding      int8
	privbit      int8
	channels     int8
	stereoMS     int8
	stereoInt    int8
	copyright    int8
	original     int8
	emphasis     int8
	bitReservoir int16
	padbits      int8
	granules     [][]*granule
	frameSize    int32
	fixedSize    int32
	mainSize     int32
	auxSize      int32
	mainBits     int32
	nchannels    int8
	next         *mp3Frame
	prev         *mp3Frame
}
