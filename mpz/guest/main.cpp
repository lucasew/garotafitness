// WASM export over the RetDec MP3Model transcription (FULL.c).
// INV-03: this module is built here; MpzSlimmer.dll is never loaded.

#include "host.h"

#include <stdint.h>
#include <stdlib.h>

extern "C" {
int32_t _3f__3f_0MP3Model_40__40_QAE_40_XZ(int32_t *th, int32_t *result);
void _3f_Input_40_MP3_40__40_QAEXPAU_iobuf_40__40__40_Z(int32_t *th, int32_t *fp);
void _3f_Output_40_MP3_40__40_QAEXPAU_iobuf_40__40__40_Z(int32_t *th, int32_t *fp);
void _3f_Process_40_MP3Model_40__40_QAEXXZ(void);
int32_t _3f_ReadFrame_40_MP3_40__40_QAEHXZ(void);
int32_t _3f_WriteFrame_40_MP3_40__40_QAEHXZ(void);
void _3f_FlushModel_40_MP3Model_40__40_QAEXXZ(void);
void _3f_WriteFlush_40_MP3_40__40_QAEXXZ(void);
int32_t function_100164c4(void);
}

// Object spans FlushModel +0x171f0c and Process +0x19260 tables.
static const uint32_t kModelSize = 0x200000;

extern "C" int32_t mpz_decode(uint8_t *src, uint32_t slen, uint8_t *dst, uint32_t dcap,
                             uint32_t orig, uint32_t frames) {
  if (src == nullptr || dst == nullptr || slen == 0 || dcap == 0) {
    return 0;
  }
  pe_image_init();
  host_set_files(src, slen, dst, dcap);
  uint8_t *model = static_cast<uint8_t *>(calloc(1, kModelSize));
  if (model == nullptr) {
    return 0;
  }
  g_self = static_cast<int32_t>(reinterpret_cast<uintptr_t>(model));
  int32_t *self = reinterpret_cast<int32_t *>(model);
  _3f__3f_0MP3Model_40__40_QAE_40_XZ(self, self);
  function_100164c4();
  int32_t in_tag = 1;
  int32_t out_tag = 2;
  _3f_Input_40_MP3_40__40_QAEXPAU_iobuf_40__40__40_Z(self, &in_tag);
  _3f_Output_40_MP3_40__40_QAEXPAU_iobuf_40__40__40_Z(self, &out_tag);
  uint32_t n = frames == 0 ? 1 : frames;
  if (n > 65536) {
    n = 65536;
  }
  for (uint32_t i = 0; i < n; i++) {
    (void)_3f_ReadFrame_40_MP3_40__40_QAEHXZ();
    _3f_Process_40_MP3Model_40__40_QAEXXZ();
    (void)_3f_WriteFrame_40_MP3_40__40_QAEHXZ();
    if (host_written() >= dcap) {
      break;
    }
  }
  _3f_FlushModel_40_MP3Model_40__40_QAEXXZ();
  _3f_WriteFlush_40_MP3_40__40_QAEXXZ();
  uint32_t wrote = host_written();
  free(model);
  if (orig != 0 && wrote > orig) {
    wrote = orig;
  }
  return static_cast<int32_t>(wrote);
}
