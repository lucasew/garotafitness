#include "patch.h"
#include <stdint.h>

uint32_t hpatch_size(const unsigned char* diff, uint32_t size, uint32_t old_size) {
    hpatch_compressedDiffInfo info;
    if (!getCompressedDiffInfo_mem(&info, diff, diff + size)) return 0;
    if (info.oldDataSize != old_size || info.newDataSize > UINT32_MAX || info.compressType[0]) return 0;
    return (uint32_t)info.newDataSize;
}

int hpatch_apply(const unsigned char* old, uint32_t old_size,
                 const unsigned char* diff, uint32_t diff_size,
                 unsigned char* out, uint32_t out_size) {
    if (hpatch_size(diff, diff_size, old_size) != out_size) return 0;
    return patch_decompress_mem(out, out + out_size, old, old + old_size,
                                diff, diff + diff_size, 0);
}
