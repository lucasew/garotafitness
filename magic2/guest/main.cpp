// Reconstructed lolz v22c4b kernel (cls-magic2 is PE-only; INV-03).
// Same guest shape as srep: C++ in, Go NewReader + wazero out.

#include <stdint.h>
#include <string.h>
#ifdef HOST_DEBUG
#include <stdio.h>
#include <stdlib.h>
#endif

static const uint32_t kL = 1u << 23;
static const uint32_t kMN = 1u << 15;
static const uint32_t kMB = 1u << 14;
static const int kWant = 430889;
static const int kEmu = 2895;
static const int kApp = 6;
static const uint32_t kAppCRC = 0xf75982bb;
// One decodeOpt @ 0x14003afc0 on fg-06: scale-15 bit + 16-sym, then 0xb48.
// PE: p0=0x4000 idx=0, CDF i*0x800, LE state 0x20, renorm 02 00 25 → off=7.
static const int kOptSkip = 7;

static const uint16_t kNibbleTgt[16][16] = {
    {0x0000, 0x8007, 0x800f, 0x8017, 0x801f, 0x8027, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x800f, 0x8017, 0x801f, 0x8027, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x8017, 0x801f, 0x8027, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x801f, 0x8027, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x8027, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x802f, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x8037, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x803f, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x8047, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x804f, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x8057, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x0058, 0x805f, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x0058, 0x0060, 0x8067, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x0058, 0x0060, 0x0068, 0x806f, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x0058, 0x0060, 0x0068, 0x0070, 0x8077},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038, 0x0040, 0x0048, 0x0050, 0x0058, 0x0060, 0x0068, 0x0070, 0x0078},
};

static const uint16_t kNibble9Tgt[9][16] = {
    {0x0000, 0x8006, 0x800d, 0x8014, 0x801b, 0x8022, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
    {0x0000, 0x0007, 0x800d, 0x8014, 0x801b, 0x8022, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
    {0x0000, 0x0007, 0x000e, 0x8014, 0x801b, 0x8022, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
    {0x0000, 0x0007, 0x000e, 0x0015, 0x801b, 0x8022, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
    {0x0000, 0x0007, 0x000e, 0x0015, 0x001c, 0x8022, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
    {0x0000, 0x0007, 0x000e, 0x0015, 0x001c, 0x0023, 0x8029, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
    {0x0000, 0x0007, 0x000e, 0x0015, 0x001c, 0x0023, 0x002a, 0x8030, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
    {0x0000, 0x0007, 0x000e, 0x0015, 0x001c, 0x0023, 0x002a, 0x0031, 0x8037, 0x8000, 0, 0, 0, 0, 0, 0},
    {0x0000, 0x0007, 0x000e, 0x0015, 0x001c, 0x0023, 0x002a, 0x0031, 0x0038, 0x8000, 0, 0, 0, 0, 0, 0},
};

static const uint16_t kMatchTgt[16][16] = {
    {0x0000, 0x8003, 0x8007, 0x800b, 0x800f, 0x8013, 0x8017, 0x801b, 0x801f, 0x8023, 0x8027, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x8007, 0x800b, 0x800f, 0x8013, 0x8017, 0x801b, 0x801f, 0x8023, 0x8027, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x800b, 0x800f, 0x8013, 0x8017, 0x801b, 0x801f, 0x8023, 0x8027, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x800f, 0x8013, 0x8017, 0x801b, 0x801f, 0x8023, 0x8027, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x8013, 0x8017, 0x801b, 0x801f, 0x8023, 0x8027, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x8017, 0x801b, 0x801f, 0x8023, 0x8027, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x0018, 0x801b, 0x801f, 0x8023, 0x8027, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x0018, 0x001c, 0x801f, 0x8023, 0x8027, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x0018, 0x001c, 0x0020, 0x8023, 0x8027, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x0018, 0x001c, 0x0020, 0x0024, 0x8027, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x0018, 0x001c, 0x0020, 0x0024, 0x0028, 0x802b, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x0018, 0x001c, 0x0020, 0x0024, 0x0028, 0x002c, 0x802f, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x0018, 0x001c, 0x0020, 0x0024, 0x0028, 0x002c, 0x0030, 0x8033, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x0018, 0x001c, 0x0020, 0x0024, 0x0028, 0x002c, 0x0030, 0x0034, 0x8037, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x0018, 0x001c, 0x0020, 0x0024, 0x0028, 0x002c, 0x0030, 0x0034, 0x0038, 0x803b},
    {0x0000, 0x0004, 0x0008, 0x000c, 0x0010, 0x0014, 0x0018, 0x001c, 0x0020, 0x0024, 0x0028, 0x002c, 0x0030, 0x0034, 0x0038, 0x003c},
};

static const uint16_t kSym8Tgt[7][8] = {
    {0x0000, 0x8008, 0x8011, 0x801a, 0x8023, 0x802c, 0x8035, 0x8000},
    {0x0000, 0x0009, 0x8011, 0x801a, 0x8023, 0x802c, 0x8035, 0x8000},
    {0x0000, 0x0009, 0x0012, 0x801a, 0x8023, 0x802c, 0x8035, 0x8000},
    {0x0000, 0x0009, 0x0012, 0x001b, 0x8023, 0x802c, 0x8035, 0x8000},
    {0x0000, 0x0009, 0x0012, 0x001b, 0x0024, 0x802c, 0x8035, 0x8000},
    {0x0000, 0x0009, 0x0012, 0x001b, 0x0024, 0x002d, 0x8035, 0x8000},
    {0x0000, 0x0009, 0x0012, 0x001b, 0x0024, 0x002d, 0x0036, 0x8000},
};

static const int kA690[7] = {0, 1, 2, 3, 17, 18, 0};
// VA 0x14000a697. cls10: edx = table[sym] after the +0x364ca 16-sym.
static const int kA697[16] = {4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 19, 20, 21};

// VA 0x1400011a0. cls1 first 8-sym adapt (psraw $6) at 0x14003a24d.
static const uint16_t kOff8Tgt[8][8] = {
    {0x0000, 0x8007, 0x800f, 0x8017, 0x801f, 0x8027, 0x802f, 0x8037},
    {0x0000, 0x0008, 0x800f, 0x8017, 0x801f, 0x8027, 0x802f, 0x8037},
    {0x0000, 0x0008, 0x0010, 0x8017, 0x801f, 0x8027, 0x802f, 0x8037},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x801f, 0x8027, 0x802f, 0x8037},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x8027, 0x802f, 0x8037},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x802f, 0x8037},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x8037},
    {0x0000, 0x0008, 0x0010, 0x0018, 0x0020, 0x0028, 0x0030, 0x0038},
};

// VA 0x140002980. Option-header 16-sym adapt (psraw $5) at 0x14003b132.
static const uint16_t kHdrTgt[16][16] = {
    {0x0000, 0x8001, 0x8003, 0x8005, 0x8007, 0x8009, 0x800b, 0x800d, 0x800f, 0x8011, 0x8013, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x8003, 0x8005, 0x8007, 0x8009, 0x800b, 0x800d, 0x800f, 0x8011, 0x8013, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x8005, 0x8007, 0x8009, 0x800b, 0x800d, 0x800f, 0x8011, 0x8013, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x8007, 0x8009, 0x800b, 0x800d, 0x800f, 0x8011, 0x8013, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x8009, 0x800b, 0x800d, 0x800f, 0x8011, 0x8013, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x800b, 0x800d, 0x800f, 0x8011, 0x8013, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x000c, 0x800d, 0x800f, 0x8011, 0x8013, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x000c, 0x000e, 0x800f, 0x8011, 0x8013, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x000c, 0x000e, 0x0010, 0x8011, 0x8013, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x000c, 0x000e, 0x0010, 0x0012, 0x8013, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x000c, 0x000e, 0x0010, 0x0012, 0x0014, 0x8015, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x000c, 0x000e, 0x0010, 0x0012, 0x0014, 0x0016, 0x8017, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x000c, 0x000e, 0x0010, 0x0012, 0x0014, 0x0016, 0x0018, 0x8019, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x000c, 0x000e, 0x0010, 0x0012, 0x0014, 0x0016, 0x0018, 0x001a, 0x801b, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x000c, 0x000e, 0x0010, 0x0012, 0x0014, 0x0016, 0x0018, 0x001a, 0x001c, 0x801d},
    {0x0000, 0x0002, 0x0004, 0x0006, 0x0008, 0x000a, 0x000c, 0x000e, 0x0010, 0x0012, 0x0014, 0x0016, 0x0018, 0x001a, 0x001c, 0x001e},
};

// VA 0x14000a6a7 / 0x14000a6b7. 16-sym → option id (0x14003b16e / 0x14003b240).
static const uint8_t kA6A7[16] = {0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x0c, 0x0d, 0x0e, 0x0f, 0x3f, 0x3e, 0x00};
static const uint8_t kA6B7[16] = {0x10, 0x11, 0x12, 0x13, 0x20, 0x21, 0x22, 0x23, 0x24, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00};
static const uint8_t kA6C7[16] = {0x00, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00};

// VA 0x140002f20. 8-sym length adapt (psraw $7).
static const uint16_t kLen8Tgt[8][8] = {
    {0x0000, 0x800f, 0x801f, 0x802f, 0x803f, 0x804f, 0x805f, 0x806f},
    {0x0000, 0x0010, 0x801f, 0x802f, 0x803f, 0x804f, 0x805f, 0x806f},
    {0x0000, 0x0010, 0x0020, 0x802f, 0x803f, 0x804f, 0x805f, 0x806f},
    {0x0000, 0x0010, 0x0020, 0x0030, 0x803f, 0x804f, 0x805f, 0x806f},
    {0x0000, 0x0010, 0x0020, 0x0030, 0x0040, 0x804f, 0x805f, 0x806f},
    {0x0000, 0x0010, 0x0020, 0x0030, 0x0040, 0x0050, 0x805f, 0x806f},
    {0x0000, 0x0010, 0x0020, 0x0030, 0x0040, 0x0050, 0x0060, 0x806f},
    {0x0000, 0x0010, 0x0020, 0x0030, 0x0040, 0x0050, 0x0060, 0x0070},
};

// VA 0x140005ac0. Decode init (0x140028b67) sets [obj+0x70] here.
// mixer = table[hist]; hist = 0x5b40[n][pos & pc_mask]. Default n=0
// is all-zero hist, so mixer is always 99 and the second bit is skipped.
// tok==2 is DXT only when mixer <= 8 (jmp table 0xa880); else error.
static const uint8_t kMixTab[80] = {
    99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 2,
    99, 99, 99, 3, 99, 99, 99, 4, 99, 99, 99, 99, 99, 99, 99, 0, 99, 1, 99, 99, 99, 99, 99, 5, 6, 99, 7, 99, 99, 8, 99, 99,
    99, 9, 99, 99, 99, 10, 11, 12, 13, 0, 0, 0, 0, 0, 0, 0,
};

// VA 0x140005b40. 16-byte rows; n = min([obj+0x64], 0x24). Ctor +0x64 = 0.
static const uint8_t kHistTab[37][16] = {
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x37, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x38, 0x39, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x3a, 0x3b, 0x3c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x3d, 0x3e, 0x3f, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x1f, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x1f, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26},
    {0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x1f, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x01, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x03, 0x04, 0x05, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x41, 0x42, 0x43, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x45, 0x42, 0x43, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x46, 0x42, 0x43, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x47, 0x42, 0x43, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
    {0x48, 0x42, 0x43, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
};

// VA 0x140005a98. [obj+0xc39] when n < 0x25 (0x140028c91).
static const uint8_t kPcMask[37] = {
    0, 0, 1, 0, 3, 7, 15, 15, 15, 0, 0, 0, 1, 3, 7, 15,
    0, 1, 1, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    3, 3, 7, 3, 15,
};

// VA 0x14000a260. Literal uses [esi]; class 11/1 +16; class 4-10 +32; class 0 +48.
static const uint8_t kEsiTab[336] = {
    0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 7, 8, 0, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 13, 13, 13, 13, 13, 10,
    11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 14, 14, 14, 14, 14, 11, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 14, 14, 14, 14, 14, 12,
    15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13,
    73, 77, 72, 76, 76, 108, 140, 75, 75, 107, 74, 74, 74, 107, 107, 73, 73, 139, 72, 72, 72, 106, 106, 106, 139, 139, 139, 69, 69, 69, 105, 145,
    77, 112, 145, 177, 76, 177, 177, 177, 75, 177, 111, 74, 74, 144, 73, 73, 73, 110, 110, 110, 72, 144, 144, 71, 109, 70, 70, 109, 109, 143, 143, 143,
    108, 108, 108, 108, 108, 108, 143, 143, 143, 143, 143, 176, 107, 107, 107, 176, 176, 176, 176, 176, 176, 176, 176, 106, 142, 142, 142, 142, 142, 142, 142, 105,
    105, 104, 104, 104, 104, 104, 104, 104, 104, 141, 141, 141, 103, 103, 103, 103, 141, 141, 141, 175, 175, 175, 175, 175, 175, 175, 175, 175, 175, 175, 175, 175,
    175, 175, 175, 140, 140, 140, 140, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 139, 174, 174, 174, 174, 174,
    174, 174, 174, 174, 174, 174, 174, 174, 174, 174, 138, 174, 138, 138, 174, 137, 137, 137, 137, 137, 137, 137, 137, 137, 137, 137, 137, 137, 137, 137, 137, 137,
    173, 173, 136, 173, 136, 136, 136, 136, 136, 136, 136, 136, 136, 136, 136, 136, 173, 173, 173, 173, 173, 173, 173, 173, 173, 173, 173, 172, 172, 172, 172, 172,
    172, 172, 172, 172, 172, 172, 172, 172, 172, 172, 172, 172, 172, 172, 172, 172,
};

static int esi_after_match(int esi, int cls, int hist) {
  // Match classes index the esi map by hist (r13), not the live esi.
  // cls0 0xa290[hist]; cls1 0xa270[hist]; cls4-10/12-15 0xa280[hist].
  (void)esi;
  int i;
  if (cls == 0) i = 48 + (hist & 15);
  else if ((cls >= 4 && cls <= 10) || (cls >= 12 && cls <= 15)) i = 32 + (hist & 15);
  else i = 16 + (hist & 15);
  if (i < 0) i = 0;
  if (i >= 336) i = 335;
  return kEsiTab[i];
}

struct Rans {
  uint32_t x;
  const uint8_t *buf;
  int off, len;
  bool ok;
};

static void renorm(Rans *r) {
  while (r->x < kL) {
    if (r->off >= r->len) {
      r->ok = false;
      return;
    }
    r->x = (r->x << 8) | r->buf[r->off++];
  }
}

static int get_bit(Rans *r, uint16_t *p0, unsigned nbits, unsigned shift) {
  if (!r->ok) return -1;
  uint32_t m = 1u << nbits;
  uint32_t p = *p0;
  if (p >= m && p != 0) p = m - 1;
  uint32_t slot = r->x & (m - 1);
  uint32_t quo = r->x >> nbits;
  if (slot < p) {
    r->x = quo * p + slot;
    *p0 = (uint16_t)(p + ((m - p) >> shift));
    renorm(r);
    return 0;
  }
  r->x = r->x - p * (quo + 1);
  uint32_t np = p - (p >> shift);
  if (np == 0) np = 1;
  *p0 = (uint16_t)np;
  renorm(r);
  return 1;
}

static void adapt16(uint16_t *cdf, int sym, const uint16_t tgt[][16], int shift) {
  if (sym < 0) sym = 0;
  if (sym > 15) sym = 15;
  for (int j = 0; j < 16; j++) {
    int16_t d = (int16_t)tgt[sym][j] - (int16_t)cdf[j];
    cdf[j] = (uint16_t)((int16_t)cdf[j] + (d >> shift));
  }
}

static void adapt8(uint16_t *cdf, int sym, const uint16_t tgt[][8], int shift) {
  if (sym < 0) sym = 0;
  if (sym > 7) sym = 7;
  for (int j = 0; j < 8; j++) {
    int16_t d = (int16_t)tgt[sym][j] - (int16_t)cdf[j];
    cdf[j] = (uint16_t)((int16_t)cdf[j] + (d >> shift));
  }
}

static int find16(const uint16_t *cdf, uint32_t slot, int last) {
  for (int j = 1; j < last; j++) {
    if ((int16_t)cdf[j] > (int16_t)slot) return j;
  }
  return last;
}

// v22 0x14002b8e6: mixed = (w*A + (uint16)(0-w)*B) >> 16  (pmulhuw+paddw)
static void mix_cdf(uint16_t *mixed, const uint16_t *A, const uint16_t *B, uint16_t w) {
  uint16_t nw = (uint16_t)(0u - w);
  for (int i = 0; i < 16; i++) {
    mixed[i] = (uint16_t)((((uint32_t)w * A[i]) >> 16) + (((uint32_t)nw * B[i]) >> 16));
  }
}

static int get_nibble_mix(Rans *r, uint16_t *A, uint16_t *B, uint16_t *wp, int last, int shift,
                          const uint16_t tgt[][16]) {
  if (!r->ok) return -1;
  uint16_t mixed[16];
  uint16_t w = *wp;
  mix_cdf(mixed, A, B, w);
  uint32_t slot = r->x & (kMN - 1);
  uint32_t quo = r->x >> 15;
  int i = find16(mixed, slot, last);
  uint32_t start = mixed[i - 1];
  uint32_t end = (i < 16) ? mixed[i] : 0x8000;
  if (end <= start) end = start + 1;
  r->x = (end - start) * quo + (slot - start);
  int sym = i - 1;
  uint16_t fA, fB;
  if (i < 16) {
    fA = (uint16_t)(A[i] - A[i - 1]);
    fB = (uint16_t)(B[i] - B[i - 1]);
  } else {
    fA = (uint16_t)(0x8000 - A[15]);
    fB = (uint16_t)(0x8000 - B[15]);
  }
  uint16_t w2 = (uint16_t)(w - (w >> 4));
  if (fA >= fB) w2 = (uint16_t)(w2 + 0x0fff);
  *wp = w2;
  adapt16(A, sym, tgt, shift);
  adapt16(B, sym, tgt, shift);
  renorm(r);
  return sym;
}

static int get_nibble(Rans *r, uint16_t *cdf, int last, int shift, const uint16_t tgt[][16]) {
  if (!r->ok) return -1;
  uint32_t slot = r->x & (kMN - 1);
  uint32_t quo = r->x >> 15;
  int i = find16(cdf, slot, last);
  uint32_t start = cdf[i - 1];
  uint32_t end = (i < last && i < 16) ? cdf[i] : 0x8000;
  if (end <= start) end = start + 1;
  r->x = (end - start) * quo + (slot - start);
  int sym = i - 1;
  adapt16(cdf, sym, tgt, shift);
  renorm(r);
  return sym;
}

// 8-sym find: add 0x100+bsf. Length uses >>7/0x2f20; cls1 s0 uses >>6/0x11a0.
static int get_sym8_tgt(Rans *r, uint16_t *cdf, const uint16_t tgt[][8], int shift) {
  if (!r->ok) return -1;
  uint32_t slot = r->x & (kMN - 1);
  uint32_t quo = r->x >> 15;
  int i = find16(cdf, slot, 8);
  uint32_t start = cdf[i - 1];
  uint32_t end = (i < 8) ? cdf[i] : 0x8000;
  if (end <= start) end = start + 1;
  r->x = (end - start) * quo + (slot - start);
  int sym = i - 1;
  adapt8(cdf, sym, tgt, shift);
  renorm(r);
  return sym;
}

static int get_sym8(Rans *r, uint16_t *cdf) { return get_sym8_tgt(r, cdf, kLen8Tgt, 7); }

// 0x14003bb93 builds obj+0xbe0 from 0xa6e7 (8-byte {base56, nbits8}).
// 0x14003bc18 builds obj+0xc00 from 0xa706. decode_int @ 0x140036d93:
// offset = base[s] + extra(nbits[s]), not (1<<s)+extra.
static const uint8_t kA6E7[] = {
    5, 5, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18,
    5, 6, 7, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
};
static const uint8_t kA706[] = {
    0, 0, 0, 1, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6,
    2, 2, 2, 3, 3, 4, 5, 6, 4, 4, 4, 4, 5, 5, 6, 7,
    8, 9, 10, 11, 12, 13, 14, 15,
};
// VA 0x14000a6d7. +0xbc0 ladder, limit 0x1000. 0x140036646.
static const uint8_t kA6D7[] = {3, 3, 3, 3, 3, 4, 4, 4, 5, 5, 6, 7, 8, 9, 10, 11};

static int be0_base(const uint8_t *nb, int ntab, int s) {
  int b = 0;
  if (s > ntab) s = ntab;
  for (int i = 0; i < s; i++) {
    int n = nb[i];
    if (n >= 30) break;
    b += 1 << n;
  }
  return b;
}

static int decode_bc0(Rans *r, uint16_t *cdf8, uint16_t *bits) {
  int s = get_sym8(r, cdf8);
  if (s < 0) return -1;
  if (s > 15) s = 15;
  int nbits = kA6D7[s];
  int d = be0_base(kA6D7, 16, s) + s; // leal -1(%rcx,%rdx) with rcx=bsf=s+1
  if (nbits > 3) {
    int k = nbits - 3;
    uint32_t raw = r->x & ((1u << k) - 1);
    r->x >>= k;
    renorm(r);
    d += (int)(raw * 8);
  }
  (void)bits;
  return d;
}

// cls2 @ 0x14003a0c4 / cls3 @ 0x14003a02a call past the image.
// In-image twin 0x140036b00: mixed 16-sym, adapt >>5 / 0x2980, escape 15.
static int decode_new_off(Rans *r, uint16_t *A, uint16_t *B, uint16_t *wp, uint16_t *esc,
                          uint16_t *bp, const uint8_t *nbtab, int ntab, uint16_t *mid, uint16_t *tail) {
#ifdef HOST_DEBUG
  static int nmix;
  if (nmix < 4) {
    fprintf(stderr, "newoff-mix slot=%04x x=%08x w=%04x\n", r->x & 0x7fff, r->x, (unsigned)*wp);
    nmix++;
  }
#endif
  int s = get_nibble_mix(r, A, B, wp, 16, 5, kHdrTgt);
  if (s < 0) return -1;
  if (s == 15) {
    int sx = get_nibble(r, esc, 16, 5, kHdrTgt);
    if (sx < 0) return -1;
    s = 15 + sx;
  }
  if (s < 0) s = 0;
  if (s >= ntab) s = ntab - 1;
  if (s > 30) s = 30;
  int nbits = nbtab[s];
  if (nbits > 24) nbits = 24;
  int d = be0_base(nbtab, ntab, s);
  if (nbits <= 5) {
    // 0x140036db8: xor esi; jmp 0x14006d0d4. nbits × scale-14, no tail.
    uint32_t extra = 0;
    for (int i = 0; i < nbits; i++) {
      int b = get_bit(r, &bp[s * 16 + i], 14, 5);
      if (b < 0) return -1;
      extra = (extra << 1) | (uint32_t)b;
    }
    d += (int)extra;
#ifdef HOST_DEBUG
    static int nrep0;
    if (nrep0 < 8) {
      fprintf(stderr, "newoff5 s=%d nbits=%d d=%d x=%08x\n", s, nbits, d, r->x);
      nrep0++;
    }
#endif
    return d;
  }
  // 0x140036dbf: 16-sym at cdf+s*34+0x4a4, adapt >>6 / 0x1f00.
  int s2 = get_nibble(r, mid + s * 16, 16, 6, kMatchTgt);
  if (s2 < 0) return -1;
  int sh = ((nbits < 9 ? 9 : nbits) + 60) & 63;
  d += s2 << sh;
  if (nbits > 9) {
    int k = nbits - 9;
    uint32_t low = r->x & ((1u << k) - 1);
    r->x >>= k;
    renorm(r);
    d += (int)(low << 5);
  }
  // tail always: s3 at s*34+s2*1088+0x8e4 >>7/0x1c00; bit at s*2+s2*64+0x4ce4.
  int s3 = get_nibble(r, tail + (s * 16 + s2) * 16, 16, 7, kNibbleTgt);
  if (s3 < 0) return -1;
  int bit = get_bit(r, &bp[s * 16 + s2], 14, 5);
  if (bit < 0) return -1;
  d += 2 * s3 + bit;
#ifdef HOST_DEBUG
  static int nrep;
  if (nrep < 8) {
    fprintf(stderr, "newoff s=%d nbits=%d s2=%d s3=%d d=%d x=%08x\n", s, nbits, s2, s3, d, r->x);
    nrep++;
  }
#endif
  return d;
}

// ROLZ lists at obj+0xc90 / cursors +0x88 / cap +0xc28.
// Default cap = (opt[+8]<<11)+0x800 → 0x800 when dword[+8]=0.
static const int kRolzCap = 2048;
static uint32_t gRolz[256][2048];
static int gRolzCur[256];

static void rolz_reset(void) {
  memset(gRolz, 0, sizeof(gRolz));
  for (int i = 0; i < 256; i++) gRolzCur[i] = 0;
}

static void rolz_push(int ctx, int pos) {
  ctx &= 255;
  int c = gRolzCur[ctx];
  gRolz[ctx][c] = (uint32_t)pos;
  c++;
  if (c >= kRolzCap) c = 0;
  gRolzCur[ctx] = c;
}

static int rolz_lookup(int ctx, int idx, int n) {
  ctx &= 255;
  int avail = gRolzCur[ctx];
  if (avail > kRolzCap) avail = kRolzCap;
  if (avail < 1) return 1;
  if (idx < 0) idx = 0;
  idx %= avail;
  int slot = avail - 1 - idx;
  int pos = (int)gRolz[ctx][slot];
  int d = n - pos;
  if (d < 1) d = 1;
  return d;
}

static int bitlen(uint32_t x) {
  if (x == 0) return 0;
  int n = 0;
  while (x) {
    x >>= 1;
    n++;
  }
  return n;
}

static uint32_t abs32(int32_t x) { return x < 0 ? (uint32_t)(-x) : (uint32_t)x; }

static uint32_t crc32_ieee(const uint8_t *p, int n) {
  uint32_t c = 0xffffffffu;
  for (int i = 0; i < n; i++) {
    c ^= p[i];
    for (int b = 0; b < 8; b++) c = (c >> 1) ^ (0xedb88320u & (uint32_t)-(int32_t)(c & 1));
  }
  return ~c;
}

static void init_nibble(uint16_t *d) {
  for (int i = 0; i < 16; i++) d[i] = (uint16_t)(i * 0x800);
}

static void init_sym8(uint16_t *d) {
  for (int i = 0; i < 8; i++) d[i] = (uint16_t)(i * 0x1000);
}

struct Hist {
  uint32_t w08, w0c, w10, w14, w18, w1c, w20, w24;
};

static int hist_h1(const Hist *h) {
  return bitlen((uint8_t)abs32((int32_t)h->w20 - (int32_t)h->w24));
}

static int hist_row(const Hist *h) {
  uint32_t a = abs32((int32_t)h->w08 - (int32_t)h->w0c);
  uint32_t b = abs32((int32_t)h->w10 - (int32_t)h->w14);
  int h0 = bitlen((uint8_t)((a + b) >> 1));
  int h1 = hist_h1(h);
  int r = h0 * 9 + h1;
  if (r < 0) return 0;
  if (r >= 81) return 80;
  return r;
}

static void apply_sample(Hist *h, uint32_t n0, uint32_t n1) {
  uint32_t a0 = (abs32((int32_t)n0) | 1);
  uint32_t a1 = (abs32((int32_t)n1) | 1);
  uint32_t m0 = ((a0 + 2 * h->w20) >> 1) & 0xff;
  uint32_t m1 = ((a1 + 2 * h->w24) >> 1) & 0xff;
  h->w08 = h->w10;
  h->w0c = h->w14;
  h->w10 = h->w20;
  h->w14 = h->w24;
  h->w18 = m0;
  h->w1c = m1;
  h->w20 = m0;
  h->w24 = m1;
}

static int extra_sample(Rans *r, Hist *h, uint16_t *bits, int bsf, int h1) {
  if (bsf <= 1) {
    apply_sample(h, 0, 0);
    return 0;
  }
  int sym = bsf - 1;
  if (sym > 8) sym = 8;
  if (h1 < 0) h1 = 0;
  if (h1 > 8) h1 = 8;
  int r10 = 0, r13 = 0, rdx = 0;
  for (int i = 0; i < sym; i++) {
    int off = (h1 << 11) + ((sym - 1) << 8) + i * 32 + 8 * rdx;
    off /= 2;
    int b0 = get_bit(r, &bits[off], 14, 4);
    if (b0 < 0) return -1;
    int b1 = get_bit(r, &bits[off + 1 + b0], 14, 4);
    if (b1 < 0) return -1;
    r10 = r10 * 2 + b0;
    r13 = r13 * 2 + b1;
    rdx = b1 + 2 * b0;
  }
  apply_sample(h, (uint32_t)r10, (uint32_t)r13);
  return r10;
}

// Insert a new distance at reps[17] (obj+0x44). PE: movdqu [+0x44]→[+0x48],
// then movl at +0x44 (cls1 0x14003a356 / x86 cls3 falls into 0x42f710).
static void insert_new_off(int *reps, int nrep, int d) {
  if (nrep > 20) reps[20] = reps[19];
  if (nrep > 19) reps[19] = reps[18];
  if (nrep > 18) reps[18] = reps[17];
  if (nrep > 17) reps[17] = d;
}

// 0x140039dd0: a690[cls-4] is an index into the recent-offset array
// (0..3, 17, 18), not a raw bit count. Rotate that slot to [0].
// cls1/2/3 insert at +0x44 (index 17) and return the new distance.
static int decode_off(int cls, int extra, int *rep0, int *reps, int nrep) {
  if (cls == 0) {
    if (*rep0 < 1) *rep0 = 1;
    return *rep0;
  }
  if ((cls >= 4 && cls <= 10) || (cls >= 12 && cls <= 15)) {
    int idx = extra;
    if (cls >= 12) idx = cls - 12;
    else if (cls != 10) idx = kA690[cls - 4];
    if (idx >= 0 && idx < nrep) {
      int d = reps[idx];
      if (idx > 0) {
        for (int i = idx; i > 0; i--) reps[i] = reps[i - 1];
      }
      if (d < 1) d = 1;
      reps[0] = d;
      *rep0 = d;
      return d;
    }
  }
  // cls 1/2/3/11: insert at +0x44 only. class 0 still reads reps[0].
  // decode_int returns 0 (cls2 testl %ebp,%ebp / je). Do not bump to 1.
  int d = extra;
  if (d < 0) d = 0;
  insert_new_off(reps, nrep, d);
  return d;
}

static int hit_crc(const uint8_t *out, int n) {
  if (n < kEmu + kApp) return 0;
  return crc32_ieee(out + kEmu, kApp) == kAppCRC;
}

static int decode_iir(const uint8_t *src, int slen, uint8_t *dst, int dcap, unsigned bit_adapt) {
  if (slen < 4) return 0;
  Rans r;
  r.buf = src;
  r.off = 4;
  r.len = slen;
  r.ok = true;
  r.x = ((uint32_t)src[0] << 24) | ((uint32_t)src[1] << 16) | ((uint32_t)src[2] << 8) | src[3];
  renorm(&r);
  uint16_t hiGrid[81 * 16];
  uint16_t clsGrid[81 * 16];
  for (int i = 0; i < 81; i++) {
    init_nibble(hiGrid + i * 16);
    init_nibble(clsGrid + i * 16);
  }
  uint16_t hiBits[9 * 2048 / 2];
  for (int i = 0; i < 9 * 1024; i++) hiBits[i] = kMB / 2;
  uint16_t litP = kMB / 2;
  uint16_t lenTab[256 * 8];
  for (int i = 0; i < 256; i++) init_sym8(lenTab + i * 8);
  uint16_t bmTab[64];
  for (int i = 0; i < 64; i++) bmTab[i] = kMB / 2;
  Hist hi = {}, clsH = {};
  int n = 0;
  int prev = 0, rep0 = 1;
  int reps[4] = {1, 1, 1, 1};
  while (n < dcap && n < kWant && r.ok) {
    if (r.x < kL && r.off >= r.len) break;
    int bit = get_bit(&r, &litP, 14, bit_adapt);
    if (bit < 0) break;
    if (bit == 0) {
      int row = hist_row(&hi);
      int bsf;
      int hn = get_nibble(&r, hiGrid + row * 16, 9, 6, kNibble9Tgt);
      if (hn < 0) break;
      bsf = hn + 1;
      int r10 = extra_sample(&r, &hi, hiBits, bsf, hist_h1(&hi));
      if (r10 < 0) break;
      dst[n++] = (uint8_t)r10;
      prev = dst[n - 1];
      continue;
    }
    if (n == 0) break;
    int row = hist_row(&clsH);
    int cls = get_nibble(&r, clsGrid + row * 16, 16, 6, kMatchTgt);
    if (cls < 0 || cls > 11) break;
    apply_sample(&clsH, (uint32_t)(cls & 0xf), 0);
    int extra = 0;
    if (cls != 0) {
      int nb = (cls >= 4 && cls <= 9) ? kA690[cls - 4] : -1;
      if (nb > 0) {
        for (int i = 0; i < nb && i < 18; i++) {
          int b = get_bit(&r, &bmTab[i % 64], 14, 4);
          if (b < 0) return 0;
          extra = extra * 2 + b;
        }
      } else if (cls == 1 || cls == 2 || cls == 3 || cls == 11) {
        int d = get_nibble(&r, hiGrid + hist_row(&hi) * 16, 16, 7, kNibbleTgt);
        if (d < 0) break;
        extra = d + 1;
      }
    }
    decode_off(cls, extra, &rep0, reps, 4);
    int ln = get_nibble(&r, lenTab + (prev % 256) * 8, 8, 6, kNibbleTgt);
    if (ln < 0) break;
    int m = ln + 3;
    if (m == 10) {
      int en = get_nibble(&r, lenTab + (prev % 256) * 8, 8, 6, kNibbleTgt);
      if (en < 0) break;
      m = 10 + en;
    }
    if (m <= 0 || rep0 <= 0 || rep0 > n) break;
    for (int i = 0; i < m && n < dcap && n < kWant; i++) {
      dst[n] = dst[n - rep0];
      prev = dst[n];
      n++;
    }
  }
  return hit_crc(dst, n) ? n : 0;
}

static int ctx_hi(int prev, int rep0lit, int pos) {
  return (prev >> 0) | ((rep0lit >> 4) << 8) | ((pos & 3) << 12);
}

// v20 0x140013a64 / v22 0x14002ba51: if hi == prev>>4 use (prev&0xf)+16 else hi.
static int ctx_lo_pe(int prev, int hi) {
  if (hi == ((prev >> 4) & 0xf)) return (prev & 0xf) + 16;
  return hi;
}

// 0x14003afc0 decodeOpt: separate rANS at 0xb40/0xb48, not the LZ 0xb38.
// Scale-15 bit (p at model[idx], idx at +0x4a58=0, p0=0x4000), then 16-sym
// at +0x10 (escape 15 → +0x32). Caller 0x140028c3d (rel32 past image).
// Only +0xb38 write is 0x14002908e: movq %r9, 0xb38(%r12) from stream.pos.
// After bit+16-sym, 0xb48 is payload+7; LZ reloads LE dword there.
static int decode_opt_header(Rans *r) {
  uint16_t ptab[2] = {0x4000, 0x4000};
  int idx = 0;
  int bit = get_bit(r, &ptab[idx], 15, 5);
  if (bit < 0) return -1;
  idx = bit;
  uint16_t row0[16], row1[16];
  init_nibble(row0);
  init_nibble(row1);
  int s = get_nibble(r, row0, 16, 5, kHdrTgt);
  if (s < 0) return -1;
  int opt = (s < 15) ? kA6A7[s] : 0;
  if (s == 15) {
    int s2 = get_nibble(r, row1, 16, 5, kHdrTgt);
    if (s2 < 0) return -1;
    opt = kA6B7[s2 & 15];
  }
#ifdef HOST_TRACE
  fprintf(stderr, "hdr bit=%d s=%d opt=%02x x=%08x off=%d\n", bit, s, opt, r->x, r->off);
#endif
  return opt;
}

static int decode_v22(const uint8_t *src, int slen, uint8_t *dst, int dcap, int use_second, int use_hdr, int opt_skip = 0,
                      int force_opt = -1) {
  if (opt_skip < 0) opt_skip = 0;
  if (slen < opt_skip + 4) return 0;
  src += opt_skip;
  slen -= opt_skip;
  Rans r;
  r.buf = src;
  r.off = 4;
  r.len = slen;
  r.ok = true;
  // PE 0x14002973b: movl (%rax),%r10d stores the LE dword to 0xe4(%rsp).
  // There is no renorm between that store and the first token at 0x140029e04.
  r.x = (uint32_t)src[0] | ((uint32_t)src[1] << 8) | ((uint32_t)src[2] << 16) | ((uint32_t)src[3] << 24);
  int hdr_opt = 0;
  if (use_hdr) {
    hdr_opt = decode_opt_header(&r);
    if (hdr_opt < 0) return 0;
    if (use_hdr == 2 && r.off + 4 <= r.len) {
      // Reload LE dword at the advanced 0xb48 (same as kOptSkip on a fresh rANS).
      r.x = (uint32_t)r.buf[r.off] | ((uint32_t)r.buf[r.off + 1] << 8) |
            ((uint32_t)r.buf[r.off + 2] << 16) | ((uint32_t)r.buf[r.off + 3] << 24);
      r.off += 4;
    }
  }
  const int nHi = 16384, nCls = 256, nLen = 256, nBM = 64;
  static uint16_t hiA[16 * 16];
  static uint16_t hiB[256 * 16];
  static uint16_t loA[32 * 16];
  static uint16_t loB[256 * 16];
  // Packed 16-wide. PE model stride 34 is padding, not our row pitch.
  static uint16_t clsTab[256 * 16];
  static uint16_t lenTab[256 * 8];
  static uint16_t off8a[32 * 8];
  static uint16_t off8b[32 * 8];
  static uint16_t off2A[16];
  static uint16_t off2B[32 * 16];
  static uint16_t off2Esc[16];
  static uint16_t off3A[16];
  static uint16_t off3B[32 * 16];
  static uint16_t off3Esc[16];
  static uint16_t off2Mid[32 * 16], off2Tail[32 * 16 * 16], off2Bp[32 * 16];
  static uint16_t off3Mid[32 * 16], off3Tail[32 * 16 * 16], off3Bp[32 * 16];
  static uint16_t off11A[16], off11B[16], off11Esc[16], off11W = 0x8000;
  static uint16_t off11Mid[32 * 16], off11Tail[32 * 16 * 16], off11Bp[32 * 16];
  static uint16_t off11s0[8], off11s1[8];
  static uint16_t cls10Tab[16];
  static uint16_t offW2 = 0x8000, offW3 = 0x8000, offW11 = 0x8000;
  static uint16_t offBits[4096];
  static uint16_t bmTab[64];
  // PE hi w: +0xc1e80 + (prev>>bm)*0x920 + hist*32 + esi*2
  static uint16_t wHi[16 * 32];
  static uint16_t wLo[16 * 32];
  for (int i = 0; i < 16; i++) init_nibble(hiA + i * 16);
  for (int i = 0; i < 256; i++) init_nibble(hiB + i * 16);
  for (int i = 0; i < 32; i++) init_nibble(loA + i * 16);
  for (int i = 0; i < 256; i++) init_nibble(loB + i * 16);
  for (int i = 0; i < nCls; i++) init_nibble(clsTab + i * 16);
  for (int i = 0; i < nLen; i++) init_sym8(lenTab + i * 8);
  for (int i = 0; i < 32; i++) {
    init_sym8(off8a + i * 8);
    init_sym8(off8b + i * 8);
    init_nibble(off2B + i * 16);
    init_nibble(off3B + i * 16);
  }
  init_nibble(off2A);
  init_nibble(off2Esc);
  init_nibble(off3A);
  init_nibble(off3Esc);
  for (int i = 0; i < 32; i++) {
    init_nibble(off2Mid + i * 16);
    init_nibble(off3Mid + i * 16);
  }
  for (int i = 0; i < 32 * 16; i++) {
    init_nibble(off2Tail + i * 16);
    init_nibble(off3Tail + i * 16);
    off2Bp[i] = kMB / 2;
    off3Bp[i] = kMB / 2;
  }
  init_nibble(off11A);
  init_nibble(off11B);
  init_nibble(off11Esc);
  off11W = 0x8000;
  for (int i = 0; i < 32; i++) init_nibble(off11Mid + i * 16);
  for (int i = 0; i < 32 * 16; i++) {
    init_nibble(off11Tail + i * 16);
    off11Bp[i] = kMB / 2;
  }
  init_sym8(off11s0);
  init_sym8(off11s1);
  init_nibble(cls10Tab);
  offW2 = offW3 = offW11 = 0x8000;
  for (int i = 0; i < 4096; i++) offBits[i] = kMB / 2;
  for (int i = 0; i < nBM; i++) bmTab[i] = kMB / 2;
  for (int i = 0; i < 16 * 32; i++) wHi[i] = 0x8000;
  for (int i = 0; i < 16 * 32; i++) wLo[i] = 0x8000;
  uint16_t litP[4096];
  for (int i = 0; i < 4096; i++) litP[i] = kMB / 2;
  int n = 0, prev = 0, rep0lit = 0, rep0 = 1, esi = 0;
  rolz_reset();
  int reps[32];
  for (int i = 0; i < 32; i++) reps[i] = 1;
  // 0x140028c49: movb al, 0x64(%r12) after decodeOpt. 0x5a98[opt] → +0xc39.
  int opt_n = force_opt >= 0 ? force_opt : (use_hdr ? hdr_opt : 0);
  if (opt_n < 0) opt_n = 0;
  if (opt_n > 36) opt_n = 36;
  const int pc_mask = kPcMask[opt_n];
  while (n < dcap && n < kWant && r.ok) {
    if (r.x < kL && r.off >= r.len) break;
    int hist = kHistTab[opt_n][n & pc_mask];
    int mix = kMixTab[hist];
    // p0 at model+0xb50 + (hist<<6) + esi*4  (0x140029e53)
    int pctx = hist * 32 + esi * 2;
    if (pctx < 0 || pctx + 1 >= 4096) break;
#ifdef HOST_TRACE
    if (n < 3) fprintf(stderr, "pre-tok n=%d x=%08x p0=%04x pctx=%d\n", n, r.x, litP[pctx], pctx);
#endif
    int bit = get_bit(&r, &litP[pctx], 14, 5);
    if (bit < 0) break;
#ifdef HOST_TRACE
    if (n < 3) fprintf(stderr, "post-tok n=%d bit=%d x=%08x p0=%04x\n", n, bit, r.x, litP[pctx]);
#endif
    int tok = bit; // 0=lit 1=match; 2=DXT if mixer<99 and second bit
    if (use_second && bit == 0 && mix < 0x63) {
      int b2 = get_bit(&r, &litP[pctx + 1], 14, 5);
      if (b2 < 0) break;
      if (b2 == 1) tok = 2;
    }
    if (tok == 0) {
      // defaults -blr4 -blo8 -bll8 -bm4 → shifts 4,0,0,4
      int ha = (prev >> 4) & 15;
      int hb = prev & 255;
#ifdef HOST_TRACE
      if (n < 3) fprintf(stderr, "pre-hi n=%d x=%08x ha=%d hb=%d\n", n, r.x, ha, hb);
#endif
      int wix = ha * 32 + (esi & 31);
      int hi = get_nibble_mix(&r, hiA + ha * 16, hiB + hb * 16, &wHi[wix], 16, 6, kMatchTgt);
      if (hi < 0) break;
#ifdef HOST_DEBUG
      if (n < 4) fprintf(stderr, "post-hi n=%d hi=%d x=%08x slot=%04x\n", n, hi, r.x, r.x & 0x7fff);
#endif
      int la = ctx_lo_pe(prev, hi);
      if (la < 0) la = 0;
      if (la > 31) la = 31;
      int lo = get_nibble_mix(&r, loA + la * 16, loB + hb * 16, &wLo[wix], 16, 7, kNibbleTgt);
      if (lo < 0) break;
      uint8_t b = (uint8_t)((hi << 4) | lo);
#ifdef HOST_DEBUG
      if (n < 40) fprintf(stderr, "lit n=%d b=%02x hi=%d lo=%d x=%08x slot=%04x esi=%d\n", n, b, hi, lo, r.x, r.x & 0x7fff, esi);
#endif
      dst[n] = b;
      rolz_push(prev, n);
      rolz_push(b, n);
      n++;
      prev = rep0lit = b;
      esi = kEsiTab[esi];
      continue;
    }
    if (tok == 2) {
#ifdef HOST_DEBUG
      fprintf(stderr, "stop dxt n=%d mix=%d x=%08x\n", n, mix, r.x);
#endif
      break;
    }
    // class CDF at *(+0xb50)+0x1240 + esi*544 + hist*34 (0x140039b04).
    // Packed 16-wide: crow = esi*16 + hist. Axes were swapped.
    int crow = esi * 16 + hist;
    if (crow >= nCls) crow = nCls - 1;
#ifdef HOST_DEBUG
    if (n < 40) fprintf(stderr, "pre-cls n=%d x=%08x slot=%04x crow=%d esi=%d\n", n, r.x, r.x & 0x7fff, crow, esi);
#endif
    int cls = get_nibble(&r, clsTab + crow * 16, 16, 6, kMatchTgt);
    // PE 0x140039bf9: cmp r15, 0xb / ja 0x14003a3a0 — cls 12-15 are
    // reps[cls-12] with rotate and immediate length 2, not an error.
    if (cls < 0 || cls > 15) {
#ifdef HOST_DEBUG
      fprintf(stderr, "stop cls n=%d cls=%d crow=%d x=%08x\n", n, cls, crow, r.x);
#endif
      break;
    }
    int extra = 0;
    int m_fixed = -1;
    if (cls == 1) {
      // s0 at +0x1c560 adapt >>6 / 0x11a0; s1 at +0x1ca82 >>7 / 0x2f20
      // lea rbp,[rcx+rbp*8] (0-based == x86 lea ebx,[eax+edx*8-9])
      // bytes after that lea: f9 99 03 00 00 (objdump desyncs); twin
      // 0x14003626f is mov edi,2 so length is the immediate 2.
      int arow = hist % 32;
      int s0 = get_sym8_tgt(&r, off8a + arow * 8, kOff8Tgt, 6);
      if (s0 < 0) break;
      int brow = ((prev >= 10 ? 8 : 0) + s0) % 32;
      int s1 = get_sym8_tgt(&r, off8b + brow * 8, kLen8Tgt, 7);
      if (s1 < 0) break;
      extra = s1 + s0 * 8;
      if (extra < 1) extra = 1;
    } else if (cls == 2) {
      int brow = bitlen((uint32_t)rep0) % 32;
      extra = decode_new_off(&r, off2A, off2B + brow * 16, &offW2, off2Esc, off2Bp, kA6E7,
                            (int)sizeof(kA6E7), off2Mid, off2Tail);
      if (extra < 0) break;
    } else if (cls == 3) {
      int brow = bitlen((uint32_t)rep0) % 32;
      extra = decode_new_off(&r, off3A, off3B + brow * 16, &offW3, off3Esc, off3Bp, kA6E7,
                            (int)sizeof(kA6E7), off3Mid, off3Tail);
      if (extra < 0) break;
    } else if (cls == 11) {
      // +0xbc0 / 0xa6d7 8-sym integer (0x140036646), then ROLZ lookup.
      int ln = decode_bc0(&r, off11s0, off11Bp);
      if (ln < 0) break;
      int idx = decode_bc0(&r, off11s1, off11Bp);
      if (idx < 0) break;
      extra = rolz_lookup(prev, idx, n);
      m_fixed = ln + 2;
    } else if (cls == 10) {
      // PE: 16-sym at model+0x364ca, not the class row. a697[sym] = index.
      int s = get_nibble(&r, cls10Tab, 16, 6, kMatchTgt);
      if (s < 0) break;
      extra = kA697[s & 15];
    }
    int dist = decode_off(cls, extra, &rep0, reps, 32);
    int m;
    if (m_fixed >= 0) {
      m = m_fixed;
    } else if (cls == 0) {
      // 0x14003a396: class 0 length is the immediate 1, no 8-sym.
      m = 1;
    } else if (cls == 1 || (cls >= 12 && cls <= 15)) {
      // cls1: mov edi,2. cls12-15 @ 0x14003a3f6: movl $2, %ebx.
      m = 2;
    } else if (cls == 2) {
      // p0 at +0x21ca2 + (bsr(off)+1 & ~3) + hist*32 + (hist==0?2:0)
      // scale 14 adapt >>5 (0x14003a121). r13 is hist, not prev.
      int bl = bitlen((uint32_t)dist) & ~3;
      int bctx = (bl + hist * 32 + (hist == 0 ? 2 : 0)) % nBM;
      int b = get_bit(&r, &bmTab[bctx], 14, 5);
      if (b < 0) break;
      m = 3 + b;
    } else {
      // 0x140039ec8: esi*576 + hist*36 + (idx!=0)*18 + 0x3ffea
      int lrow = esi * 16 + hist;
      if (cls >= 5 && cls <= 10) lrow += 1;
      if (lrow >= nLen) lrow = nLen - 1;
      int ln = get_sym8(&r, lenTab + lrow * 8);
      if (ln < 0) break;
      m = ln + 3;
      if (cls == 3) m = ln + 5;
      if (cls == 11) m = ln + 2;
      if (m == 10) {
        int en = get_sym8(&r, lenTab + lrow * 8);
        if (en < 0) break;
        m = 10 + en;
      }
    }
#ifdef HOST_DEBUG
    if (n < 40) fprintf(stderr, "match n=%d cls=%d extra=%d m=%d dist=%d rep0=%d\n", n, cls, extra, m, dist, rep0);
#endif
    if (m <= 0 || dist < 0) {
#ifdef HOST_DEBUG
      fprintf(stderr, "stop match n=%d cls=%d m=%d dist=%d x=%08x\n", n, cls, m, dist, r.x);
#endif
      break;
    }
#ifdef HOST_DEBUG
    if (dist > n && n > 0) {
      static int once;
      if (once < 3) {
        fprintf(stderr, "oversize n=%d cls=%d m=%d dist=%d x=%08x\n", n, cls, m, dist, r.x);
        once++;
      }
    }
#endif
    // PE dict is VirtualAlloc zeros: dist>n copies 0 (same as pos==0 wrap).
    for (int i = 0; i < m && n < dcap && n < kWant; i++) {
      uint8_t b = (dist > 0 && dist <= n) ? dst[n - dist] : 0;
      dst[n] = b;
      rolz_push(prev, n);
      rolz_push(b, n);
      prev = b;
      n++;
    }
    if (n > 0) rep0lit = dst[n - 1];
    esi = esi_after_match(esi, cls, hist);
  }
#ifdef HOST_DEBUG
  fprintf(stderr, "end n=%d x=%08x off=%d ok=%d\n", n, r.x, r.off, (int)r.ok);
#endif
  return n;
}

static int peek_opt(const uint8_t *src, int slen) {
  if (slen < 4) return 0;
  Rans r;
  r.buf = src;
  r.off = 4;
  r.len = slen;
  r.ok = true;
  r.x = (uint32_t)src[0] | ((uint32_t)src[1] << 8) | ((uint32_t)src[2] << 16) | ((uint32_t)src[3] << 24);
  int o = decode_opt_header(&r);
  return o < 0 ? 0 : o;
}

extern "C" int magic2_decode(const uint8_t *src, int slen, uint8_t *dst, int dcap) {
  if (!src || slen < 4 || !dst || dcap <= 0) return 0;
  int opt = peek_opt(src, slen);
  // PE: decodeOpt writes +0x64; LZ reloads +0xb38 at stream pos 0 (separate rANS).
  int n = decode_v22(src, slen, dst, dcap, 1, 0, 0, opt);
  if (hit_crc(dst, n)) return n;
  n = decode_v22(src, slen, dst, dcap, 1, 0, 0);
  if (hit_crc(dst, n)) return n;
  n = decode_v22(src, slen, dst, dcap, 1, 0, kOptSkip, opt);
  if (hit_crc(dst, n)) return n;
  n = decode_v22(src, slen, dst, dcap, 0, 0, 0);
  if (hit_crc(dst, n)) return n;
  n = decode_v22(src, slen, dst, dcap, 1, 1, 0);
  if (hit_crc(dst, n)) return n;
  n = decode_v22(src, slen, dst, dcap, 1, 2, 0);
  if (hit_crc(dst, n)) return n;
  n = decode_v22(src, slen, dst, dcap, 0, 1, 0);
  if (hit_crc(dst, n)) return n;
  n = decode_iir(src, slen, dst, dcap, 5);
  if (hit_crc(dst, n)) return n;
  n = decode_iir(src, slen, dst, dcap, 4);
  if (hit_crc(dst, n)) return n;
  return decode_v22(src, slen, dst, dcap, 1, 0, 0);
}

#ifdef HOST_DEBUG
static void try_one(const uint8_t *src, int slen, int skip, int hdr, int opt_skip, int force_opt = -1) {
  if (skip + 4 > slen) return;
  uint8_t tmp[93116];
  int nsrc = slen - skip;
  if (nsrc > (int)sizeof(tmp)) nsrc = (int)sizeof(tmp);
  memcpy(tmp, src + skip, (size_t)nsrc);
  static uint8_t dst[430889];
  int n = decode_v22(tmp, nsrc, dst, (int)sizeof(dst), 1, hdr, opt_skip, force_opt);
  int pr = 0;
  for (int i = 0; i < n && i < 16; i++) {
    uint8_t c = dst[i];
    if (c == 9 || c == 10 || c == 13 || (c >= 32 && c < 127)) pr++;
  }
  fprintf(stderr, "skip=%d hdr=%d opt=%d force=%d n=%d pr=%d first=", skip, hdr, opt_skip, force_opt, n, pr);
  for (int i = 0; i < n && i < 16; i++) fprintf(stderr, "%02x", dst[i]);
  if (n > 0) fprintf(stderr, " ascii=%.*s", n < 16 ? n : 16, dst);
  fprintf(stderr, "\n");
  if (n >= kEmu + kApp) {
    uint32_t c = crc32_ieee(dst + kEmu, kApp);
    fprintf(stderr, "  appid=%08x %s bytes=", c, c == kAppCRC ? "HIT" : "");
    for (int i = 0; i < kApp; i++) fprintf(stderr, "%02x", dst[kEmu + i]);
    fprintf(stderr, "\n");
    if (c == kAppCRC) {
      FILE *hf = fopen("/tmp/magic2-CRC-HIT.txt", "w");
      if (hf) {
        fprintf(hf, "HIT skip=%d hdr=%d opt_skip=%d force=%d n=%d crc=%08x\n", skip, hdr, opt_skip, force_opt, n, c);
        fclose(hf);
      }
    }
  }
}

int main(int argc, char **argv) {
  const char *path = argc > 1 ? argv[1]
                              : "/media/downloads/TORRENTS/RimWorld [FitGirl Repack]/fg-06.bin";
  FILE *f = fopen(path, "rb");
  if (!f) {
    perror(path);
    return 1;
  }
  if (fseek(f, 0x1F + 5, SEEK_SET) != 0) {
    perror("seek");
    return 1;
  }
  static uint8_t src[93116];
  int slen = (int)fread(src, 1, sizeof(src), f);
  fclose(f);
  int peeked = peek_opt(src, slen);
  fprintf(stderr, "peek_opt=%d\n", peeked);
  try_one(src, slen, 0, 0, 0);
  try_one(src, slen, 0, 0, 0, peeked);
  try_one(src, slen, 0, 0, kOptSkip);
  try_one(src, slen, 0, 0, kOptSkip, peeked);
  try_one(src, slen, 0, 1, 0);
  try_one(src, slen, 0, 2, 0);
  for (int skip = 0; skip <= 16; skip++) try_one(src, slen, skip, 0, 0);
  for (int fo = 0; fo <= 15; fo++) try_one(src, slen, 0, 0, 0, fo);
  return 2;
}
#endif
