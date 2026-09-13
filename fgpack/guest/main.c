#include "LzmaEnc.h"
#include "Alloc.h"
#include <stdint.h>

// LZMA SDK 19.00 console: e -d19 -fb64 -lc3 -lp0 -pb2, default bt4/algo1.
uint32_t fgpack_encode(const Byte* src, uint32_t size, Byte* out, uint32_t capacity) {
    if (capacity < 13) return 0;
    CLzmaEncProps props;
    LzmaEncProps_Init(&props);
    props.dictSize = 1 << 19;
    props.fb = 64;
    props.lc = 3; props.lp = 0; props.pb = 2;
    props.algo = 1; props.btMode = 1; props.numHashBytes = 4;
    props.numThreads = 1;
    SizeT out_size = capacity - 13, prop_size = 5;
    SRes res = LzmaEncode(out + 13, &out_size, src, size, &props,
                         out, &prop_size, 0, 0, &g_Alloc, &g_Alloc);
    if (res != SZ_OK) return 0;
    for (unsigned i = 0; i < 8; i++) out[5+i] = (Byte)((uint64_t)size >> (8*i));
    return (uint32_t)out_size + 13;
}
