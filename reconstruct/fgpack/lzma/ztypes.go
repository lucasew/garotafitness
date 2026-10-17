package lzma

import "unsafe"

type Byte byte
type Int32 int32
type SRes int32
type BoolInt int32
type UInt64 uint64
type SizeT uint32
type Int64 int64

const (
	SZ_OK                SRes    = 0
	SZ_ERROR_DATA        SRes    = 1
	SZ_ERROR_MEM         SRes    = 2
	SZ_ERROR_CRC         SRes    = 3
	SZ_ERROR_UNSUPPORTED SRes    = 4
	SZ_ERROR_PARAM       SRes    = 5
	SZ_ERROR_INPUT_EOF   SRes    = 6
	SZ_ERROR_OUTPUT_EOF  SRes    = 7
	SZ_ERROR_READ        SRes    = 8
	SZ_ERROR_WRITE       SRes    = 9
	SZ_ERROR_PROGRESS    SRes    = 10
	SZ_ERROR_FAIL        SRes    = 11
	SZ_ERROR_THREAD      SRes    = 12
	True                 BoolInt = 1
	False                BoolInt = 0
)

type ISzAlloc struct {
	Alloc func(p ISzAllocPtr, size uint32) unsafe.Pointer
	Free  func(p ISzAllocPtr, address unsafe.Pointer)
}
type ISzAllocPtr = *ISzAlloc

type ISeqInStream struct {
	Read func(p *ISeqInStream, buf unsafe.Pointer, size *uint32) SRes
}
type ISeqOutStream struct {
	Write func(p *ISeqOutStream, buf unsafe.Pointer, size uint32) uint32
}
type ICompressProgress struct {
	Progress func(p *ICompressProgress, inSize, outSize UInt64) SRes
}

func cBoolToInt(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func cMemcpy(dst, src unsafe.Pointer, n int) {
	if n <= 0 {
		return
	}
	copy(unsafe.Slice((*byte)(dst), n), unsafe.Slice((*byte)(src), n))
}

func cMemmove(dst, src unsafe.Pointer, n int) {
	cMemcpy(dst, src, n)
}

func cMemset(p unsafe.Pointer, ch byte, n int) {
	b := unsafe.Slice((*byte)(p), n)
	for i := range b {
		b[i] = ch
	}
}

func cMalloc(sz int) unsafe.Pointer {
	if sz <= 0 {
		return nil
	}
	return MyAlloc(uint32(sz))
}
