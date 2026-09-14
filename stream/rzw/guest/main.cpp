// WASM export over the RetDec RAZOR 1.00 transcription (FULL.c).
// INV-03: this module is built here; rz.exe / razor.dll are never loaded.

#include "host.h"

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

extern "C" {
int64_t function_409050(int64_t a1, int64_t a2, int64_t a3);
int64_t function_40ec30(int64_t a1, int64_t a2, int64_t a3, int64_t a4);
int64_t function_411f10(int64_t a1);
int64_t function_41ba10(int64_t a1);
int64_t function_41b5e0(int64_t a1);
}

// Decoder object spans rANS tables at +0x14f68 / +0x15098 and window ptrs.
static const uint32_t kObjSize = 0x40000;

extern "C" int32_t rzw_decode(uint8_t *src, uint32_t slen, uint8_t *dst, uint32_t dcap) {
    if (src == nullptr || dst == nullptr || slen == 0 || dcap == 0) {
        return 0;
    }
    pe_image_init();
    host_set_files(src, slen, dst, dcap);

    uint8_t *parent = static_cast<uint8_t *>(calloc(1, 4096));
    uint8_t *aux = static_cast<uint8_t *>(calloc(1, 4096));
    uint8_t *obj = static_cast<uint8_t *>(calloc(1, kObjSize));
    uint8_t *tab = static_cast<uint8_t *>(calloc(1, 36u * 8192u));
    if (parent == nullptr || aux == nullptr || obj == nullptr || tab == nullptr) {
        free(parent);
        free(aux);
        free(obj);
        free(tab);
        return 0;
    }

    // PE constructor (function_40ec30) mallocs the rANS/window objects.
    function_40ec30(reinterpret_cast<int64_t>(parent), 0, reinterpret_cast<int64_t>(aux),
                    static_cast<int64_t>(dcap));
    // Decoder object the vtable at g51 / 0x409050 expects.
    int64_t self = reinterpret_cast<int64_t>(obj);
    g_self = self;
    function_41ba10(self);

    *reinterpret_cast<int64_t *>(obj + 88) = reinterpret_cast<int64_t>(dst);
    *reinterpret_cast<int32_t *>(obj + 104) = static_cast<int32_t>(dcap > 0 ? dcap - 1 : 0);
    *reinterpret_cast<int32_t *>(obj + 108) = static_cast<int32_t>(dcap);
    *reinterpret_cast<int32_t *>(obj + 116) = 0;
    *reinterpret_cast<int64_t *>(obj + 128) = reinterpret_cast<int64_t>(tab);
    *reinterpret_cast<int32_t *>(obj + 136) = 0;
    *reinterpret_cast<int32_t *>(obj + 140) = 0;
    *reinterpret_cast<int32_t *>(obj + 144) = 0;
    *reinterpret_cast<int32_t *>(obj + 148) = 0;

    function_409050(self, 0, 0);

    uint32_t wrote = host_written();
    int32_t pos = *reinterpret_cast<int32_t *>(obj + 148);
    if (wrote == 0 && pos > 0 && static_cast<uint32_t>(pos) <= dcap) {
        wrote = static_cast<uint32_t>(pos);
    }
    if (wrote > dcap) {
        wrote = dcap;
    }
    free(parent);
    free(aux);
    free(obj);
    free(tab);
    return static_cast<int32_t>(wrote);
}
