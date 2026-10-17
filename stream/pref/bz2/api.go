package bz2

import (
	"fmt"
	"runtime"
)

// ioChunk is precomp's CHUNK in def_part_bzip2.
const ioChunk = 262144

// Compress matches precomp def_part_bzip2 for a whole buffer:
// BZ2_bzCompressInit(level, 0, 0), then 256KiB BZ_RUN slices and BZ_FINISH.
// The caller truncates the result to the length stored in the PCF.
func Compress(src []byte, level int) ([]byte, error) {
	if level < 1 || level > 9 {
		return nil, fmt.Errorf("bz2: level %d", level)
	}
	var strm bz_stream
	if ret := BZ2_bzCompressInit(&strm, int32(level), 0, 0); ret != BZ_OK {
		return nil, fmt.Errorf("bz2: init %d", ret)
	}
	defer BZ2_bzCompressEnd(&strm)
	var out []byte
	pos := 0
	var ret int32
	for {
		flush := int32(BZ_FINISH)
		n := len(src) - pos
		if n > ioChunk {
			n = ioChunk
			flush = BZ_RUN
		}
		if n > 0 {
			strm.Next_in = &src[pos]
			strm.Avail_in = uint32(n)
			pos += n
		} else {
			strm.Next_in = nil
			strm.Avail_in = 0
		}
		for {
			buf := make([]byte, ioChunk)
			strm.Next_out = &buf[0]
			strm.Avail_out = ioChunk
			ret = BZ2_bzCompress(&strm, flush)
			have := ioChunk - int(strm.Avail_out)
			if have > 0 {
				out = append(out, buf[:have]...)
			}
			if strm.Avail_out != 0 {
				break
			}
		}
		if flush == BZ_FINISH {
			break
		}
	}
	runtime.KeepAlive(src)
	if ret != BZ_STREAM_END {
		return nil, fmt.Errorf("bz2: compress %d", ret)
	}
	return out, nil
}
