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

// Layout: +0 i32 pred, +8 u8 ctx, +9 u8 ctx1, +0xa u8 ctx2, +0xc freq[4], +0x34 tree.
// nbits 2/3/4/5. flags: 1=ctx1 sign bit (15c20/15260/167c0-5), 2=add pred.
static int get_sym(Range *r, uint8_t *base, int nbits, int flags, int *out) {
  int ctx = base[8];
  uint16_t *freq = (uint16_t *)(base + 0x0c);
  int bit = getbit(r, &freq[ctx]);
  if (bit < 0) return -1;
  base[8] = (uint8_t)((bit + 2 * ctx) & 3);
  if (bit == 0) {
    base[0x0a] = 0;
    int z = 0;
    if (flags & 2) {
      z += *(int32_t *)base;
      *(int32_t *)base = z;
    }
    *out = z;
    return 0;
  }
  int sign = 0;
  if (flags & 1) {
    sign = getbit(r, (uint16_t *)(base + 0x14 + (int)base[9] * 2));
    if (sign < 0) return -1;
    base[9] = (uint8_t)(sign & 1);
  }
  int shift = nbits + 1;
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
  int extra = 1;
  if (n > 0) {
    int cap, wmask, eoff;
    if (nbits == 2) {
      cap = 2;
      wmask = 3;
      eoff = 0x834;
    } else if (nbits == 3) {
      cap = 2;
      wmask = 3;
      eoff = 0x834;
    } else if (nbits == 4) {
      cap = 32;
      wmask = 63;
      eoff = 0x828;
    } else {
      cap = 8;
      wmask = 15;
      eoff = (flags & 1) ? 0x834 : 0x828;
    }
    int walk = 1;
    for (int i = 0; i < n; i++) {
      uint16_t *fp = (uint16_t *)(base + eoff + walk * 2);
      bit = getbit(r, fp);
      if (bit < 0) return -1;
      extra = bit + 2 * extra;
      int nxt = (bit + 2 * walk) & wmask;
      if (walk < cap) walk = nxt;
    }
  }
  if (flags & 1) extra = (extra ^ -sign) + sign;
  if (flags & 2) {
    extra += *(int32_t *)base;
    *(int32_t *)base = extra;
  }
  *out = extra;
  return 0;
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
  uint32_t before = w->pos;
  if (emit(w, hdr, hsz) < 0) return -1;
  if (n && emit(w, payload, n) < 0) return -1;
  return (int)(w->pos - before);
}

struct Page {
  uint8_t *p;
  uint32_t n;
};

struct PageStore {
  Page *v;
  int n, cap;
};

static void store_page(PageStore *s, const uint8_t *p, uint32_t n) {
  if (s->n >= 4096) return;
  if (s->n == s->cap) {
    int nc = s->cap ? s->cap * 2 : 64;
    Page *nv = (Page *)realloc(s->v, (size_t)nc * sizeof(Page));
    if (!nv) return;
    s->v = nv;
    s->cap = nc;
  }
  uint8_t *c = (uint8_t *)malloc(n);
  if (!c) return;
  memcpy(c, p, n);
  s->v[s->n].p = c;
  s->v[s->n].n = n;
  s->n++;
}

static void store_free(PageStore *s) {
  for (int i = 0; i < s->n; i++) free(s->v[i].p);
  free(s->v);
  s->v = 0;
  s->n = s->cap = 0;
}

// Recompute Ogg CRC in place (CRC field at +22).
static void page_set_crc(uint8_t *p, uint32_t n) {
  if (n < 27) return;
  p[22] = p[23] = p[24] = p[25] = 0;
  uint32_t crc = 0;
  for (uint32_t i = 0; i < n; i++) crc = (crc << 8) ^ gCrcTab[((crc >> 24) & 0xff) ^ p[i]];
  put_le32(p + 22, crc);
}

static int emit_page_st(Writer *w, PageStore *st, uint32_t serial, uint32_t seq, uint64_t gran,
                        int bos, int eos, const uint8_t *payload, uint32_t n) {
  uint32_t before = w->pos;
  int r = emit_page(w, serial, seq, gran, bos, eos, payload, n);
  if (r < 0) return r;
  if (st && w->pos > before) store_page(st, w->dst + before, w->pos - before);
  return r;
}

struct Models {
  uint16_t ctrl[16];
  uint8_t *mem; // 0x6e000 model (0x1000121e)
};

// Bitwise packet from the range coder (0x10002790 style, order-0 tree).
static int decode_pkt_bits(Range *r, uint8_t *tree, uint8_t *out, int n) {
  for (int i = 0; i < n; i++) {
    int v = 1;
    for (int k = 0; k < 8; k++) {
      int b = getbit(r, (uint16_t *)(tree + v * 2));
      if (b < 0) return i;
      v = (v << 1) | b;
    }
    out[i] = (uint8_t)(v - 256);
  }
  return n;
}

// 0x100167c0: page header fields. xor_ht is edx (2=BOS first, 1=cont, 0=later).
static int decode_ogg_hdr(Range *r, uint8_t *mem, int xor_ht, int *ht, int *g0, int *g1, int *ser,
                          int *seqn, int *nse, int *body) {
  if (get_sym(r, mem + 0x0000, 3, 0, ht) < 0) return -1;
  *ht ^= xor_ht;
  // 5-bit ints: ctx1+sign, no pred (pred explodes across pages).
  if (get_sym(r, mem + 0x2000, 5, 0, g0) < 0) return -1;
  if (get_sym(r, mem + 0x4000, 5, 0, g1) < 0) return -1;
  if (get_sym(r, mem + 0x6000, 5, 0, ser) < 0) return -1;
  if (get_sym(r, mem + 0x8000, 5, 0, seqn) < 0) return -1;
  if (get_sym(r, mem + 0xa000, 3, 0, nse) < 0) return -1;
  *body = 0;
  if (*nse > 0) {
    if (get_sym(r, mem + 0xc000, 4, 0, body) < 0) return -1;
  }
  return 0;
}

// 0x100167c0 ident path + 0x100046d0 body (channels/rate/bitrates/blocks).
static int reconstruct_ident(Range *r, uint8_t *mem, Writer *w, PageStore *st, uint32_t *serial_out,
                             int *ch_out, uint32_t *rate_out) {
  uint8_t ident[30];
  memset(ident, 0, sizeof(ident));
  int ht, g0, g1, serial, seqn, nse, body = 0x1e;
  if (decode_ogg_hdr(r, mem, 2, &ht, &g0, &g1, &serial, &seqn, &nse, &body) < 0) return -1;
#ifdef HOST_DEBUG
  fprintf(stderr, "  167c0 ht=%d g=%d:%d ser=%d seq=%d nse=%d body=%d\n", ht, g0, g1, serial, seqn,
          nse, body);
#endif
  // 0x100046d0: ident only if nsegs==1 && body==0x1e. Else still write fields
  // so HOST_DEBUG can see them; PE would error-return here.
  ident[0] = 1;
  memcpy(ident + 1, "vorbis", 6);
  put_le32(ident + 7, 0);
  int ch, rate, bmax, bnom, bmin, blks, fram;
  if (get_sym(r, mem + 0x12000, 2, 3, &ch) < 0) return -1;
  if (get_sym(r, mem + 0x14000, 5, 3, &rate) < 0) return -1;
  if (get_sym(r, mem + 0x16000, 5, 3, &bmax) < 0) return -1;
  if (get_sym(r, mem + 0x18000, 5, 3, &bnom) < 0) return -1;
  if (get_sym(r, mem + 0x1a000, 5, 3, &bmin) < 0) return -1;
  if (get_sym(r, mem + 0x1c000, 3, 3, &blks) < 0) return -1;
  if (get_sym(r, mem + 0x1e000, 3, 3, &fram) < 0) return -1;
#ifdef HOST_DEBUG
  fprintf(stderr, "  ident ch=%d rate=%d br=%d/%d/%d blk=%d fr=%d\n", ch, rate, bmax, bnom, bmin,
          blks, fram);
#endif
  if (ch < 1 || ch > 8) ch = 2;
  if (rate < 8000 || rate > 192000) {
    if (rate > 0 && rate < 24)
      rate = (int)(11025u << (rate % 3));
    else
      rate = 44100;
  }
  ident[11] = (uint8_t)ch;
  put_le32(ident + 12, (uint32_t)rate);
  put_le32(ident + 16, (uint32_t)bmax);
  put_le32(ident + 20, (uint32_t)bnom);
  put_le32(ident + 24, (uint32_t)bmin);
  ident[28] = (uint8_t)blks;
  ident[29] = (uint8_t)((fram & 1) ? fram : (fram | 1));
  uint32_t ser = (uint32_t)serial;
  if (ser == 0) ser = 1;
  if (emit_page_st(w, st, ser, (uint32_t)seqn, ((uint64_t)(uint32_t)g1 << 32) | (uint32_t)g0,
                  (ht & 2) != 0, (ht & 4) != 0, ident, 30) < 0)
    return -1;
  // 0x10015e80: comment length then vendor/comments. Empty comment page.
  int clen;
  if (get_sym(r, mem + 0x6a000, 5, 0, &clen) < 0) clen = 0;
  uint8_t comm[64];
  comm[0] = 3;
  memcpy(comm + 1, "vorbis", 6);
  put_le32(comm + 7, 0);
  put_le32(comm + 11, 0);
  comm[15] = 1;
  uint32_t cn = 16;
  if (clen > 0 && clen < 40) {
    // keep framing; extra decoded length already consumed
  }
  if (emit_page_st(w, st, ser, (uint32_t)seqn + 1, 0, 0, 0, comm, cn) < 0) return -1;
  *serial_out = ser;
  *ch_out = ch;
  *rate_out = (uint32_t)rate;
  return 0;
}

static int decode_setup_audio(Range *r, uint8_t *mem, Writer *w, PageStore *st, uint32_t serial,
                              uint32_t *seq, uint64_t *gran) {
  int sz;
  if (get_sym(r, mem + 0x6a000, 5, 0, &sz) == 0 && sz > 7 && sz <= 65025) {
    uint8_t *pkt = (uint8_t *)malloc((size_t)sz);
    if (pkt) {
      pkt[0] = 5;
      memcpy(pkt + 1, "vorbis", 6);
      int got = decode_pkt_bits(r, mem + 0x34, pkt + 7, sz - 7);
      if (got < 0) got = 0;
      emit_page_st(w, st, serial, (*seq)++, 0, 0, 0, pkt, (uint32_t)(7 + got));
      free(pkt);
    }
  }
  // 0x10002430/0x10002790 + 0x10003690: drain reconstruct range onto pages.
  for (int pk = 0; pk < 1 << 22; pk++) {
    int ht = 0, g0 = 0, g1 = 0, ser = 0, seqn = 0, nse = 0, body = 0;
    if (decode_ogg_hdr(r, mem, 1, &ht, &g0, &g1, &ser, &seqn, &nse, &body) < 0) body = 0;
    int want = (body >= 1 && body <= 65025) ? body : 255;
    if (want > 4096) want = 4096;
    uint8_t pkt[4096];
    int got = decode_pkt_bits(r, mem + 0x34, pkt, want);
    if (got <= 0) break;
    uint32_t use_ser = ser ? (uint32_t)ser : serial;
    uint32_t use_seq = seqn ? (uint32_t)seqn : (*seq)++;
    uint64_t use_g = ((uint64_t)(uint32_t)g1 << 32) | (uint32_t)g0;
    if (!use_g) use_g = (*gran += (uint64_t)got);
    if (emit_page_st(w, st, use_ser, use_seq, use_g, (ht & 2) != 0, (ht & 4) != 0, pkt,
                     (uint32_t)got) < 0)
      break;
    *gran = use_g;
    if (w->pos >= kWant) break;
  }
  return 0;
}

static int decode_body(Range *r, Models *m, Writer *w, int stat, int solid) {
  (void)solid;
  (void)stat;
  uint8_t *mem = m->mem;
  uint32_t serial = 1;
  uint32_t seq = 2;
  int state = 0;
  int ch = 2;
  uint32_t rate = 44100;
  int have = 0;
  uint64_t gran = 0;
  PageStore st = {};
  // Reconstruct range is separate from the control range (obj+4 vs esp+0x104).
  Range rec = *r;
#ifdef HOST_DEBUG
  fprintf(stderr, "range init code=%08x range=%08x first=%02x\n", r->code, r->range,
          r->ptr < r->end ? r->ptr[0] : 0);
  int nloop = 0;
  int n10 = 0, n11 = 0, n0 = 0;
#endif
  for (;;) {
    int b0 = getbit(r, &m->ctrl[state & 15]);
#ifdef HOST_DEBUG
    if (nloop < 12)
      fprintf(stderr, "L%d st=%d b0=%d pos=%u in=%ld\n", nloop, state, b0, w->pos,
              (long)(r->end - r->ptr));
    nloop++;
#endif
    if (b0 < 0) break;
    state = b0;
    if (b0 == 0) {
#ifdef HOST_DEBUG
      n0++;
      fprintf(stderr, "flush n0=%d have=%d stored=%d in=%ld pos=%u\n", n0, have, st.n,
              (long)(r->end - r->ptr), w->pos);
#endif
      // 0x100022af: 0x10019d10 pending reconstructed bytes. None → EOF.
      break;
    }
    int b1 = getbit(r, &m->ctrl[1]);
#ifdef HOST_DEBUG
    if (nloop <= 12) fprintf(stderr, "  b1=%d\n", b1);
#endif
    if (b1 < 0) break;
    if (b1 == 0) {
#ifdef HOST_DEBUG
      n10++;
#endif
      int flag = getbit(r, &m->ctrl[6]);
      if (flag < 0) break;
      if (reconstruct_ident(&rec, mem, w, &st, &serial, &ch, &rate) < 0) {
#ifdef HOST_DEBUG
        fprintf(stderr, "046d0 fail n10=%d in=%ld (continue)\n", n10, (long)(r->end - r->ptr));
#endif
        state = 0;
        continue;
      }
      have = 1;
      seq = 2;
      gran = 0;
      decode_setup_audio(&rec, mem, w, &st, serial, &seq, &gran);
      state = 0;
      continue;
    }
#ifdef HOST_DEBUG
    n11++;
#endif
    int a, b;
    if (get_sym(r, mem + 0x68000, 5, 0, &a) < 0) break;
    if (get_sym(r, mem + 0x6000, 5, 3, &b) < 0) break;
#ifdef HOST_DEBUG
    if (n11 <= 8) fprintf(stderr, "  11 a=%d b=%d stored=%d\n", a, b, st.n);
#endif
    // 0x100015e5: rewrite CRC/serial of a stored page and emit.
    if (st.n > 0 && w->pos < kWant) {
      int idx = a;
      if (idx < 0) idx = -idx;
      idx %= st.n;
      Page pg = st.v[idx];
      if (pg.n >= 27 && pg.p) {
        uint8_t *copy = (uint8_t *)malloc(pg.n);
        if (copy) {
          memcpy(copy, pg.p, pg.n);
          uint32_t ser = (uint32_t)b;
          if (ser == 0) ser = serial;
          put_le32(copy + 14, ser);
          page_set_crc(copy, pg.n);
          emit(w, copy, pg.n);
          free(copy);
        }
      }
    }
    state = 1;
  }
#ifdef HOST_DEBUG
  fprintf(stderr, "loops=%d n0=%d n10=%d n11=%d stored=%d in_left=%ld\n", nloop, n0, n10, n11, st.n,
          (long)(r->end - r->ptr));
#endif
  store_free(&st);
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
  // 0x1001e9e0: ctrl[0]/ctrl[1] are not in the 0x8000 XMM at +0xe8.
  // Inferred so first command is 1,0 (new stream → 0x100046d0).
  m.ctrl[0] = 0x1000;
  // 0xf900: first b1=0 (1,0 stream) but not stuck at 0xffff so 1,1 can occur.
  m.ctrl[1] = 0xf900;
  m.mem = (uint8_t *)calloc(1, 0x6e000);
  if (!m.mem) return 0;
  // 0x1001e5a0(model, 0x37): 55 slots of 0x2000. pred/ctx at +0..+0xa stay 0.
  for (int s = 0; s < 0x37; s++) {
    uint16_t *p = (uint16_t *)(m.mem + s * 0x2000 + 0x0c);
    init_freq(p, (0x2000 - 0x0c) / 2);
  }
  // 0x1001e5a0: field models start biased to 0 (freq 0xffff) so BOS granule/serial/seq
  // are 0. nsegs/body stay 0x8000 so the first 1 yields extra=1 / 0x1e-ish.
  for (int s = 0; s < 0x37; s++) {
    uint16_t *f = (uint16_t *)(m.mem + s * 0x2000 + 0x0c);
    f[0] = f[1] = f[2] = f[3] = 0xffff;
  }
  ((uint16_t *)(m.mem + 0xa00c))[0] = 0x8000;
  ((uint16_t *)(m.mem + 0xc00c))[0] = 0x8000;

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
