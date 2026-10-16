package bz2

const BZ_VERSION = "1.0.6, 6-Sept-2010"
const BZ_HDR_B = 0x42
const BZ_HDR_Z = 0x5A
const BZ_HDR_h = 0x68
const BZ_HDR_0 = 0x30
const BZ_MAX_ALPHA_SIZE = 258
const BZ_MAX_CODE_LEN = 23
const BZ_RUNA = 0
const BZ_RUNB = 1
const BZ_N_GROUPS = 6
const BZ_G_SIZE = 50
const BZ_N_ITERS = 4
const BZ_MAX_SELECTORS = 0x4652
const BZ_M_IDLE = 1
const BZ_M_RUNNING = 2
const BZ_M_FLUSHING = 3
const BZ_M_FINISHING = 4
const BZ_S_OUTPUT = 1
const BZ_S_INPUT = 2
const BZ_N_RADIX = 2
const BZ_N_QSORT = 12
const BZ_N_SHELL = 18
const BZ_N_OVERSHOOT = 34
const BZ_X_IDLE = 1
const BZ_X_OUTPUT = 2
const BZ_X_MAGIC_1 = 10
const BZ_X_MAGIC_2 = 11
const BZ_X_MAGIC_3 = 12
const BZ_X_MAGIC_4 = 13
const BZ_X_BLKHDR_1 = 14
const BZ_X_BLKHDR_2 = 15
const BZ_X_BLKHDR_3 = 16
const BZ_X_BLKHDR_4 = 17
const BZ_X_BLKHDR_5 = 18
const BZ_X_BLKHDR_6 = 19
const BZ_X_BCRC_1 = 20
const BZ_X_BCRC_2 = 21
const BZ_X_BCRC_3 = 22
const BZ_X_BCRC_4 = 23
const BZ_X_RANDBIT = 24
const BZ_X_ORIGPTR_1 = 25
const BZ_X_ORIGPTR_2 = 26
const BZ_X_ORIGPTR_3 = 27
const BZ_X_MAPPING_1 = 28
const BZ_X_MAPPING_2 = 29
const BZ_X_SELECTOR_1 = 30
const BZ_X_SELECTOR_2 = 31
const BZ_X_SELECTOR_3 = 32
const BZ_X_CODING_1 = 33
const BZ_X_CODING_2 = 34
const BZ_X_CODING_3 = 35
const BZ_X_MTF_1 = 36
const BZ_X_MTF_2 = 37
const BZ_X_MTF_3 = 38
const BZ_X_MTF_4 = 39
const BZ_X_MTF_5 = 40
const BZ_X_MTF_6 = 41
const BZ_X_ENDHDR_2 = 42
const BZ_X_ENDHDR_3 = 43
const BZ_X_ENDHDR_4 = 44
const BZ_X_ENDHDR_5 = 45
const BZ_X_ENDHDR_6 = 46
const BZ_X_CCRC_1 = 47
const BZ_X_CCRC_2 = 48
const BZ_X_CCRC_3 = 49
const BZ_X_CCRC_4 = 50
const MTFA_SIZE = 4096
const MTFL_SIZE = 16

type Char int8
type Bool uint8
type UChar uint8
type Int32 int32
type UInt32 uint32
type Int16 int16
type UInt16 uint16
type EState struct {
	Strm            *bz_stream
	Mode            Int32
	State           Int32
	Avail_in_expect UInt32
	Arr1            *UInt32
	Arr2            *UInt32
	Ftab            *UInt32
	OrigPtr         Int32
	Ptr             *UInt32
	Block           *UChar
	Mtfv            *UInt16
	Zbits           *UChar
	WorkFactor      Int32
	State_in_ch     UInt32
	State_in_len    Int32
	RNToGo          Int32
	RTPos           Int32
	Nblock          Int32
	NblockMAX       Int32
	NumZ            Int32
	State_out_pos   Int32
	NInUse          Int32
	InUse           [256]Bool
	UnseqToSeq      [256]UChar
	BsBuff          UInt32
	BsLive          Int32
	BlockCRC        UInt32
	CombinedCRC     UInt32
	Verbosity       Int32
	BlockNo         Int32
	BlockSize100k   Int32
	NMTF            Int32
	MtfFreq         [258]Int32
	Selector        [18002]UChar
	SelectorMtf     [18002]UChar
	Len             [6][258]UChar
	Code            [6][258]Int32
	Rfreq           [6][258]Int32
	Len_pack        [258][4]UInt32
}
type DState struct {
	Strm                  *bz_stream
	State                 Int32
	State_out_ch          UChar
	State_out_len         Int32
	BlockRandomised       Bool
	RNToGo                Int32
	RTPos                 Int32
	BsBuff                UInt32
	BsLive                Int32
	BlockSize100k         Int32
	SmallDecompress       Bool
	CurrBlockNo           Int32
	Verbosity             Int32
	OrigPtr               Int32
	TPos                  UInt32
	K0                    Int32
	Unzftab               [256]Int32
	Nblock_used           Int32
	Cftab                 [257]Int32
	CftabCopy             [257]Int32
	Tt                    *UInt32
	Ll16                  *UInt16
	Ll4                   *UChar
	StoredBlockCRC        UInt32
	StoredCombinedCRC     UInt32
	CalculatedBlockCRC    UInt32
	CalculatedCombinedCRC UInt32
	NInUse                Int32
	InUse                 [256]Bool
	InUse16               [16]Bool
	SeqToUnseq            [256]UChar
	Mtfa                  [4096]UChar
	Mtfbase               [16]Int32
	Selector              [18002]UChar
	SelectorMtf           [18002]UChar
	Len                   [6][258]UChar
	Limit                 [6][258]Int32
	Base                  [6][258]Int32
	Perm                  [6][258]Int32
	MinLens               [6]Int32
	Save_i                Int32
	Save_j                Int32
	Save_t                Int32
	Save_alphaSize        Int32
	Save_nGroups          Int32
	Save_nSelectors       Int32
	Save_EOB              Int32
	Save_groupNo          Int32
	Save_groupPos         Int32
	Save_nextSym          Int32
	Save_nblockMAX        Int32
	Save_nblock           Int32
	Save_es               Int32
	Save_N                Int32
	Save_curr             Int32
	Save_zt               Int32
	Save_zn               Int32
	Save_zvec             Int32
	Save_zj               Int32
	Save_gSel             Int32
	Save_gMinlen          Int32
	Save_gLimit           *Int32
	Save_gBase            *Int32
	Save_gPerm            *Int32
}
