#ifndef SREP_GUEST_KERNEL_H
#define SREP_GUEST_KERNEL_H

#define _FILE_OFFSET_BITS 64
#define FREEARC_UNIX
#define FREEARC_INTEL_BYTE_ORDER
#define BULAT_ZIGANSHIN_SIGNATURE 0x26351817

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <vector>
#include <set>
#include <stack>
#include <new>

typedef unsigned uint;
typedef uint8_t uint8;
typedef uint16_t uint16;
typedef uint32_t uint32;
typedef uint64_t uint64;
typedef uint32 STAT;
typedef uint64 Offset;

#define b_ (1u)
#define kb (1024 * b_)
#define mb (1024 * kb)
#define mymax(a, b) ((a) > (b) ? (a) : (b))
#define mymin(a, b) ((a) < (b) ? (a) : (b))
#define file_seek(stream, pos) fseeko((stream), (off_t)(pos), SEEK_SET)
#define file_seek_cur(stream, pos) fseeko((stream), (off_t)(pos), SEEK_CUR)
#define file_read(file, buf, size) fread((buf), 1, (size), (file))
#define file_write(file, buf, size) fwrite((buf), 1, (size), (file))
#define CHECK(code, cond, msg)                                                                     \
  do {                                                                                             \
    if (!(cond))                                                                                   \
      abort();                                                                                     \
  } while (0)

#define STATS_PER_MATCH(ROUND_MATCHES) ((ROUND_MATCHES) ? 3 : 4)

#define DECODE_LZ_MATCH(stat, FUTURE_LZ, ROUND_MATCHES, L, basic_pos, lit_len, LZ_MATCH_TYPE,      \
                        lz_match)                                                                  \
  unsigned L1 = ((ROUND_MATCHES) ? (L) : 1);                                                       \
  unsigned lit_len = *stat++;                                                                      \
  Offset lz_match_offset = *stat++;                                                                \
  if (!(ROUND_MATCHES))                                                                            \
    lz_match_offset += Offset(*stat++) << 32;                                                      \
  lz_match_offset *= L1;                                                                           \
  LZ_MATCH_TYPE lz_match;                                                                          \
  lz_match.len = (*stat++) * L1 + (L);                                                             \
  if (!(FUTURE_LZ)) {                                                                              \
    lz_match.dest = (basic_pos) + lit_len;                                                         \
    lz_match.src = lz_match.dest / L1 * L1 - lz_match_offset;                                      \
  } else {                                                                                         \
    lz_match.src = (basic_pos) + lit_len;                                                          \
    lz_match.dest = lz_match.src + lz_match_offset;                                                \
  }

struct LZ_MATCH {
  LZ_MATCH() : src(Offset(-1)), dest(Offset(-1)), len(STAT(-1)) {}
  Offset src, dest;
  STAT len;
};

static inline void memcpy_lz_match(void *_dest, void *_src, unsigned len) {
  if (len) {
    char *dest = (char *)_dest, *src = (char *)_src;
    do {
      *dest++ = *src++;
    } while (--len);
  }
}

#endif
