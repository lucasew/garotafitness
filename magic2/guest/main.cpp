// Reconstructed lolz v22c4b kernel (cls-magic2 is PE-only; INV-03).
// Same guest shape as srep: C++ in, Go NewReader + wazero out.

#include <stdint.h>
#include <string.h>

static const uint32_t kL = 1u << 23;
static const uint32_t kMN = 1u << 15;
static const uint32_t kMB = 1u << 14;
static const int kWant = 430889;
static const int kEmu = 2895;
static const int kApp = 6;
static const uint32_t kAppCRC = 0xf75982bb;

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
  if (p == 0) p = 1;
  if (p >= m) p = m - 1;
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

static int find16(const uint16_t *cdf, uint32_t slot, int last) {
  for (int j = 1; j < last; j++) {
    if ((int16_t)cdf[j] > (int16_t)slot) return j;
  }
  return last;
}

static int get_nibble(Rans *r, uint16_t *cdf, int last, int shift, const uint16_t tgt[][16]) {
  if (!r->ok) return -1;
  uint32_t slot = r->x & (kMN - 1);
  uint32_t quo = r->x >> 15;
  int i = find16(cdf, slot, last);
  uint32_t start = cdf[i - 1];
  uint32_t end = (i < 16) ? cdf[i] : 0x8000;
  if (end <= start) end = start + 1;
  r->x = (end - start) * quo + (slot - start);
  int sym = i - 1;
  adapt16(cdf, sym, tgt, shift);
  renorm(r);
  return sym;
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
  d[7] = 0x8000;
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

static int decode_off(int cls, int extra, int *rep0, int *reps) {
  if (cls == 0) {
    if (*rep0 < 1) *rep0 = 1;
    return *rep0;
  }
  if (cls >= 4 && cls <= 10) {
    int n = (cls == 10) ? extra : kA690[cls - 4];
    if (n >= 0 && n < 4) {
      int d = reps[n];
      if (n > 0) {
        for (int i = n; i > 0; i--) reps[i] = reps[i - 1];
      }
      if (d < 1) d = 1;
      reps[0] = d;
      *rep0 = d;
      return d;
    }
  }
  int d = extra;
  if (d < 1) d = 1;
  for (int i = 3; i > 0; i--) reps[i] = reps[i - 1];
  reps[0] = d;
  *rep0 = d;
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
    decode_off(cls, extra, &rep0, reps);
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

static int ctx_lo(int prev, int hi) { return ((prev >> 0) << 4) | hi; }

static int decode_v22(const uint8_t *src, int slen, uint8_t *dst, int dcap) {
  if (slen < 4) return 0;
  Rans r;
  r.buf = src;
  r.off = 4;
  r.len = slen;
  r.ok = true;
  r.x = ((uint32_t)src[0] << 24) | ((uint32_t)src[1] << 16) | ((uint32_t)src[2] << 8) | src[3];
  renorm(&r);
  const int nHi = 16384, nLo = 4096, nCls = 256, nLen = 256, nBM = 64;
  static uint16_t hiTab[16384 * 16];
  static uint16_t loTab[4096 * 16];
  static uint16_t clsTab[256 * 16];
  static uint16_t lenTab[256 * 8];
  static uint16_t bmTab[64];
  for (int i = 0; i < nHi; i++) init_nibble(hiTab + i * 16);
  for (int i = 0; i < nLo; i++) init_nibble(loTab + i * 16);
  for (int i = 0; i < nCls; i++) init_nibble(clsTab + i * 16);
  for (int i = 0; i < nLen; i++) init_sym8(lenTab + i * 8);
  for (int i = 0; i < nBM; i++) bmTab[i] = kMB / 2;
  uint16_t litP = kMB / 2;
  int n = 0, prev = 0, rep0lit = 0, rep0 = 1;
  int reps[4] = {1, 1, 1, 1};
  while (n < dcap && n < kWant && r.ok) {
    if (r.x < kL && r.off >= r.len) break;
    int bit = get_bit(&r, &litP, 14, 5);
    if (bit < 0) break;
    if (bit == 0) {
      int hi = get_nibble(&r, hiTab + (ctx_hi(prev, rep0lit, n) % nHi) * 16, 16, 7, kNibbleTgt);
      if (hi < 0) break;
      int lo = get_nibble(&r, loTab + (ctx_lo(prev, hi) % nLo) * 16, 16, 7, kNibbleTgt);
      if (lo < 0) break;
      uint8_t b = (uint8_t)((hi << 4) | lo);
      dst[n++] = b;
      prev = rep0lit = b;
      continue;
    }
    if (n == 0) break;
    int cls = get_nibble(&r, clsTab + (prev % nCls) * 16, 16, 6, kMatchTgt);
    if (cls < 0 || cls > 11) break;
    int extra = 0;
    if (cls != 0) {
      int nb = (cls >= 4 && cls <= 9) ? kA690[cls - 4] : -1;
      if (nb > 0) {
        for (int i = 0; i < nb && i < 18; i++) {
          int b = get_bit(&r, &bmTab[(rep0lit + i) % nBM], 14, 4);
          if (b < 0) return 0;
          extra = extra * 2 + b;
        }
      } else if (cls == 1 || cls == 2 || cls == 3 || cls == 11) {
        int d = get_nibble(&r, hiTab + (ctx_hi(prev, rep0lit, n) % nHi) * 16, 16, 7, kNibbleTgt);
        if (d < 0) break;
        extra = d + 1;
      }
    }
    decode_off(cls, extra, &rep0, reps);
    int m;
    if (cls == 2) {
      int b = get_bit(&r, &bmTab[rep0lit % nBM], 14, 4);
      if (b < 0) break;
      m = 3 + b;
    } else {
      int ln = get_nibble(&r, lenTab + (prev % nLen) * 8, 8, 6, kNibbleTgt);
      if (ln < 0) break;
      m = ln + 3;
      if (cls == 3) m = ln + 5;
      if (cls == 11) m = ln + 2;
      if (m == 10) {
        int en = get_nibble(&r, lenTab + (prev % nLen) * 8, 8, 6, kNibbleTgt);
        if (en < 0) break;
        m = 10 + en;
      }
    }
    if (m <= 0 || rep0 <= 0 || rep0 > n) break;
    for (int i = 0; i < m && n < dcap && n < kWant; i++) {
      dst[n] = dst[n - rep0];
      prev = dst[n];
      n++;
    }
    if (n > 0) rep0lit = dst[n - 1];
  }
  return hit_crc(dst, n) ? n : 0;
}

extern "C" int magic2_decode(const uint8_t *src, int slen, uint8_t *dst, int dcap) {
  if (!src || slen < 4 || !dst || dcap <= 0) return 0;
  int n = decode_iir(src, slen, dst, dcap, 5);
  if (n > 0) return n;
  n = decode_iir(src, slen, dst, dcap, 4);
  if (n > 0) return n;
  return decode_v22(src, slen, dst, dcap);
}
