package lzma

import (
	"unsafe"
)

const kBlockMoveAlign = 128
const kBlockSizeAlign = 0x10000
const kBlockSizeReserveMin = 0x1000000
const kEmptyHashValue = 0
const kCrcPoly = 0xEDB88320
const kLzHash_CrcShift_1 = 5
const kLzHash_CrcShift_2 = 10
const kFix5HashSize = 0x10400
const NUM_REFS_ALIGN_MASK = 0xF
const CYC_TO_POS_OFFSET = 0
const LZFIND_NORM_ALIGN_BLOCK_SIZE = 128

type UInt32 uint32
type CLzRef UInt32
type CMatchFinder struct {
	Buffer              *Byte
	Pos                 UInt32
	PosLimit            UInt32
	StreamPos           UInt32
	LenLimit            UInt32
	CyclicBufferPos     UInt32
	CyclicBufferSize    UInt32
	StreamEndWasReached Byte
	BtMode              Byte
	BigHash             Byte
	DirectInput         Byte
	MatchMaxLen         UInt32
	Hash                *CLzRef
	Son                 *CLzRef
	HashMask            UInt32
	CutValue            UInt32
	BufBase             *Byte
	Stream              ISeqInStreamPtr
	BlockSize           UInt32
	KeepSizeBefore      UInt32
	KeepSizeAfter       UInt32
	NumHashBytes        UInt32
	DirectInputRem      uint32
	HistorySize         UInt32
	FixedHashSize       UInt32
	NumHashBytes_Min    Byte
	NumHashOutBits      Byte
	_pad2_              [2]Byte
	Result              SRes
	Crc                 [256]UInt32
	NumRefs             uint32
	ExpectedDataSize    UInt64
}
type Mf_Init_Func func(object unsafe.Pointer)
type Mf_GetNumAvailableBytes_Func func(object unsafe.Pointer) UInt32
type Mf_GetPointerToCurrentPos_Func func(object unsafe.Pointer) *Byte
type Mf_GetMatches_Func func(object unsafe.Pointer, distances *UInt32) *UInt32
type Mf_Skip_Func func(object unsafe.Pointer, num UInt32)
type IMatchFinder2 struct {
	Init                   Mf_Init_Func
	GetNumAvailableBytes   Mf_GetNumAvailableBytes_Func
	GetPointerToCurrentPos Mf_GetPointerToCurrentPos_Func
	GetMatches             Mf_GetMatches_Func
	Skip                   Mf_Skip_Func
}

func LzInWindow_Free(p *CMatchFinder, alloc ISzAllocPtr) {
	alloc.Free(alloc, unsafe.Pointer(p.BufBase))
	p.BufBase = nil
}
func LzInWindow_Create2(p *CMatchFinder, blockSize UInt32, alloc ISzAllocPtr) int32 {
	if blockSize == 0 {
		return 0
	}
	if p.BufBase == nil || p.BlockSize != blockSize {
		LzInWindow_Free(p, alloc)
		p.BlockSize = blockSize
		p.BufBase = (*Byte)(alloc.Alloc(alloc, uint32(blockSize)))
	}
	return cBoolToInt(p.BufBase != nil)
}
func MatchFinder_GetPointerToCurrentPos(p unsafe.Pointer) *Byte {
	return ((*CMatchFinder)(p)).Buffer
}
func MatchFinder_GetNumAvailableBytes(p unsafe.Pointer) UInt32 {
	return ((*CMatchFinder)(p)).StreamPos - ((*CMatchFinder)(p)).Pos
}
func MatchFinder_ReadBlock(p *CMatchFinder) {
	if p.StreamEndWasReached != 0 || p.Result != SZ_OK {
		return
	}
	if p.DirectInput != 0 {
		var curSize UInt32 = 0xFFFFFFFF - (p.StreamPos - p.Pos)
		if curSize > UInt32(p.DirectInputRem) {
			curSize = UInt32(p.DirectInputRem)
		}
		p.StreamPos += curSize
		p.DirectInputRem -= uint32(curSize)
		if p.DirectInputRem == 0 {
			p.StreamEndWasReached = 1
		}
		return
	}
	for {
		var (
			dest *Byte  = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), p.StreamPos-p.Pos))
			size uint32 = uint32(int32(uintptr(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(p.BufBase), p.BlockSize)))) - uintptr(unsafe.Pointer(dest))))
		)
		if size == 0 {
			return
		}
		p.Result = p.Stream.Read(p.Stream, unsafe.Add(unsafe.Pointer(p.BufBase), int32(uintptr(unsafe.Pointer(dest))-uintptr(unsafe.Pointer(p.BufBase)))), &size)
		if p.Result != SZ_OK {
			return
		}
		if size == 0 {
			p.StreamEndWasReached = 1
			return
		}
		p.StreamPos += UInt32(size)
		if (p.StreamPos - p.Pos) > p.KeepSizeAfter {
			return
		}
	}
}
func MatchFinder_MoveBlock(p *CMatchFinder) {
	var (
		offset     uint32 = uint32(int32(uintptr(unsafe.Pointer(p.Buffer))-uintptr(unsafe.Pointer(p.BufBase)))) - uint32(p.KeepSizeBefore)
		keepBefore uint32 = (offset & ((1 << 7) - 1)) + uint32(p.KeepSizeBefore)
	)
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.BufBase), keepBefore))
	cMemmove(unsafe.Pointer(p.BufBase), unsafe.Add(unsafe.Pointer(p.BufBase), offset & ^uint32((1<<7)-1)), int(keepBefore+uint32(p.StreamPos-p.Pos)))
}
func MatchFinder_NeedMove(p *CMatchFinder) int32 {
	if p.DirectInput != 0 {
		return 0
	}
	if p.StreamEndWasReached != 0 || p.Result != SZ_OK {
		return 0
	}
	return cBoolToInt(uint32(int32(uintptr(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(p.BufBase), p.BlockSize))))-uintptr(unsafe.Pointer(p.Buffer)))) <= uint32(p.KeepSizeAfter))
}
func MatchFinder_ReadIfRequired(p *CMatchFinder) {
	if p.KeepSizeAfter >= (p.StreamPos - p.Pos) {
		MatchFinder_ReadBlock(p)
	}
}
func MatchFinder_SetDefaultSettings(p *CMatchFinder) {
	p.CutValue = 32
	p.BtMode = 1
	p.NumHashBytes = 4
	p.NumHashBytes_Min = 2
	p.NumHashOutBits = 0
	p.BigHash = 0
}
func MatchFinder_Construct(p *CMatchFinder) {
	var i uint32
	p.Buffer = nil
	p.BufBase = nil
	p.DirectInput = 0
	p.Stream = nil
	p.Hash = nil
	p.ExpectedDataSize = 18446744073709551615
	MatchFinder_SetDefaultSettings(p)
	for i = 0; i < 256; i++ {
		var (
			r UInt32 = UInt32(i)
			j uint32
		)
		for j = 0; j < 8; j++ {
			r = (r >> 1) ^ (kCrcPoly & (UInt32(0) - (r & 1)))
		}
		p.Crc[i] = r
	}
}
func MatchFinder_FreeThisClassMemory(p *CMatchFinder, alloc ISzAllocPtr) {
	alloc.Free(alloc, unsafe.Pointer(p.Hash))
	p.Hash = nil
}
func MatchFinder_Free(p *CMatchFinder, alloc ISzAllocPtr) {
	MatchFinder_FreeThisClassMemory(p, alloc)
	LzInWindow_Free(p, alloc)
	unpinAll(unsafe.Pointer(p))
}
func AllocRefs(num uint32, alloc ISzAllocPtr) *CLzRef {
	var sizeInBytes uint32 = num * uint32(unsafe.Sizeof(CLzRef(0)))
	if sizeInBytes/uint32(unsafe.Sizeof(CLzRef(0))) != num {
		return nil
	}
	return (*CLzRef)(alloc.Alloc(alloc, sizeInBytes))
}
func GetBlockSize(p *CMatchFinder, historySize UInt32) UInt32 {
	var blockSize UInt32 = (p.KeepSizeBefore + p.KeepSizeAfter)
	if p.KeepSizeBefore < historySize || blockSize < p.KeepSizeBefore {
		return 0
	}
	{
		var (
			kBlockSizeMax UInt32 = ^UInt32((1 << 16) - 1)
			rem           UInt32 = kBlockSizeMax - blockSize
			reserve       UInt32 = (blockSize >> UInt32(func() int32 {
				if blockSize < (1 << 30) {
					return 1
				}
				return 2
			}())) + (1 << 12) + (1 << 7) + (1 << 16)
		)
		if blockSize >= kBlockSizeMax || rem < (1<<24) {
			return 0
		}
		if reserve >= rem {
			blockSize = kBlockSizeMax
		} else {
			blockSize += reserve
			blockSize &= ^UInt32((1 << 16) - 1)
		}
	}
	return blockSize
}
func MatchFinder_GetHashMask2(p *CMatchFinder, hs UInt32) UInt32 {
	if p.NumHashBytes == 2 {
		return (1 << 16) - 1
	}
	if hs != 0 {
		hs--
	}
	hs |= hs >> 1
	hs |= hs >> 2
	hs |= hs >> 4
	hs |= hs >> 8
	if hs >= (1 << 24) {
		if p.NumHashBytes == 3 {
			hs = (1 << 24) - 1
		}
	}
	hs |= (1 << 16) - 1
	if p.NumHashBytes >= 5 {
		hs |= UInt32((int32(256 << kLzHash_CrcShift_2)) - 1)
	}
	return hs
}
func MatchFinder_GetHashMask(p *CMatchFinder, hs UInt32) UInt32 {
	if p.NumHashBytes == 2 {
		return (1 << 16) - 1
	}
	if hs != 0 {
		hs--
	}
	hs |= hs >> 1
	hs |= hs >> 2
	hs |= hs >> 4
	hs |= hs >> 8
	hs >>= 1
	if hs >= (1 << 24) {
		if p.NumHashBytes == 3 {
			hs = (1 << 24) - 1
		} else {
			hs >>= 1
		}
	}
	hs |= (1 << 16) - 1
	if p.NumHashBytes >= 5 {
		hs |= UInt32((int32(256 << kLzHash_CrcShift_2)) - 1)
	}
	return hs
}
func MatchFinder_Create(p *CMatchFinder, historySize UInt32, keepAddBufferBefore UInt32, matchMaxLen UInt32, keepAddBufferAfter UInt32, alloc ISzAllocPtr) int32 {
	p.KeepSizeBefore = historySize + keepAddBufferBefore + 1
	keepAddBufferAfter += matchMaxLen
	if keepAddBufferAfter < p.NumHashBytes {
		keepAddBufferAfter = p.NumHashBytes
	}
	p.KeepSizeAfter = keepAddBufferAfter
	if p.DirectInput != 0 {
		p.BlockSize = 0
	}
	if p.DirectInput != 0 || LzInWindow_Create2(p, GetBlockSize(p, historySize), alloc) != 0 {
		var hashSizeSum uint32
		{
			var (
				hs    UInt32
				hsCur UInt32
			)
			if p.NumHashOutBits != 0 {
				var (
					numBits uint32 = uint32(p.NumHashOutBits)
					nbMax   uint32 = uint32(func() int32 {
						if p.NumHashBytes == 2 {
							return 16
						}
						if p.NumHashBytes == 3 {
							return 24
						}
						return 32
					}())
				)
				if numBits >= nbMax {
					numBits = nbMax
				}
				if numBits >= 32 {
					hs = ^UInt32(0)
				} else {
					hs = UInt32((1 << numBits) - 1)
				}
				hs |= (1 << 16) - 1
				if p.NumHashBytes >= 5 {
					hs |= UInt32((int32(256 << kLzHash_CrcShift_2)) - 1)
				}
				{
					var hs2 UInt32 = MatchFinder_GetHashMask2(p, historySize)
					if hs >= hs2 {
						hs = hs2
					}
				}
				hsCur = hs
				if p.ExpectedDataSize < UInt64(historySize) {
					var hs2 UInt32 = MatchFinder_GetHashMask2(p, UInt32(p.ExpectedDataSize))
					if hsCur >= hs2 {
						hsCur = hs2
					}
				}
			} else {
				hs = MatchFinder_GetHashMask(p, historySize)
				hsCur = hs
				if p.ExpectedDataSize < UInt64(historySize) {
					hsCur = MatchFinder_GetHashMask(p, UInt32(p.ExpectedDataSize))
					if hsCur >= hs {
						hsCur = hs
					}
				}
			}
			p.HashMask = hsCur
			hashSizeSum = uint32(hs)
			hashSizeSum++
			if hashSizeSum < uint32(hs) {
				return 0
			}
			{
				var fixedHashSize UInt32 = 0
				if p.NumHashBytes > 2 && p.NumHashBytes_Min <= 2 {
					fixedHashSize += 1 << 10
				}
				if p.NumHashBytes > 3 && p.NumHashBytes_Min <= 3 {
					fixedHashSize += 1 << 16
				}
				hashSizeSum += uint32(fixedHashSize)
				p.FixedHashSize = fixedHashSize
			}
		}
		p.MatchMaxLen = matchMaxLen
		{
			var (
				newSize             uint32
				numSons             uint32
				newCyclicBufferSize UInt32 = historySize + 1
			)
			p.HistorySize = historySize
			p.CyclicBufferSize = newCyclicBufferSize
			numSons = uint32(newCyclicBufferSize)
			if p.BtMode != 0 {
				numSons <<= 1
			}
			newSize = hashSizeSum + numSons
			if numSons < uint32(newCyclicBufferSize) || newSize < numSons {
				return 0
			}
			newSize = (newSize + NUM_REFS_ALIGN_MASK) & ^uint32(NUM_REFS_ALIGN_MASK)
			if p.Hash != nil && p.NumRefs >= newSize {
				return 1
			}
			MatchFinder_FreeThisClassMemory(p, alloc)
			p.NumRefs = newSize
			p.Hash = AllocRefs(newSize, alloc)
			if p.Hash != nil {
				p.Son = (*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hashSizeSum)))
				return 1
			}
		}
	}
	MatchFinder_Free(p, alloc)
	return 0
}
func MatchFinder_SetLimits(p *CMatchFinder) {
	var (
		k UInt32
		n UInt32 = 0 - p.Pos
	)
	if n == 0 {
		n = 4294967295
	}
	k = p.CyclicBufferSize - p.CyclicBufferPos
	if k < n {
		n = k
	}
	k = p.StreamPos - p.Pos
	{
		var (
			ksa UInt32 = p.KeepSizeAfter
			mm  UInt32 = p.MatchMaxLen
		)
		if k > ksa {
			k -= ksa
		} else if k >= mm {
			k -= mm
			k++
		} else {
			mm = k
			if k != 0 {
				k = 1
			}
		}
		p.LenLimit = mm
	}
	if k < n {
		n = k
	}
	p.PosLimit = p.Pos + n
}
func MatchFinder_Init_LowHash(p *CMatchFinder) {
	var (
		i        uint32
		items    *CLzRef = p.Hash
		numItems uint32  = uint32(p.FixedHashSize)
	)
	for i = 0; i < numItems; i++ {
		*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*uintptr(i))) = kEmptyHashValue
	}
}
func MatchFinder_Init_HighHash(p *CMatchFinder) {
	var (
		i        uint32
		items    *CLzRef = (*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(p.FixedHashSize)))
		numItems uint32  = uint32(p.HashMask) + 1
	)
	for i = 0; i < numItems; i++ {
		*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*uintptr(i))) = kEmptyHashValue
	}
}
func MatchFinder_Init_4(p *CMatchFinder) {
	if p.DirectInput == 0 {
		p.Buffer = p.BufBase
	}
	{
		p.Pos = func() UInt32 {
			p_ := &p.StreamPos
			*p_ = 1
			return *p_
		}()
	}
	p.Result = SZ_OK
	p.StreamEndWasReached = 0
}
func MatchFinder_Init(_p unsafe.Pointer) {
	var p *CMatchFinder = (*CMatchFinder)(_p)
	MatchFinder_Init_HighHash(p)
	MatchFinder_Init_LowHash(p)
	MatchFinder_Init_4(p)
	MatchFinder_ReadBlock(p)
	p.CyclicBufferPos = p.Pos - CYC_TO_POS_OFFSET
	MatchFinder_SetLimits(p)
}
func LzFind_SaturSub_32(subValue UInt32, items *CLzRef, lim *CLzRef) {
	for {
		{
			var v UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)))
			if v < subValue {
				v = subValue
			}
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)) = CLzRef(v - subValue)
		}
		{
			var v UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*1)))
			if v < subValue {
				v = subValue
			}
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*1)) = CLzRef(v - subValue)
		}
		items = (*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*2))
		{
			var v UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)))
			if v < subValue {
				v = subValue
			}
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)) = CLzRef(v - subValue)
		}
		{
			var v UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*1)))
			if v < subValue {
				v = subValue
			}
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*1)) = CLzRef(v - subValue)
		}
		items = (*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*2))
		{
			var v UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)))
			if v < subValue {
				v = subValue
			}
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)) = CLzRef(v - subValue)
		}
		{
			var v UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*1)))
			if v < subValue {
				v = subValue
			}
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*1)) = CLzRef(v - subValue)
		}
		items = (*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*2))
		{
			var v UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)))
			if v < subValue {
				v = subValue
			}
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)) = CLzRef(v - subValue)
		}
		{
			var v UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*1)))
			if v < subValue {
				v = subValue
			}
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*1)) = CLzRef(v - subValue)
		}
		items = (*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*2))
		if items == lim {
			break
		}
	}
}
func MatchFinder_Normalize3(subValue UInt32, items *CLzRef, numItems uint32) {
	for ; numItems != 0 && (uint32(int32(uintptr(unsafe.Pointer(items))))&((1<<7)-1)) != 0; numItems-- {
		{
			var v UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)))
			if v < subValue {
				v = subValue
			}
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)) = CLzRef(v - subValue)
		}
		items = (*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*1))
	}
	{
		var (
			k_Align_Mask uint32  = ((1<<7)/4 - 1)
			lim          *CLzRef = (*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*uintptr(numItems & ^k_Align_Mask)))
		)
		numItems &= k_Align_Mask
		if items != lim {
			LzFind_SaturSub_32(subValue, items, lim)
		}
		items = lim
	}
	for ; numItems != 0; numItems-- {
		{
			var v UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)))
			if v < subValue {
				v = subValue
			}
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*0)) = CLzRef(v - subValue)
		}
		items = (*CLzRef)(unsafe.Add(unsafe.Pointer(items), unsafe.Sizeof(CLzRef(0))*1))
	}
}
func MatchFinder_CheckLimits(p *CMatchFinder) {
	if p.KeepSizeAfter == (p.StreamPos - p.Pos) {
		if MatchFinder_NeedMove(p) != 0 {
			MatchFinder_MoveBlock(p)
		}
		MatchFinder_ReadBlock(p)
	}
	if p.Pos == 0 {
		if (p.StreamPos - p.Pos) >= p.NumHashBytes {
			var subValue UInt32 = (p.Pos - p.HistorySize - 1)
			p.Pos -= subValue
			p.StreamPos -= subValue
			MatchFinder_Normalize3(subValue, p.Hash, uint32(p.HashMask)+1+uint32(p.FixedHashSize))
			{
				var numSonRefs uint32 = uint32(p.CyclicBufferSize)
				if p.BtMode != 0 {
					numSonRefs <<= 1
				}
				MatchFinder_Normalize3(subValue, p.Son, numSonRefs)
			}
		}
	}
	if p.CyclicBufferPos == p.CyclicBufferSize {
		p.CyclicBufferPos = 0
	}
	MatchFinder_SetLimits(p)
}
func Hc_GetMatchesSpec(lenLimit uint32, curMatch UInt32, pos UInt32, cur *Byte, son *CLzRef, _cyclicBufferPos uint32, _cyclicBufferSize UInt32, cutValue UInt32, d *UInt32, maxLen uint32) *UInt32 {
	var lim *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), lenLimit))
	*(*CLzRef)(unsafe.Add(unsafe.Pointer(son), unsafe.Sizeof(CLzRef(0))*uintptr(_cyclicBufferPos))) = CLzRef(curMatch)
	for {
		{
			var delta UInt32
			if curMatch == 0 {
				break
			}
			delta = pos - curMatch
			if delta >= _cyclicBufferSize {
				break
			}
			{
				var diff int32
				curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(son), unsafe.Sizeof(CLzRef(0))*uintptr(_cyclicBufferPos-uint32(delta)+uint32(func() UInt32 {
					if _cyclicBufferPos < uint32(delta) {
						return _cyclicBufferSize
					}
					return 0
				}())))))
				diff = 0 - int32(delta)
				if *(*Byte)(unsafe.Add(unsafe.Pointer(cur), maxLen)) == *(*Byte)(unsafe.Add(unsafe.Pointer(cur), int32(maxLen)+diff)) {
					var c *Byte = cur
					for *c == *(*Byte)(unsafe.Add(unsafe.Pointer(c), diff)) {
						if func() *Byte {
							p_ := &c
							*p_ = (*Byte)(unsafe.Add(unsafe.Pointer(*p_), 1))
							return *p_
						}() == lim {
							*(*UInt32)(unsafe.Add(unsafe.Pointer(d), unsafe.Sizeof(UInt32(0))*0)) = UInt32(int32(uintptr(unsafe.Pointer(lim)) - uintptr(unsafe.Pointer(cur))))
							*(*UInt32)(unsafe.Add(unsafe.Pointer(d), unsafe.Sizeof(UInt32(0))*1)) = delta - 1
							return (*UInt32)(unsafe.Add(unsafe.Pointer(d), unsafe.Sizeof(UInt32(0))*2))
						}
					}
					{
						var len_ uint32 = uint32(int32(uintptr(unsafe.Pointer(c)) - uintptr(unsafe.Pointer(cur))))
						if maxLen < len_ {
							maxLen = len_
							*(*UInt32)(unsafe.Add(unsafe.Pointer(d), unsafe.Sizeof(UInt32(0))*0)) = UInt32(len_)
							*(*UInt32)(unsafe.Add(unsafe.Pointer(d), unsafe.Sizeof(UInt32(0))*1)) = delta - 1
							d = (*UInt32)(unsafe.Add(unsafe.Pointer(d), unsafe.Sizeof(UInt32(0))*2))
						}
					}
				}
			}
		}
		if func() UInt32 {
			p_ := &cutValue
			*p_--
			return *p_
		}() == 0 {
			break
		}
	}
	return d
}
func GetMatchesSpec1(lenLimit UInt32, curMatch UInt32, pos UInt32, cur *Byte, son *CLzRef, _cyclicBufferPos uint32, _cyclicBufferSize UInt32, cutValue UInt32, d *UInt32, maxLen UInt32) *UInt32 {
	var ptr0 *CLzRef = (*CLzRef)(unsafe.Add(unsafe.Pointer((*CLzRef)(unsafe.Add(unsafe.Pointer(son), unsafe.Sizeof(CLzRef(0))*uintptr(_cyclicBufferPos<<1)))), unsafe.Sizeof(CLzRef(0))*1))
	_ = ptr0
	var ptr1 *CLzRef = (*CLzRef)(unsafe.Add(unsafe.Pointer(son), unsafe.Sizeof(CLzRef(0))*uintptr(_cyclicBufferPos<<1)))
	var len0 uint32 = 0
	var len1 uint32 = 0
	var cmCheck UInt32
	cmCheck = pos - _cyclicBufferSize
	if pos < _cyclicBufferSize {
		cmCheck = 0
	}
	if cmCheck < curMatch {
		for {
			{
				var delta UInt32 = pos - curMatch
				{
					var (
						pair *CLzRef = (*CLzRef)(unsafe.Add(unsafe.Pointer(son), unsafe.Sizeof(CLzRef(0))*uintptr((_cyclicBufferPos-uint32(delta)+uint32(func() UInt32 {
							if _cyclicBufferPos < uint32(delta) {
								return _cyclicBufferSize
							}
							return 0
						}()))<<1)))
						pb   *Byte  = (*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(delta)))
						len_ uint32 = (func() uint32 {
							if len0 < len1 {
								return len0
							}
							return len1
						}())
						pair0 UInt32 = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(pair), unsafe.Sizeof(CLzRef(0))*0)))
					)
					if *(*Byte)(unsafe.Add(unsafe.Pointer(pb), len_)) == *(*Byte)(unsafe.Add(unsafe.Pointer(cur), len_)) {
						if func() uint32 {
							p_ := &len_
							*p_++
							return *p_
						}() != uint32(lenLimit) && *(*Byte)(unsafe.Add(unsafe.Pointer(pb), len_)) == *(*Byte)(unsafe.Add(unsafe.Pointer(cur), len_)) {
							for func() uint32 {
								p_ := &len_
								*p_++
								return *p_
							}() != uint32(lenLimit) {
								if *(*Byte)(unsafe.Add(unsafe.Pointer(pb), len_)) != *(*Byte)(unsafe.Add(unsafe.Pointer(cur), len_)) {
									break
								}
							}
						}
						if maxLen < UInt32(len_) {
							maxLen = UInt32(len_)
							*func() *UInt32 {
								p_ := &d
								x := *p_
								*p_ = (*UInt32)(unsafe.Add(unsafe.Pointer(*p_), unsafe.Sizeof(UInt32(0))*1))
								return x
							}() = UInt32(len_)
							*func() *UInt32 {
								p_ := &d
								x := *p_
								*p_ = (*UInt32)(unsafe.Add(unsafe.Pointer(*p_), unsafe.Sizeof(UInt32(0))*1))
								return x
							}() = delta - 1
							if len_ == uint32(lenLimit) {
								*ptr1 = CLzRef(pair0)
								*ptr0 = *(*CLzRef)(unsafe.Add(unsafe.Pointer(pair), unsafe.Sizeof(CLzRef(0))*1))
								return d
							}
						}
					}
					if *(*Byte)(unsafe.Add(unsafe.Pointer(pb), len_)) < *(*Byte)(unsafe.Add(unsafe.Pointer(cur), len_)) {
						*ptr1 = CLzRef(curMatch)
						curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(pair), unsafe.Sizeof(CLzRef(0))*1)))
						ptr1 = (*CLzRef)(unsafe.Add(unsafe.Pointer(pair), unsafe.Sizeof(CLzRef(0))*1))
						len1 = len_
					} else {
						*ptr0 = CLzRef(curMatch)
						curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(pair), unsafe.Sizeof(CLzRef(0))*0)))
						ptr0 = pair
						len0 = len_
					}
				}
			}
			if func() UInt32 {
				p_ := &cutValue
				*p_--
				return *p_
			}() == 0 || cmCheck >= curMatch {
				break
			}
		}
	}
	*ptr0 = func() CLzRef {
		p_ := ptr1
		*p_ = kEmptyHashValue
		return *p_
	}()
	return d
}
func SkipMatchesSpec(lenLimit UInt32, curMatch UInt32, pos UInt32, cur *Byte, son *CLzRef, _cyclicBufferPos uint32, _cyclicBufferSize UInt32, cutValue UInt32) {
	var ptr0 *CLzRef = (*CLzRef)(unsafe.Add(unsafe.Pointer((*CLzRef)(unsafe.Add(unsafe.Pointer(son), unsafe.Sizeof(CLzRef(0))*uintptr(_cyclicBufferPos<<1)))), unsafe.Sizeof(CLzRef(0))*1))
	_ = ptr0
	var ptr1 *CLzRef = (*CLzRef)(unsafe.Add(unsafe.Pointer(son), unsafe.Sizeof(CLzRef(0))*uintptr(_cyclicBufferPos<<1)))
	var len0 uint32 = 0
	var len1 uint32 = 0
	var cmCheck UInt32
	cmCheck = pos - _cyclicBufferSize
	if pos < _cyclicBufferSize {
		cmCheck = 0
	}
	if cmCheck < curMatch {
		for {
			{
				var delta UInt32 = pos - curMatch
				{
					var (
						pair *CLzRef = (*CLzRef)(unsafe.Add(unsafe.Pointer(son), unsafe.Sizeof(CLzRef(0))*uintptr((_cyclicBufferPos-uint32(delta)+uint32(func() UInt32 {
							if _cyclicBufferPos < uint32(delta) {
								return _cyclicBufferSize
							}
							return 0
						}()))<<1)))
						pb   *Byte  = (*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(delta)))
						len_ uint32 = (func() uint32 {
							if len0 < len1 {
								return len0
							}
							return len1
						}())
					)
					if *(*Byte)(unsafe.Add(unsafe.Pointer(pb), len_)) == *(*Byte)(unsafe.Add(unsafe.Pointer(cur), len_)) {
						for func() uint32 {
							p_ := &len_
							*p_++
							return *p_
						}() != uint32(lenLimit) {
							if *(*Byte)(unsafe.Add(unsafe.Pointer(pb), len_)) != *(*Byte)(unsafe.Add(unsafe.Pointer(cur), len_)) {
								break
							}
						}
						if len_ == uint32(lenLimit) {
							*ptr1 = *(*CLzRef)(unsafe.Add(unsafe.Pointer(pair), unsafe.Sizeof(CLzRef(0))*0))
							*ptr0 = *(*CLzRef)(unsafe.Add(unsafe.Pointer(pair), unsafe.Sizeof(CLzRef(0))*1))
							return
						}
					}
					if *(*Byte)(unsafe.Add(unsafe.Pointer(pb), len_)) < *(*Byte)(unsafe.Add(unsafe.Pointer(cur), len_)) {
						*ptr1 = CLzRef(curMatch)
						curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(pair), unsafe.Sizeof(CLzRef(0))*1)))
						ptr1 = (*CLzRef)(unsafe.Add(unsafe.Pointer(pair), unsafe.Sizeof(CLzRef(0))*1))
						len1 = len_
					} else {
						*ptr0 = CLzRef(curMatch)
						curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(pair), unsafe.Sizeof(CLzRef(0))*0)))
						ptr0 = pair
						len0 = len_
					}
				}
			}
			if func() UInt32 {
				p_ := &cutValue
				*p_--
				return *p_
			}() == 0 || cmCheck >= curMatch {
				break
			}
		}
	}
	*ptr0 = func() CLzRef {
		p_ := ptr1
		*p_ = kEmptyHashValue
		return *p_
	}()
	return
}
func MatchFinder_MovePos(p *CMatchFinder) {
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	{
		var pos1 UInt32 = p.Pos + 1
		p.Pos = pos1
		if pos1 == p.PosLimit {
			MatchFinder_CheckLimits(p)
		}
	}
}
func Bt2_MatchFinder_GetMatches(_p unsafe.Pointer, distances *UInt32) *UInt32 {
	var (
		p        *CMatchFinder = (*CMatchFinder)(_p)
		hv       UInt32
		cur      *Byte
		curMatch UInt32
		lenLimit UInt32 = p.LenLimit
	)
	if lenLimit < 2 {
		MatchFinder_MovePos(p)
		return distances
	}
	cur = p.Buffer
	hv = UInt32(UInt16(*cur) | UInt16(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))<<8)
	curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
	*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
	distances = GetMatchesSpec1(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue, distances, 1)
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	{
		var pos1 UInt32 = p.Pos + 1
		p.Pos = pos1
		if pos1 == p.PosLimit {
			MatchFinder_CheckLimits(p)
		}
	}
	return distances
}
func Bt3Zip_MatchFinder_GetMatches(p *CMatchFinder, distances *UInt32) *UInt32 {
	var (
		hv       UInt32
		cur      *Byte
		curMatch UInt32
		lenLimit UInt32 = p.LenLimit
	)
	if lenLimit < 3 {
		MatchFinder_MovePos(p)
		return distances
	}
	cur = p.Buffer
	hv = ((UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) | UInt32(*cur)<<8) ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))]) & 0xFFFF
	curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
	*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
	distances = GetMatchesSpec1(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue, distances, 2)
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	{
		var pos1 UInt32 = p.Pos + 1
		p.Pos = pos1
		if pos1 == p.PosLimit {
			MatchFinder_CheckLimits(p)
		}
	}
	return distances
}
func Bt3_MatchFinder_GetMatches(_p unsafe.Pointer, distances *UInt32) *UInt32 {
	var (
		p        *CMatchFinder = (*CMatchFinder)(_p)
		mmm      UInt32
		h2       UInt32
		d2       UInt32
		pos      UInt32
		maxLen   uint32
		hash     *UInt32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
		lenLimit UInt32 = p.LenLimit
	)
	if lenLimit < 3 {
		MatchFinder_MovePos(p)
		return distances
	}
	cur = p.Buffer
	{
		var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
		h2 = temp & ((1 << 10) - 1)
		hv = (temp ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2)))<<8) & p.HashMask
	}
	hash = (*UInt32)(unsafe.Pointer(p.Hash))
	pos = p.Pos
	d2 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2)))
	curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
	*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(hv))) = pos
	mmm = p.CyclicBufferSize
	if pos < mmm {
		mmm = pos
	}
	maxLen = 2
	if d2 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))) == *cur {
		{
			var (
				diff int32 = 0 - int32(d2)
				c    *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), maxLen))
				lim  *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), lenLimit))
			)
			for ; c != lim; c = (*Byte)(unsafe.Add(unsafe.Pointer(c), 1)) {
				if *((*Byte)(unsafe.Add(unsafe.Pointer(c), diff))) != *c {
					break
				}
			}
			maxLen = uint32(int32(uintptr(unsafe.Pointer(c)) - uintptr(unsafe.Pointer(cur))))
		}
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*0)) = UInt32(maxLen)
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d2 - 1
		distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
		if maxLen == uint32(lenLimit) {
			SkipMatchesSpec(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			{
				var pos1 UInt32 = p.Pos + 1
				p.Pos = pos1
				if pos1 == p.PosLimit {
					MatchFinder_CheckLimits(p)
				}
			}
			return distances
		}
	}
	distances = GetMatchesSpec1(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue, distances, UInt32(maxLen))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	{
		var pos1 UInt32 = p.Pos + 1
		p.Pos = pos1
		if pos1 == p.PosLimit {
			MatchFinder_CheckLimits(p)
		}
	}
	return distances
}
func Bt4_MatchFinder_GetMatches(_p unsafe.Pointer, distances *UInt32) *UInt32 {
	var (
		p        *CMatchFinder = (*CMatchFinder)(_p)
		mmm      UInt32
		h2       UInt32
		h3       UInt32
		d2       UInt32
		d3       UInt32
		pos      UInt32
		maxLen   uint32
		hash     *UInt32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
		lenLimit UInt32 = p.LenLimit
	)
	if lenLimit < 4 {
		MatchFinder_MovePos(p)
		return distances
	}
	cur = p.Buffer
	{
		var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
		h2 = temp & ((1 << 10) - 1)
		temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
		h3 = temp & ((1 << 16) - 1)
		hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))]<<kLzHash_CrcShift_1) & p.HashMask
	}
	hash = (*UInt32)(unsafe.Pointer(p.Hash))
	pos = p.Pos
	d2 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2)))
	d3 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3)))
	curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
	*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv))) = pos
	mmm = p.CyclicBufferSize
	if pos < mmm {
		mmm = pos
	}
	maxLen = 3
	for {
		if d2 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))) == *cur {
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*0)) = 2
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d2 - 1
			distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
			if *((*Byte)(unsafe.Add(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))), 2))) == *(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2)) {
			} else if d3 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d3)))) == *cur {
				d2 = d3
				*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d3 - 1
				distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
			} else {
				break
			}
		} else if d3 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d3)))) == *cur {
			d2 = d3
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d3 - 1
			distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
		} else {
			break
		}
		{
			var (
				diff int32 = 0 - int32(d2)
				c    *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), maxLen))
				lim  *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), lenLimit))
			)
			for ; c != lim; c = (*Byte)(unsafe.Add(unsafe.Pointer(c), 1)) {
				if *((*Byte)(unsafe.Add(unsafe.Pointer(c), diff))) != *c {
					break
				}
			}
			maxLen = uint32(int32(uintptr(unsafe.Pointer(c)) - uintptr(unsafe.Pointer(cur))))
		}
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), -int(unsafe.Sizeof(UInt32(0))*2))) = UInt32(maxLen)
		if maxLen == uint32(lenLimit) {
			SkipMatchesSpec(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			{
				var pos1 UInt32 = p.Pos + 1
				p.Pos = pos1
				if pos1 == p.PosLimit {
					MatchFinder_CheckLimits(p)
				}
			}
			return distances
		}
		break
	}
	distances = GetMatchesSpec1(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue, distances, UInt32(maxLen))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	{
		var pos1 UInt32 = p.Pos + 1
		p.Pos = pos1
		if pos1 == p.PosLimit {
			MatchFinder_CheckLimits(p)
		}
	}
	return distances
}
func Bt5_MatchFinder_GetMatches(_p unsafe.Pointer, distances *UInt32) *UInt32 {
	var (
		p        *CMatchFinder = (*CMatchFinder)(_p)
		mmm      UInt32
		h2       UInt32
		h3       UInt32
		d2       UInt32
		d3       UInt32
		pos      UInt32
		maxLen   uint32
		hash     *UInt32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
		lenLimit UInt32 = p.LenLimit
	)
	if lenLimit < 5 {
		MatchFinder_MovePos(p)
		return distances
	}
	cur = p.Buffer
	{
		var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
		h2 = temp & ((1 << 10) - 1)
		temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
		h3 = temp & ((1 << 16) - 1)
		temp ^= p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))] << kLzHash_CrcShift_1
		hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 4))]<<kLzHash_CrcShift_2) & p.HashMask
	}
	hash = (*UInt32)(unsafe.Pointer(p.Hash))
	pos = p.Pos
	d2 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2)))
	d3 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3)))
	curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
	*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv))) = pos
	mmm = p.CyclicBufferSize
	if pos < mmm {
		mmm = pos
	}
	maxLen = 4
	for {
		if d2 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))) == *cur {
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*0)) = 2
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d2 - 1
			distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
			if *((*Byte)(unsafe.Add(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))), 2))) == *(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2)) {
			} else if d3 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d3)))) == *cur {
				*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d3 - 1
				distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
				d2 = d3
			} else {
				break
			}
		} else if d3 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d3)))) == *cur {
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d3 - 1
			distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
			d2 = d3
		} else {
			break
		}
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), -int(unsafe.Sizeof(UInt32(0))*2))) = 3
		if *((*Byte)(unsafe.Add(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))), 3))) != *(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3)) {
			break
		}
		{
			var (
				diff int32 = 0 - int32(d2)
				c    *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), maxLen))
				lim  *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), lenLimit))
			)
			for ; c != lim; c = (*Byte)(unsafe.Add(unsafe.Pointer(c), 1)) {
				if *((*Byte)(unsafe.Add(unsafe.Pointer(c), diff))) != *c {
					break
				}
			}
			maxLen = uint32(int32(uintptr(unsafe.Pointer(c)) - uintptr(unsafe.Pointer(cur))))
		}
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), -int(unsafe.Sizeof(UInt32(0))*2))) = UInt32(maxLen)
		if maxLen == uint32(lenLimit) {
			SkipMatchesSpec(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			{
				var pos1 UInt32 = p.Pos + 1
				p.Pos = pos1
				if pos1 == p.PosLimit {
					MatchFinder_CheckLimits(p)
				}
			}
			return distances
		}
		break
	}
	distances = GetMatchesSpec1(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue, distances, UInt32(maxLen))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	{
		var pos1 UInt32 = p.Pos + 1
		p.Pos = pos1
		if pos1 == p.PosLimit {
			MatchFinder_CheckLimits(p)
		}
	}
	return distances
}
func Hc4_MatchFinder_GetMatches(_p unsafe.Pointer, distances *UInt32) *UInt32 {
	var (
		p        *CMatchFinder = (*CMatchFinder)(_p)
		mmm      UInt32
		h2       UInt32
		h3       UInt32
		d2       UInt32
		d3       UInt32
		pos      UInt32
		maxLen   uint32
		hash     *UInt32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
		lenLimit UInt32 = p.LenLimit
	)
	if lenLimit < 4 {
		MatchFinder_MovePos(p)
		return distances
	}
	cur = p.Buffer
	{
		var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
		h2 = temp & ((1 << 10) - 1)
		temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
		h3 = temp & ((1 << 16) - 1)
		hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))]<<kLzHash_CrcShift_1) & p.HashMask
	}
	hash = (*UInt32)(unsafe.Pointer(p.Hash))
	pos = p.Pos
	d2 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2)))
	d3 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3)))
	curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
	*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv))) = pos
	mmm = p.CyclicBufferSize
	if pos < mmm {
		mmm = pos
	}
	maxLen = 3
	for {
		if d2 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))) == *cur {
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*0)) = 2
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d2 - 1
			distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
			if *((*Byte)(unsafe.Add(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))), 2))) == *(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2)) {
			} else if d3 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d3)))) == *cur {
				d2 = d3
				*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d3 - 1
				distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
			} else {
				break
			}
		} else if d3 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d3)))) == *cur {
			d2 = d3
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d3 - 1
			distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
		} else {
			break
		}
		{
			var (
				diff int32 = 0 - int32(d2)
				c    *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), maxLen))
				lim  *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), lenLimit))
			)
			for ; c != lim; c = (*Byte)(unsafe.Add(unsafe.Pointer(c), 1)) {
				if *((*Byte)(unsafe.Add(unsafe.Pointer(c), diff))) != *c {
					break
				}
			}
			maxLen = uint32(int32(uintptr(unsafe.Pointer(c)) - uintptr(unsafe.Pointer(cur))))
		}
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), -int(unsafe.Sizeof(UInt32(0))*2))) = UInt32(maxLen)
		if maxLen == uint32(lenLimit) {
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Son), unsafe.Sizeof(CLzRef(0))*uintptr(p.CyclicBufferPos))) = CLzRef(curMatch)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			{
				var pos1 UInt32 = p.Pos + 1
				p.Pos = pos1
				if pos1 == p.PosLimit {
					MatchFinder_CheckLimits(p)
				}
			}
			return distances
		}
		break
	}
	distances = Hc_GetMatchesSpec(uint32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue, distances, uint32(UInt32(maxLen)))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	{
		var pos1 UInt32 = p.Pos + 1
		p.Pos = pos1
		if pos1 == p.PosLimit {
			MatchFinder_CheckLimits(p)
		}
	}
	return distances
}
func Hc5_MatchFinder_GetMatches(_p unsafe.Pointer, distances *UInt32) *UInt32 {
	var (
		p        *CMatchFinder = (*CMatchFinder)(_p)
		mmm      UInt32
		h2       UInt32
		h3       UInt32
		d2       UInt32
		d3       UInt32
		pos      UInt32
		maxLen   uint32
		hash     *UInt32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
		lenLimit UInt32 = p.LenLimit
	)
	if lenLimit < 5 {
		MatchFinder_MovePos(p)
		return distances
	}
	cur = p.Buffer
	{
		var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
		h2 = temp & ((1 << 10) - 1)
		temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
		h3 = temp & ((1 << 16) - 1)
		temp ^= p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))] << kLzHash_CrcShift_1
		hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 4))]<<kLzHash_CrcShift_2) & p.HashMask
	}
	hash = (*UInt32)(unsafe.Pointer(p.Hash))
	pos = p.Pos
	d2 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2)))
	d3 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3)))
	curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
	*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv))) = pos
	mmm = p.CyclicBufferSize
	if pos < mmm {
		mmm = pos
	}
	maxLen = 4
	for {
		if d2 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))) == *cur {
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*0)) = 2
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d2 - 1
			distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
			if *((*Byte)(unsafe.Add(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))), 2))) == *(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2)) {
			} else if d3 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d3)))) == *cur {
				*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d3 - 1
				distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
				d2 = d3
			} else {
				break
			}
		} else if d3 < mmm && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d3)))) == *cur {
			*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d3 - 1
			distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*2))
			d2 = d3
		} else {
			break
		}
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), -int(unsafe.Sizeof(UInt32(0))*2))) = 3
		if *((*Byte)(unsafe.Add(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))), 3))) != *(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3)) {
			break
		}
		{
			var (
				diff int32 = 0 - int32(d2)
				c    *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), maxLen))
				lim  *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(cur), lenLimit))
			)
			for ; c != lim; c = (*Byte)(unsafe.Add(unsafe.Pointer(c), 1)) {
				if *((*Byte)(unsafe.Add(unsafe.Pointer(c), diff))) != *c {
					break
				}
			}
			maxLen = uint32(int32(uintptr(unsafe.Pointer(c)) - uintptr(unsafe.Pointer(cur))))
		}
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), -int(unsafe.Sizeof(UInt32(0))*2))) = UInt32(maxLen)
		if maxLen == uint32(lenLimit) {
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Son), unsafe.Sizeof(CLzRef(0))*uintptr(p.CyclicBufferPos))) = CLzRef(curMatch)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			{
				var pos1 UInt32 = p.Pos + 1
				p.Pos = pos1
				if pos1 == p.PosLimit {
					MatchFinder_CheckLimits(p)
				}
			}
			return distances
		}
		break
	}
	distances = Hc_GetMatchesSpec(uint32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue, distances, uint32(UInt32(maxLen)))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	{
		var pos1 UInt32 = p.Pos + 1
		p.Pos = pos1
		if pos1 == p.PosLimit {
			MatchFinder_CheckLimits(p)
		}
	}
	return distances
}
func Hc3Zip_MatchFinder_GetMatches(p *CMatchFinder, distances *UInt32) *UInt32 {
	var (
		hv       UInt32
		cur      *Byte
		curMatch UInt32
		lenLimit UInt32 = p.LenLimit
	)
	if lenLimit < 3 {
		MatchFinder_MovePos(p)
		return distances
	}
	cur = p.Buffer
	hv = ((UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) | UInt32(*cur)<<8) ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))]) & 0xFFFF
	curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
	*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
	distances = Hc_GetMatchesSpec(uint32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue, distances, 2)
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	{
		var pos1 UInt32 = p.Pos + 1
		p.Pos = pos1
		if pos1 == p.PosLimit {
			MatchFinder_CheckLimits(p)
		}
	}
	return distances
}
func Bt2_MatchFinder_Skip(_p unsafe.Pointer, num UInt32) {
	var p *CMatchFinder = (*CMatchFinder)(_p)
	for {
		{
			var (
				hv       UInt32
				cur      *Byte
				curMatch UInt32
				lenLimit UInt32 = p.LenLimit
			)
			if lenLimit < 2 {
				MatchFinder_MovePos(p)
				num--
				if num == 0 {
					return
				}
				continue
			}
			cur = p.Buffer
			{
				hv = UInt32(UInt16(*cur) | UInt16(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))<<8)
				curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
				*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
			}
			SkipMatchesSpec(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			{
				var pos1 UInt32 = p.Pos + 1
				p.Pos = pos1
				if pos1 == p.PosLimit {
					MatchFinder_CheckLimits(p)
				}
			}
		}
		if func() UInt32 {
			p_ := &num
			*p_--
			return *p_
		}() == 0 {
			break
		}
	}
}
func Bt3Zip_MatchFinder_Skip(p *CMatchFinder, num UInt32) {
	for {
		{
			var (
				hv       UInt32
				cur      *Byte
				curMatch UInt32
				lenLimit UInt32 = p.LenLimit
			)
			if lenLimit < 3 {
				MatchFinder_MovePos(p)
				num--
				if num == 0 {
					return
				}
				continue
			}
			cur = p.Buffer
			{
				hv = ((UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) | UInt32(*cur)<<8) ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))]) & 0xFFFF
				curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
				*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
			}
			SkipMatchesSpec(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			{
				var pos1 UInt32 = p.Pos + 1
				p.Pos = pos1
				if pos1 == p.PosLimit {
					MatchFinder_CheckLimits(p)
				}
			}
		}
		if func() UInt32 {
			p_ := &num
			*p_--
			return *p_
		}() == 0 {
			break
		}
	}
}
func Bt3_MatchFinder_Skip(_p unsafe.Pointer, num UInt32) {
	var p *CMatchFinder = (*CMatchFinder)(_p)
	for {
		{
			var (
				hv       UInt32
				cur      *Byte
				curMatch UInt32
				lenLimit UInt32 = p.LenLimit
			)
			if lenLimit < 3 {
				MatchFinder_MovePos(p)
				num--
				if num == 0 {
					return
				}
				continue
			}
			cur = p.Buffer
			{
				var (
					h2   UInt32
					hash *UInt32
				)
				{
					var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
					h2 = temp & ((1 << 10) - 1)
					hv = (temp ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2)))<<8) & p.HashMask
				}
				hash = (*UInt32)(unsafe.Pointer(p.Hash))
				curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
				*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = func() UInt32 {
					p_ := (*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
					*p_ = p.Pos
					return *p_
				}()
			}
			SkipMatchesSpec(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			{
				var pos1 UInt32 = p.Pos + 1
				p.Pos = pos1
				if pos1 == p.PosLimit {
					MatchFinder_CheckLimits(p)
				}
			}
		}
		if func() UInt32 {
			p_ := &num
			*p_--
			return *p_
		}() == 0 {
			break
		}
	}
}
func Bt4_MatchFinder_Skip(_p unsafe.Pointer, num UInt32) {
	var p *CMatchFinder = (*CMatchFinder)(_p)
	for {
		{
			var (
				hv       UInt32
				cur      *Byte
				curMatch UInt32
				lenLimit UInt32 = p.LenLimit
			)
			if lenLimit < 4 {
				MatchFinder_MovePos(p)
				num--
				if num == 0 {
					return
				}
				continue
			}
			cur = p.Buffer
			{
				var (
					h2   UInt32
					h3   UInt32
					hash *UInt32
				)
				{
					var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
					h2 = temp & ((1 << 10) - 1)
					temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
					h3 = temp & ((1 << 16) - 1)
					hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))]<<kLzHash_CrcShift_1) & p.HashMask
				}
				hash = (*UInt32)(unsafe.Pointer(p.Hash))
				curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
				*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = func() UInt32 {
					p_ := (*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3)))
					*p_ = func() UInt32 {
						p_ := (*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
						*p_ = p.Pos
						return *p_
					}()
					return *p_
				}()
			}
			SkipMatchesSpec(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			{
				var pos1 UInt32 = p.Pos + 1
				p.Pos = pos1
				if pos1 == p.PosLimit {
					MatchFinder_CheckLimits(p)
				}
			}
		}
		if func() UInt32 {
			p_ := &num
			*p_--
			return *p_
		}() == 0 {
			break
		}
	}
}
func Bt5_MatchFinder_Skip(_p unsafe.Pointer, num UInt32) {
	var p *CMatchFinder = (*CMatchFinder)(_p)
	for {
		{
			var (
				hv       UInt32
				cur      *Byte
				curMatch UInt32
				lenLimit UInt32 = p.LenLimit
			)
			if lenLimit < 5 {
				MatchFinder_MovePos(p)
				num--
				if num == 0 {
					return
				}
				continue
			}
			cur = p.Buffer
			{
				var (
					h2   UInt32
					h3   UInt32
					hash *UInt32
				)
				{
					var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
					h2 = temp & ((1 << 10) - 1)
					temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
					h3 = temp & ((1 << 16) - 1)
					temp ^= p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))] << kLzHash_CrcShift_1
					hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 4))]<<kLzHash_CrcShift_2) & p.HashMask
				}
				hash = (*UInt32)(unsafe.Pointer(p.Hash))
				curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
				*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = func() UInt32 {
					p_ := (*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3)))
					*p_ = func() UInt32 {
						p_ := (*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
						*p_ = p.Pos
						return *p_
					}()
					return *p_
				}()
			}
			SkipMatchesSpec(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, uint32(p.CyclicBufferPos), p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			{
				var pos1 UInt32 = p.Pos + 1
				p.Pos = pos1
				if pos1 == p.PosLimit {
					MatchFinder_CheckLimits(p)
				}
			}
		}
		if func() UInt32 {
			p_ := &num
			*p_--
			return *p_
		}() == 0 {
			break
		}
	}
}
func Hc4_MatchFinder_Skip(_p unsafe.Pointer, num UInt32) {
	var p *CMatchFinder = (*CMatchFinder)(_p)
	for {
		if p.LenLimit < 4 {
			MatchFinder_MovePos(p)
			num--
			if num == 0 {
				return
			}
			continue
		}
		{
			var (
				cur  *Byte
				hash *UInt32
				son  *UInt32
				pos  UInt32 = p.Pos
				num2 UInt32 = num
			)
			{
				var rem UInt32 = p.PosLimit - pos
				if num2 >= rem {
					num2 = rem
				}
			}
			num -= num2
			{
				var cycPos UInt32 = p.CyclicBufferPos
				son = (*UInt32)(unsafe.Pointer((*CLzRef)(unsafe.Add(unsafe.Pointer(p.Son), unsafe.Sizeof(CLzRef(0))*uintptr(cycPos)))))
				p.CyclicBufferPos = cycPos + num2
			}
			cur = p.Buffer
			hash = (*UInt32)(unsafe.Pointer(p.Hash))
			for {
				{
					var (
						curMatch UInt32
						hv       UInt32
						h2       UInt32
						h3       UInt32
					)
					{
						var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
						h2 = temp & ((1 << 10) - 1)
						temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
						h3 = temp & ((1 << 16) - 1)
						hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))]<<kLzHash_CrcShift_1) & p.HashMask
					}
					curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = func() UInt32 {
						p_ := (*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3)))
						*p_ = func() UInt32 {
							p_ := (*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
							*p_ = pos
							return *p_
						}()
						return *p_
					}()
					cur = (*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))
					pos++
					*func() *UInt32 {
						p_ := &son
						x := *p_
						*p_ = (*UInt32)(unsafe.Add(unsafe.Pointer(*p_), unsafe.Sizeof(UInt32(0))*1))
						return x
					}() = curMatch
				}
				if func() UInt32 {
					p_ := &num2
					*p_--
					return *p_
				}() == 0 {
					break
				}
			}
			p.Buffer = cur
			p.Pos = pos
			if pos == p.PosLimit {
				MatchFinder_CheckLimits(p)
			}
		}
		if num == 0 {
			break
		}
	}
}
func Hc5_MatchFinder_Skip(_p unsafe.Pointer, num UInt32) {
	var p *CMatchFinder = (*CMatchFinder)(_p)
	for {
		if p.LenLimit < 5 {
			MatchFinder_MovePos(p)
			num--
			if num == 0 {
				return
			}
			continue
		}
		{
			var (
				cur  *Byte
				hash *UInt32
				son  *UInt32
				pos  UInt32 = p.Pos
				num2 UInt32 = num
			)
			{
				var rem UInt32 = p.PosLimit - pos
				if num2 >= rem {
					num2 = rem
				}
			}
			num -= num2
			{
				var cycPos UInt32 = p.CyclicBufferPos
				son = (*UInt32)(unsafe.Pointer((*CLzRef)(unsafe.Add(unsafe.Pointer(p.Son), unsafe.Sizeof(CLzRef(0))*uintptr(cycPos)))))
				p.CyclicBufferPos = cycPos + num2
			}
			cur = p.Buffer
			hash = (*UInt32)(unsafe.Pointer(p.Hash))
			for {
				{
					var (
						curMatch UInt32
						hv       UInt32
						h2       UInt32
						h3       UInt32
					)
					{
						var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
						h2 = temp & ((1 << 10) - 1)
						temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
						h3 = temp & ((1 << 16) - 1)
						temp ^= p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))] << kLzHash_CrcShift_1
						hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 4))]<<kLzHash_CrcShift_2) & p.HashMask
					}
					curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = func() UInt32 {
						p_ := (*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3)))
						*p_ = func() UInt32 {
							p_ := (*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
							*p_ = pos
							return *p_
						}()
						return *p_
					}()
					cur = (*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))
					pos++
					*func() *UInt32 {
						p_ := &son
						x := *p_
						*p_ = (*UInt32)(unsafe.Add(unsafe.Pointer(*p_), unsafe.Sizeof(UInt32(0))*1))
						return x
					}() = curMatch
				}
				if func() UInt32 {
					p_ := &num2
					*p_--
					return *p_
				}() == 0 {
					break
				}
			}
			p.Buffer = cur
			p.Pos = pos
			if pos == p.PosLimit {
				MatchFinder_CheckLimits(p)
			}
		}
		if num == 0 {
			break
		}
	}
}
func Hc3Zip_MatchFinder_Skip(p *CMatchFinder, num UInt32) {
	for {
		if p.LenLimit < 3 {
			MatchFinder_MovePos(p)
			num--
			if num == 0 {
				return
			}
			continue
		}
		{
			var (
				cur  *Byte
				hash *UInt32
				son  *UInt32
				pos  UInt32 = p.Pos
				num2 UInt32 = num
			)
			{
				var rem UInt32 = p.PosLimit - pos
				if num2 >= rem {
					num2 = rem
				}
			}
			num -= num2
			{
				var cycPos UInt32 = p.CyclicBufferPos
				son = (*UInt32)(unsafe.Pointer((*CLzRef)(unsafe.Add(unsafe.Pointer(p.Son), unsafe.Sizeof(CLzRef(0))*uintptr(cycPos)))))
				p.CyclicBufferPos = cycPos + num2
			}
			cur = p.Buffer
			hash = (*UInt32)(unsafe.Pointer(p.Hash))
			for {
				{
					var (
						curMatch UInt32
						hv       UInt32
					)
					hv = ((UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) | UInt32(*cur)<<8) ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))]) & 0xFFFF
					curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
					*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(hv))) = pos
					cur = (*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))
					pos++
					*func() *UInt32 {
						p_ := &son
						x := *p_
						*p_ = (*UInt32)(unsafe.Add(unsafe.Pointer(*p_), unsafe.Sizeof(UInt32(0))*1))
						return x
					}() = curMatch
				}
				if func() UInt32 {
					p_ := &num2
					*p_--
					return *p_
				}() == 0 {
					break
				}
			}
			p.Buffer = cur
			p.Pos = pos
			if pos == p.PosLimit {
				MatchFinder_CheckLimits(p)
			}
		}
		if num == 0 {
			break
		}
	}
}
func MatchFinder_CreateVTable(p *CMatchFinder, vTable *IMatchFinder2) {
	vTable.Init = func(object unsafe.Pointer) {
		func(object unsafe.Pointer) {
			MatchFinder_Init(object)
		}(object)
	}
	vTable.GetNumAvailableBytes = func(object unsafe.Pointer) UInt32 {
		return func(object unsafe.Pointer) UInt32 {
			return MatchFinder_GetNumAvailableBytes(object)
		}(object)
	}
	vTable.GetPointerToCurrentPos = func(object unsafe.Pointer) *Byte {
		return func(object unsafe.Pointer) *Byte {
			return MatchFinder_GetPointerToCurrentPos(object)
		}(object)
	}
	if p.BtMode == 0 {
		if p.NumHashBytes <= 4 {
			vTable.GetMatches = func(object unsafe.Pointer, distances *UInt32) *UInt32 {
				return func(object unsafe.Pointer, distances *UInt32) *UInt32 {
					return Hc4_MatchFinder_GetMatches(object, distances)
				}(object, distances)
			}
			vTable.Skip = func(object unsafe.Pointer, num UInt32) {
				func(object unsafe.Pointer, num UInt32) {
					Hc4_MatchFinder_Skip(object, num)
				}(object, num)
			}
		} else {
			vTable.GetMatches = func(object unsafe.Pointer, distances *UInt32) *UInt32 {
				return func(object unsafe.Pointer, distances *UInt32) *UInt32 {
					return Hc5_MatchFinder_GetMatches(object, distances)
				}(object, distances)
			}
			vTable.Skip = func(object unsafe.Pointer, num UInt32) {
				func(object unsafe.Pointer, num UInt32) {
					Hc5_MatchFinder_Skip(object, num)
				}(object, num)
			}
		}
	} else if p.NumHashBytes == 2 {
		vTable.GetMatches = func(object unsafe.Pointer, distances *UInt32) *UInt32 {
			return func(object unsafe.Pointer, distances *UInt32) *UInt32 {
				return Bt2_MatchFinder_GetMatches(object, distances)
			}(object, distances)
		}
		vTable.Skip = func(object unsafe.Pointer, num UInt32) {
			func(object unsafe.Pointer, num UInt32) {
				Bt2_MatchFinder_Skip(object, num)
			}(object, num)
		}
	} else if p.NumHashBytes == 3 {
		vTable.GetMatches = func(object unsafe.Pointer, distances *UInt32) *UInt32 {
			return func(object unsafe.Pointer, distances *UInt32) *UInt32 {
				return Bt3_MatchFinder_GetMatches(object, distances)
			}(object, distances)
		}
		vTable.Skip = func(object unsafe.Pointer, num UInt32) {
			func(object unsafe.Pointer, num UInt32) {
				Bt3_MatchFinder_Skip(object, num)
			}(object, num)
		}
	} else if p.NumHashBytes == 4 {
		vTable.GetMatches = func(object unsafe.Pointer, distances *UInt32) *UInt32 {
			return func(object unsafe.Pointer, distances *UInt32) *UInt32 {
				return Bt4_MatchFinder_GetMatches(object, distances)
			}(object, distances)
		}
		vTable.Skip = func(object unsafe.Pointer, num UInt32) {
			func(object unsafe.Pointer, num UInt32) {
				Bt4_MatchFinder_Skip(object, num)
			}(object, num)
		}
	} else {
		vTable.GetMatches = func(object unsafe.Pointer, distances *UInt32) *UInt32 {
			return func(object unsafe.Pointer, distances *UInt32) *UInt32 {
				return Bt5_MatchFinder_GetMatches(object, distances)
			}(object, distances)
		}
		vTable.Skip = func(object unsafe.Pointer, num UInt32) {
			func(object unsafe.Pointer, num UInt32) {
				Bt5_MatchFinder_Skip(object, num)
			}(object, num)
		}
	}
	// The vtable lives in the mmap'd encoder, which the garbage collector
	// does not scan. Pin the closures or a collection can free them mid-encode.
	pinValue(unsafe.Pointer(p), vTable.Init)
	pinValue(unsafe.Pointer(p), vTable.GetNumAvailableBytes)
	pinValue(unsafe.Pointer(p), vTable.GetPointerToCurrentPos)
	pinValue(unsafe.Pointer(p), vTable.GetMatches)
	pinValue(unsafe.Pointer(p), vTable.Skip)
}
func LzFindPrepare() {
}
