// WASM export over the RetDec CLS-OGGRE transcription (CLS-OGGRE.c).
// INV-03: this module is built here; CLS-OGGRE.dll is never loaded.

#include "host.h"

#include <stdint.h>
#include <stdlib.h>

// ClsMain is entry_point @ 0x1000. a1==4 is CLS_DECOMPRESS.
// unknown_5c001052 is the RetDec decode symbol (call from ClsMain).

extern "C" int32_t mpzz_decode(uint8_t *src, uint32_t slen, uint8_t *dst, uint32_t dcap) {
  if (src == nullptr || dst == nullptr || slen == 0 || dcap == 0) {
    return 0;
  }
  pe_image_init();
  host_set_files(src, slen, dst, dcap);
  int32_t cb = static_cast<int32_t>(reinterpret_cast<intptr_t>(&host_cls_callback));
  int32_t inst = 1;
  (void)entry_point(4, cb, inst);
  return static_cast<int32_t>(host_written());
}
