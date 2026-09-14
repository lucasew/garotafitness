#include "xdelta3.h"
#include <stdint.h>

uint32_t xdelta_apply(const uint8_t* old, uint32_t old_size,
                      const uint8_t* diff, uint32_t diff_size,
                      uint8_t* out, uint32_t capacity) {
    usize_t size = 0;
    int err = xd3_decode_memory(diff, diff_size, old, old_size, out, &size, capacity, 0);
    if (err) return 0;
    return (uint32_t)size;
}
