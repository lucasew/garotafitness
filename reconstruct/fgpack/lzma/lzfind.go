package lzma

import (
	"unsafe"
)

const kEmptyHashValue = 0
const kNormalizeStepMin = 1024
const kStartMaxLen = 3
const kCrcPoly = 0xEDB88320

type UInt32 uint32
type CLzRef UInt32
type _CMatchFinder struct {
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
	BufferBase          *Byte
	Stream              *ISeqInStream
	BlockSize           UInt32
	KeepSizeBefore      UInt32
	KeepSizeAfter       UInt32
	NumHashBytes        UInt32
	DirectInputRem      uint32
	HistorySize         UInt32
	FixedHashSize       UInt32
	HashSizeSum         UInt32
	Result              SRes
	Crc                 [256]UInt32
	NumRefs             uint32
	ExpectedDataSize    UInt64
}
type CMatchFinder _CMatchFinder
type Mf_Init_Func func(object unsafe.Pointer)
type Mf_GetNumAvailableBytes_Func func(object unsafe.Pointer) UInt32
type Mf_GetPointerToCurrentPos_Func func(object unsafe.Pointer) *Byte
type Mf_GetMatches_Func func(object unsafe.Pointer, distances *UInt32) UInt32
type Mf_Skip_Func func(object unsafe.Pointer, num UInt32)
type _IMatchFinder struct {
	Init                   Mf_Init_Func
	GetNumAvailableBytes   Mf_GetNumAvailableBytes_Func
	GetPointerToCurrentPos Mf_GetPointerToCurrentPos_Func
	GetMatches             Mf_GetMatches_Func
	Skip                   Mf_Skip_Func
}
type IMatchFinder _IMatchFinder

func LzInWindow_Free(p *CMatchFinder, alloc ISzAllocPtr) {
	if p.DirectInput == 0 {
		alloc.Free(alloc, unsafe.Pointer(p.BufferBase))
		p.BufferBase = nil
	}
}
func LzInWindow_Create(p *CMatchFinder, keepSizeReserv UInt32, alloc ISzAllocPtr) int32 {
	var blockSize UInt32 = p.KeepSizeBefore + p.KeepSizeAfter + keepSizeReserv
	if p.DirectInput != 0 {
		p.BlockSize = blockSize
		return 1
	}
	if p.BufferBase == nil || p.BlockSize != blockSize {
		LzInWindow_Free(p, alloc)
		p.BlockSize = blockSize
		p.BufferBase = (*Byte)(alloc.Alloc(alloc, uint32(blockSize)))
	}
	return cBoolToInt(p.BufferBase != nil)
}
func MatchFinder_GetPointerToCurrentPos(p *CMatchFinder) *Byte {
	return p.Buffer
}
func MatchFinder_GetNumAvailableBytes(p *CMatchFinder) UInt32 {
	return p.StreamPos - p.Pos
}
func MatchFinder_ReduceOffsets(p *CMatchFinder, subValue UInt32) {
	p.PosLimit -= subValue
	p.Pos -= subValue
	p.StreamPos -= subValue
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
		p.DirectInputRem -= uint32(curSize)
		p.StreamPos += curSize
		if p.DirectInputRem == 0 {
			p.StreamEndWasReached = 1
		}
		return
	}
	for {
		var (
			dest *Byte  = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), p.StreamPos-p.Pos))
			size uint32 = uint32(int32(uintptr(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(p.BufferBase), p.BlockSize)))) - uintptr(unsafe.Pointer(dest))))
		)
		if size == 0 {
			return
		}
		p.Result = p.Stream.Read(p.Stream, unsafe.Pointer(dest), &size)
		if p.Result != SZ_OK {
			return
		}
		if size == 0 {
			p.StreamEndWasReached = 1
			return
		}
		p.StreamPos += UInt32(size)
		if p.StreamPos-p.Pos > p.KeepSizeAfter {
			return
		}
	}
}
func MatchFinder_MoveBlock(p *CMatchFinder) {
	cMemmove(unsafe.Pointer(p.BufferBase), unsafe.Add(unsafe.Pointer(p.Buffer), -int(p.KeepSizeBefore)), int(uint32(p.StreamPos-p.Pos)+uint32(p.KeepSizeBefore)))
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.BufferBase), p.KeepSizeBefore))
}
func MatchFinder_NeedMove(p *CMatchFinder) int32 {
	if p.DirectInput != 0 {
		return 0
	}
	return cBoolToInt(uint32(int32(uintptr(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(p.BufferBase), p.BlockSize))))-uintptr(unsafe.Pointer(p.Buffer)))) <= uint32(p.KeepSizeAfter))
}
func MatchFinder_ReadIfRequired(p *CMatchFinder) {
	if p.StreamEndWasReached != 0 {
		return
	}
	if p.KeepSizeAfter >= p.StreamPos-p.Pos {
		MatchFinder_ReadBlock(p)
	}
}
func MatchFinder_CheckAndMoveAndRead(p *CMatchFinder) {
	if MatchFinder_NeedMove(p) != 0 {
		MatchFinder_MoveBlock(p)
	}
	MatchFinder_ReadBlock(p)
}
func MatchFinder_SetDefaultSettings(p *CMatchFinder) {
	p.CutValue = 32
	p.BtMode = 1
	p.NumHashBytes = 4
	p.BigHash = 0
}
func MatchFinder_Construct(p *CMatchFinder) {
	var i uint32
	p.BufferBase = nil
	p.DirectInput = 0
	p.Hash = nil
	p.ExpectedDataSize = 18446744073709551615
	MatchFinder_SetDefaultSettings(p)
	for i = 0; i < 256; i++ {
		var (
			r UInt32 = UInt32(i)
			j uint32
		)
		for j = 0; j < 8; j++ {
			r = (r >> 1) ^ (kCrcPoly & (0 - (r & 1)))
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
func MatchFinder_Create(p *CMatchFinder, historySize UInt32, keepAddBufferBefore UInt32, matchMaxLen UInt32, keepAddBufferAfter UInt32, alloc ISzAllocPtr) int32 {
	var sizeReserv UInt32
	if historySize > (7 << 29) {
		MatchFinder_Free(p, alloc)
		return 0
	}
	sizeReserv = historySize >> 1
	if historySize >= (3 << 30) {
		sizeReserv = historySize >> 3
	} else if historySize >= (2 << 30) {
		sizeReserv = historySize >> 2
	}
	sizeReserv += (keepAddBufferBefore+matchMaxLen+keepAddBufferAfter)/2 + (1 << 19)
	p.KeepSizeBefore = historySize + keepAddBufferBefore + 1
	p.KeepSizeAfter = matchMaxLen + keepAddBufferAfter
	if LzInWindow_Create(p, sizeReserv, alloc) != 0 {
		var (
			newCyclicBufferSize UInt32 = historySize + 1
			hs                  UInt32
		)
		p.MatchMaxLen = matchMaxLen
		p.FixedHashSize = 0
		if p.NumHashBytes == 2 {
			hs = (1 << 16) - 1
		} else {
			hs = historySize
			if UInt64(hs) > p.ExpectedDataSize {
				hs = UInt32(p.ExpectedDataSize)
			}
			if hs != 0 {
				hs--
			}
			hs |= hs >> 1
			hs |= hs >> 2
			hs |= hs >> 4
			hs |= hs >> 8
			hs >>= 1
			hs |= 0xFFFF
			if hs > (1 << 24) {
				if p.NumHashBytes == 3 {
					hs = (1 << 24) - 1
				} else {
					hs >>= 1
				}
			}
		}
		p.HashMask = hs
		hs++
		if p.NumHashBytes > 2 {
			p.FixedHashSize += 1 << 10
		}
		if p.NumHashBytes > 3 {
			p.FixedHashSize += 1 << 16
		}
		if p.NumHashBytes > 4 {
			p.FixedHashSize += 1 << 20
		}
		hs += p.FixedHashSize
		{
			var (
				newSize uint32
				numSons uint32
			)
			p.HistorySize = historySize
			p.HashSizeSum = hs
			p.CyclicBufferSize = newCyclicBufferSize
			numSons = uint32(newCyclicBufferSize)
			if p.BtMode != 0 {
				numSons <<= 1
			}
			newSize = uint32(hs + UInt32(numSons))
			if p.Hash != nil && p.NumRefs == newSize {
				return 1
			}
			MatchFinder_FreeThisClassMemory(p, alloc)
			p.NumRefs = newSize
			p.Hash = AllocRefs(newSize, alloc)
			if p.Hash != nil {
				p.Son = (*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(p.HashSizeSum)))
				return 1
			}
		}
	}
	MatchFinder_Free(p, alloc)
	return 0
}
func MatchFinder_SetLimits(p *CMatchFinder) {
	var (
		limit  UInt32 = 0xFFFFFFFF - p.Pos
		limit2 UInt32 = p.CyclicBufferSize - p.CyclicBufferPos
	)
	if limit2 < limit {
		limit = limit2
	}
	limit2 = p.StreamPos - p.Pos
	if limit2 <= p.KeepSizeAfter {
		if limit2 > 0 {
			limit2 = 1
		}
	} else {
		limit2 -= p.KeepSizeAfter
	}
	if limit2 < limit {
		limit = limit2
	}
	{
		var lenLimit UInt32 = p.StreamPos - p.Pos
		if lenLimit > p.MatchMaxLen {
			lenLimit = p.MatchMaxLen
		}
		p.LenLimit = lenLimit
	}
	p.PosLimit = p.Pos + limit
}
func MatchFinder_Init_LowHash(p *CMatchFinder) {
	items := unsafe.Slice(p.Hash, p.FixedHashSize)
	for i := range items {
		items[i] = kEmptyHashValue
	}
}
func MatchFinder_Init_HighHash(p *CMatchFinder) {
	items := unsafe.Slice((*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(p.FixedHashSize))), int(p.HashMask)+1)
	for i := range items {
		items[i] = kEmptyHashValue
	}
}
func MatchFinder_Init_3(p *CMatchFinder, readData int32) {
	p.CyclicBufferPos = 0
	p.Buffer = p.BufferBase
	p.Pos = func() UInt32 {
		p_ := &p.StreamPos
		*p_ = p.CyclicBufferSize
		return *p_
	}()
	p.Result = SZ_OK
	p.StreamEndWasReached = 0
	if readData != 0 {
		MatchFinder_ReadBlock(p)
	}
	MatchFinder_SetLimits(p)
}
func MatchFinder_Init(p *CMatchFinder) {
	MatchFinder_Init_HighHash(p)
	MatchFinder_Init_LowHash(p)
	MatchFinder_Init_3(p, int32(True))
}
func MatchFinder_GetSubValue(p *CMatchFinder) UInt32 {
	return (p.Pos - p.HistorySize - 1) & (^UInt32((1 << 10) - 1))
}
func MatchFinder_Normalize3(subValue UInt32, items *CLzRef, numItems uint32) {
	refs := unsafe.Slice(items, numItems)
	for i, value := range refs {
		if value <= CLzRef(subValue) {
			value = kEmptyHashValue
		} else {
			value -= CLzRef(subValue)
		}
		refs[i] = value
	}
}
func MatchFinder_Normalize(p *CMatchFinder) {
	var subValue UInt32 = MatchFinder_GetSubValue(p)
	MatchFinder_Normalize3(subValue, p.Hash, p.NumRefs)
	MatchFinder_ReduceOffsets(p, subValue)
}
func MatchFinder_CheckLimits(p *CMatchFinder) {
	if p.Pos == 0xFFFFFFFF {
		MatchFinder_Normalize(p)
	}
	if p.StreamEndWasReached == 0 && p.KeepSizeAfter == p.StreamPos-p.Pos {
		MatchFinder_CheckAndMoveAndRead(p)
	}
	if p.CyclicBufferPos == p.CyclicBufferSize {
		p.CyclicBufferPos = 0
	}
	MatchFinder_SetLimits(p)
}
func byteAt(p *Byte, off int) byte {
	return *(*byte)(unsafe.Add(unsafe.Pointer(p), off))
}

func Hc_GetMatchesSpec(lenLimit uint32, curMatch UInt32, pos UInt32, cur *Byte, son *CLzRef, _cyclicBufferPos UInt32, _cyclicBufferSize UInt32, cutValue UInt32, distances *UInt32, maxLen uint32) *UInt32 {
	sonS := unsafe.Slice(son, int(_cyclicBufferSize))
	sonS[_cyclicBufferPos] = CLzRef(curMatch)
	curP := cur
	for {
		delta := pos - curMatch
		if delta >= _cyclicBufferSize {
			break
		}
		idx := _cyclicBufferPos - delta
		if delta > _cyclicBufferPos {
			idx += _cyclicBufferSize
		}
		curMatch = UInt32(sonS[idx])
		pb := (*Byte)(unsafe.Add(unsafe.Pointer(curP), -int(delta)))
		if byteAt(curP, int(maxLen)) == byteAt(pb, int(maxLen)) {
			cOff := 0
			for byteAt(curP, cOff) == byteAt(pb, cOff) {
				cOff++
				if uint32(cOff) == lenLimit {
					*distances = UInt32(lenLimit)
					*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), 4)) = delta - 1
					return (*UInt32)(unsafe.Add(unsafe.Pointer(distances), 8))
				}
			}
			if maxLen < uint32(cOff) {
				maxLen = uint32(cOff)
				*distances = UInt32(cOff)
				*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), 4)) = delta - 1
				distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), 8))
			}
		}
		cutValue--
		if cutValue == 0 {
			break
		}
	}
	return distances
}

func GetMatchesSpec1(lenLimit UInt32, curMatch UInt32, pos UInt32, cur *Byte, son *CLzRef, _cyclicBufferPos UInt32, _cyclicBufferSize UInt32, cutValue UInt32, distances *UInt32, maxLen UInt32) *UInt32 {
	sonS := unsafe.Slice(son, int(_cyclicBufferSize)<<1)
	ptr0 := int(_cyclicBufferPos)<<1 + 1
	ptr1 := int(_cyclicBufferPos) << 1
	len0 := uint32(0)
	len1 := uint32(0)
	for {
		delta := pos - curMatch
		oldCut := cutValue
		cutValue--
		if oldCut == 0 || delta >= _cyclicBufferSize {
			sonS[ptr0] = kEmptyHashValue
			sonS[ptr1] = kEmptyHashValue
			return distances
		}
		cyc := _cyclicBufferPos - delta
		if delta > _cyclicBufferPos {
			cyc += _cyclicBufferSize
		}
		pair := int(cyc) << 1
		pb := (*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(delta)))
		ln := len0
		if len1 < ln {
			ln = len1
		}
		pair0 := sonS[pair]
		if byteAt(pb, int(ln)) == byteAt(cur, int(ln)) {
			ln++
			if ln != uint32(lenLimit) && byteAt(pb, int(ln)) == byteAt(cur, int(ln)) {
				for {
					ln++
					if ln == uint32(lenLimit) || byteAt(pb, int(ln)) != byteAt(cur, int(ln)) {
						break
					}
				}
			}
			if maxLen < UInt32(ln) {
				maxLen = UInt32(ln)
				*distances = UInt32(ln)
				distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), 4))
				*distances = delta - 1
				distances = (*UInt32)(unsafe.Add(unsafe.Pointer(distances), 4))
				if ln == uint32(lenLimit) {
					sonS[ptr1] = pair0
					sonS[ptr0] = sonS[pair+1]
					return distances
				}
			}
		}
		if byteAt(pb, int(ln)) < byteAt(cur, int(ln)) {
			sonS[ptr1] = CLzRef(curMatch)
			ptr1 = pair + 1
			curMatch = UInt32(sonS[ptr1])
			len1 = ln
		} else {
			sonS[ptr0] = CLzRef(curMatch)
			ptr0 = pair
			curMatch = UInt32(sonS[ptr0])
			len0 = ln
		}
	}
}

func SkipMatchesSpec(lenLimit UInt32, curMatch UInt32, pos UInt32, cur *Byte, son *CLzRef, _cyclicBufferPos UInt32, _cyclicBufferSize UInt32, cutValue UInt32) {
	sonS := unsafe.Slice(son, int(_cyclicBufferSize)<<1)
	ptr0 := int(_cyclicBufferPos)<<1 + 1
	ptr1 := int(_cyclicBufferPos) << 1
	len0 := uint32(0)
	len1 := uint32(0)
	for {
		delta := pos - curMatch
		oldCut := cutValue
		cutValue--
		if oldCut == 0 || delta >= _cyclicBufferSize {
			sonS[ptr0] = kEmptyHashValue
			sonS[ptr1] = kEmptyHashValue
			return
		}
		cyc := _cyclicBufferPos - delta
		if delta > _cyclicBufferPos {
			cyc += _cyclicBufferSize
		}
		pair := int(cyc) << 1
		pb := (*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(delta)))
		ln := len0
		if len1 < ln {
			ln = len1
		}
		if byteAt(pb, int(ln)) == byteAt(cur, int(ln)) {
			for {
				ln++
				if ln == uint32(lenLimit) || byteAt(pb, int(ln)) != byteAt(cur, int(ln)) {
					break
				}
			}
			if ln == uint32(lenLimit) {
				sonS[ptr1] = sonS[pair]
				sonS[ptr0] = sonS[pair+1]
				return
			}
		}
		if byteAt(pb, int(ln)) < byteAt(cur, int(ln)) {
			sonS[ptr1] = CLzRef(curMatch)
			ptr1 = pair + 1
			curMatch = UInt32(sonS[ptr1])
			len1 = ln
		} else {
			sonS[ptr0] = CLzRef(curMatch)
			ptr0 = pair
			curMatch = UInt32(sonS[ptr0])
			len0 = ln
		}
	}
}

func MatchFinder_MovePos(p *CMatchFinder) {
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	p.Pos++
	if p.Pos == p.PosLimit {
		MatchFinder_CheckLimits(p)
	}
}
func Bt2_MatchFinder_GetMatches(p *CMatchFinder, distances *UInt32) UInt32 {
	var (
		offset   uint32
		lenLimit uint32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
	)
	lenLimit = uint32(p.LenLimit)
	{
		if lenLimit < 2 {
			MatchFinder_MovePos(p)
			return 0
		}
	}
	cur = p.Buffer
	hv = UInt32(*cur) | UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))<<8
	curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
	*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
	offset = 0
	offset = uint32(int32((uintptr(unsafe.Pointer(GetMatchesSpec1(UInt32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue, (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*uintptr(offset))), 1))) - uintptr(unsafe.Pointer(distances))) / 4))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	if func() UInt32 {
		p_ := &p.Pos
		*p_++
		return *p_
	}() == p.PosLimit {
		MatchFinder_CheckLimits(p)
	}
	return UInt32(offset)
}
func Bt3Zip_MatchFinder_GetMatches(p *CMatchFinder, distances *UInt32) UInt32 {
	var (
		offset   uint32
		lenLimit uint32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
	)
	lenLimit = uint32(p.LenLimit)
	{
		if lenLimit < 3 {
			MatchFinder_MovePos(p)
			return 0
		}
	}
	cur = p.Buffer
	hv = ((UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) | UInt32(*cur)<<8) ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))]) & 0xFFFF
	curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
	*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
	offset = 0
	offset = uint32(int32((uintptr(unsafe.Pointer(GetMatchesSpec1(UInt32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue, (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*uintptr(offset))), 2))) - uintptr(unsafe.Pointer(distances))) / 4))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	if func() UInt32 {
		p_ := &p.Pos
		*p_++
		return *p_
	}() == p.PosLimit {
		MatchFinder_CheckLimits(p)
	}
	return UInt32(offset)
}
func Bt3_MatchFinder_GetMatches(p *CMatchFinder, distances *UInt32) UInt32 {
	var (
		h2       UInt32
		d2       UInt32
		pos      UInt32
		maxLen   uint32
		offset   uint32
		hash     *UInt32
		lenLimit uint32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
	)
	lenLimit = uint32(p.LenLimit)
	{
		if lenLimit < 3 {
			MatchFinder_MovePos(p)
			return 0
		}
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
	maxLen = 2
	offset = 0
	if d2 < p.CyclicBufferSize && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))) == *cur {
		{
			var (
				diff int32 = int32(0 - d2)
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
		offset = 2
		if maxLen == lenLimit {
			SkipMatchesSpec(UInt32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			if func() UInt32 {
				p_ := &p.Pos
				*p_++
				return *p_
			}() == p.PosLimit {
				MatchFinder_CheckLimits(p)
			}
			return UInt32(offset)
		}
	}
	offset = uint32(int32((uintptr(unsafe.Pointer(GetMatchesSpec1(UInt32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue, (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*uintptr(offset))), UInt32(maxLen)))) - uintptr(unsafe.Pointer(distances))) / 4))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	if func() UInt32 {
		p_ := &p.Pos
		*p_++
		return *p_
	}() == p.PosLimit {
		MatchFinder_CheckLimits(p)
	}
	return UInt32(offset)
}
func Bt4_MatchFinder_GetMatches(p *CMatchFinder, distances *UInt32) UInt32 {
	var (
		h2       UInt32
		h3       UInt32
		d2       UInt32
		d3       UInt32
		pos      UInt32
		maxLen   uint32
		offset   uint32
		hash     *UInt32
		lenLimit uint32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
	)
	lenLimit = uint32(p.LenLimit)
	{
		if lenLimit < 4 {
			MatchFinder_MovePos(p)
			return 0
		}
	}
	cur = p.Buffer
	{
		var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
		h2 = temp & ((1 << 10) - 1)
		temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
		h3 = temp & ((1 << 16) - 1)
		hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))]<<5) & p.HashMask
	}
	hash = (*UInt32)(unsafe.Pointer(p.Hash))
	pos = p.Pos
	d2 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2)))
	d3 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3)))
	curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
	*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv))) = pos
	maxLen = 0
	offset = 0
	if d2 < p.CyclicBufferSize && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))) == *cur {
		maxLen = 2
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*0)) = 2
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d2 - 1
		offset = 2
	}
	if d2 != d3 && d3 < p.CyclicBufferSize && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d3)))) == *cur {
		maxLen = 3
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*uintptr(offset+1))) = d3 - 1
		offset += 2
		d2 = d3
	}
	if offset != 0 {
		{
			var (
				diff int32 = int32(0 - d2)
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
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*uintptr(offset-2))) = UInt32(maxLen)
		if maxLen == lenLimit {
			SkipMatchesSpec(UInt32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			if func() UInt32 {
				p_ := &p.Pos
				*p_++
				return *p_
			}() == p.PosLimit {
				MatchFinder_CheckLimits(p)
			}
			return UInt32(offset)
		}
	}
	if maxLen < 3 {
		maxLen = 3
	}
	offset = uint32(int32((uintptr(unsafe.Pointer(GetMatchesSpec1(UInt32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue, (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*uintptr(offset))), UInt32(maxLen)))) - uintptr(unsafe.Pointer(distances))) / 4))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	if func() UInt32 {
		p_ := &p.Pos
		*p_++
		return *p_
	}() == p.PosLimit {
		MatchFinder_CheckLimits(p)
	}
	return UInt32(offset)
}
func Hc4_MatchFinder_GetMatches(p *CMatchFinder, distances *UInt32) UInt32 {
	var (
		h2       UInt32
		h3       UInt32
		d2       UInt32
		d3       UInt32
		pos      UInt32
		maxLen   uint32
		offset   uint32
		hash     *UInt32
		lenLimit uint32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
	)
	lenLimit = uint32(p.LenLimit)
	{
		if lenLimit < 4 {
			MatchFinder_MovePos(p)
			return 0
		}
	}
	cur = p.Buffer
	{
		var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
		h2 = temp & ((1 << 10) - 1)
		temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
		h3 = temp & ((1 << 16) - 1)
		hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))]<<5) & p.HashMask
	}
	hash = (*UInt32)(unsafe.Pointer(p.Hash))
	pos = p.Pos
	d2 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2)))
	d3 = pos - *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3)))
	curMatch = *(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv)))
	*(*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*uintptr(h2))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*(1<<10)))), unsafe.Sizeof(UInt32(0))*uintptr(h3))) = pos
	*(*UInt32)(unsafe.Add(unsafe.Pointer((*UInt32)(unsafe.Add(unsafe.Pointer(hash), unsafe.Sizeof(UInt32(0))*((1<<10)+(1<<16))))), unsafe.Sizeof(UInt32(0))*uintptr(hv))) = pos
	maxLen = 0
	offset = 0
	if d2 < p.CyclicBufferSize && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d2)))) == *cur {
		maxLen = 2
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*0)) = 2
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*1)) = d2 - 1
		offset = 2
	}
	if d2 != d3 && d3 < p.CyclicBufferSize && *((*Byte)(unsafe.Add(unsafe.Pointer(cur), -int(d3)))) == *cur {
		maxLen = 3
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*uintptr(offset+1))) = d3 - 1
		offset += 2
		d2 = d3
	}
	if offset != 0 {
		{
			var (
				diff int32 = int32(0 - d2)
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
		*(*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*uintptr(offset-2))) = UInt32(maxLen)
		if maxLen == lenLimit {
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Son), unsafe.Sizeof(CLzRef(0))*uintptr(p.CyclicBufferPos))) = CLzRef(curMatch)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			if func() UInt32 {
				p_ := &p.Pos
				*p_++
				return *p_
			}() == p.PosLimit {
				MatchFinder_CheckLimits(p)
			}
			return UInt32(offset)
		}
	}
	if maxLen < 3 {
		maxLen = 3
	}
	offset = uint32(int32((uintptr(unsafe.Pointer(Hc_GetMatchesSpec(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue, (*UInt32)(unsafe.Add(unsafe.Pointer(distances), unsafe.Sizeof(UInt32(0))*uintptr(offset))), maxLen))) - uintptr(unsafe.Pointer(distances))) / 4))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	if func() UInt32 {
		p_ := &p.Pos
		*p_++
		return *p_
	}() == p.PosLimit {
		MatchFinder_CheckLimits(p)
	}
	return UInt32(offset)
}
func Hc3Zip_MatchFinder_GetMatches(p *CMatchFinder, distances *UInt32) UInt32 {
	var (
		offset   uint32
		lenLimit uint32
		hv       UInt32
		cur      *Byte
		curMatch UInt32
	)
	lenLimit = uint32(p.LenLimit)
	{
		if lenLimit < 3 {
			MatchFinder_MovePos(p)
			return 0
		}
	}
	cur = p.Buffer
	hv = ((UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) | UInt32(*cur)<<8) ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))]) & 0xFFFF
	curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
	*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
	offset = uint32(int32((uintptr(unsafe.Pointer(Hc_GetMatchesSpec(lenLimit, curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue, distances, 2))) - uintptr(unsafe.Pointer(distances))) / 4))
	p.CyclicBufferPos++
	p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
	if func() UInt32 {
		p_ := &p.Pos
		*p_++
		return *p_
	}() == p.PosLimit {
		MatchFinder_CheckLimits(p)
	}
	return UInt32(offset)
}
func Bt2_MatchFinder_Skip(p *CMatchFinder, num UInt32) {
	for {
		{
			var (
				lenLimit uint32
				hv       UInt32
				cur      *Byte
				curMatch UInt32
			)
			lenLimit = uint32(p.LenLimit)
			{
				if lenLimit < 2 {
					MatchFinder_MovePos(p)
					num--
					if num == 0 {
						return
					}
					continue
				}
			}
			cur = p.Buffer
			hv = UInt32(*cur) | UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))<<8
			curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
			SkipMatchesSpec(UInt32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			if func() UInt32 {
				p_ := &p.Pos
				*p_++
				return *p_
			}() == p.PosLimit {
				MatchFinder_CheckLimits(p)
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
				lenLimit uint32
				hv       UInt32
				cur      *Byte
				curMatch UInt32
			)
			lenLimit = uint32(p.LenLimit)
			{
				if lenLimit < 3 {
					MatchFinder_MovePos(p)
					num--
					if num == 0 {
						return
					}
					continue
				}
			}
			cur = p.Buffer
			hv = ((UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) | UInt32(*cur)<<8) ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))]) & 0xFFFF
			curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
			SkipMatchesSpec(UInt32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			if func() UInt32 {
				p_ := &p.Pos
				*p_++
				return *p_
			}() == p.PosLimit {
				MatchFinder_CheckLimits(p)
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
func Bt3_MatchFinder_Skip(p *CMatchFinder, num UInt32) {
	for {
		{
			var (
				h2       UInt32
				hash     *UInt32
				lenLimit uint32
				hv       UInt32
				cur      *Byte
				curMatch UInt32
			)
			lenLimit = uint32(p.LenLimit)
			{
				if lenLimit < 3 {
					MatchFinder_MovePos(p)
					num--
					if num == 0 {
						return
					}
					continue
				}
			}
			cur = p.Buffer
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
			SkipMatchesSpec(UInt32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			if func() UInt32 {
				p_ := &p.Pos
				*p_++
				return *p_
			}() == p.PosLimit {
				MatchFinder_CheckLimits(p)
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
func Bt4_MatchFinder_Skip(p *CMatchFinder, num UInt32) {
	for {
		{
			var (
				h2       UInt32
				h3       UInt32
				hash     *UInt32
				lenLimit uint32
				hv       UInt32
				cur      *Byte
				curMatch UInt32
			)
			lenLimit = uint32(p.LenLimit)
			{
				if lenLimit < 4 {
					MatchFinder_MovePos(p)
					num--
					if num == 0 {
						return
					}
					continue
				}
			}
			cur = p.Buffer
			{
				var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
				h2 = temp & ((1 << 10) - 1)
				temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
				h3 = temp & ((1 << 16) - 1)
				hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))]<<5) & p.HashMask
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
			SkipMatchesSpec(UInt32(lenLimit), curMatch, p.Pos, p.Buffer, p.Son, p.CyclicBufferPos, p.CyclicBufferSize, p.CutValue)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			if func() UInt32 {
				p_ := &p.Pos
				*p_++
				return *p_
			}() == p.PosLimit {
				MatchFinder_CheckLimits(p)
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
func Hc4_MatchFinder_Skip(p *CMatchFinder, num UInt32) {
	for {
		{
			var (
				h2       UInt32
				h3       UInt32
				hash     *UInt32
				lenLimit uint32
				hv       UInt32
				cur      *Byte
				curMatch UInt32
			)
			lenLimit = uint32(p.LenLimit)
			{
				if lenLimit < 4 {
					MatchFinder_MovePos(p)
					num--
					if num == 0 {
						return
					}
					continue
				}
			}
			cur = p.Buffer
			{
				var temp UInt32 = p.Crc[*cur] ^ UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1)))
				h2 = temp & ((1 << 10) - 1)
				temp ^= UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) << 8
				h3 = temp & ((1 << 16) - 1)
				hv = (temp ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 3))]<<5) & p.HashMask
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
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Son), unsafe.Sizeof(CLzRef(0))*uintptr(p.CyclicBufferPos))) = CLzRef(curMatch)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			if func() UInt32 {
				p_ := &p.Pos
				*p_++
				return *p_
			}() == p.PosLimit {
				MatchFinder_CheckLimits(p)
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
func Hc3Zip_MatchFinder_Skip(p *CMatchFinder, num UInt32) {
	for {
		{
			var (
				lenLimit uint32
				hv       UInt32
				cur      *Byte
				curMatch UInt32
			)
			lenLimit = uint32(p.LenLimit)
			{
				if lenLimit < 3 {
					MatchFinder_MovePos(p)
					num--
					if num == 0 {
						return
					}
					continue
				}
			}
			cur = p.Buffer
			hv = ((UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 2))) | UInt32(*cur)<<8) ^ p.Crc[*(*Byte)(unsafe.Add(unsafe.Pointer(cur), 1))]) & 0xFFFF
			curMatch = UInt32(*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))))
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Hash), unsafe.Sizeof(CLzRef(0))*uintptr(hv))) = CLzRef(p.Pos)
			*(*CLzRef)(unsafe.Add(unsafe.Pointer(p.Son), unsafe.Sizeof(CLzRef(0))*uintptr(p.CyclicBufferPos))) = CLzRef(curMatch)
			p.CyclicBufferPos++
			p.Buffer = (*Byte)(unsafe.Add(unsafe.Pointer(p.Buffer), 1))
			if func() UInt32 {
				p_ := &p.Pos
				*p_++
				return *p_
			}() == p.PosLimit {
				MatchFinder_CheckLimits(p)
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
func MatchFinder_CreateVTable(p *CMatchFinder, vTable *IMatchFinder) {
	vTable.Init = func(object unsafe.Pointer) {
		MatchFinder_Init((*CMatchFinder)(object))
	}
	vTable.GetNumAvailableBytes = func(object unsafe.Pointer) UInt32 {
		return MatchFinder_GetNumAvailableBytes((*CMatchFinder)(object))
	}
	vTable.GetPointerToCurrentPos = func(object unsafe.Pointer) *Byte {
		return MatchFinder_GetPointerToCurrentPos((*CMatchFinder)(object))
	}
	if p.BtMode == 0 {
		vTable.GetMatches = func(object unsafe.Pointer, distances *UInt32) UInt32 {
			return Hc4_MatchFinder_GetMatches((*CMatchFinder)(object), distances)
		}
		vTable.Skip = func(object unsafe.Pointer, num UInt32) {
			Hc4_MatchFinder_Skip((*CMatchFinder)(object), num)
		}
	} else if p.NumHashBytes == 2 {
		vTable.GetMatches = func(object unsafe.Pointer, distances *UInt32) UInt32 {
			return Bt2_MatchFinder_GetMatches((*CMatchFinder)(object), distances)
		}
		vTable.Skip = func(object unsafe.Pointer, num UInt32) {
			Bt2_MatchFinder_Skip((*CMatchFinder)(object), num)
		}
	} else if p.NumHashBytes == 3 {
		vTable.GetMatches = func(object unsafe.Pointer, distances *UInt32) UInt32 {
			return Bt3_MatchFinder_GetMatches((*CMatchFinder)(object), distances)
		}
		vTable.Skip = func(object unsafe.Pointer, num UInt32) {
			Bt3_MatchFinder_Skip((*CMatchFinder)(object), num)
		}
	} else {
		vTable.GetMatches = func(object unsafe.Pointer, distances *UInt32) UInt32 {
			return Bt4_MatchFinder_GetMatches((*CMatchFinder)(object), distances)
		}
		vTable.Skip = func(object unsafe.Pointer, num UInt32) {
			Bt4_MatchFinder_Skip((*CMatchFinder)(object), num)
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
