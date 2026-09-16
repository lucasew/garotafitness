#include "precomp_xz.h"

#include <stdint.h>

// Restore only: official init_decoder from compress_easy_mt.cpp.
// init_encoder_mt needs stream_encoder_mt (pthreads); unused on -r.

bool init_encoder_mt(lzma_stream *, int, uint64_t, uint64_t &, uint64_t &,
                     const lzma_init_mt_extra_parameters &) {
	return false;
}

bool init_decoder(lzma_stream *strm) {
	return lzma_auto_decoder(strm, UINT64_MAX, 0) == LZMA_OK;
}
