// OGGRE kernel reconstructed from unfiltered CLS-OGGRE (VA 0x10001000).
// INV-03: PE is data only; this module is built here.

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

namespace {

const uint32_t kTop = 0x1000000u;
const uint32_t kWant = 255994514u;
const uint32_t kWantCRC = 0xf7a300d7u;

struct Range {
  uint32_t code;
  uint32_t range;
  const uint8_t *ptr;
  const uint8_t *end;
};

static int refill(Range *r) {
  while (r->range < kTop) {
    if (r->ptr >= r->end) return -1;
    r->code = (r->code << 8) | *r->ptr++;
    r->range <<= 8;
  }
  return 0;
}

// getbit @ 0x100038c0 thiscall: ecx=Range*, edx=uint16* freq.
static int getbit(Range *r, uint16_t *freq) {
  if (refill(r) < 0) return -1;
  uint32_t range = r->range;
  uint32_t code = r->code;
  uint32_t f = *freq;
  uint32_t bound = (range >> 16) * f;
  if (code >= bound) {
    r->code = code - bound;
    r->range = range - bound;
    *freq = (uint16_t)(f - (f >> 6));
    return 1;
  }
  r->range = bound;
  *freq = (uint16_t)(f + ((0xffffu - f) >> 6));
  return 0;
}

static void init_freq(uint16_t *p, uint32_t n) {
  for (uint32_t i = 0; i < n; i++) p[i] = 0x8000;
}

// Ogg CRC-32, poly 0x04c11db7, unreflected (libogg).
static uint32_t gCrcTab[256];

static void crc_init(void) {
  for (uint32_t i = 0; i < 256; i++) {
    uint32_t r = i << 24;
    for (int j = 0; j < 8; j++) {
      if (r & 0x80000000u)
        r = (r << 1) ^ 0x04c11db7u;
      else
        r <<= 1;
    }
    gCrcTab[i] = r;
  }
}

static void put_le32(uint8_t *p, uint32_t v) {
  p[0] = (uint8_t)v;
  p[1] = (uint8_t)(v >> 8);
  p[2] = (uint8_t)(v >> 16);
  p[3] = (uint8_t)(v >> 24);
}

static void put_le64(uint8_t *p, uint64_t v) {
  put_le32(p, (uint32_t)v);
  put_le32(p + 4, (uint32_t)(v >> 32));
}

// Integer used at 0x10001615 (nbits=5) and 0x10016816 (nbits=3).
// Layout: +8 u8 ctx, +0xa u8 ctx2, +0xc uint16 freq[4], +0x34 tree.
static int get_sym(Range *r, uint8_t *base, int nbits) {
  int ctx = base[8];
  uint16_t *freq = (uint16_t *)(base + 0x0c);
  int bit = getbit(r, &freq[ctx]);
  if (bit < 0) return -1;
  base[8] = (uint8_t)((bit + 2 * ctx) & 3);
  if (bit == 0) {
    base[0x0a] = 0;
    return 0;
  }
  int shift = nbits == 3 ? 4 : 6;
  uint8_t *node = base + 0x34 + ((int)base[0x0a] << shift);
  bit = getbit(r, (uint16_t *)(node + 2));
  if (bit < 0) return -1;
  int idx = 4 + 2 * bit;
  bit = getbit(r, (uint16_t *)(node + idx));
  if (bit < 0) return -1;
  int v = bit + idx;
  int left = nbits - 2;
  for (int i = 0; i < left; i++) {
    bit = getbit(r, (uint16_t *)(node + v * 2));
    if (bit < 0) return -1;
    v = bit + 2 * v;
  }
  int n = v & ((1 << nbits) - 1);
  base[0x0a] = (uint8_t)(n + 1);
  if (n == 0) return 1;
  int extra = 1;
  int cap = nbits == 3 ? 2 : 8;
  int walk = 1;
  for (int i = 0; i < n; i++) {
    uint16_t *fp = (uint16_t *)(base + (nbits == 3 ? 0x834 : 0x828) + walk * 2);
    bit = getbit(r, fp);
    if (bit < 0) return -1;
    extra = bit + 2 * extra;
    int nxt = (bit + 2 * walk) & (nbits == 3 ? 3 : 15);
    if (walk < cap) walk = nxt;
  }
  return extra;
}

struct Writer {
  uint8_t *dst;
  uint32_t cap;
  uint32_t pos;
};

static int emit(Writer *w, const uint8_t *p, uint32_t n) {
  if (n > w->cap - w->pos) n = w->cap - w->pos;
  if (n && p) memcpy(w->dst + w->pos, p, n);
  w->pos += n;
  return (int)n;
}

// Rebuild one Ogg page over payload p[0..n).
static int emit_page(Writer *w, uint32_t serial, uint32_t seq, uint64_t gran,
                     int bos, int eos, const uint8_t *payload, uint32_t n) {
  uint8_t hdr[27 + 255];
  uint32_t nseg = (n + 254) / 255;
  if (nseg == 0) nseg = 1;
  if (nseg > 255) return -1;
  memset(hdr, 0, sizeof(hdr));
  hdr[0] = 'O';
  hdr[1] = 'g';
  hdr[2] = 'g';
  hdr[3] = 'S';
  hdr[4] = 0;
  hdr[5] = (uint8_t)((bos ? 2 : 0) | (eos ? 4 : 0) | (seq == 0 && !bos ? 1 : 0));
  put_le64(hdr + 6, gran);
  put_le32(hdr + 14, serial);
  put_le32(hdr + 18, seq);
  put_le32(hdr + 22, 0);
  hdr[26] = (uint8_t)nseg;
  uint32_t left = n;
  for (uint32_t i = 0; i < nseg; i++) {
    uint32_t s = left > 255 ? 255 : left;
    hdr[27 + i] = (uint8_t)s;
    left -= s;
  }
  uint32_t hsz = 27 + nseg;
  // CRC over header+payload with CRC field zero.
  uint32_t crc = 0;
  for (uint32_t i = 0; i < hsz; i++) crc = (crc << 8) ^ gCrcTab[((crc >> 24) & 0xff) ^ hdr[i]];
  for (uint32_t i = 0; i < n; i++) crc = (crc << 8) ^ gCrcTab[((crc >> 24) & 0xff) ^ payload[i]];
  put_le32(hdr + 22, crc);
  if (emit(w, hdr, hsz) < 0) return -1;
  if (n && emit(w, payload, n) < 0) return -1;
  return 0;
}

struct Models {
  uint16_t ctrl[16];
  uint8_t *mem; // 0x6e000 model (0x1000121e)
};

static int decode_body(Range *r, Models *m, Writer *w, int stat, int solid) {
  (void)solid;
  (void)stat;
  uint8_t *mem = m->mem;
  uint32_t serial = 1;
  uint32_t seq = 0;
  int state = 0;
#ifdef HOST_DEBUG
  fprintf(stderr, "range init code=%08x range=%08x first=%02x\n", r->code, r->range,
          r->ptr < r->end ? r->ptr[0] : 0);
  int nloop = 0;
#endif
  for (;;) {
    int b0 = getbit(r, &m->ctrl[state & 15]);
#ifdef HOST_DEBUG
    if (nloop < 8)
      fprintf(stderr, "L%d st=%d b0=%d pos=%u in=%ld\n", nloop, state, b0, w->pos,
              (long)(r->end - r->ptr));
    nloop++;
#endif
    if (b0 < 0) break;
    state = b0;
    if (b0 == 0) {
      int n = get_sym(r, mem + 0x6000, 5);
      if (n < 0) break;
      if (n == 0) {
        if ((long)(r->end - r->ptr) < 8) break;
        n = 1;
      }
      if (n > 8192) n = 8192;
      uint8_t pkt[8192];
      int got = 0;
      for (int i = 0; i < n; i++) {
        int v = 1;
        for (int k = 0; k < 8; k++) {
          int b = getbit(r, (uint16_t *)(mem + 0x34 + v * 2));
          if (b < 0) goto done;
          v = (v << 1) | b;
        }
        pkt[got++] = (uint8_t)(v - 256);
      }
      if (got > 0) {
        if (emit_page(w, serial, seq, (uint64_t)seq * 256ull, 0, 0, pkt, (uint32_t)got) < 0)
          break;
        seq++;
      }
      continue;
    }
    int b1 = getbit(r, &m->ctrl[1]);
#ifdef HOST_DEBUG
    if (nloop <= 32) fprintf(stderr, "  b1=%d\n", b1);
#endif
    if (b1 < 0) break;
    if (b1 == 0) {
      // 0x10001bb9: one more control bit, then ident via 0x100167c0.
      int flag = getbit(r, &m->ctrl[6]);
      if (flag < 0) break;
      int v = get_sym(r, mem + 0x6000, 3);
      if (v < 0) break;
      int sr = get_sym(r, mem + 0x2000, 5);
      if (sr < 0) sr = 0;
      uint32_t rate = 44100;
      if (sr > 0 && sr < 24) rate = 11025u << (sr % 3);
      int ch = v & 7;
      if (ch < 1) ch = 2;
      if (ch > 8) ch = 2;
      uint8_t ident[30];
      memset(ident, 0, sizeof(ident));
      ident[0] = 1;
      memcpy(ident + 1, "vorbis", 6);
      ident[11] = (uint8_t)ch;
      put_le32(ident + 12, rate);
      ident[28] = (uint8_t)((8 << 4) | 11);
      ident[29] = 1;
      serial++;
      seq = 0;
      if (emit_page(w, serial, seq, 0, 1, 0, ident, 30) < 0) return 0;
      seq++;
      uint8_t comm[16];
      comm[0] = 3;
      memcpy(comm + 1, "vorbis", 6);
      put_le32(comm + 7, 0);
      put_le32(comm + 11, 0);
      comm[15] = 1;
      if (emit_page(w, serial, seq, 0, 0, 0, comm, 16) < 0) return 0;
      seq++;
      state = 0;
      continue;
    }
    // 0x100015e5: two get_sym on the 0x68000 / 0x6000 models.
    int a = get_sym(r, mem + 0x68000, 5);
    if (a < 0) break;
    int b = get_sym(r, mem + 0x6000, 5);
    if (b < 0) break;
    (void)a;
    (void)b;
    state = 1;
  }
done:
  return (int)w->pos;
}

static int32_t decode(const uint8_t *src, uint32_t slen, uint8_t *dst, uint32_t dcap) {
  if (!src || !dst || slen < 7 || dcap == 0) return 0;
  if (src[0] != 'O' || src[1] != 'G' || src[2] != 'G' || src[3] != 'R' || src[4] != 'E')
    return 0;
  if (src[5] != 0) return 0;
  uint8_t flags = src[6];
  int stat = flags & 7;
  int solid = flags & 8;
  if (stat > 3) return 0;

  crc_init();

  Range r;
  r.ptr = src + 7;
  r.end = src + slen;
  r.range = 0xffffffffu;
  r.code = 0;
  for (int i = 0; i < 4; i++) {
    if (r.ptr >= r.end) return 0;
    r.code = (r.code << 8) | *r.ptr++;
  }

  Models m;
  memset(&m, 0, sizeof(m));
  init_freq(m.ctrl, 16);
  m.ctrl[0] = 0x1000;
  m.ctrl[1] = 0xffff;
  m.mem = (uint8_t *)calloc(1, 0x6e000);
  if (!m.mem) return 0;
  for (uint32_t i = 0; i < 0x6e000 / 2; i++) ((uint16_t *)m.mem)[i] = 0x8000;
  m.mem[0x6008] = 0;
  m.mem[0x600a] = 0;
  m.mem[0x68008] = 0;
  m.mem[0x6800a] = 0;

  Writer w = {dst, dcap, 0};
  decode_body(&r, &m, &w, stat, solid);

  free(m.mem);
  return (int32_t)w.pos;
}

} // namespace

extern "C" int32_t mpzz_decode(uint8_t *src, uint32_t slen, uint8_t *dst, uint32_t dcap) {
  return decode(src, slen, dst, dcap);
}

#ifdef HOST_DEBUG
int main(int argc, char **argv) {
  const char *path = argc > 1 ? argv[1] : "/tmp/gf-extract/fg01-after-srep.bin";
  FILE *f = fopen(path, "rb");
  if (!f) {
    perror(path);
    return 1;
  }
  fseek(f, 0, SEEK_END);
  long n = ftell(f);
  fseek(f, 0, SEEK_SET);
  uint8_t *src = (uint8_t *)malloc((size_t)n);
  if (!src || fread(src, 1, (size_t)n, f) != (size_t)n) {
    fprintf(stderr, "read failed\n");
    return 1;
  }
  fclose(f);
  uint32_t cap = kWant + (4u << 20);
  uint8_t *dst = (uint8_t *)malloc(cap);
  if (!dst) {
    fprintf(stderr, "oom\n");
    return 1;
  }
  int32_t got = mpzz_decode(src, (uint32_t)n, dst, cap);
  uint32_t crc = 0;
  if (got > 0) {
    crc = 0xffffffffu;
    static uint32_t ieee[256];
    static int once;
    if (!once) {
      for (uint32_t i = 0; i < 256; i++) {
        uint32_t c = i;
        for (int j = 0; j < 8; j++) c = (c >> 1) ^ (0xedb88320u & -(c & 1));
        ieee[i] = c;
      }
      once = 1;
    }
    crc = 0xffffffffu;
    for (int32_t i = 0; i < got; i++) crc = ieee[(crc ^ dst[i]) & 0xff] ^ (crc >> 8);
    crc ^= 0xffffffffu;
  }
  fprintf(stderr, "size %d want %u crc %08x want %08x head ", got, kWant, crc, kWantCRC);
  for (int i = 0; i < 16 && i < got; i++) fprintf(stderr, "%02x", dst[i]);
  fprintf(stderr, "\n");
  int ok = (uint32_t)got == kWant && crc == kWantCRC;
  free(src);
  free(dst);
  return ok ? 0 : 2;
}
#endif
