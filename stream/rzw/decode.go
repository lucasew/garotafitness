package rzw

import (
	"fmt"
	"hash/crc32"
)

// Token order in the rz 1.00 decoder jump table at 0x42a000.
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

// DecodeStreams runs the 1.00 logical-stream pipeline on framed packets.
func DecodeStreams(streams [8][][]byte) ([]byte, error) {
	return (&archive{streams: streams}).decode()
}

// Decode each logical stream, restore the instruction and duplicate transforms,
// and check every stored file CRC before exposing any bytes to the caller.
func (a *archive) decode() ([]byte, error) {
	m, err := readMetadata(a.streams[0])
	if err != nil {
		return nil, err
	}
	size := uint64(0)
	for _, f := range m.files {
		if f.attributes&0x10 != 0 {
			if f.size != 0 {
				return nil, fmt.Errorf("rzw: nonempty directory")
			}
			continue
		}
		if size > maxPlain || f.size > maxPlain-size {
			return nil, errTooLarge
		}
		size += f.size
	}
	for _, s := range a.streams[5:] {
		if len(s) != 0 {
			return nil, fmt.Errorf("rzw: unexpected logical stream")
		}
	}
	records, n, err := readDuplicates(a.streams[1], m.window, int(size))
	if err != nil {
		return nil, err
	}
	operands, err := decodeFrames(a.streams[2], n)
	if err != nil {
		return nil, err
	}
	data, err := decodeFrames(a.streams[4], n)
	if err != nil {
		return nil, err
	}
	data, err = restoreInstructions(a.streams[3], data, operands, n)
	if err != nil {
		return nil, err
	}
	data, err = restoreDuplicates(data, records, int(size))
	if err != nil {
		return nil, err
	}
	off := 0
	for _, f := range m.files {
		if f.attributes&0x10 != 0 {
			continue
		}
		end := off + int(f.size)
		if got := crc32.ChecksumIEEE(data[off:end]); got != f.crc {
			return nil, fmt.Errorf("rzw: %s CRC %08x, want %08x", f.name, got, f.crc)
		}
		off = end
	}
	return data, nil
}
