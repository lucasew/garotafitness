package bz2

import (
	"unsafe"
)

const BZ_RUN = 0
const BZ_FLUSH = 1
const BZ_FINISH = 2
const BZ_OK = 0
const BZ_RUN_OK = 1
const BZ_FLUSH_OK = 2
const BZ_FINISH_OK = 3
const BZ_STREAM_END = 4
const BZ_SEQUENCE_ERROR = -1
const BZ_PARAM_ERROR = -2
const BZ_MEM_ERROR = -3
const BZ_DATA_ERROR = -4
const BZ_DATA_ERROR_MAGIC = -5
const BZ_IO_ERROR = -6
const BZ_UNEXPECTED_EOF = -7
const BZ_OUTBUFF_FULL = -8
const BZ_CONFIG_ERROR = -9

type bz_stream struct {
	Next_in        *byte
	Avail_in       uint32
	Total_in_lo32  uint32
	Total_in_hi32  uint32
	Next_out       *byte
	Avail_out      uint32
	Total_out_lo32 uint32
	Total_out_hi32 uint32
	State          unsafe.Pointer
	Bzalloc        func(unsafe.Pointer, int32, int32) unsafe.Pointer
	Bzfree         func(unsafe.Pointer, unsafe.Pointer)
	Opaque         unsafe.Pointer
}

func bz_config_ok() int32 {
	if unsafe.Sizeof(int32(0)) != 4 {
		return 0
	}
	if unsafe.Sizeof(int16(0)) != 2 {
		return 0
	}
	if unsafe.Sizeof(int8(0)) != 1 {
		return 0
	}
	return 1
}
func default_bzalloc(opaque unsafe.Pointer, items Int32, size Int32) unsafe.Pointer {
	if items <= 0 || size <= 0 {
		return nil
	}
	return bmalloc(int(items) * int(size))
}
func default_bzfree(opaque unsafe.Pointer, addr unsafe.Pointer) {
	bfree(addr)
}
func prepare_new_block(s *EState) {
	var i Int32
	s.Nblock = 0
	s.NumZ = 0
	s.State_out_pos = 0
	{
		s.BlockCRC = 0xFFFFFFFF
	}
	for i = 0; i < 256; i++ {
		s.InUse[i] = 0
	}
	s.BlockNo++
}
func init_RL(s *EState) {
	s.State_in_ch = 256
	s.State_in_len = 0
}
func isempty_RL(s *EState) Bool {
	if s.State_in_ch < 256 && s.State_in_len > 0 {
		return 0
	} else {
		return 1
	}
}
func BZ2_bzCompressInit(strm *bz_stream, blockSize100k int32, verbosity int32, workFactor int32) int32 {
	var (
		n Int32
		s *EState
	)
	if bz_config_ok() == 0 {
		return -9
	}
	if strm == nil || blockSize100k < 1 || blockSize100k > 9 || workFactor < 0 || workFactor > 250 {
		return -2
	}
	if workFactor == 0 {
		workFactor = 30
	}
	if strm.Bzalloc == nil {
		strm.Bzalloc = func(arg1 unsafe.Pointer, arg2 int32, arg3 int32) unsafe.Pointer {
			return default_bzalloc(arg1, Int32(arg2), Int32(arg3))
		}
	}
	if strm.Bzfree == nil {
		strm.Bzfree = default_bzfree
	}
	s = (*EState)(strm.Bzalloc(strm.Opaque, int32(uint32(unsafe.Sizeof(EState{}))), 1))
	if s == nil {
		return -3
	}
	s.Strm = strm
	s.Arr1 = nil
	s.Arr2 = nil
	s.Ftab = nil
	n = Int32(uint32(blockSize100k) * 100000)
	s.Arr1 = (*UInt32)(strm.Bzalloc(strm.Opaque, int32(uint32(uintptr(n)*unsafe.Sizeof(UInt32(0)))), 1))
	s.Arr2 = (*UInt32)(strm.Bzalloc(strm.Opaque, int32(uint32(uintptr(n+Int32(int32(BZ_N_RADIX+BZ_N_QSORT)+BZ_N_SHELL+2))*unsafe.Sizeof(UInt32(0)))), 1))
	s.Ftab = (*UInt32)(strm.Bzalloc(strm.Opaque, int32(uint32(65537*unsafe.Sizeof(UInt32(0)))), 1))
	if s.Arr1 == nil || s.Arr2 == nil || s.Ftab == nil {
		if s.Arr1 != nil {
			strm.Bzfree(strm.Opaque, unsafe.Pointer(s.Arr1))
		}
		if s.Arr2 != nil {
			strm.Bzfree(strm.Opaque, unsafe.Pointer(s.Arr2))
		}
		if s.Ftab != nil {
			strm.Bzfree(strm.Opaque, unsafe.Pointer(s.Ftab))
		}
		if s != nil {
			strm.Bzfree(strm.Opaque, unsafe.Pointer(s))
		}
		return -3
	}
	s.BlockNo = 0
	s.State = BZ_S_INPUT
	s.Mode = BZ_M_RUNNING
	s.CombinedCRC = 0
	s.BlockSize100k = Int32(blockSize100k)
	s.NblockMAX = Int32(uint32(blockSize100k)*100000 - 19)
	s.Verbosity = Int32(verbosity)
	s.WorkFactor = Int32(workFactor)
	s.Block = (*UChar)(unsafe.Pointer(s.Arr2))
	s.Mtfv = (*UInt16)(unsafe.Pointer(s.Arr1))
	s.Zbits = nil
	s.Ptr = s.Arr1
	strm.State = unsafe.Pointer(s)
	strm.Total_in_lo32 = 0
	strm.Total_in_hi32 = 0
	strm.Total_out_lo32 = 0
	strm.Total_out_hi32 = 0
	init_RL(s)
	prepare_new_block(s)
	return BZ_OK
}
func add_pair_to_block(s *EState) {
	var (
		i  Int32
		ch UChar = UChar(s.State_in_ch)
	)
	for i = 0; i < s.State_in_len; i++ {
		s.BlockCRC = (s.BlockCRC << 8) ^ BZ2_crc32Table[(s.BlockCRC>>24)^UInt32(ch)]
	}
	s.InUse[s.State_in_ch] = 1
	switch s.State_in_len {
	case 1:
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
		s.Nblock++
	case 2:
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
		s.Nblock++
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
		s.Nblock++
	case 3:
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
		s.Nblock++
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
		s.Nblock++
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
		s.Nblock++
	default:
		s.InUse[s.State_in_len-4] = 1
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
		s.Nblock++
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
		s.Nblock++
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
		s.Nblock++
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
		s.Nblock++
		*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = UChar(int8(s.State_in_len - 4))
		s.Nblock++
	}
}
func flush_RL(s *EState) {
	if s.State_in_ch < 256 {
		add_pair_to_block(s)
	}
	init_RL(s)
}
func copy_input_until_stop(s *EState) Bool {
	var progress_in Bool = 0
	if s.Mode == BZ_M_RUNNING {
		for {
			if s.Nblock >= s.NblockMAX {
				break
			}
			if s.Strm.Avail_in == 0 {
				break
			}
			progress_in = 1
			{
				var zchh UInt32 = (UInt32(*((*UChar)(unsafe.Pointer(s.Strm.Next_in)))))
				if zchh != s.State_in_ch && s.State_in_len == 1 {
					var ch UChar = UChar(s.State_in_ch)
					s.BlockCRC = (s.BlockCRC << 8) ^ BZ2_crc32Table[(s.BlockCRC>>24)^UInt32(ch)]
					s.InUse[s.State_in_ch] = 1
					*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
					s.Nblock++
					s.State_in_ch = zchh
				} else if zchh != s.State_in_ch || s.State_in_len == 255 {
					if s.State_in_ch < 256 {
						add_pair_to_block(s)
					}
					s.State_in_ch = zchh
					s.State_in_len = 1
				} else {
					s.State_in_len++
				}
			}
			s.Strm.Next_in = (*byte)(unsafe.Add(unsafe.Pointer(s.Strm.Next_in), 1))
			s.Strm.Avail_in--
			s.Strm.Total_in_lo32++
			if s.Strm.Total_in_lo32 == 0 {
				s.Strm.Total_in_hi32++
			}
		}
	} else {
		for {
			if s.Nblock >= s.NblockMAX {
				break
			}
			if s.Strm.Avail_in == 0 {
				break
			}
			if s.Avail_in_expect == 0 {
				break
			}
			progress_in = 1
			{
				var zchh UInt32 = (UInt32(*((*UChar)(unsafe.Pointer(s.Strm.Next_in)))))
				if zchh != s.State_in_ch && s.State_in_len == 1 {
					var ch UChar = UChar(s.State_in_ch)
					s.BlockCRC = (s.BlockCRC << 8) ^ BZ2_crc32Table[(s.BlockCRC>>24)^UInt32(ch)]
					s.InUse[s.State_in_ch] = 1
					*(*UChar)(unsafe.Add(unsafe.Pointer(s.Block), s.Nblock)) = ch
					s.Nblock++
					s.State_in_ch = zchh
				} else if zchh != s.State_in_ch || s.State_in_len == 255 {
					if s.State_in_ch < 256 {
						add_pair_to_block(s)
					}
					s.State_in_ch = zchh
					s.State_in_len = 1
				} else {
					s.State_in_len++
				}
			}
			s.Strm.Next_in = (*byte)(unsafe.Add(unsafe.Pointer(s.Strm.Next_in), 1))
			s.Strm.Avail_in--
			s.Strm.Total_in_lo32++
			if s.Strm.Total_in_lo32 == 0 {
				s.Strm.Total_in_hi32++
			}
			s.Avail_in_expect--
		}
	}
	return progress_in
}
func copy_output_until_stop(s *EState) Bool {
	var progress_out Bool = 0
	for {
		if s.Strm.Avail_out == 0 {
			break
		}
		if s.State_out_pos >= s.NumZ {
			break
		}
		progress_out = 1
		*s.Strm.Next_out = byte(*(*UChar)(unsafe.Add(unsafe.Pointer(s.Zbits), s.State_out_pos)))
		s.State_out_pos++
		s.Strm.Avail_out--
		s.Strm.Next_out = (*byte)(unsafe.Add(unsafe.Pointer(s.Strm.Next_out), 1))
		s.Strm.Total_out_lo32++
		if s.Strm.Total_out_lo32 == 0 {
			s.Strm.Total_out_hi32++
		}
	}
	return progress_out
}
func handle_compress(strm *bz_stream) Bool {
	var (
		progress_in  Bool    = 0
		progress_out Bool    = 0
		s            *EState = (*EState)(strm.State)
	)
	for {
		if s.State == BZ_S_OUTPUT {
			progress_out |= copy_output_until_stop(s)
			if s.State_out_pos < s.NumZ {
				break
			}
			if s.Mode == BZ_M_FINISHING && s.Avail_in_expect == 0 && isempty_RL(s) != 0 {
				break
			}
			prepare_new_block(s)
			s.State = BZ_S_INPUT
			if s.Mode == BZ_M_FLUSHING && s.Avail_in_expect == 0 && isempty_RL(s) != 0 {
				break
			}
		}
		if s.State == BZ_S_INPUT {
			progress_in |= copy_input_until_stop(s)
			if s.Mode != BZ_M_RUNNING && s.Avail_in_expect == 0 {
				flush_RL(s)
				BZ2_compressBlock(s, cbool(s.Mode == BZ_M_FINISHING))
				s.State = BZ_S_OUTPUT
			} else if s.Nblock >= s.NblockMAX {
				BZ2_compressBlock(s, 0)
				s.State = BZ_S_OUTPUT
			} else if s.Strm.Avail_in == 0 {
				break
			}
		}
	}
	return cbool(progress_in != 0 || progress_out != 0)
}
func BZ2_bzCompress(strm *bz_stream, action int32) int32 {
	var (
		progress Bool
		s        *EState
	)
	if strm == nil {
		return -2
	}
	s = (*EState)(strm.State)
	if s == nil {
		return -2
	}
	if s.Strm != strm {
		return -2
	}
preswitch:
	switch s.Mode {
	case BZ_M_IDLE:
		return -1
	case BZ_M_RUNNING:
		if action == BZ_RUN {
			progress = handle_compress(strm)
			if progress != 0 {
				return BZ_RUN_OK
			}
			return -2
		} else if action == BZ_FLUSH {
			s.Avail_in_expect = UInt32(strm.Avail_in)
			s.Mode = BZ_M_FLUSHING
			goto preswitch
		} else if action == BZ_FINISH {
			s.Avail_in_expect = UInt32(strm.Avail_in)
			s.Mode = BZ_M_FINISHING
			goto preswitch
		} else {
			return -2
		}
	case BZ_M_FLUSHING:
		if action != BZ_FLUSH {
			return -1
		}
		if s.Avail_in_expect != UInt32(s.Strm.Avail_in) {
			return -1
		}
		progress = handle_compress(strm)
		if s.Avail_in_expect > 0 || isempty_RL(s) == 0 || s.State_out_pos < s.NumZ {
			return BZ_FLUSH_OK
		}
		s.Mode = BZ_M_RUNNING
		return BZ_RUN_OK
	case BZ_M_FINISHING:
		if action != BZ_FINISH {
			return -1
		}
		if s.Avail_in_expect != UInt32(s.Strm.Avail_in) {
			return -1
		}
		progress = handle_compress(strm)
		if progress == 0 {
			return -1
		}
		if s.Avail_in_expect > 0 || isempty_RL(s) == 0 || s.State_out_pos < s.NumZ {
			return BZ_FINISH_OK
		}
		s.Mode = BZ_M_IDLE
		return BZ_STREAM_END
	}
	return BZ_OK
}
func BZ2_bzCompressEnd(strm *bz_stream) int32 {
	var s *EState
	if strm == nil {
		return -2
	}
	s = (*EState)(strm.State)
	if s == nil {
		return -2
	}
	if s.Strm != strm {
		return -2
	}
	if s.Arr1 != nil {
		strm.Bzfree(strm.Opaque, unsafe.Pointer(s.Arr1))
	}
	if s.Arr2 != nil {
		strm.Bzfree(strm.Opaque, unsafe.Pointer(s.Arr2))
	}
	if s.Ftab != nil {
		strm.Bzfree(strm.Opaque, unsafe.Pointer(s.Ftab))
	}
	strm.Bzfree(strm.Opaque, strm.State)
	strm.State = nil
	return BZ_OK
}
func BZ2_indexIntoF(indx Int32, cftab *Int32) Int32 {
	var (
		nb  Int32
		na  Int32
		mid Int32
	)
	nb = 0
	na = 256
	for {
		mid = (nb + na) >> 1
		if indx >= *(*Int32)(unsafe.Add(unsafe.Pointer(cftab), unsafe.Sizeof(Int32(0))*uintptr(mid))) {
			nb = mid
		} else {
			na = mid
		}
		if na-nb == 1 {
			break
		}
	}
	return nb
}
func BZ2_bzBuffToBuffCompress(dest *byte, destLen *uint32, source *byte, sourceLen uint32, blockSize100k int32, verbosity int32, workFactor int32) int32 {
	var (
		strm bz_stream
		ret  int32
	)
	if dest == nil || destLen == nil || source == nil || blockSize100k < 1 || blockSize100k > 9 || verbosity < 0 || verbosity > 4 || workFactor < 0 || workFactor > 250 {
		return -2
	}
	if workFactor == 0 {
		workFactor = 30
	}
	strm.Bzalloc = nil
	strm.Bzfree = nil
	strm.Opaque = nil
	ret = BZ2_bzCompressInit(&strm, blockSize100k, verbosity, workFactor)
	if ret != BZ_OK {
		return ret
	}
	strm.Next_in = source
	strm.Next_out = dest
	strm.Avail_in = sourceLen
	strm.Avail_out = *destLen
	ret = BZ2_bzCompress(&strm, BZ_FINISH)
	if ret == BZ_FINISH_OK {
		goto output_overflow
	}
	if ret != BZ_STREAM_END {
		goto errhandler
	}
	*destLen -= strm.Avail_out
	BZ2_bzCompressEnd(&strm)
	return BZ_OK
output_overflow:
	BZ2_bzCompressEnd(&strm)
	return -8
errhandler:
	BZ2_bzCompressEnd(&strm)
	return ret
}
func BZ2_bzlibVersion() *byte {
	return &bzVersion[0]
}
