#include "precomp_xz.h"

bool init_encoder_mt(lzma_stream *, int, uint64_t, uint64_t &, uint64_t &,
                     const lzma_init_mt_extra_parameters &) {
	return false;
}

bool init_decoder(lzma_stream *) { return false; }
