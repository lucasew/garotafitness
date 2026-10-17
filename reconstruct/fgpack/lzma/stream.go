package lzma

import (
	"encoding/binary"
	"runtime"
	"unsafe"
)

// Encode is the fgpack guest: config is 20 or 36 bytes of little-endian
// uint32 parameters. The result is 5 LZMA props, an 8-byte unpacked size,
// and the payload with no end marker. A nil slice means the guest returned 0.
func Encode(src, config []byte) []byte {
	if len(config) != 20 && len(config) != 36 {
		return nil
	}
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(config[off:]) }
	dictLog := u32(0)
	if dictLog < 12 || dictLog > 29 || len(src) > int(^uint32(0)) {
		return nil
	}
	var props CLzmaEncProps
	LzmaEncProps_Init(&props)
	props.DictSize = UInt32(1) << dictLog
	props.Fb = int32(u32(4))
	props.Lc = int32(u32(8))
	props.Lp = int32(u32(12))
	props.Pb = int32(u32(16))
	props.Algo = 1
	props.BtMode = 1
	props.NumHashBytes = 4
	props.NumThreads = 1
	if len(config) >= 32 {
		props.Algo = int32(u32(20))
		props.BtMode = int32(u32(24))
		props.NumHashBytes = int32(u32(28))
	}
	if len(config) >= 36 {
		props.ReduceSize = UInt64(len(src))
	}
	capn := len(src) + len(src)/3 + 65536
	if capn < 13 {
		capn = 13
	}
	// Tail padding keeps a C end pointer (base+len) inside the Go object.
	dst := make([]byte, capn+64)
	destLen := SizeT(capn - 13)
	propSize := SizeT(5)
	var dummy [64]Byte
	srcPtr := &dummy[0]
	srcKeep := src
	if len(src) > 0 {
		buf := make([]byte, len(src)+64)
		copy(buf, src)
		srcKeep = buf
		srcPtr = (*Byte)(unsafe.Pointer(&buf[0]))
	}
	res := LzmaEncode((*Byte)(unsafe.Pointer(&dst[13])), &destLen, srcPtr, SizeT(len(src)), &props, (*Byte)(unsafe.Pointer(&dst[0])), &propSize, 0, nil, AllocPtr(), AllocPtr())
	runtime.KeepAlive(srcKeep)
	runtime.KeepAlive(dst)
	if res != SZ_OK {
		return nil
	}
	sz := uint64(len(src))
	for i := 0; i < 8; i++ {
		dst[5+i] = byte(sz >> (8 * i))
	}
	return append([]byte(nil), dst[:13+int(destLen)]...)
}
