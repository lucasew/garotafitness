package lzma

import (
	"unsafe"
)

const LZMA_PROPS_SIZE = 5
const kNumTopBits = 24
const kNumBitModelTotalBits = 11
const kBitModelTotal = 2048
const kNumMoveBits = 5
const kProbInitValue = 1024
const kNumMoveReducingBits = 4
const kNumBitPriceShiftBits = 4
const kBitPrice = 16
const REP_LEN_COUNT = 64
const LZMA_NUM_REPS = 4
const kNumOpts = 2048
const kPackReserve = 0x4000
const kNumLenToPosStates = 4
const kNumPosSlotBits = 6
const kDicLogSizeMin = 0
const kDicLogSizeMax = 32
const kDistTableSizeMax = 64
const kNumAlignBits = 4
const kAlignTableSize = 16
const kAlignMask = 15
const kStartPosModelIndex = 4
const kEndPosModelIndex = 14
const kNumFullDistances = 128
const LZMA_PB_MAX = 4
const LZMA_LC_MAX = 8
const LZMA_LP_MAX = 4
const LZMA_NUM_PB_STATES_MAX = 16
const kLenNumLowBits = 3
const kLenNumLowSymbols = 8
const kLenNumHighBits = 8
const kLenNumHighSymbols = 256
const kLenNumSymbolsTotal = 272
const LZMA_MATCH_LEN_MIN = 2
const LZMA_MATCH_LEN_MAX = 273
const kNumStates = 12
const kState_Start = 0
const kState_LitAfterMatch = 4
const kState_LitAfterRep = 5
const kState_MatchAfterLit = 7
const kState_RepAfterLit = 8
const kInfinityPrice = 0x40000000
const RC_BUF_SIZE = 0x10000

type _CLzmaEncProps struct {
	Level        int32
	DictSize     UInt32
	Lc           int32
	Lp           int32
	Pb           int32
	Algo         int32
	Fb           int32
	BtMode       int32
	NumHashBytes int32
	Mc           UInt32
	WriteEndMark uint32
	NumThreads   int32
	ReduceSize   UInt64
}
type CLzmaEncProps _CLzmaEncProps
type CLzmaEncHandle = unsafe.Pointer

func LzmaEncProps_Init(p *CLzmaEncProps) {
	p.Level = 5
	p.DictSize = func() UInt32 {
		p_ := &p.Mc
		*p_ = 0
		return *p_
	}()
	p.ReduceSize = 18446744073709551615
	p.Lc = func() int32 {
		p_ := &p.Lp
		*p_ = func() int32 {
			p_ := &p.Pb
			*p_ = func() int32 {
				p_ := &p.Algo
				*p_ = func() int32 {
					p_ := &p.Fb
					*p_ = func() int32 {
						p_ := &p.BtMode
						*p_ = func() int32 {
							p_ := &p.NumHashBytes
							*p_ = func() int32 {
								p_ := &p.NumThreads
								*p_ = -1
								return *p_
							}()
							return *p_
						}()
						return *p_
					}()
					return *p_
				}()
				return *p_
			}()
			return *p_
		}()
		return *p_
	}()
	p.WriteEndMark = 0
}
func LzmaEncProps_Normalize(p *CLzmaEncProps) {
	var level int32 = p.Level
	if level < 0 {
		level = 5
	}
	p.Level = level
	if p.DictSize == 0 {
		if level <= 5 {
			p.DictSize = UInt32(1 << (level*2 + 14))
		} else if level <= 7 {
			p.DictSize = 1 << 25
		} else {
			p.DictSize = 1 << 26
		}
	}
	if UInt64(p.DictSize) > p.ReduceSize {
		var (
			i          uint32
			reduceSize UInt32 = UInt32(p.ReduceSize)
		)
		for i = 11; i <= 30; i++ {
			if reduceSize <= UInt32(2<<i) {
				p.DictSize = UInt32(2 << i)
				break
			}
			if reduceSize <= UInt32(3<<i) {
				p.DictSize = UInt32(3 << i)
				break
			}
		}
	}
	if p.Lc < 0 {
		p.Lc = 3
	}
	if p.Lp < 0 {
		p.Lp = 0
	}
	if p.Pb < 0 {
		p.Pb = 2
	}
	if p.Algo < 0 {
		if level < 5 {
			p.Algo = 0
		} else {
			p.Algo = 1
		}
	}
	if p.Fb < 0 {
		if level < 7 {
			p.Fb = 32
		} else {
			p.Fb = 64
		}
	}
	if p.BtMode < 0 {
		if p.Algo == 0 {
			p.BtMode = 0
		} else {
			p.BtMode = 1
		}
	}
	if p.NumHashBytes < 0 {
		p.NumHashBytes = 4
	}
	if p.Mc == 0 {
		p.Mc = UInt32(((p.Fb >> 1) + 16) >> (func() int32 {
			if p.BtMode != 0 {
				return 0
			}
			return 1
		}()))
	}
	if p.NumThreads < 0 {
		p.NumThreads = 1
	}
}
func LzmaEncProps_GetDictSize(props2 *CLzmaEncProps) UInt32 {
	var props CLzmaEncProps = *props2
	LzmaEncProps_Normalize(&props)
	return props.DictSize
}
func LzmaEnc_FastPosInit(g_FastPos *Byte) {
	var slot uint32
	*g_FastPos = 0
	*(*Byte)(unsafe.Add(unsafe.Pointer(g_FastPos), 1)) = 1
	g_FastPos = (*Byte)(unsafe.Add(unsafe.Pointer(g_FastPos), 2))
	for slot = 2; slot < uint32((9+unsafe.Sizeof(uint32(0))/2)*2); slot++ {
		var (
			k uint32 = (1 << ((slot >> 1) - 1))
			j uint32
		)
		for j = 0; j < k; j++ {
			*(*Byte)(unsafe.Add(unsafe.Pointer(g_FastPos), j)) = Byte(slot)
		}
		g_FastPos = (*Byte)(unsafe.Add(unsafe.Pointer(g_FastPos), k))
	}
}

type UInt16 uint16
type CState UInt16
type CExtra UInt16
type COptimal struct {
	Price UInt32
	State CState
	Extra CExtra
	Len   UInt32
	Dist  UInt32
	Reps  [4]UInt32
}
type CLzmaProb UInt16
type CLenEnc struct {
	Low  [256]CLzmaProb
	High [256]CLzmaProb
}
type CLenPriceEnc struct {
	TableSize uint32
	Prices    [16][272]UInt32
}
type CRangeEnc struct {
	Range     UInt32
	Cache     uint32
	Low       UInt64
	CacheSize UInt64
	Buf       *Byte
	BufLim    *Byte
	BufBase   *Byte
	OutStream *ISeqOutStream
	Processed UInt64
	Res       SRes
}
type CSaveState struct {
	LitProbs        *CLzmaProb
	State           uint32
	Reps            [4]UInt32
	PosAlignEncoder [16]CLzmaProb
	IsRep           [12]CLzmaProb
	IsRepG0         [12]CLzmaProb
	IsRepG1         [12]CLzmaProb
	IsRepG2         [12]CLzmaProb
	IsMatch         [12][16]CLzmaProb
	IsRep0Long      [12][16]CLzmaProb
	PosSlotEncoder  [4][64]CLzmaProb
	PosEncoders     [128]CLzmaProb
	LenProbs        CLenEnc
	RepLenProbs     CLenEnc
}
type CProbPrice UInt32
type CLzmaEnc struct {
	MatchFinderObj   unsafe.Pointer
	MatchFinder      IMatchFinder
	OptCur           uint32
	OptEnd           uint32
	LongestMatchLen  uint32
	NumPairs         uint32
	NumAvail         UInt32
	State            uint32
	NumFastBytes     uint32
	AdditionalOffset uint32
	Reps             [4]UInt32
	LpMask           uint32
	PbMask           uint32
	LitProbs         *CLzmaProb
	Rc               CRangeEnc
	BackRes          UInt32
	Lc               uint32
	Lp               uint32
	Pb               uint32
	Lclp             uint32
	FastMode         BoolInt
	WriteEndMark     BoolInt
	Finished         BoolInt
	MultiThread      BoolInt
	NeedInit         BoolInt
	NowPos64         UInt64
	MatchPriceCount  uint32
	RepLenEncCounter int32
	DistTableSize    uint32
	DictSize         UInt32
	Result           SRes
	MatchFinderBase  CMatchFinder
	ProbPrices       [128]CProbPrice
	Matches          [549]UInt32
	AlignPrices      [16]UInt32
	PosSlotPrices    [4][64]UInt32
	DistancesPrices  [4][128]UInt32
	PosAlignEncoder  [16]CLzmaProb
	IsRep            [12]CLzmaProb
	IsRepG0          [12]CLzmaProb
	IsRepG1          [12]CLzmaProb
	IsRepG2          [12]CLzmaProb
	IsMatch          [12][16]CLzmaProb
	IsRep0Long       [12][16]CLzmaProb
	PosSlotEncoder   [4][64]CLzmaProb
	PosEncoders      [128]CLzmaProb
	LenProbs         CLenEnc
	RepLenProbs      CLenEnc
	G_FastPos        [2048]Byte
	LenEnc           CLenPriceEnc
	RepLenEnc        CLenPriceEnc
	Opt              [2048]COptimal
	SaveState        CSaveState
}

func LzmaEnc_SaveState(pp CLzmaEncHandle) {
	var (
		p    *CLzmaEnc   = (*CLzmaEnc)(pp)
		dest *CSaveState = &p.SaveState
	)
	dest.State = p.State
	dest.LenProbs = p.LenProbs
	dest.RepLenProbs = p.RepLenProbs
	cMemcpy(unsafe.Pointer(&dest.Reps[0]), unsafe.Pointer(&p.Reps[0]), int(unsafe.Sizeof([4]UInt32{})))
	cMemcpy(unsafe.Pointer(&dest.PosAlignEncoder[0]), unsafe.Pointer(&p.PosAlignEncoder[0]), int(unsafe.Sizeof([16]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsRep[0]), unsafe.Pointer(&p.IsRep[0]), int(unsafe.Sizeof([12]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsRepG0[0]), unsafe.Pointer(&p.IsRepG0[0]), int(unsafe.Sizeof([12]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsRepG1[0]), unsafe.Pointer(&p.IsRepG1[0]), int(unsafe.Sizeof([12]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsRepG2[0]), unsafe.Pointer(&p.IsRepG2[0]), int(unsafe.Sizeof([12]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsMatch[0][0]), unsafe.Pointer(&p.IsMatch[0][0]), int(unsafe.Sizeof([12][16]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsRep0Long[0][0]), unsafe.Pointer(&p.IsRep0Long[0][0]), int(unsafe.Sizeof([12][16]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.PosSlotEncoder[0][0]), unsafe.Pointer(&p.PosSlotEncoder[0][0]), int(unsafe.Sizeof([4][64]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.PosEncoders[0]), unsafe.Pointer(&p.PosEncoders[0]), int(unsafe.Sizeof([128]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(dest.LitProbs), unsafe.Pointer(p.LitProbs), int((0x300<<p.Lclp)*uint32(unsafe.Sizeof(CLzmaProb(0)))))
}
func LzmaEnc_RestoreState(pp CLzmaEncHandle) {
	var (
		dest *CLzmaEnc   = (*CLzmaEnc)(pp)
		p    *CSaveState = &dest.SaveState
	)
	dest.State = p.State
	dest.LenProbs = p.LenProbs
	dest.RepLenProbs = p.RepLenProbs
	cMemcpy(unsafe.Pointer(&dest.Reps[0]), unsafe.Pointer(&p.Reps[0]), int(unsafe.Sizeof([4]UInt32{})))
	cMemcpy(unsafe.Pointer(&dest.PosAlignEncoder[0]), unsafe.Pointer(&p.PosAlignEncoder[0]), int(unsafe.Sizeof([16]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsRep[0]), unsafe.Pointer(&p.IsRep[0]), int(unsafe.Sizeof([12]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsRepG0[0]), unsafe.Pointer(&p.IsRepG0[0]), int(unsafe.Sizeof([12]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsRepG1[0]), unsafe.Pointer(&p.IsRepG1[0]), int(unsafe.Sizeof([12]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsRepG2[0]), unsafe.Pointer(&p.IsRepG2[0]), int(unsafe.Sizeof([12]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsMatch[0][0]), unsafe.Pointer(&p.IsMatch[0][0]), int(unsafe.Sizeof([12][16]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.IsRep0Long[0][0]), unsafe.Pointer(&p.IsRep0Long[0][0]), int(unsafe.Sizeof([12][16]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.PosSlotEncoder[0][0]), unsafe.Pointer(&p.PosSlotEncoder[0][0]), int(unsafe.Sizeof([4][64]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(&dest.PosEncoders[0]), unsafe.Pointer(&p.PosEncoders[0]), int(unsafe.Sizeof([128]CLzmaProb{})))
	cMemcpy(unsafe.Pointer(dest.LitProbs), unsafe.Pointer(p.LitProbs), int((0x300<<dest.Lclp)*uint32(unsafe.Sizeof(CLzmaProb(0)))))
}
func LzmaEnc_SetProps(pp CLzmaEncHandle, props2 *CLzmaEncProps) SRes {
	var (
		p     *CLzmaEnc     = (*CLzmaEnc)(pp)
		props CLzmaEncProps = *props2
	)
	LzmaEncProps_Normalize(&props)
	if props.Lc > LZMA_LC_MAX || props.Lp > LZMA_LP_MAX || props.Pb > LZMA_PB_MAX || props.DictSize > UInt32(1<<(((9+unsafe.Sizeof(uint32(0))/2)-1)*2+7)) || props.DictSize > (3<<29) {
		return SZ_ERROR_PARAM
	}
	p.DictSize = props.DictSize
	{
		var fb uint32 = uint32(props.Fb)
		if fb < 5 {
			fb = 5
		}
		if fb > uint32(LZMA_MATCH_LEN_MIN+((int32(1<<kLenNumLowBits))*2+(int32(1<<kLenNumHighBits)))-1) {
			fb = uint32(LZMA_MATCH_LEN_MIN + ((int32(1<<kLenNumLowBits))*2 + (int32(1 << kLenNumHighBits))) - 1)
		}
		p.NumFastBytes = fb
	}
	p.Lc = uint32(props.Lc)
	p.Lp = uint32(props.Lp)
	p.Pb = uint32(props.Pb)
	p.FastMode = BoolInt(cBoolToInt(props.Algo == 0))
	if props.BtMode != 0 {
		p.MatchFinderBase.BtMode = 1
	} else {
		p.MatchFinderBase.BtMode = 0
	}
	{
		var numHashBytes uint32 = 4
		if props.BtMode != 0 {
			if props.NumHashBytes < 2 {
				numHashBytes = 2
			} else if props.NumHashBytes < 4 {
				numHashBytes = uint32(props.NumHashBytes)
			}
		}
		p.MatchFinderBase.NumHashBytes = UInt32(numHashBytes)
	}
	p.MatchFinderBase.CutValue = props.Mc
	p.WriteEndMark = BoolInt(props.WriteEndMark)
	return SZ_OK
}
func LzmaEnc_SetDataSize(pp CLzmaEncHandle, expectedDataSiize UInt64) {
	var p *CLzmaEnc = (*CLzmaEnc)(pp)
	p.MatchFinderBase.ExpectedDataSize = expectedDataSiize
}

var kLiteralNextStates [12]Byte = [12]Byte{0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 4, 5}
var kMatchNextStates [12]Byte = [12]Byte{7, 7, 7, 7, 7, 7, 7, 10, 10, 10, 10, 10}
var kRepNextStates [12]Byte = [12]Byte{8, 8, 8, 8, 8, 8, 8, 11, 11, 11, 11, 11}
var kShortRepNextStates [12]Byte = [12]Byte{9, 9, 9, 9, 9, 9, 9, 11, 11, 11, 11, 11}

func RangeEnc_Construct(p *CRangeEnc) {
	p.OutStream = nil
	p.BufBase = nil
}
func RangeEnc_Alloc(p *CRangeEnc, alloc ISzAllocPtr) int32 {
	if p.BufBase == nil {
		p.BufBase = (*Byte)(alloc.Alloc(alloc, 1<<16))
		if p.BufBase == nil {
			return 0
		}
		p.BufLim = (*Byte)(unsafe.Add(unsafe.Pointer(p.BufBase), 1<<16))
	}
	return 1
}
func RangeEnc_Free(p *CRangeEnc, alloc ISzAllocPtr) {
	alloc.Free(alloc, unsafe.Pointer(p.BufBase))
	p.BufBase = nil
}
func RangeEnc_Init(p *CRangeEnc) {
	p.Range = 0xFFFFFFFF
	p.Cache = 0
	p.Low = 0
	p.CacheSize = 0
	p.Buf = p.BufBase
	p.Processed = 0
	p.Res = SZ_OK
}
func RangeEnc_FlushStream(p *CRangeEnc) {
	var num uint32
	if p.Res != SZ_OK {
		return
	}
	num = uint32(int32(uintptr(unsafe.Pointer(p.Buf)) - uintptr(unsafe.Pointer(p.BufBase))))
	if num != p.OutStream.Write(p.OutStream, unsafe.Pointer(p.BufBase), num) {
		p.Res = SZ_ERROR_WRITE
	}
	p.Processed += UInt64(num)
	p.Buf = p.BufBase
}
func RangeEnc_ShiftLow(p *CRangeEnc) {
	var (
		low  UInt32 = UInt32(p.Low)
		high uint32 = uint32(p.Low >> 32)
	)
	p.Low = UInt64(low << 8)
	if low < 0xFF000000 || high != 0 {
		{
			var buf *Byte = p.Buf
			*func() *Byte {
				p_ := &buf
				x := *p_
				*p_ = (*Byte)(unsafe.Add(unsafe.Pointer(*p_), 1))
				return x
			}() = Byte(p.Cache + high)
			p.Cache = uint32(low >> 24)
			p.Buf = buf
			if buf == p.BufLim {
				RangeEnc_FlushStream(p)
			}
			if p.CacheSize == 0 {
				return
			}
		}
		high += 0xFF
		for {
			var buf *Byte = p.Buf
			*func() *Byte {
				p_ := &buf
				x := *p_
				*p_ = (*Byte)(unsafe.Add(unsafe.Pointer(*p_), 1))
				return x
			}() = Byte(high)
			p.Buf = buf
			if buf == p.BufLim {
				RangeEnc_FlushStream(p)
			}
			if func() UInt64 {
				p_ := &p.CacheSize
				*p_--
				return *p_
			}() == 0 {
				return
			}
		}
	}
	p.CacheSize++
}
func RangeEnc_FlushData(p *CRangeEnc) {
	var i int32
	for i = 0; i < 5; i++ {
		RangeEnc_ShiftLow(p)
	}
}
func RangeEnc_EncodeBit_0(p *CRangeEnc, prob *CLzmaProb) {
	var (
		range_   UInt32
		ttt      UInt32
		newBound UInt32
	)
	range_ = p.Range
	ttt = UInt32(*prob)
	newBound = (range_ >> kNumBitModelTotalBits) * ttt
	range_ = newBound
	*prob = CLzmaProb(ttt + UInt32(((int32(1<<kNumBitModelTotalBits))-int32(ttt))>>kNumMoveBits))
	if range_ < UInt32(int32(1<<kNumTopBits)) {
		range_ <<= 8
		RangeEnc_ShiftLow(p)
	}
	p.Range = range_
}
func LitEnc_Encode(p *CRangeEnc, probs *CLzmaProb, sym UInt32) {
	var range_ UInt32 = p.Range
	sym |= 0x100
	for {
		{
			var (
				ttt      UInt32
				newBound UInt32
				prob     *CLzmaProb = (*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(sym>>8)))
				bit      UInt32     = (sym >> 7) & 1
			)
			sym <<= 1
			{
				var mask UInt32
				ttt = UInt32(*prob)
				newBound = (range_ >> kNumBitModelTotalBits) * ttt
				mask = 0 - bit
				range_ &= mask
				mask &= newBound
				range_ -= mask
				p.Low += UInt64(mask)
				mask = bit - 1
				range_ += newBound & mask
				mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
				mask += UInt32((int32(1 << kNumMoveBits)) - 1)
				ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
				*prob = CLzmaProb(ttt)
				if range_ < UInt32(int32(1<<kNumTopBits)) {
					range_ <<= 8
					RangeEnc_ShiftLow(p)
				}
			}
		}
		if sym >= 0x10000 {
			break
		}
	}
	p.Range = range_
}
func LitEnc_EncodeMatched(p *CRangeEnc, probs *CLzmaProb, sym UInt32, matchByte UInt32) {
	var (
		range_ UInt32 = p.Range
		offs   UInt32 = 0x100
	)
	sym |= 0x100
	for {
		{
			var (
				ttt      UInt32
				newBound UInt32
				prob     *CLzmaProb
				bit      UInt32
			)
			matchByte <<= 1
			prob = (*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(offs+(matchByte&offs)+(sym>>8))))
			bit = (sym >> 7) & 1
			sym <<= 1
			offs &= ^(matchByte ^ sym)
			{
				var mask UInt32
				ttt = UInt32(*prob)
				newBound = (range_ >> kNumBitModelTotalBits) * ttt
				mask = 0 - bit
				range_ &= mask
				mask &= newBound
				range_ -= mask
				p.Low += UInt64(mask)
				mask = bit - 1
				range_ += newBound & mask
				mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
				mask += UInt32((int32(1 << kNumMoveBits)) - 1)
				ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
				*prob = CLzmaProb(ttt)
				if range_ < UInt32(int32(1<<kNumTopBits)) {
					range_ <<= 8
					RangeEnc_ShiftLow(p)
				}
			}
		}
		if sym >= 0x10000 {
			break
		}
	}
	p.Range = range_
}
func LzmaEnc_InitPriceTables(ProbPrices *CProbPrice) {
	var i UInt32
	for i = 0; i < UInt32((int32(1<<kNumBitModelTotalBits))>>kNumMoveReducingBits); i++ {
		var (
			kCyclesBits uint32 = kNumBitPriceShiftBits
			w           UInt32 = (i << kNumMoveReducingBits) + UInt32(1<<(int32(kNumMoveReducingBits-1)))
			bitCount    uint32 = 0
			j           uint32
		)
		for j = 0; j < kCyclesBits; j++ {
			w = w * w
			bitCount <<= 1
			for w >= (1 << 16) {
				w >>= 1
				bitCount++
			}
		}
		*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr(i))) = CProbPrice((kNumBitModelTotalBits << kCyclesBits) - 15 - bitCount)
	}
}
func LitEnc_GetPrice(probs *CLzmaProb, sym UInt32, ProbPrices *CProbPrice) UInt32 {
	var price UInt32 = 0
	sym |= 0x100
	for {
		{
			var bit uint32 = uint32(sym & 1)
			sym >>= 1
			price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(sym))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
		}
		if sym < 2 {
			break
		}
	}
	return price
}
func LitEnc_Matched_GetPrice(probs *CLzmaProb, sym UInt32, matchByte UInt32, ProbPrices *CProbPrice) UInt32 {
	var (
		price UInt32 = 0
		offs  UInt32 = 0x100
	)
	sym |= 0x100
	for {
		matchByte <<= 1
		price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(offs+(matchByte&offs)+(sym>>8)))))^uint32((-int32((sym>>7)&1))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
		sym <<= 1
		offs &= ^(matchByte ^ sym)
		if sym >= 0x10000 {
			break
		}
	}
	return price
}
func RcTree_ReverseEncode(rc *CRangeEnc, probs *CLzmaProb, numBits uint32, sym uint32) {
	var (
		range_ UInt32 = rc.Range
		m      uint32 = 1
	)
	for {
		{
			var (
				ttt      UInt32
				newBound UInt32
				bit      uint32 = sym & 1
			)
			sym >>= 1
			{
				var mask UInt32
				ttt = UInt32(*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m)))))
				newBound = (range_ >> kNumBitModelTotalBits) * ttt
				mask = 0 - UInt32(bit)
				range_ &= mask
				mask &= newBound
				range_ -= mask
				rc.Low += UInt64(mask)
				mask = UInt32(bit) - 1
				range_ += newBound & mask
				mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
				mask += UInt32((int32(1 << kNumMoveBits)) - 1)
				ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
				*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m)))) = CLzmaProb(ttt)
				if range_ < UInt32(int32(1<<kNumTopBits)) {
					range_ <<= 8
					RangeEnc_ShiftLow(rc)
				}
			}
			m = (m << 1) | bit
		}
		if func() uint32 {
			p_ := &numBits
			*p_--
			return *p_
		}() == 0 {
			break
		}
	}
	rc.Range = range_
}
func LenEnc_Init(p *CLenEnc) {
	var i uint32
	for i = 0; i < uint32((int32(1<<LZMA_PB_MAX))<<(int32(kLenNumLowBits+1))); i++ {
		p.Low[i] = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
	}
	for i = 0; i < uint32(int32(1<<kLenNumHighBits)); i++ {
		p.High[i] = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
	}
}
func LenEnc_Encode(p *CLenEnc, rc *CRangeEnc, sym uint32, posState uint32) {
	var (
		range_   UInt32
		ttt      UInt32
		newBound UInt32
		probs    *CLzmaProb = &p.Low[0]
	)
	range_ = rc.Range
	ttt = UInt32(*probs)
	newBound = (range_ >> kNumBitModelTotalBits) * ttt
	if sym >= uint32(int32(1<<kLenNumLowBits)) {
		range_ -= newBound
		rc.Low += UInt64(newBound)
		*probs = CLzmaProb(ttt - (ttt >> kNumMoveBits))
		if range_ < UInt32(int32(1<<kNumTopBits)) {
			range_ <<= 8
			RangeEnc_ShiftLow(rc)
		}
		probs = (*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(int32(1<<kLenNumLowBits))))
		ttt = UInt32(*probs)
		newBound = (range_ >> kNumBitModelTotalBits) * ttt
		if sym >= uint32((int32(1<<kLenNumLowBits))*2) {
			range_ -= newBound
			rc.Low += UInt64(newBound)
			*probs = CLzmaProb(ttt - (ttt >> kNumMoveBits))
			if range_ < UInt32(int32(1<<kNumTopBits)) {
				range_ <<= 8
				RangeEnc_ShiftLow(rc)
			}
			rc.Range = range_
			LitEnc_Encode(rc, &p.High[0], UInt32(sym-uint32((int32(1<<kLenNumLowBits))*2)))
			return
		}
		sym -= uint32(int32(1 << kLenNumLowBits))
	}
	{
		var (
			m   uint32
			bit uint32
		)
		range_ = newBound
		*probs = CLzmaProb(ttt + UInt32(((int32(1<<kNumBitModelTotalBits))-int32(ttt))>>kNumMoveBits))
		if range_ < UInt32(int32(1<<kNumTopBits)) {
			range_ <<= 8
			RangeEnc_ShiftLow(rc)
		}
		probs = (*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(posState<<uint32(int32(kLenNumLowBits+1)))))
		bit = sym >> 2
		{
			var mask UInt32
			ttt = UInt32(*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*1))))
			newBound = (range_ >> kNumBitModelTotalBits) * ttt
			mask = 0 - UInt32(bit)
			range_ &= mask
			mask &= newBound
			range_ -= mask
			rc.Low += UInt64(mask)
			mask = UInt32(bit) - 1
			range_ += newBound & mask
			mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
			mask += UInt32((int32(1 << kNumMoveBits)) - 1)
			ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
			*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*1))) = CLzmaProb(ttt)
			if range_ < UInt32(int32(1<<kNumTopBits)) {
				range_ <<= 8
				RangeEnc_ShiftLow(rc)
			}
		}
		m = bit + (1 << 1)
		bit = (sym >> 1) & 1
		{
			var mask UInt32
			ttt = UInt32(*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m)))))
			newBound = (range_ >> kNumBitModelTotalBits) * ttt
			mask = 0 - UInt32(bit)
			range_ &= mask
			mask &= newBound
			range_ -= mask
			rc.Low += UInt64(mask)
			mask = UInt32(bit) - 1
			range_ += newBound & mask
			mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
			mask += UInt32((int32(1 << kNumMoveBits)) - 1)
			ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
			*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m)))) = CLzmaProb(ttt)
			if range_ < UInt32(int32(1<<kNumTopBits)) {
				range_ <<= 8
				RangeEnc_ShiftLow(rc)
			}
		}
		m = (m << 1) + bit
		bit = sym & 1
		{
			var mask UInt32
			ttt = UInt32(*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m)))))
			newBound = (range_ >> kNumBitModelTotalBits) * ttt
			mask = 0 - UInt32(bit)
			range_ &= mask
			mask &= newBound
			range_ -= mask
			rc.Low += UInt64(mask)
			mask = UInt32(bit) - 1
			range_ += newBound & mask
			mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
			mask += UInt32((int32(1 << kNumMoveBits)) - 1)
			ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
			*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m)))) = CLzmaProb(ttt)
			if range_ < UInt32(int32(1<<kNumTopBits)) {
				range_ <<= 8
				RangeEnc_ShiftLow(rc)
			}
		}
		rc.Range = range_
	}
}
func SetPrices_3(probs *CLzmaProb, startPrice UInt32, prices *UInt32, ProbPrices *CProbPrice) {
	var i uint32
	for i = 0; i < 8; i += 2 {
		var (
			price UInt32 = startPrice
			prob  UInt32
		)
		price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*1)))^uint32((-int32(i>>2))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
		price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr((i>>2)+2))))^uint32((-int32((i>>1)&1))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
		prob = UInt32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr((i>>1)+4))))
		*(*UInt32)(unsafe.Add(unsafe.Pointer(prices), unsafe.Sizeof(UInt32(0))*uintptr(i))) = price + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr(prob>>kNumMoveReducingBits))))
		*(*UInt32)(unsafe.Add(unsafe.Pointer(prices), unsafe.Sizeof(UInt32(0))*uintptr(i+1))) = price + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((prob^UInt32((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits))))
	}
}
func LenPriceEnc_UpdateTables(p *CLenPriceEnc, numPosStates uint32, enc *CLenEnc, ProbPrices *CProbPrice) {
	var b UInt32
	{
		var (
			prob     uint32 = uint32(enc.Low[0])
			a        UInt32
			c        UInt32
			posState uint32
		)
		b = UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((prob^uint32((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits))))
		a = UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr(prob>>kNumMoveReducingBits))))
		c = b + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((enc.Low[int32(1<<kLenNumLowBits)])>>kNumMoveReducingBits))))
		for posState = 0; posState < numPosStates; posState++ {
			var (
				prices *UInt32    = &p.Prices[posState][0]
				probs  *CLzmaProb = &enc.Low[posState<<uint32(int32(kLenNumLowBits+1))]
			)
			SetPrices_3(probs, a, prices, ProbPrices)
			SetPrices_3((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(int32(1<<kLenNumLowBits)))), c, (*UInt32)(unsafe.Add(unsafe.Pointer(prices), unsafe.Sizeof(UInt32(0))*uintptr(int32(1<<kLenNumLowBits)))), ProbPrices)
		}
	}
	{
		var i uint32 = p.TableSize
		if i > uint32((int32(1<<kLenNumLowBits))*2) {
			var (
				probs  *CLzmaProb = &enc.High[0]
				prices *UInt32    = &p.Prices[0][(int32(1<<kLenNumLowBits))*2]
			)
			i -= uint32((int32(1<<kLenNumLowBits))*2 - 1)
			i >>= 1
			b += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((int32(enc.Low[int32(1<<kLenNumLowBits)])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits))))
			for {
				{
					var (
						sym uint32 = func() uint32 {
							p_ := &i
							*p_--
							return *p_
						}() + uint32(1<<(int32(kLenNumHighBits-1)))
						price UInt32 = b
					)
					for {
						{
							var bit uint32 = sym & 1
							sym >>= 1
							price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(sym))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
						}
						if sym < 2 {
							break
						}
					}
					{
						var prob uint32 = uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(i+uint32(1<<(int32(kLenNumHighBits-1)))))))
						*(*UInt32)(unsafe.Add(unsafe.Pointer(prices), unsafe.Sizeof(UInt32(0))*uintptr(i*2))) = price + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr(prob>>kNumMoveReducingBits))))
						*(*UInt32)(unsafe.Add(unsafe.Pointer(prices), unsafe.Sizeof(UInt32(0))*uintptr(i*2+1))) = price + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((prob^uint32((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits))))
					}
				}
				if i == 0 {
					break
				}
			}
			{
				var (
					posState uint32
					num      uint32 = (p.TableSize - uint32((int32(1<<kLenNumLowBits))*2)) * uint32(unsafe.Sizeof(UInt32(0)))
				)
				for posState = 1; posState < numPosStates; posState++ {
					cMemcpy(unsafe.Pointer(&p.Prices[posState][(int32(1<<kLenNumLowBits))*2]), unsafe.Pointer(&p.Prices[0][(int32(1<<kLenNumLowBits))*2]), int(num))
				}
			}
		}
	}
}
func ReadMatchDistances(p *CLzmaEnc, numPairsRes *uint32) uint32 {
	var numPairs uint32
	p.AdditionalOffset++
	p.NumAvail = p.MatchFinder.GetNumAvailableBytes(p.MatchFinderObj)
	numPairs = uint32(p.MatchFinder.GetMatches(p.MatchFinderObj, &p.Matches[0]))
	*numPairsRes = numPairs
	if numPairs == 0 {
		return 0
	}
	{
		var len_ uint32 = uint32(p.Matches[numPairs-2])
		if len_ != p.NumFastBytes {
			return len_
		}
		{
			var numAvail UInt32 = p.NumAvail
			if numAvail > UInt32(LZMA_MATCH_LEN_MIN+((int32(1<<kLenNumLowBits))*2+(int32(1<<kLenNumHighBits)))-1) {
				numAvail = UInt32(LZMA_MATCH_LEN_MIN + ((int32(1<<kLenNumLowBits))*2 + (int32(1 << kLenNumHighBits))) - 1)
			}
			{
				var (
					p1  *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(p.MatchFinder.GetPointerToCurrentPos(p.MatchFinderObj)), -1))
					p2  *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(p1), len_))
					dif int32 = int32(4294967295 - p.Matches[numPairs-1])
					lim *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(p1), numAvail))
				)
				for ; p2 != lim && *p2 == *(*Byte)(unsafe.Add(unsafe.Pointer(p2), dif)); p2 = (*Byte)(unsafe.Add(unsafe.Pointer(p2), 1)) {
				}
				return uint32(int32(uintptr(unsafe.Pointer(p2)) - uintptr(unsafe.Pointer(p1))))
			}
		}
	}
}
func GetPrice_PureRep(p *CLzmaEnc, repIndex uint32, state uint32, posState uint32) UInt32 {
	var (
		price UInt32
		prob  UInt32 = UInt32(p.IsRepG0[state])
	)
	if repIndex == 0 {
		price = UInt32(p.ProbPrices[prob>>kNumMoveReducingBits])
		price += UInt32(p.ProbPrices[(int32(p.IsRep0Long[state][posState])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits])
	} else {
		price = UInt32(p.ProbPrices[(prob^UInt32((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits])
		prob = UInt32(p.IsRepG1[state])
		if repIndex == 1 {
			price += UInt32(p.ProbPrices[prob>>kNumMoveReducingBits])
		} else {
			price += UInt32(p.ProbPrices[(prob^UInt32((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits])
			price += UInt32(p.ProbPrices[(uint32(p.IsRepG2[state])^uint32((-int32(repIndex-2))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits])
		}
	}
	return price
}
func Backward(p *CLzmaEnc, cur uint32) uint32 {
	var wr uint32 = cur + 1
	p.OptEnd = wr
	for {
		var (
			dist  UInt32 = p.Opt[cur].Dist
			len_  uint32 = uint32(p.Opt[cur].Len)
			extra uint32 = uint32(p.Opt[cur].Extra)
		)
		cur -= len_
		if extra != 0 {
			wr--
			p.Opt[wr].Len = UInt32(len_)
			cur -= extra
			len_ = extra
			if extra == 1 {
				p.Opt[wr].Dist = dist
				dist = 4294967295
			} else {
				p.Opt[wr].Dist = 0
				len_--
				wr--
				p.Opt[wr].Dist = 4294967295
				p.Opt[wr].Len = 1
			}
		}
		if cur == 0 {
			p.BackRes = dist
			p.OptCur = wr
			return len_
		}
		wr--
		p.Opt[wr].Dist = dist
		p.Opt[wr].Len = UInt32(len_)
	}
}
func GetOptimum(p *CLzmaEnc, position UInt32) uint32 {
	var (
		last    uint32
		cur     uint32
		reps    [4]UInt32
		repLens [4]uint32
		matches *UInt32
	)
	{
		var (
			numAvail      UInt32
			numPairs      uint32
			mainLen       uint32
			repMaxIndex   uint32
			i             uint32
			posState      uint32
			matchPrice    UInt32
			repMatchPrice UInt32
			data          *Byte
			curByte       Byte
			matchByte     Byte
		)
		p.OptCur = func() uint32 {
			p_ := &p.OptEnd
			*p_ = 0
			return *p_
		}()
		if p.AdditionalOffset == 0 {
			mainLen = ReadMatchDistances(p, &numPairs)
		} else {
			mainLen = p.LongestMatchLen
			numPairs = p.NumPairs
		}
		numAvail = p.NumAvail
		if numAvail < 2 {
			p.BackRes = 4294967295
			return 1
		}
		if numAvail > UInt32(LZMA_MATCH_LEN_MIN+((int32(1<<kLenNumLowBits))*2+(int32(1<<kLenNumHighBits)))-1) {
			numAvail = UInt32(LZMA_MATCH_LEN_MIN + ((int32(1<<kLenNumLowBits))*2 + (int32(1 << kLenNumHighBits))) - 1)
		}
		data = (*Byte)(unsafe.Add(unsafe.Pointer(p.MatchFinder.GetPointerToCurrentPos(p.MatchFinderObj)), -1))
		repMaxIndex = 0
		for i = 0; i < LZMA_NUM_REPS; i++ {
			var (
				len_  uint32
				data2 *Byte
			)
			reps[i] = p.Reps[i]
			data2 = (*Byte)(unsafe.Add(unsafe.Pointer(data), -int(reps[i])))
			if *data != *data2 || *(*Byte)(unsafe.Add(unsafe.Pointer(data), 1)) != *(*Byte)(unsafe.Add(unsafe.Pointer(data2), 1)) {
				repLens[i] = 0
				continue
			}
			for len_ = 2; len_ < uint32(numAvail) && *(*Byte)(unsafe.Add(unsafe.Pointer(data), len_)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len_)); len_++ {
			}
			repLens[i] = len_
			if len_ > repLens[repMaxIndex] {
				repMaxIndex = i
			}
		}
		if repLens[repMaxIndex] >= p.NumFastBytes {
			var len_ uint32
			p.BackRes = UInt32(repMaxIndex)
			len_ = repLens[repMaxIndex]
			p.AdditionalOffset += len_ - 1
			p.MatchFinder.Skip(p.MatchFinderObj, UInt32(len_-1))
			return len_
		}
		matches = &p.Matches[0]
		if mainLen >= p.NumFastBytes {
			p.BackRes = *(*UInt32)(unsafe.Add(unsafe.Pointer(matches), unsafe.Sizeof(UInt32(0))*uintptr(numPairs-1))) + LZMA_NUM_REPS
			p.AdditionalOffset += mainLen - 1
			p.MatchFinder.Skip(p.MatchFinderObj, UInt32(mainLen-1))
			return mainLen
		}
		curByte = *data
		matchByte = *((*Byte)(unsafe.Add(unsafe.Pointer(data), -int(reps[0]))))
		last = repLens[repMaxIndex]
		if last <= mainLen {
			last = mainLen
		}
		if last < 2 && curByte != matchByte {
			p.BackRes = 4294967295
			return 1
		}
		p.Opt[0].State = CState(p.State)
		posState = uint32(position & UInt32(p.PbMask))
		{
			var probs *CLzmaProb = ((*CLzmaProb)(unsafe.Add(unsafe.Pointer(p.LitProbs), unsafe.Sizeof(CLzmaProb(0))*uintptr(((((position<<8)+UInt32(*((*Byte)(unsafe.Add(unsafe.Pointer(data), -1)))))&UInt32(p.LpMask))<<UInt32(p.Lc))*3))))
			p.Opt[1].Price = UInt32(p.ProbPrices[(p.IsMatch[p.State][posState])>>kNumMoveReducingBits] + CProbPrice(func() UInt32 {
				if p.State >= 7 {
					return LitEnc_Matched_GetPrice(probs, UInt32(curByte), UInt32(matchByte), &p.ProbPrices[0])
				}
				return LitEnc_GetPrice(probs, UInt32(curByte), &p.ProbPrices[0])
			}()))
		}
		{
			p.Opt[1].Dist = 4294967295
			p.Opt[1].Extra = 0
		}
		matchPrice = UInt32(p.ProbPrices[(int32(p.IsMatch[p.State][posState])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits])
		repMatchPrice = matchPrice + UInt32(p.ProbPrices[(int32(p.IsRep[p.State])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits])
		if matchByte == curByte && repLens[0] == 0 {
			var shortRepPrice UInt32 = repMatchPrice + UInt32(p.ProbPrices[(p.IsRepG0[p.State])>>kNumMoveReducingBits]+p.ProbPrices[(p.IsRep0Long[p.State][posState])>>kNumMoveReducingBits])
			if shortRepPrice < p.Opt[1].Price {
				p.Opt[1].Price = shortRepPrice
				p.Opt[1].Dist = 0
				p.Opt[1].Extra = 0
			}
			if last < 2 {
				p.BackRes = p.Opt[1].Dist
				return 1
			}
		}
		p.Opt[1].Len = 1
		p.Opt[0].Reps[0] = reps[0]
		p.Opt[0].Reps[1] = reps[1]
		p.Opt[0].Reps[2] = reps[2]
		p.Opt[0].Reps[3] = reps[3]
		for i = 0; i < LZMA_NUM_REPS; i++ {
			var (
				repLen uint32 = repLens[i]
				price  UInt32
			)
			if repLen < 2 {
				continue
			}
			price = repMatchPrice + GetPrice_PureRep(p, i, p.State, posState)
			for {
				{
					var (
						price2 UInt32    = price + (p.RepLenEnc.Prices[posState][repLen-LZMA_MATCH_LEN_MIN])
						opt    *COptimal = &p.Opt[repLen]
					)
					if price2 < opt.Price {
						opt.Price = price2
						opt.Len = UInt32(repLen)
						opt.Dist = UInt32(i)
						opt.Extra = 0
					}
				}
				if func() uint32 {
					p_ := &repLen
					*p_--
					return *p_
				}() < 2 {
					break
				}
			}
		}
		{
			var len_ uint32 = repLens[0] + 1
			if len_ <= mainLen {
				var (
					offs             uint32 = 0
					normalMatchPrice UInt32 = matchPrice + UInt32(p.ProbPrices[(p.IsRep[p.State])>>kNumMoveReducingBits])
				)
				if len_ < 2 {
					len_ = 2
				} else {
					for len_ > uint32(*(*UInt32)(unsafe.Add(unsafe.Pointer(matches), unsafe.Sizeof(UInt32(0))*uintptr(offs)))) {
						offs += 2
					}
				}
				for ; ; len_++ {
					var (
						opt           *COptimal
						dist          UInt32 = *(*UInt32)(unsafe.Add(unsafe.Pointer(matches), unsafe.Sizeof(UInt32(0))*uintptr(offs+1)))
						price         UInt32 = normalMatchPrice + (p.LenEnc.Prices[posState][len_-LZMA_MATCH_LEN_MIN])
						lenToPosState uint32 = (func() uint32 {
							if len_ < uint32(int32(kNumLenToPosStates+1)) {
								return len_ - 2
							}
							return uint32(int32(kNumLenToPosStates - 1))
						}())
					)
					if dist < UInt32(1<<(int32(kEndPosModelIndex>>1))) {
						price += p.DistancesPrices[lenToPosState][dist&UInt32((1<<(int32(kEndPosModelIndex>>1)))-1)]
					} else {
						var slot uint32
						{
							var zz uint32
							if dist < UInt32(1<<((9+unsafe.Sizeof(uint32(0))/2)+6)) {
								zz = 6
							} else {
								zz = uint32(6 + (9 + unsafe.Sizeof(uint32(0))/2) - 1)
							}
							slot = uint32(p.G_FastPos[dist>>UInt32(zz)]) + zz*2
						}
						price += p.AlignPrices[dist&UInt32((int32(1<<kNumAlignBits))-1)]
						price += p.PosSlotPrices[lenToPosState][slot]
					}
					opt = &p.Opt[len_]
					if price < opt.Price {
						opt.Price = price
						opt.Len = UInt32(len_)
						opt.Dist = dist + LZMA_NUM_REPS
						opt.Extra = 0
					}
					if len_ == uint32(*(*UInt32)(unsafe.Add(unsafe.Pointer(matches), unsafe.Sizeof(UInt32(0))*uintptr(offs)))) {
						offs += 2
						if offs == numPairs {
							break
						}
					}
				}
			}
		}
		cur = 0
	}
	for {
		var (
			numAvail      uint32
			numAvailFull  UInt32
			newLen        uint32
			numPairs      uint32
			prev          uint32
			state         uint32
			posState      uint32
			startLen      uint32
			litPrice      UInt32
			matchPrice    UInt32
			repMatchPrice UInt32
			nextIsLit     BoolInt
			curByte       Byte
			matchByte     Byte
			data          *Byte
			curOpt        *COptimal
			nextOpt       *COptimal
		)
		if func() uint32 {
			p_ := &cur
			*p_++
			return *p_
		}() == last {
			break
		}
		if cur >= (1<<11)-64 {
			var (
				j     uint32
				best  uint32
				price UInt32 = p.Opt[cur].Price
			)
			best = cur
			for j = cur + 1; j <= last; j++ {
				var price2 UInt32 = p.Opt[j].Price
				if price >= price2 {
					price = price2
					best = j
				}
			}
			{
				var delta uint32 = best - cur
				if delta != 0 {
					p.AdditionalOffset += delta
					p.MatchFinder.Skip(p.MatchFinderObj, UInt32(delta))
				}
			}
			cur = best
			break
		}
		newLen = ReadMatchDistances(p, &numPairs)
		if newLen >= p.NumFastBytes {
			p.NumPairs = numPairs
			p.LongestMatchLen = newLen
			break
		}
		curOpt = &p.Opt[cur]
		position++
		prev = cur - uint32(curOpt.Len)
		if curOpt.Len == 1 {
			state = uint32(p.Opt[prev].State)
			if curOpt.Dist == 0 {
				state = uint32(kShortRepNextStates[state])
			} else {
				state = uint32(kLiteralNextStates[state])
			}
		} else {
			var (
				prevOpt *COptimal
				b0      UInt32
				dist    UInt32 = curOpt.Dist
			)
			if curOpt.Extra != 0 {
				prev -= uint32(curOpt.Extra)
				state = kState_RepAfterLit
				if curOpt.Extra == 1 {
					if dist < LZMA_NUM_REPS {
						state = kState_RepAfterLit
					} else {
						state = kState_MatchAfterLit
					}
				}
			} else {
				state = uint32(p.Opt[prev].State)
				if dist < LZMA_NUM_REPS {
					state = uint32(kRepNextStates[state])
				} else {
					state = uint32(kMatchNextStates[state])
				}
			}
			prevOpt = &p.Opt[prev]
			b0 = prevOpt.Reps[0]
			if dist < LZMA_NUM_REPS {
				if dist == 0 {
					reps[0] = b0
					reps[1] = prevOpt.Reps[1]
					reps[2] = prevOpt.Reps[2]
					reps[3] = prevOpt.Reps[3]
				} else {
					reps[1] = b0
					b0 = prevOpt.Reps[1]
					if dist == 1 {
						reps[0] = b0
						reps[2] = prevOpt.Reps[2]
						reps[3] = prevOpt.Reps[3]
					} else {
						reps[2] = b0
						reps[0] = prevOpt.Reps[dist]
						reps[3] = prevOpt.Reps[dist^1]
					}
				}
			} else {
				reps[0] = dist - LZMA_NUM_REPS + 1
				reps[1] = b0
				reps[2] = prevOpt.Reps[1]
				reps[3] = prevOpt.Reps[2]
			}
		}
		curOpt.State = CState(state)
		curOpt.Reps[0] = reps[0]
		curOpt.Reps[1] = reps[1]
		curOpt.Reps[2] = reps[2]
		curOpt.Reps[3] = reps[3]
		data = (*Byte)(unsafe.Add(unsafe.Pointer(p.MatchFinder.GetPointerToCurrentPos(p.MatchFinderObj)), -1))
		curByte = *data
		matchByte = *((*Byte)(unsafe.Add(unsafe.Pointer(data), -int(reps[0]))))
		posState = uint32(position & UInt32(p.PbMask))
		{
			var (
				curPrice UInt32 = curOpt.Price
				prob     uint32 = uint32(p.IsMatch[state][posState])
			)
			matchPrice = curPrice + UInt32(p.ProbPrices[(prob^uint32((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits])
			litPrice = curPrice + UInt32(p.ProbPrices[prob>>kNumMoveReducingBits])
		}
		nextOpt = &p.Opt[cur+1]
		nextIsLit = False
		if nextOpt.Price < (1<<30) && matchByte == curByte || litPrice > nextOpt.Price {
			litPrice = 0
		} else {
			var probs *CLzmaProb = ((*CLzmaProb)(unsafe.Add(unsafe.Pointer(p.LitProbs), unsafe.Sizeof(CLzmaProb(0))*uintptr(((((position<<8)+UInt32(*((*Byte)(unsafe.Add(unsafe.Pointer(data), -1)))))&UInt32(p.LpMask))<<UInt32(p.Lc))*3))))
			if state >= 7 {
				litPrice += LitEnc_Matched_GetPrice(probs, UInt32(curByte), UInt32(matchByte), &p.ProbPrices[0])
			} else {
				litPrice += LitEnc_GetPrice(probs, UInt32(curByte), &p.ProbPrices[0])
			}
			if litPrice < nextOpt.Price {
				nextOpt.Price = litPrice
				nextOpt.Len = 1
				nextOpt.Dist = 4294967295
				nextOpt.Extra = 0
				nextIsLit = True
			}
		}
		repMatchPrice = matchPrice + UInt32(p.ProbPrices[(int32(p.IsRep[state])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits])
		numAvailFull = p.NumAvail
		{
			var temp uint32 = (1 << 11) - 1 - cur
			if numAvailFull > UInt32(temp) {
				numAvailFull = UInt32(temp)
			}
		}
		if state < 7 {
			if matchByte == curByte {
				if repMatchPrice < nextOpt.Price {
					if nextOpt.Len < 2 || nextOpt.Dist != 0 {
						var shortRepPrice UInt32 = repMatchPrice + UInt32(p.ProbPrices[(p.IsRepG0[state])>>kNumMoveReducingBits]+p.ProbPrices[(p.IsRep0Long[state][posState])>>kNumMoveReducingBits])
						if shortRepPrice < nextOpt.Price {
							nextOpt.Price = shortRepPrice
							nextOpt.Len = 1
							nextOpt.Dist = 0
							nextOpt.Extra = 0
							nextIsLit = False
						}
					}
				}
			}
		}
		if numAvailFull < 2 {
			continue
		}
		if numAvailFull <= UInt32(p.NumFastBytes) {
			numAvail = uint32(numAvailFull)
		} else {
			numAvail = p.NumFastBytes
		}
		if nextIsLit == 0 && litPrice != 0 && matchByte != curByte && numAvailFull > 2 {
			var data2 *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(data), -int(reps[0])))
			if *(*Byte)(unsafe.Add(unsafe.Pointer(data), 1)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), 1)) && *(*Byte)(unsafe.Add(unsafe.Pointer(data), 2)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), 2)) {
				var (
					len_  uint32
					limit uint32 = p.NumFastBytes + 1
				)
				if limit > uint32(numAvailFull) {
					limit = uint32(numAvailFull)
				}
				for len_ = 3; len_ < limit && *(*Byte)(unsafe.Add(unsafe.Pointer(data), len_)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len_)); len_++ {
				}
				{
					var (
						state2    uint32 = uint32(kLiteralNextStates[state])
						posState2 uint32 = uint32((position + 1) & UInt32(p.PbMask))
						price     UInt32 = litPrice + UInt32(p.ProbPrices[(int32(p.IsMatch[state2][posState2])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits]+p.ProbPrices[(int32(p.IsRep0Long[state2][posState2])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits]) + UInt32(p.ProbPrices[(int32(p.IsRep[state2])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits]) + UInt32(p.ProbPrices[(p.IsRepG0[state2])>>kNumMoveReducingBits])
					)
					{
						var offset uint32 = cur + len_
						if last < offset {
							last = offset
						}
						{
							var (
								price2 UInt32
								opt    *COptimal
							)
							len_--
							price2 = price + (p.RepLenEnc.Prices[posState2][len_-LZMA_MATCH_LEN_MIN])
							opt = &p.Opt[offset]
							if price2 < opt.Price {
								opt.Price = price2
								opt.Len = UInt32(len_)
								opt.Dist = 0
								opt.Extra = 1
							}
						}
					}
				}
			}
		}
		startLen = 2
		{
			var repIndex uint32 = 0
			for ; repIndex < LZMA_NUM_REPS; repIndex++ {
				var (
					len_  uint32
					price UInt32
					data2 *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(data), -int(reps[repIndex])))
				)
				if *data != *data2 || *(*Byte)(unsafe.Add(unsafe.Pointer(data), 1)) != *(*Byte)(unsafe.Add(unsafe.Pointer(data2), 1)) {
					continue
				}
				for len_ = 2; len_ < numAvail && *(*Byte)(unsafe.Add(unsafe.Pointer(data), len_)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len_)); len_++ {
				}
				{
					var offset uint32 = cur + len_
					if last < offset {
						last = offset
					}
				}
				{
					var len2 uint32 = len_
					price = repMatchPrice + GetPrice_PureRep(p, repIndex, state, posState)
					for {
						{
							var (
								price2 UInt32    = price + (p.RepLenEnc.Prices[posState][len2-LZMA_MATCH_LEN_MIN])
								opt    *COptimal = &p.Opt[cur+len2]
							)
							if price2 < opt.Price {
								opt.Price = price2
								opt.Len = UInt32(len2)
								opt.Dist = UInt32(repIndex)
								opt.Extra = 0
							}
						}
						if func() uint32 {
							p_ := &len2
							*p_--
							return *p_
						}() < 2 {
							break
						}
					}
				}
				if repIndex == 0 {
					startLen = len_ + 1
				}
				{
					var (
						len2  uint32 = len_ + 1
						limit uint32 = len2 + p.NumFastBytes
					)
					if limit > uint32(numAvailFull) {
						limit = uint32(numAvailFull)
					}
					len2 += 2
					if len2 <= limit {
						if *(*Byte)(unsafe.Add(unsafe.Pointer(data), len2-2)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len2-2)) {
							if *(*Byte)(unsafe.Add(unsafe.Pointer(data), len2-1)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len2-1)) {
								var (
									state2    uint32 = uint32(kRepNextStates[state])
									posState2 uint32 = uint32((position + UInt32(len_)) & UInt32(p.PbMask))
								)
								price += (p.RepLenEnc.Prices[posState][len_-LZMA_MATCH_LEN_MIN]) + UInt32(p.ProbPrices[(p.IsMatch[state2][posState2])>>kNumMoveReducingBits]) + LitEnc_Matched_GetPrice((*CLzmaProb)(unsafe.Add(unsafe.Pointer(p.LitProbs), unsafe.Sizeof(CLzmaProb(0))*uintptr((((((position+UInt32(len_))<<8)+UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(data), len_-1))))&UInt32(p.LpMask))<<UInt32(p.Lc))*3))), UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(data), len_))), UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(data2), len_))), &p.ProbPrices[0])
								state2 = kState_LitAfterRep
								posState2 = (posState2 + 1) & p.PbMask
								price += UInt32((p.ProbPrices[(int32(p.IsMatch[state2][posState2])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits] + p.ProbPrices[(int32(p.IsRep0Long[state2][posState2])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits]) + p.ProbPrices[(int32(p.IsRep[state2])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits] + p.ProbPrices[(p.IsRepG0[state2])>>kNumMoveReducingBits])
								for ; len2 < limit && *(*Byte)(unsafe.Add(unsafe.Pointer(data), len2)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len2)); len2++ {
								}
								len2 -= len_
								{
									var offset uint32 = cur + len_ + len2
									if last < offset {
										last = offset
									}
									{
										var (
											price2 UInt32
											opt    *COptimal
										)
										len2--
										price2 = price + (p.RepLenEnc.Prices[posState2][len2-LZMA_MATCH_LEN_MIN])
										opt = &p.Opt[offset]
										if price2 < opt.Price {
											opt.Price = price2
											opt.Len = UInt32(len2)
											opt.Extra = CExtra(len_ + 1)
											opt.Dist = UInt32(repIndex)
										}
									}
								}
							}
						}
					}
				}
			}
		}
		if newLen > numAvail {
			newLen = numAvail
			for numPairs = 0; newLen > uint32(*(*UInt32)(unsafe.Add(unsafe.Pointer(matches), unsafe.Sizeof(UInt32(0))*uintptr(numPairs)))); numPairs += 2 {
			}
			*(*UInt32)(unsafe.Add(unsafe.Pointer(matches), unsafe.Sizeof(UInt32(0))*uintptr(numPairs))) = UInt32(newLen)
			numPairs += 2
		}
		if newLen >= startLen {
			var (
				normalMatchPrice UInt32 = matchPrice + UInt32(p.ProbPrices[(p.IsRep[state])>>kNumMoveReducingBits])
				dist             UInt32
				offs             uint32
				posSlot          uint32
				len_             uint32
			)
			{
				var offset uint32 = cur + newLen
				if last < offset {
					last = offset
				}
			}
			offs = 0
			for startLen > uint32(*(*UInt32)(unsafe.Add(unsafe.Pointer(matches), unsafe.Sizeof(UInt32(0))*uintptr(offs)))) {
				offs += 2
			}
			dist = *(*UInt32)(unsafe.Add(unsafe.Pointer(matches), unsafe.Sizeof(UInt32(0))*uintptr(offs+1)))
			{
				var zz uint32
				if dist < UInt32(1<<((9+unsafe.Sizeof(uint32(0))/2)+6)) {
					zz = 6
				} else {
					zz = uint32(6 + (9 + unsafe.Sizeof(uint32(0))/2) - 1)
				}
				posSlot = uint32(p.G_FastPos[dist>>UInt32(zz)]) + zz*2
			}
			for len_ = startLen; ; len_++ {
				var price UInt32 = normalMatchPrice + (p.LenEnc.Prices[posState][len_-LZMA_MATCH_LEN_MIN])
				{
					var (
						opt     *COptimal
						lenNorm uint32 = len_ - 2
					)
					// GetLenToPosState2: keep lenNorm when it is already below the table.
					if lenNorm >= uint32(int32(kNumLenToPosStates-1)) {
						lenNorm = uint32(int32(kNumLenToPosStates - 1))
					}
					if dist < UInt32(1<<(int32(kEndPosModelIndex>>1))) {
						price += p.DistancesPrices[lenNorm][dist&UInt32((1<<(int32(kEndPosModelIndex>>1)))-1)]
					} else {
						price += p.PosSlotPrices[lenNorm][posSlot] + p.AlignPrices[dist&UInt32((int32(1<<kNumAlignBits))-1)]
					}
					opt = &p.Opt[cur+len_]
					if price < opt.Price {
						opt.Price = price
						opt.Len = UInt32(len_)
						opt.Dist = dist + LZMA_NUM_REPS
						opt.Extra = 0
					}
				}
				if len_ == uint32(*(*UInt32)(unsafe.Add(unsafe.Pointer(matches), unsafe.Sizeof(UInt32(0))*uintptr(offs)))) {
					var (
						data2 *Byte  = (*Byte)(unsafe.Add(unsafe.Pointer((*Byte)(unsafe.Add(unsafe.Pointer(data), -int(dist)))), -1))
						len2  uint32 = len_ + 1
						limit uint32 = len2 + p.NumFastBytes
					)
					if limit > uint32(numAvailFull) {
						limit = uint32(numAvailFull)
					}
					len2 += 2
					if len2 <= limit {
						if *(*Byte)(unsafe.Add(unsafe.Pointer(data), len2-2)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len2-2)) {
							if *(*Byte)(unsafe.Add(unsafe.Pointer(data), len2-1)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len2-1)) {
								for ; len2 < limit && *(*Byte)(unsafe.Add(unsafe.Pointer(data), len2)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len2)); len2++ {
								}
								len2 -= len_
								{
									var (
										state2    uint32 = uint32(kMatchNextStates[state])
										posState2 uint32 = uint32((position + UInt32(len_)) & UInt32(p.PbMask))
										offset    uint32
									)
									price += UInt32(p.ProbPrices[(p.IsMatch[state2][posState2])>>kNumMoveReducingBits])
									price += LitEnc_Matched_GetPrice((*CLzmaProb)(unsafe.Add(unsafe.Pointer(p.LitProbs), unsafe.Sizeof(CLzmaProb(0))*uintptr((((((position+UInt32(len_))<<8)+UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(data), len_-1))))&UInt32(p.LpMask))<<UInt32(p.Lc))*3))), UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(data), len_))), UInt32(*(*Byte)(unsafe.Add(unsafe.Pointer(data2), len_))), &p.ProbPrices[0])
									state2 = kState_LitAfterMatch
									posState2 = (posState2 + 1) & p.PbMask
									price += UInt32((p.ProbPrices[(int32(p.IsMatch[state2][posState2])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits] + p.ProbPrices[(int32(p.IsRep0Long[state2][posState2])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits]) + p.ProbPrices[(int32(p.IsRep[state2])^((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits] + p.ProbPrices[(p.IsRepG0[state2])>>kNumMoveReducingBits])
									offset = cur + len_ + len2
									if last < offset {
										last = offset
									}
									{
										var (
											price2 UInt32
											opt    *COptimal
										)
										len2--
										price2 = price + (p.RepLenEnc.Prices[posState2][len2-LZMA_MATCH_LEN_MIN])
										opt = &p.Opt[offset]
										if price2 < opt.Price {
											opt.Price = price2
											opt.Len = UInt32(len2)
											opt.Extra = CExtra(len_ + 1)
											opt.Dist = dist + LZMA_NUM_REPS
										}
									}
								}
							}
						}
					}
					offs += 2
					if offs == numPairs {
						break
					}
					dist = *(*UInt32)(unsafe.Add(unsafe.Pointer(matches), unsafe.Sizeof(UInt32(0))*uintptr(offs+1)))
					{
						var zz uint32
						if dist < UInt32(1<<((9+unsafe.Sizeof(uint32(0))/2)+6)) {
							zz = 6
						} else {
							zz = uint32(6 + (9 + unsafe.Sizeof(uint32(0))/2) - 1)
						}
						posSlot = uint32(p.G_FastPos[dist>>UInt32(zz)]) + zz*2
					}
				}
			}
		}
	}
	for {
		p.Opt[last].Price = 1 << 30
		if func() uint32 {
			p_ := &last
			*p_--
			return *p_
		}() == 0 {
			break
		}
	}
	return Backward(p, cur)
}
func GetOptimumFast(p *CLzmaEnc) uint32 {
	var (
		numAvail UInt32
		mainDist UInt32
		mainLen  uint32
		numPairs uint32
		repIndex uint32
		repLen   uint32
		i        uint32
		data     *Byte
	)
	if p.AdditionalOffset == 0 {
		mainLen = ReadMatchDistances(p, &numPairs)
	} else {
		mainLen = p.LongestMatchLen
		numPairs = p.NumPairs
	}
	numAvail = p.NumAvail
	p.BackRes = 4294967295
	if numAvail < 2 {
		return 1
	}
	if numAvail > UInt32(LZMA_MATCH_LEN_MIN+((int32(1<<kLenNumLowBits))*2+(int32(1<<kLenNumHighBits)))-1) {
		numAvail = UInt32(LZMA_MATCH_LEN_MIN + ((int32(1<<kLenNumLowBits))*2 + (int32(1 << kLenNumHighBits))) - 1)
	}
	data = (*Byte)(unsafe.Add(unsafe.Pointer(p.MatchFinder.GetPointerToCurrentPos(p.MatchFinderObj)), -1))
	repLen = func() uint32 {
		repIndex = 0
		return repIndex
	}()
	for i = 0; i < LZMA_NUM_REPS; i++ {
		var (
			len_  uint32
			data2 *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(data), -int(p.Reps[i])))
		)
		if *data != *data2 || *(*Byte)(unsafe.Add(unsafe.Pointer(data), 1)) != *(*Byte)(unsafe.Add(unsafe.Pointer(data2), 1)) {
			continue
		}
		for len_ = 2; len_ < uint32(numAvail) && *(*Byte)(unsafe.Add(unsafe.Pointer(data), len_)) == *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len_)); len_++ {
		}
		if len_ >= p.NumFastBytes {
			p.BackRes = UInt32(i)
			p.AdditionalOffset += len_ - 1
			p.MatchFinder.Skip(p.MatchFinderObj, UInt32(len_-1))
			return len_
		}
		if len_ > repLen {
			repIndex = i
			repLen = len_
		}
	}
	if mainLen >= p.NumFastBytes {
		p.BackRes = p.Matches[numPairs-1] + LZMA_NUM_REPS
		p.AdditionalOffset += mainLen - 1
		p.MatchFinder.Skip(p.MatchFinderObj, UInt32(mainLen-1))
		return mainLen
	}
	mainDist = 0
	if mainLen >= 2 {
		mainDist = p.Matches[numPairs-1]
		for numPairs > 2 {
			var dist2 UInt32
			if mainLen != uint32(p.Matches[numPairs-4]+1) {
				break
			}
			dist2 = p.Matches[numPairs-3]
			if (mainDist >> 7) <= dist2 {
				break
			}
			numPairs -= 2
			mainLen--
			mainDist = dist2
		}
		if mainLen == 2 && mainDist >= 0x80 {
			mainLen = 1
		}
	}
	if repLen >= 2 {
		if repLen+1 >= mainLen || repLen+2 >= mainLen && mainDist >= (1<<9) || repLen+3 >= mainLen && mainDist >= (1<<15) {
			p.BackRes = UInt32(repIndex)
			p.AdditionalOffset += repLen - 1
			p.MatchFinder.Skip(p.MatchFinderObj, UInt32(repLen-1))
			return repLen
		}
	}
	if mainLen < 2 || numAvail <= 2 {
		return 1
	}
	{
		var len1 uint32 = ReadMatchDistances(p, &p.NumPairs)
		p.LongestMatchLen = len1
		if len1 >= 2 {
			var newDist UInt32 = p.Matches[p.NumPairs-1]
			if len1 >= mainLen && newDist < mainDist || len1 == mainLen+1 && (newDist>>7) <= mainDist || len1 > mainLen+1 || len1+1 >= mainLen && mainLen >= 3 && (mainDist>>7) > newDist {
				return 1
			}
		}
	}
	data = (*Byte)(unsafe.Add(unsafe.Pointer(p.MatchFinder.GetPointerToCurrentPos(p.MatchFinderObj)), -1))
	for i = 0; i < LZMA_NUM_REPS; i++ {
		var (
			len_  uint32
			limit uint32
			data2 *Byte = (*Byte)(unsafe.Add(unsafe.Pointer(data), -int(p.Reps[i])))
		)
		if *data != *data2 || *(*Byte)(unsafe.Add(unsafe.Pointer(data), 1)) != *(*Byte)(unsafe.Add(unsafe.Pointer(data2), 1)) {
			continue
		}
		limit = mainLen - 1
		for len_ = 2; ; len_++ {
			if len_ >= limit {
				return 1
			}
			if *(*Byte)(unsafe.Add(unsafe.Pointer(data), len_)) != *(*Byte)(unsafe.Add(unsafe.Pointer(data2), len_)) {
				break
			}
		}
	}
	p.BackRes = mainDist + LZMA_NUM_REPS
	if mainLen != 2 {
		p.AdditionalOffset += mainLen - 2
		p.MatchFinder.Skip(p.MatchFinderObj, UInt32(mainLen-2))
	}
	return mainLen
}
func WriteEndMarker(p *CLzmaEnc, posState uint32) {
	var range_ UInt32
	range_ = p.Rc.Range
	{
		var (
			ttt      UInt32
			newBound UInt32
			prob     *CLzmaProb = &p.IsMatch[p.State][posState]
		)
		ttt = UInt32(*prob)
		newBound = (range_ >> kNumBitModelTotalBits) * ttt
		range_ -= newBound
		p.Rc.Low += UInt64(newBound)
		*prob = CLzmaProb(ttt - (ttt >> kNumMoveBits))
		if range_ < UInt32(int32(1<<kNumTopBits)) {
			range_ <<= 8
			RangeEnc_ShiftLow(&p.Rc)
		}
		prob = &p.IsRep[p.State]
		ttt = UInt32(*prob)
		newBound = (range_ >> kNumBitModelTotalBits) * ttt
		range_ = newBound
		*prob = CLzmaProb(ttt + UInt32(((int32(1<<kNumBitModelTotalBits))-int32(ttt))>>kNumMoveBits))
		if range_ < UInt32(int32(1<<kNumTopBits)) {
			range_ <<= 8
			RangeEnc_ShiftLow(&p.Rc)
		}
	}
	p.State = uint32(kMatchNextStates[p.State])
	p.Rc.Range = range_
	LenEnc_Encode(&p.LenProbs, &p.Rc, 0, posState)
	range_ = p.Rc.Range
	{
		var (
			probs *CLzmaProb = &p.PosSlotEncoder[0][0]
			m     uint32     = 1
		)
		for {
			{
				var (
					ttt      UInt32
					newBound UInt32
				)
				ttt = UInt32(*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m)))))
				newBound = (range_ >> kNumBitModelTotalBits) * ttt
				range_ -= newBound
				p.Rc.Low += UInt64(newBound)
				*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m)))) = CLzmaProb(ttt - (ttt >> kNumMoveBits))
				if range_ < UInt32(int32(1<<kNumTopBits)) {
					range_ <<= 8
					RangeEnc_ShiftLow(&p.Rc)
				}
				m = (m << 1) + 1
			}
			if m >= uint32(int32(1<<kNumPosSlotBits)) {
				break
			}
		}
	}
	{
		var numBits uint32 = uint32(int32(30 - kNumAlignBits))
		for {
			range_ >>= 1
			p.Rc.Low += UInt64(range_)
			if range_ < UInt32(int32(1<<kNumTopBits)) {
				range_ <<= 8
				RangeEnc_ShiftLow(&p.Rc)
			}
			if func() uint32 {
				p_ := &numBits
				*p_--
				return *p_
			}() == 0 {
				break
			}
		}
	}
	{
		var (
			probs *CLzmaProb = &p.PosAlignEncoder[0]
			m     uint32     = 1
		)
		for {
			{
				var (
					ttt      UInt32
					newBound UInt32
				)
				ttt = UInt32(*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m)))))
				newBound = (range_ >> kNumBitModelTotalBits) * ttt
				range_ -= newBound
				p.Rc.Low += UInt64(newBound)
				*((*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m)))) = CLzmaProb(ttt - (ttt >> kNumMoveBits))
				if range_ < UInt32(int32(1<<kNumTopBits)) {
					range_ <<= 8
					RangeEnc_ShiftLow(&p.Rc)
				}
				m = (m << 1) + 1
			}
			if m >= uint32(int32(1<<kNumAlignBits)) {
				break
			}
		}
	}
	p.Rc.Range = range_
}
func CheckErrors(p *CLzmaEnc) SRes {
	if p.Result != SZ_OK {
		return p.Result
	}
	if p.Rc.Res != SZ_OK {
		p.Result = SZ_ERROR_WRITE
	}
	if p.MatchFinderBase.Result != SZ_OK {
		p.Result = SZ_ERROR_READ
	}
	if p.Result != SZ_OK {
		p.Finished = True
	}
	return p.Result
}
func Flush(p *CLzmaEnc, nowPos UInt32) SRes {
	p.Finished = True
	if p.WriteEndMark != 0 {
		WriteEndMarker(p, uint32(nowPos&UInt32(p.PbMask)))
	}
	RangeEnc_FlushData(&p.Rc)
	RangeEnc_FlushStream(&p.Rc)
	return CheckErrors(p)
}
func FillAlignPrices(p *CLzmaEnc) {
	var (
		i          uint32
		ProbPrices *CProbPrice = &p.ProbPrices[0]
		probs      *CLzmaProb  = &p.PosAlignEncoder[0]
	)
	for i = 0; i < uint32((int32(1<<kNumAlignBits))/2); i++ {
		var (
			price UInt32 = 0
			sym   uint32 = i
			m     uint32 = 1
			bit   uint32
			prob  UInt32
		)
		bit = sym & 1
		sym >>= 1
		price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
		m = (m << 1) + bit
		bit = sym & 1
		sym >>= 1
		price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
		m = (m << 1) + bit
		bit = sym & 1
		sym >>= 1
		price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
		m = (m << 1) + bit
		prob = UInt32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m))))
		p.AlignPrices[i] = price + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr(prob>>kNumMoveReducingBits))))
		p.AlignPrices[i+8] = price + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((prob^UInt32((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits))))
	}
}
func FillDistancesPrices(p *CLzmaEnc) {
	var (
		tempPrices [128]UInt32
		i          uint32
		lps        uint32
		ProbPrices *CProbPrice = &p.ProbPrices[0]
	)
	p.MatchPriceCount = 0
	for i = uint32(int32(kStartPosModelIndex / 2)); i < uint32((1<<(int32(kEndPosModelIndex>>1)))/2); i++ {
		var (
			posSlot    uint32     = uint32(p.G_FastPos[i])
			footerBits uint32     = (posSlot >> 1) - 1
			base       uint32     = (((posSlot & 1) | 2) << footerBits)
			probs      *CLzmaProb = &p.PosEncoders[base*2]
			price      UInt32     = 0
			m          uint32     = 1
			sym        uint32     = i
			offset     uint32     = 1 << footerBits
		)
		base += i
		if footerBits != 0 {
			for {
				{
					var bit uint32 = sym & 1
					sym >>= 1
					price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
					m = (m << 1) + bit
				}
				if func() uint32 {
					p_ := &footerBits
					*p_--
					return *p_
				}() == 0 {
					break
				}
			}
		}
		{
			var prob uint32 = uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(m))))
			tempPrices[base] = price + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr(prob>>kNumMoveReducingBits))))
			tempPrices[base+offset] = price + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((prob^uint32((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits))))
		}
	}
	for lps = 0; lps < kNumLenToPosStates; lps++ {
		var (
			slot           uint32
			distTableSize2 uint32     = (p.DistTableSize + 1) >> 1
			posSlotPrices  *UInt32    = &p.PosSlotPrices[lps][0]
			probs          *CLzmaProb = &p.PosSlotEncoder[lps][0]
		)
		for slot = 0; slot < distTableSize2; slot++ {
			var (
				price UInt32
				bit   uint32
				sym   uint32 = slot + uint32(1<<(int32(kNumPosSlotBits-1)))
				prob  uint32
			)
			bit = sym & 1
			sym >>= 1
			price = UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(sym))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
			bit = sym & 1
			sym >>= 1
			price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(sym))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
			bit = sym & 1
			sym >>= 1
			price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(sym))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
			bit = sym & 1
			sym >>= 1
			price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(sym))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
			bit = sym & 1
			sym >>= 1
			price += UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(sym))))^uint32((-int32(bit))&((int32(1<<kNumBitModelTotalBits))-1)))>>kNumMoveReducingBits))))
			prob = uint32(*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(slot+uint32(1<<(int32(kNumPosSlotBits-1)))))))
			*(*UInt32)(unsafe.Add(unsafe.Pointer(posSlotPrices), unsafe.Sizeof(UInt32(0))*uintptr(slot*2))) = price + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr(prob>>kNumMoveReducingBits))))
			*(*UInt32)(unsafe.Add(unsafe.Pointer(posSlotPrices), unsafe.Sizeof(UInt32(0))*uintptr(slot*2+1))) = price + UInt32(*(*CProbPrice)(unsafe.Add(unsafe.Pointer(ProbPrices), unsafe.Sizeof(CProbPrice(0))*uintptr((prob^uint32((int32(1<<kNumBitModelTotalBits))-1))>>kNumMoveReducingBits))))
		}
		{
			var delta UInt32 = (UInt32((int32(kEndPosModelIndex/2)-1)-kNumAlignBits) << kNumBitPriceShiftBits)
			for slot = uint32(int32(kEndPosModelIndex / 2)); slot < distTableSize2; slot++ {
				*(*UInt32)(unsafe.Add(unsafe.Pointer(posSlotPrices), unsafe.Sizeof(UInt32(0))*uintptr(slot*2))) += delta
				*(*UInt32)(unsafe.Add(unsafe.Pointer(posSlotPrices), unsafe.Sizeof(UInt32(0))*uintptr(slot*2+1))) += delta
				delta += UInt32(int32(1 << kNumBitPriceShiftBits))
			}
		}
		{
			var dp *UInt32 = &p.DistancesPrices[lps][0]
			*(*UInt32)(unsafe.Add(unsafe.Pointer(dp), unsafe.Sizeof(UInt32(0))*0)) = *(*UInt32)(unsafe.Add(unsafe.Pointer(posSlotPrices), unsafe.Sizeof(UInt32(0))*0))
			*(*UInt32)(unsafe.Add(unsafe.Pointer(dp), unsafe.Sizeof(UInt32(0))*1)) = *(*UInt32)(unsafe.Add(unsafe.Pointer(posSlotPrices), unsafe.Sizeof(UInt32(0))*1))
			*(*UInt32)(unsafe.Add(unsafe.Pointer(dp), unsafe.Sizeof(UInt32(0))*2)) = *(*UInt32)(unsafe.Add(unsafe.Pointer(posSlotPrices), unsafe.Sizeof(UInt32(0))*2))
			*(*UInt32)(unsafe.Add(unsafe.Pointer(dp), unsafe.Sizeof(UInt32(0))*3)) = *(*UInt32)(unsafe.Add(unsafe.Pointer(posSlotPrices), unsafe.Sizeof(UInt32(0))*3))
			for i = 4; i < uint32(1<<(int32(kEndPosModelIndex>>1))); i += 2 {
				var slotPrice UInt32 = *(*UInt32)(unsafe.Add(unsafe.Pointer(posSlotPrices), unsafe.Sizeof(UInt32(0))*uintptr(p.G_FastPos[i])))
				*(*UInt32)(unsafe.Add(unsafe.Pointer(dp), unsafe.Sizeof(UInt32(0))*uintptr(i))) = slotPrice + tempPrices[i]
				*(*UInt32)(unsafe.Add(unsafe.Pointer(dp), unsafe.Sizeof(UInt32(0))*uintptr(i+1))) = slotPrice + tempPrices[i+1]
			}
		}
	}
}
func LzmaEnc_Construct(p *CLzmaEnc) {
	RangeEnc_Construct(&p.Rc)
	MatchFinder_Construct(&p.MatchFinderBase)
	{
		var props CLzmaEncProps
		LzmaEncProps_Init(&props)
		LzmaEnc_SetProps(CLzmaEncHandle(p), &props)
	}
	LzmaEnc_FastPosInit(&p.G_FastPos[0])
	LzmaEnc_InitPriceTables(&p.ProbPrices[0])
	p.LitProbs = nil
	p.SaveState.LitProbs = nil
}
func LzmaEnc_Create(alloc ISzAllocPtr) CLzmaEncHandle {
	var p unsafe.Pointer
	p = alloc.Alloc(alloc, uint32(unsafe.Sizeof(CLzmaEnc{})))
	if p != nil {
		LzmaEnc_Construct((*CLzmaEnc)(p))
	}
	return p
}
func LzmaEnc_FreeLits(p *CLzmaEnc, alloc ISzAllocPtr) {
	alloc.Free(alloc, unsafe.Pointer(p.LitProbs))
	alloc.Free(alloc, unsafe.Pointer(p.SaveState.LitProbs))
	p.LitProbs = nil
	p.SaveState.LitProbs = nil
}
func LzmaEnc_Destruct(p *CLzmaEnc, alloc ISzAllocPtr, allocBig ISzAllocPtr) {
	MatchFinder_Free(&p.MatchFinderBase, allocBig)
	LzmaEnc_FreeLits(p, alloc)
	RangeEnc_Free(&p.Rc, alloc)
}
func LzmaEnc_Destroy(p CLzmaEncHandle, alloc ISzAllocPtr, allocBig ISzAllocPtr) {
	LzmaEnc_Destruct((*CLzmaEnc)(p), alloc, allocBig)
	alloc.Free(alloc, p)
}
func LzmaEnc_CodeOneBlock(p *CLzmaEnc, maxPackSize UInt32, maxUnpackSize UInt32) SRes {
	var (
		nowPos32   UInt32
		startPos32 UInt32
	)
	if p.NeedInit != 0 {
		p.MatchFinder.Init(p.MatchFinderObj)
		p.NeedInit = 0
	}
	if p.Finished != 0 {
		return p.Result
	}
	{
		var __result__ int32 = int32(CheckErrors(p))
		if __result__ != 0 {
			return SRes(__result__)
		}
	}
	nowPos32 = UInt32(p.NowPos64)
	startPos32 = nowPos32
	if p.NowPos64 == 0 {
		var (
			numPairs uint32
			curByte  Byte
		)
		if p.MatchFinder.GetNumAvailableBytes(p.MatchFinderObj) == 0 {
			return Flush(p, nowPos32)
		}
		ReadMatchDistances(p, &numPairs)
		RangeEnc_EncodeBit_0(&p.Rc, &p.IsMatch[kState_Start][0])
		curByte = *((*Byte)(unsafe.Add(unsafe.Pointer(p.MatchFinder.GetPointerToCurrentPos(p.MatchFinderObj)), -int(p.AdditionalOffset))))
		LitEnc_Encode(&p.Rc, p.LitProbs, UInt32(curByte))
		p.AdditionalOffset--
		nowPos32++
	}
	if p.MatchFinder.GetNumAvailableBytes(p.MatchFinderObj) != 0 {
		for {
			var (
				dist     UInt32
				len_     uint32
				posState uint32
				range_   UInt32
				ttt      UInt32
				newBound UInt32
				probs    *CLzmaProb
			)
			if p.FastMode != 0 {
				len_ = GetOptimumFast(p)
			} else {
				var oci uint32 = p.OptCur
				if p.OptEnd == oci {
					len_ = GetOptimum(p, nowPos32)
				} else {
					var opt *COptimal = &p.Opt[oci]
					len_ = uint32(opt.Len)
					p.BackRes = opt.Dist
					p.OptCur = oci + 1
				}
			}
			posState = uint32(nowPos32) & p.PbMask
			range_ = p.Rc.Range
			probs = &p.IsMatch[p.State][posState]
			ttt = UInt32(*probs)
			newBound = (range_ >> kNumBitModelTotalBits) * ttt
			dist = p.BackRes
			if dist == 4294967295 {
				var (
					curByte Byte
					data    *Byte
					state   uint32
				)
				range_ = newBound
				*probs = CLzmaProb(ttt + UInt32(((int32(1<<kNumBitModelTotalBits))-int32(ttt))>>kNumMoveBits))
				if range_ < UInt32(int32(1<<kNumTopBits)) {
					range_ <<= 8
					RangeEnc_ShiftLow(&p.Rc)
				}
				p.Rc.Range = range_
				data = (*Byte)(unsafe.Add(unsafe.Pointer(p.MatchFinder.GetPointerToCurrentPos(p.MatchFinderObj)), -int(p.AdditionalOffset)))
				probs = (*CLzmaProb)(unsafe.Add(unsafe.Pointer(p.LitProbs), unsafe.Sizeof(CLzmaProb(0))*uintptr(((((nowPos32<<8)+UInt32(*((*Byte)(unsafe.Add(unsafe.Pointer(data), -1)))))&UInt32(p.LpMask))<<UInt32(p.Lc))*3)))
				curByte = *data
				state = p.State
				p.State = uint32(kLiteralNextStates[state])
				if state < 7 {
					LitEnc_Encode(&p.Rc, probs, UInt32(curByte))
				} else {
					LitEnc_EncodeMatched(&p.Rc, probs, UInt32(curByte), UInt32(*((*Byte)(unsafe.Add(unsafe.Pointer(data), -int(p.Reps[0]))))))
				}
			} else {
				range_ -= newBound
				p.Rc.Low += UInt64(newBound)
				*probs = CLzmaProb(ttt - (ttt >> kNumMoveBits))
				if range_ < UInt32(int32(1<<kNumTopBits)) {
					range_ <<= 8
					RangeEnc_ShiftLow(&p.Rc)
				}
				probs = &p.IsRep[p.State]
				ttt = UInt32(*probs)
				newBound = (range_ >> kNumBitModelTotalBits) * ttt
				if dist < LZMA_NUM_REPS {
					range_ -= newBound
					p.Rc.Low += UInt64(newBound)
					*probs = CLzmaProb(ttt - (ttt >> kNumMoveBits))
					if range_ < UInt32(int32(1<<kNumTopBits)) {
						range_ <<= 8
						RangeEnc_ShiftLow(&p.Rc)
					}
					probs = &p.IsRepG0[p.State]
					ttt = UInt32(*probs)
					newBound = (range_ >> kNumBitModelTotalBits) * ttt
					if dist == 0 {
						range_ = newBound
						*probs = CLzmaProb(ttt + UInt32(((int32(1<<kNumBitModelTotalBits))-int32(ttt))>>kNumMoveBits))
						if range_ < UInt32(int32(1<<kNumTopBits)) {
							range_ <<= 8
							RangeEnc_ShiftLow(&p.Rc)
						}
						probs = &p.IsRep0Long[p.State][posState]
						ttt = UInt32(*probs)
						newBound = (range_ >> kNumBitModelTotalBits) * ttt
						if len_ != 1 {
							range_ -= newBound
							p.Rc.Low += UInt64(newBound)
							*probs = CLzmaProb(ttt - (ttt >> kNumMoveBits))
						} else {
							range_ = newBound
							*probs = CLzmaProb(ttt + UInt32(((int32(1<<kNumBitModelTotalBits))-int32(ttt))>>kNumMoveBits))
							p.State = uint32(kShortRepNextStates[p.State])
						}
					} else {
						range_ -= newBound
						p.Rc.Low += UInt64(newBound)
						*probs = CLzmaProb(ttt - (ttt >> kNumMoveBits))
						if range_ < UInt32(int32(1<<kNumTopBits)) {
							range_ <<= 8
							RangeEnc_ShiftLow(&p.Rc)
						}
						probs = &p.IsRepG1[p.State]
						ttt = UInt32(*probs)
						newBound = (range_ >> kNumBitModelTotalBits) * ttt
						if dist == 1 {
							range_ = newBound
							*probs = CLzmaProb(ttt + UInt32(((int32(1<<kNumBitModelTotalBits))-int32(ttt))>>kNumMoveBits))
							dist = p.Reps[1]
						} else {
							range_ -= newBound
							p.Rc.Low += UInt64(newBound)
							*probs = CLzmaProb(ttt - (ttt >> kNumMoveBits))
							if range_ < UInt32(int32(1<<kNumTopBits)) {
								range_ <<= 8
								RangeEnc_ShiftLow(&p.Rc)
							}
							probs = &p.IsRepG2[p.State]
							ttt = UInt32(*probs)
							newBound = (range_ >> kNumBitModelTotalBits) * ttt
							if dist == 2 {
								range_ = newBound
								*probs = CLzmaProb(ttt + UInt32(((int32(1<<kNumBitModelTotalBits))-int32(ttt))>>kNumMoveBits))
								dist = p.Reps[2]
							} else {
								range_ -= newBound
								p.Rc.Low += UInt64(newBound)
								*probs = CLzmaProb(ttt - (ttt >> kNumMoveBits))
								dist = p.Reps[3]
								p.Reps[3] = p.Reps[2]
							}
							p.Reps[2] = p.Reps[1]
						}
						p.Reps[1] = p.Reps[0]
						p.Reps[0] = dist
					}
					if range_ < UInt32(int32(1<<kNumTopBits)) {
						range_ <<= 8
						RangeEnc_ShiftLow(&p.Rc)
					}
					p.Rc.Range = range_
					if len_ != 1 {
						LenEnc_Encode(&p.RepLenProbs, &p.Rc, len_-LZMA_MATCH_LEN_MIN, posState)
						p.RepLenEncCounter--
						p.State = uint32(kRepNextStates[p.State])
					}
				} else {
					var posSlot uint32
					range_ = newBound
					*probs = CLzmaProb(ttt + UInt32(((int32(1<<kNumBitModelTotalBits))-int32(ttt))>>kNumMoveBits))
					if range_ < UInt32(int32(1<<kNumTopBits)) {
						range_ <<= 8
						RangeEnc_ShiftLow(&p.Rc)
					}
					p.Rc.Range = range_
					p.State = uint32(kMatchNextStates[p.State])
					LenEnc_Encode(&p.LenProbs, &p.Rc, len_-LZMA_MATCH_LEN_MIN, posState)
					dist -= LZMA_NUM_REPS
					p.Reps[3] = p.Reps[2]
					p.Reps[2] = p.Reps[1]
					p.Reps[1] = p.Reps[0]
					p.Reps[0] = dist + 1
					p.MatchPriceCount++
					{
						if dist < UInt32(1<<(int32(kEndPosModelIndex>>1))) {
							posSlot = uint32(p.G_FastPos[dist&UInt32((1<<(int32(kEndPosModelIndex>>1)))-1)])
						} else {
							var zz uint32
							if dist < UInt32(1<<((9+unsafe.Sizeof(uint32(0))/2)+6)) {
								zz = 6
							} else {
								zz = uint32(6 + (9 + unsafe.Sizeof(uint32(0))/2) - 1)
							}
							posSlot = uint32(p.G_FastPos[dist>>UInt32(zz)]) + zz*2
						}
					}
					{
						var sym UInt32 = UInt32(posSlot) + UInt32(int32(1<<kNumPosSlotBits))
						range_ = p.Rc.Range
						probs = &p.PosSlotEncoder[func() uint32 {
							if len_ < uint32(int32(kNumLenToPosStates+1)) {
								return len_ - 2
							}
							return uint32(int32(kNumLenToPosStates - 1))
						}()][0]
						for {
							{
								var (
									prob *CLzmaProb = (*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(sym>>kNumPosSlotBits)))
									bit  UInt32     = (sym >> UInt32(int32(kNumPosSlotBits-1))) & 1
								)
								sym <<= 1
								{
									var mask UInt32
									ttt = UInt32(*prob)
									newBound = (range_ >> kNumBitModelTotalBits) * ttt
									mask = 0 - bit
									range_ &= mask
									mask &= newBound
									range_ -= mask
									p.Rc.Low += UInt64(mask)
									mask = bit - 1
									range_ += newBound & mask
									mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
									mask += UInt32((int32(1 << kNumMoveBits)) - 1)
									ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
									*prob = CLzmaProb(ttt)
									if range_ < UInt32(int32(1<<kNumTopBits)) {
										range_ <<= 8
										RangeEnc_ShiftLow(&p.Rc)
									}
								}
							}
							if sym >= UInt32(1<<int32(kNumPosSlotBits*2)) {
								break
							}
						}
						p.Rc.Range = range_
					}
					if dist >= kStartPosModelIndex {
						var footerBits uint32 = ((posSlot >> 1) - 1)
						if dist < UInt32(1<<(int32(kEndPosModelIndex>>1))) {
							var base uint32 = (((posSlot & 1) | 2) << footerBits)
							RcTree_ReverseEncode(&p.Rc, &p.PosEncoders[base], footerBits, uint32(dist))
						} else {
							var pos2 UInt32 = (dist | 0xF) << UInt32(32-footerBits)
							range_ = p.Rc.Range
							for {
								range_ >>= 1
								p.Rc.Low += UInt64(range_ & (0 - (pos2 >> 31)))
								pos2 += pos2
								if range_ < UInt32(int32(1<<kNumTopBits)) {
									range_ <<= 8
									RangeEnc_ShiftLow(&p.Rc)
								}
								if pos2 == 0xF0000000 {
									break
								}
							}
							{
								var (
									m   uint32 = 1
									bit uint32
								)
								bit = uint32(dist & 1)
								dist >>= 1
								{
									var mask UInt32
									ttt = UInt32(p.PosAlignEncoder[m])
									newBound = (range_ >> kNumBitModelTotalBits) * ttt
									mask = 0 - UInt32(bit)
									range_ &= mask
									mask &= newBound
									range_ -= mask
									p.Rc.Low += UInt64(mask)
									mask = UInt32(bit) - 1
									range_ += newBound & mask
									mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
									mask += UInt32((int32(1 << kNumMoveBits)) - 1)
									ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
									p.PosAlignEncoder[m] = CLzmaProb(ttt)
									if range_ < UInt32(int32(1<<kNumTopBits)) {
										range_ <<= 8
										RangeEnc_ShiftLow(&p.Rc)
									}
								}
								m = (m << 1) + bit
								bit = uint32(dist & 1)
								dist >>= 1
								{
									var mask UInt32
									ttt = UInt32(p.PosAlignEncoder[m])
									newBound = (range_ >> kNumBitModelTotalBits) * ttt
									mask = 0 - UInt32(bit)
									range_ &= mask
									mask &= newBound
									range_ -= mask
									p.Rc.Low += UInt64(mask)
									mask = UInt32(bit) - 1
									range_ += newBound & mask
									mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
									mask += UInt32((int32(1 << kNumMoveBits)) - 1)
									ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
									p.PosAlignEncoder[m] = CLzmaProb(ttt)
									if range_ < UInt32(int32(1<<kNumTopBits)) {
										range_ <<= 8
										RangeEnc_ShiftLow(&p.Rc)
									}
								}
								m = (m << 1) + bit
								bit = uint32(dist & 1)
								dist >>= 1
								{
									var mask UInt32
									ttt = UInt32(p.PosAlignEncoder[m])
									newBound = (range_ >> kNumBitModelTotalBits) * ttt
									mask = 0 - UInt32(bit)
									range_ &= mask
									mask &= newBound
									range_ -= mask
									p.Rc.Low += UInt64(mask)
									mask = UInt32(bit) - 1
									range_ += newBound & mask
									mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
									mask += UInt32((int32(1 << kNumMoveBits)) - 1)
									ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
									p.PosAlignEncoder[m] = CLzmaProb(ttt)
									if range_ < UInt32(int32(1<<kNumTopBits)) {
										range_ <<= 8
										RangeEnc_ShiftLow(&p.Rc)
									}
								}
								m = (m << 1) + bit
								bit = uint32(dist & 1)
								{
									var mask UInt32
									ttt = UInt32(p.PosAlignEncoder[m])
									newBound = (range_ >> kNumBitModelTotalBits) * ttt
									mask = 0 - UInt32(bit)
									range_ &= mask
									mask &= newBound
									range_ -= mask
									p.Rc.Low += UInt64(mask)
									mask = UInt32(bit) - 1
									range_ += newBound & mask
									mask &= UInt32((int32(1 << kNumBitModelTotalBits)) - ((int32(1 << kNumMoveBits)) - 1))
									mask += UInt32((int32(1 << kNumMoveBits)) - 1)
									ttt += UInt32(Int32(mask-ttt) >> kNumMoveBits)
									p.PosAlignEncoder[m] = CLzmaProb(ttt)
									if range_ < UInt32(int32(1<<kNumTopBits)) {
										range_ <<= 8
										RangeEnc_ShiftLow(&p.Rc)
									}
								}
								p.Rc.Range = range_
							}
						}
					}
				}
			}
			nowPos32 += UInt32(len_)
			p.AdditionalOffset -= len_
			if p.AdditionalOffset == 0 {
				var processed UInt32
				if p.FastMode == 0 {
					if p.MatchPriceCount >= 64 {
						FillAlignPrices(p)
						FillDistancesPrices(p)
						LenPriceEnc_UpdateTables(&p.LenEnc, 1<<p.Pb, &p.LenProbs, &p.ProbPrices[0])
					}
					if p.RepLenEncCounter <= 0 {
						p.RepLenEncCounter = REP_LEN_COUNT
						LenPriceEnc_UpdateTables(&p.RepLenEnc, 1<<p.Pb, &p.RepLenProbs, &p.ProbPrices[0])
					}
				}
				if p.MatchFinder.GetNumAvailableBytes(p.MatchFinderObj) == 0 {
					break
				}
				processed = nowPos32 - startPos32
				if maxPackSize != 0 {
					if processed+(1<<11)+300 >= maxUnpackSize || (uint32(p.Rc.Processed)+uint32(int32(uintptr(unsafe.Pointer(p.Rc.Buf))-uintptr(unsafe.Pointer(p.Rc.BufBase))))+uint32(p.Rc.CacheSize))+(1<<11)*8 >= uint32(maxPackSize) {
						break
					}
				} else if processed >= (1 << 17) {
					p.NowPos64 += UInt64(nowPos32 - startPos32)
					return CheckErrors(p)
				}
			}
		}
	}
	p.NowPos64 += UInt64(nowPos32 - startPos32)
	return Flush(p, nowPos32)
}
func LzmaEnc_Alloc(p *CLzmaEnc, keepWindowSize UInt32, alloc ISzAllocPtr, allocBig ISzAllocPtr) SRes {
	var beforeSize UInt32 = (1 << 11)
	if RangeEnc_Alloc(&p.Rc, alloc) == 0 {
		return SZ_ERROR_MEM
	}
	{
		var lclp uint32 = p.Lc + p.Lp
		if p.LitProbs == nil || p.SaveState.LitProbs == nil || p.Lclp != lclp {
			LzmaEnc_FreeLits(p, alloc)
			p.LitProbs = (*CLzmaProb)(alloc.Alloc(alloc, (0x300<<lclp)*uint32(unsafe.Sizeof(CLzmaProb(0)))))
			p.SaveState.LitProbs = (*CLzmaProb)(alloc.Alloc(alloc, (0x300<<lclp)*uint32(unsafe.Sizeof(CLzmaProb(0)))))
			if p.LitProbs == nil || p.SaveState.LitProbs == nil {
				LzmaEnc_FreeLits(p, alloc)
				return SZ_ERROR_MEM
			}
			p.Lclp = lclp
		}
	}
	if p.DictSize > (1 << 24) {
		p.MatchFinderBase.BigHash = 1
	} else {
		p.MatchFinderBase.BigHash = 0
	}
	if beforeSize+p.DictSize < keepWindowSize {
		beforeSize = keepWindowSize - p.DictSize
	}
	{
		if MatchFinder_Create(&p.MatchFinderBase, p.DictSize, beforeSize, UInt32(p.NumFastBytes), UInt32(LZMA_MATCH_LEN_MIN+((int32(1<<kLenNumLowBits))*2+(int32(1<<kLenNumHighBits)))-1), allocBig) == 0 {
			return SZ_ERROR_MEM
		}
		p.MatchFinderObj = unsafe.Pointer(&p.MatchFinderBase)
		MatchFinder_CreateVTable(&p.MatchFinderBase, &p.MatchFinder)
	}
	return SZ_OK
}
func LzmaEnc_Init(p *CLzmaEnc) {
	var i uint32
	p.State = 0
	p.Reps[0] = func() UInt32 {
		p_ := &p.Reps[1]
		*p_ = func() UInt32 {
			p_ := &p.Reps[2]
			*p_ = func() UInt32 {
				p_ := &p.Reps[3]
				*p_ = 1
				return *p_
			}()
			return *p_
		}()
		return *p_
	}()
	RangeEnc_Init(&p.Rc)
	for i = 0; i < uint32(int32(1<<kNumAlignBits)); i++ {
		p.PosAlignEncoder[i] = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
	}
	for i = 0; i < kNumStates; i++ {
		var j uint32
		for j = 0; j < uint32(int32(1<<LZMA_PB_MAX)); j++ {
			p.IsMatch[i][j] = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
			p.IsRep0Long[i][j] = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
		}
		p.IsRep[i] = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
		p.IsRepG0[i] = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
		p.IsRepG1[i] = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
		p.IsRepG2[i] = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
	}
	{
		for i = 0; i < kNumLenToPosStates; i++ {
			var (
				probs *CLzmaProb = &p.PosSlotEncoder[i][0]
				j     uint32
			)
			for j = 0; j < uint32(int32(1<<kNumPosSlotBits)); j++ {
				*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(j))) = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
			}
		}
	}
	{
		for i = 0; i < uint32(1<<(int32(kEndPosModelIndex>>1))); i++ {
			p.PosEncoders[i] = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
		}
	}
	{
		var (
			num   UInt32 = UInt32(0x300 << (p.Lp + p.Lc))
			k     UInt32
			probs *CLzmaProb = p.LitProbs
		)
		for k = 0; k < num; k++ {
			*(*CLzmaProb)(unsafe.Add(unsafe.Pointer(probs), unsafe.Sizeof(CLzmaProb(0))*uintptr(k))) = CLzmaProb(int16((int32(1 << kNumBitModelTotalBits)) >> 1))
		}
	}
	LenEnc_Init(&p.LenProbs)
	LenEnc_Init(&p.RepLenProbs)
	p.OptEnd = 0
	p.OptCur = 0
	{
		for i = 0; i < (1 << 11); i++ {
			p.Opt[i].Price = 1 << 30
		}
	}
	p.AdditionalOffset = 0
	p.PbMask = (1 << p.Pb) - 1
	p.LpMask = (0x100 << p.Lp) - (0x100 >> p.Lc)
}
func LzmaEnc_InitPrices(p *CLzmaEnc) {
	if p.FastMode == 0 {
		FillDistancesPrices(p)
		FillAlignPrices(p)
	}
	p.LenEnc.TableSize = func() uint32 {
		p_ := &p.RepLenEnc.TableSize
		*p_ = p.NumFastBytes + 1 - LZMA_MATCH_LEN_MIN
		return *p_
	}()
	p.RepLenEncCounter = REP_LEN_COUNT
	LenPriceEnc_UpdateTables(&p.LenEnc, 1<<p.Pb, &p.LenProbs, &p.ProbPrices[0])
	LenPriceEnc_UpdateTables(&p.RepLenEnc, 1<<p.Pb, &p.RepLenProbs, &p.ProbPrices[0])
}
func LzmaEnc_AllocAndInit(p *CLzmaEnc, keepWindowSize UInt32, alloc ISzAllocPtr, allocBig ISzAllocPtr) SRes {
	var i uint32
	for i = uint32(int32(kEndPosModelIndex / 2)); i < kDicLogSizeMax; i++ {
		if p.DictSize <= UInt32(1<<i) {
			break
		}
	}
	p.DistTableSize = i * 2
	p.Finished = False
	p.Result = SZ_OK
	{
		var __result__ int32 = int32(LzmaEnc_Alloc(p, keepWindowSize, alloc, allocBig))
		if __result__ != 0 {
			return SRes(__result__)
		}
	}
	LzmaEnc_Init(p)
	LzmaEnc_InitPrices(p)
	p.NowPos64 = 0
	return SZ_OK
}
func LzmaEnc_Prepare(pp CLzmaEncHandle, outStream *ISeqOutStream, inStream *ISeqInStream, alloc ISzAllocPtr, allocBig ISzAllocPtr) SRes {
	var p *CLzmaEnc = (*CLzmaEnc)(pp)
	p.MatchFinderBase.Stream = inStream
	p.NeedInit = 1
	p.Rc.OutStream = outStream
	return LzmaEnc_AllocAndInit(p, 0, alloc, allocBig)
}
func LzmaEnc_PrepareForLzma2(pp CLzmaEncHandle, inStream *ISeqInStream, keepWindowSize UInt32, alloc ISzAllocPtr, allocBig ISzAllocPtr) SRes {
	var p *CLzmaEnc = (*CLzmaEnc)(pp)
	p.MatchFinderBase.Stream = inStream
	p.NeedInit = 1
	return LzmaEnc_AllocAndInit(p, keepWindowSize, alloc, allocBig)
}
func LzmaEnc_SetInputBuf(p *CLzmaEnc, src *Byte, srcLen SizeT) {
	p.MatchFinderBase.DirectInput = 1
	p.MatchFinderBase.BufferBase = src
	p.MatchFinderBase.DirectInputRem = uint32(srcLen)
}
func LzmaEnc_MemPrepare(pp CLzmaEncHandle, src *Byte, srcLen SizeT, keepWindowSize UInt32, alloc ISzAllocPtr, allocBig ISzAllocPtr) SRes {
	var p *CLzmaEnc = (*CLzmaEnc)(pp)
	LzmaEnc_SetInputBuf(p, src, srcLen)
	p.NeedInit = 1
	LzmaEnc_SetDataSize(pp, UInt64(srcLen))
	return LzmaEnc_AllocAndInit(p, keepWindowSize, alloc, allocBig)
}
func LzmaEnc_Finish(pp CLzmaEncHandle) {
	_ = pp
}

type CLzmaEnc_SeqOutStreamBuf struct {
	Vt       ISeqOutStream
	Data     *Byte
	Rem      SizeT
	Overflow BoolInt
}

func SeqOutStreamBuf_Write(pp *ISeqOutStream, data unsafe.Pointer, size uint32) uint32 {
	var p *CLzmaEnc_SeqOutStreamBuf = ((*CLzmaEnc_SeqOutStreamBuf)(unsafe.Pointer(pp)))
	if p.Rem < SizeT(size) {
		size = uint32(p.Rem)
		p.Overflow = True
	}
	cMemcpy(unsafe.Pointer(p.Data), data, int(size))
	p.Rem -= SizeT(size)
	p.Data = (*Byte)(unsafe.Add(unsafe.Pointer(p.Data), size))
	return size
}
func LzmaEnc_GetNumAvailableBytes(pp CLzmaEncHandle) UInt32 {
	var p *CLzmaEnc = (*CLzmaEnc)(pp)
	return p.MatchFinder.GetNumAvailableBytes(p.MatchFinderObj)
}
func LzmaEnc_GetCurBuf(pp CLzmaEncHandle) *Byte {
	var p *CLzmaEnc = (*CLzmaEnc)(pp)
	return (*Byte)(unsafe.Add(unsafe.Pointer(p.MatchFinder.GetPointerToCurrentPos(p.MatchFinderObj)), -int(p.AdditionalOffset)))
}
func LzmaEnc_CodeOneMemBlock(pp CLzmaEncHandle, reInit BoolInt, dest *Byte, destLen *uint32, desiredPackSize UInt32, unpackSize *UInt32) SRes {
	var (
		p         *CLzmaEnc = (*CLzmaEnc)(pp)
		nowPos64  UInt64
		res       SRes
		outStream CLzmaEnc_SeqOutStreamBuf
	)
	outStream.Vt.Write = SeqOutStreamBuf_Write
	outStream.Data = dest
	outStream.Rem = SizeT(*destLen)
	outStream.Overflow = False
	p.WriteEndMark = False
	p.Finished = False
	p.Result = SZ_OK
	if reInit != 0 {
		LzmaEnc_Init(p)
	}
	LzmaEnc_InitPrices(p)
	nowPos64 = p.NowPos64
	RangeEnc_Init(&p.Rc)
	p.Rc.OutStream = &outStream.Vt
	if desiredPackSize == 0 {
		return SZ_ERROR_OUTPUT_EOF
	}
	res = LzmaEnc_CodeOneBlock(p, desiredPackSize, *unpackSize)
	*unpackSize = UInt32(p.NowPos64 - nowPos64)
	*destLen -= uint32(outStream.Rem)
	if outStream.Overflow != 0 {
		return SZ_ERROR_OUTPUT_EOF
	}
	return res
}
func LzmaEnc_Encode2(p *CLzmaEnc, progress *ICompressProgress) SRes {
	var res SRes = SZ_OK
	for {
		res = LzmaEnc_CodeOneBlock(p, 0, 0)
		if res != SZ_OK || p.Finished != 0 {
			break
		}
		if progress != nil {
			res = progress.Progress(progress, p.NowPos64, p.Rc.Processed+UInt64(int32(uintptr(unsafe.Pointer(p.Rc.Buf))-uintptr(unsafe.Pointer(p.Rc.BufBase))))+p.Rc.CacheSize)
			if res != SZ_OK {
				res = SZ_ERROR_PROGRESS
				break
			}
		}
	}
	LzmaEnc_Finish(CLzmaEncHandle(p))
	return res
}
func LzmaEnc_Encode(pp CLzmaEncHandle, outStream *ISeqOutStream, inStream *ISeqInStream, progress *ICompressProgress, alloc ISzAllocPtr, allocBig ISzAllocPtr) SRes {
	{
		var __result__ int32 = int32(LzmaEnc_Prepare(pp, outStream, inStream, alloc, allocBig))
		if __result__ != 0 {
			return SRes(__result__)
		}
	}
	return LzmaEnc_Encode2((*CLzmaEnc)(pp), progress)
}
func LzmaEnc_WriteProperties(pp CLzmaEncHandle, props *Byte, size *SizeT) SRes {
	var (
		p        *CLzmaEnc = (*CLzmaEnc)(pp)
		i        uint32
		dictSize UInt32 = p.DictSize
	)
	if *size < LZMA_PROPS_SIZE {
		return SZ_ERROR_PARAM
	}
	*size = LZMA_PROPS_SIZE
	*props = Byte((p.Pb*5+p.Lp)*9 + p.Lc)
	if dictSize >= (1 << 22) {
		var kDictMask UInt32 = (1 << 20) - 1
		if dictSize < 0xFFFFFFFF-kDictMask {
			dictSize = (dictSize + kDictMask) & ^kDictMask
		}
	} else {
		for i = 11; i <= 30; i++ {
			if dictSize <= UInt32(2<<i) {
				dictSize = UInt32(2 << i)
				break
			}
			if dictSize <= UInt32(3<<i) {
				dictSize = UInt32(3 << i)
				break
			}
		}
	}
	for i = 0; i < 4; i++ {
		*(*Byte)(unsafe.Add(unsafe.Pointer(props), i+1)) = Byte(dictSize >> UInt32(i*8))
	}
	return SZ_OK
}
func LzmaEnc_IsWriteEndMark(pp CLzmaEncHandle) uint32 {
	return uint32(((*CLzmaEnc)(pp)).WriteEndMark)
}
func LzmaEnc_MemEncode(pp CLzmaEncHandle, dest *Byte, destLen *SizeT, src *Byte, srcLen SizeT, writeEndMark int32, progress *ICompressProgress, alloc ISzAllocPtr, allocBig ISzAllocPtr) SRes {
	var (
		res       SRes
		p         *CLzmaEnc = (*CLzmaEnc)(pp)
		outStream CLzmaEnc_SeqOutStreamBuf
	)
	outStream.Vt.Write = SeqOutStreamBuf_Write
	outStream.Data = dest
	outStream.Rem = *destLen
	outStream.Overflow = False
	p.WriteEndMark = BoolInt(writeEndMark)
	p.Rc.OutStream = &outStream.Vt
	res = LzmaEnc_MemPrepare(pp, src, srcLen, 0, alloc, allocBig)
	if res == SZ_OK {
		res = LzmaEnc_Encode2(p, progress)
		if res == SZ_OK && p.NowPos64 != UInt64(srcLen) {
			res = SZ_ERROR_FAIL
		}
	}
	*destLen -= outStream.Rem
	if outStream.Overflow != 0 {
		return SZ_ERROR_OUTPUT_EOF
	}
	return res
}
func LzmaEncode(dest *Byte, destLen *SizeT, src *Byte, srcLen SizeT, props *CLzmaEncProps, propsEncoded *Byte, propsSize *SizeT, writeEndMark int32, progress *ICompressProgress, alloc ISzAllocPtr, allocBig ISzAllocPtr) SRes {
	var (
		p   *CLzmaEnc = (*CLzmaEnc)(LzmaEnc_Create(alloc))
		res SRes
	)
	if p == nil {
		return SZ_ERROR_MEM
	}
	res = LzmaEnc_SetProps(CLzmaEncHandle(p), props)
	if res == SZ_OK {
		res = LzmaEnc_WriteProperties(CLzmaEncHandle(p), propsEncoded, propsSize)
		if res == SZ_OK {
			res = LzmaEnc_MemEncode(CLzmaEncHandle(p), dest, destLen, src, srcLen, writeEndMark, progress, alloc, allocBig)
		}
	}
	LzmaEnc_Destroy(CLzmaEncHandle(p), alloc, allocBig)
	return res
}
