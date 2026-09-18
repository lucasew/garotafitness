// Package sevenz writes the non-solid LZMA 7z archives used by the installer.
package sevenz

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"sort"

	"github.com/lucasew/garotafitness/reconstruct/fgpack"
)

// File is one member packed into the archive.
type File struct {
	Name string
	Data []byte
}

// Encode writes a 7z archive matching:
// 7z a -ms=off -mtc=off -mtm=off -mta=off -m0=lzma:x=4:d=512k
func Encode(ctx context.Context, files []File) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("sevenz: no files")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	var packed []byte
	packSizes := make([]uint64, len(files))
	crcs := make([]uint32, len(files))
	sizes := make([]uint64, len(files))
	for i, f := range files {
		enc, err := lzmaEncode(ctx, f.Data)
		if err != nil {
			return nil, err
		}
		packSizes[i] = uint64(len(enc))
		sizes[i] = uint64(len(f.Data))
		crcs[i] = crc32.ChecksumIEEE(f.Data)
		packed = append(packed, enc...)
	}
	header := encodeHeader(files, packSizes, sizes, crcs)
	var out bytes.Buffer
	out.Write([]byte{0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c, 0x00, 0x04})
	var start [20]byte
	binary.LittleEndian.PutUint64(start[0:8], uint64(len(packed)))
	binary.LittleEndian.PutUint64(start[8:16], uint64(len(header)))
	binary.LittleEndian.PutUint32(start[16:20], crc32.ChecksumIEEE(header))
	var crcField [4]byte
	binary.LittleEndian.PutUint32(crcField[:], crc32.ChecksumIEEE(start[:]))
	out.Write(crcField[:])
	out.Write(start[:])
	out.Write(packed)
	out.Write(header)
	return out.Bytes(), nil
}

func lzmaEncode(ctx context.Context, data []byte) ([]byte, error) {
	return fgpack.EncodeWithOptions(ctx, data, fgpack.Options{DictLog: 19, FastBytes: 32, LC: 3, LP: 0, PB: 2})
}

func encodeHeader(files []File, packSizes, unpackSizes []uint64, crcs []uint32) []byte {
	var b bytes.Buffer
	b.WriteByte(0x01) // kHeader
	b.WriteByte(0x04) // kMainStreamsInfo
	b.WriteByte(0x06) // kPackInfo
	writeNumber(&b, 0)
	writeNumber(&b, uint64(len(files)))
	b.WriteByte(0x09) // kSize
	for _, n := range packSizes {
		writeNumber(&b, n)
	}
	b.WriteByte(0x00) // kEnd pack
	b.WriteByte(0x07) // kUnpackInfo
	b.WriteByte(0x0b) // kFolder
	writeNumber(&b, uint64(len(files)))
	b.WriteByte(0) // external
	for range files {
		b.WriteByte(1)    // num coders
		b.WriteByte(0x21) // flags: idlen=1 + codec props
		b.WriteByte(0x21) // LZMA
		props := lzmaProps(3, 0, 2, 512*1024)
		writeNumber(&b, uint64(len(props)))
		b.Write(props)
	}
	b.WriteByte(0x0c) // kCodersUnpackSize
	for _, n := range unpackSizes {
		writeNumber(&b, n)
	}
	b.WriteByte(0x0a) // kCRC
	b.WriteByte(1)    // all defined
	for _, c := range crcs {
		binary.Write(&b, binary.LittleEndian, c)
	}
	b.WriteByte(0x00) // kEnd unpack
	b.WriteByte(0x08) // kSubStreamsInfo
	b.WriteByte(0x00) // kEnd substreans (one file per folder)
	b.WriteByte(0x00) // kEnd streams
	b.WriteByte(0x05) // kFilesInfo
	writeNumber(&b, uint64(len(files)))
	b.WriteByte(0x11) // kName
	var names bytes.Buffer
	for _, f := range files {
		for _, r := range f.Name {
			var u [2]byte
			binary.LittleEndian.PutUint16(u[:], uint16(r))
			names.Write(u[:])
		}
		names.Write([]byte{0, 0})
	}
	writeNumber(&b, uint64(names.Len()+1))
	b.WriteByte(0) // external
	b.Write(names.Bytes())
	b.WriteByte(0x00) // kEnd files
	b.WriteByte(0x00) // kEnd header
	return b.Bytes()
}

func lzmaProps(lc, lp, pb int, dict int) []byte {
	var p [5]byte
	p[0] = byte((pb*5+lp)*9 + lc)
	binary.LittleEndian.PutUint32(p[1:], uint32(dict))
	return p[:]
}

func writeNumber(b *bytes.Buffer, v uint64) {
	if v < 0x80 {
		b.WriteByte(byte(v))
		return
	}
	extra := 1
	limit := uint64(0x80)
	for extra < 8 && v >= limit {
		limit <<= 7
		extra++
	}
	buf := make([]byte, extra)
	for i := extra - 1; i > 0; i-- {
		buf[i] = byte(v)
		v >>= 8
	}
	buf[0] = byte(0xFF<<(8-extra)) | byte(v)
	b.Write(buf)
}
