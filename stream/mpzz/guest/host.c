#include "host.h"

uint8_t pe_image[PE_IMAGE_SIZE];
int32_t g_self;
int32_t g_cb;
int32_t g_inst;

#include "image.inc"

static const uint8_t *g_src;
static uint32_t g_slen, g_spos;
static uint8_t *g_dst;
static uint32_t g_dcap, g_dpos;
static int32_t g_fs0;
static int32_t g_last_err;

void pe_image_init(void) {
    memcpy(pe_image, kPeImageInit, PE_IMAGE_SIZE);
}

void host_set_files(const uint8_t *src, uint32_t slen, uint8_t *dst, uint32_t dcap) {
    g_src = src;
    g_slen = slen;
    g_spos = 0;
    g_dst = dst;
    g_dcap = dcap;
    g_dpos = 0;
    g_cb = 0;
    g_inst = 0;
    g_self = 0;
}

uint32_t host_written(void) { return g_dpos; }

/* CLS callback ops from cls.h */
#define CLS_MALLOC 1
#define CLS_FREE 2
#define CLS_GET_PARAMSTR 3
#define CLS_FULL_READ 4096
#define CLS_PARTIAL_READ 5120
#define CLS_FULL_WRITE 6144
#define CLS_PARTIAL_WRITE 7168

int32_t host_cls_callback(int32_t instance, int32_t op, int32_t ptr, int32_t n) {
    (void)instance;
    uint8_t *p = (uint8_t *)(intptr_t)ptr;
    if (op == CLS_MALLOC) {
        void *m = n > 0 ? calloc(1, (size_t)n) : NULL;
        if (p) *(void **)p = m;
        return m ? 0 : -3;
    }
    if (op == CLS_FREE) {
        free(p);
        return 0;
    }
    if (op == CLS_GET_PARAMSTR) {
        if (p && n > 0) p[0] = 0;
        return 0;
    }
    if (op == CLS_FULL_READ || op == CLS_PARTIAL_READ) {
        uint32_t want = n < 0 ? 0 : (uint32_t)n;
        if (want > g_slen - g_spos) want = g_slen - g_spos;
        if (p && want) memcpy(p, g_src + g_spos, want);
        g_spos += want;
        return (int32_t)want;
    }
    if (op == CLS_FULL_WRITE || op == CLS_PARTIAL_WRITE) {
        uint32_t want = n < 0 ? 0 : (uint32_t)n;
        if (want > g_dcap - g_dpos) want = g_dcap - g_dpos;
        if (p && want) memcpy(g_dst + g_dpos, p, want);
        g_dpos += want;
        return (int32_t)want;
    }
    return -2;
}

int32_t llvm_ctlz_i32(int32_t x, bool is_zero_undef) {
    if (x == 0) return is_zero_undef ? 0 : 32;
    return (int32_t)__builtin_clz((unsigned)x);
}
int32_t llvm_cttz_i32(int32_t x, bool is_zero_undef) {
    if (x == 0) return is_zero_undef ? 0 : 32;
    return (int32_t)__builtin_ctz((unsigned)x);
}

int32_t __readfsdword(int32_t off) { (void)off; return g_fs0; }
void __writefsdword(int32_t off, int32_t val) { (void)off; g_fs0 = val; }

void __asm_rep_movsd_memcpy(char *dst, char *src, int32_t ndwords) {
    if (dst && src && ndwords > 0) memcpy(dst, src, (size_t)ndwords * 4);
}
int32_t __asm_sti(void) { return 0; }
int32_t __asm_fnclex(void) { return 0; }
int32_t __asm_in(int32_t a, ...) { (void)a; return 0; }
int32_t __asm_insb(int32_t a) { (void)a; return 0; }
int32_t __asm_int3(void) { return 0; }
int32_t __asm_iretd(void) { return 0; }
int32_t __asm_maskmovq(int32_t a, int32_t b) { (void)a; (void)b; return 0; }
int32_t __asm_out(int32_t a, int32_t b) { (void)a; (void)b; return 0; }
int32_t __asm_out_6(int32_t a, int32_t b) { (void)a; (void)b; return 0; }
int32_t __asm_str(void) { return 0; }

int128_t __asm_addsd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_addss(int128_t a, ...) { (void)a; return a; }
int128_t __asm_andps(int128_t a, ...) { (void)a; return a; }
int128_t __asm_cmpeqss(int128_t a, ...) { (void)a; return a; }
int128_t __asm_cmpltsd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_cmpnlesd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_cvtsd2si(int128_t a, ...) { (void)a; return a; }
int128_t __asm_cvtsd2ss(int128_t a, ...) { (void)a; return a; }
int128_t __asm_cvtsi2sd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_cvtsi2ss(int128_t a, ...) { (void)a; return a; }
int128_t __asm_cvtss2sd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_divss(int128_t a, ...) { (void)a; return a; }
int128_t __asm_movaps(int128_t a, ...) { (void)a; return a; }
int128_t __asm_movd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_movd_2(int128_t a, ...) { (void)a; return a; }
int128_t __asm_movdqa(int128_t a, ...) { (void)a; return a; }
int128_t __asm_movdqu(int128_t a, ...) { (void)a; return a; }
int128_t __asm_movdqu_1(int128_t a, ...) { (void)a; return a; }
int128_t __asm_movsd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_movsd_3(int128_t a, ...) { (void)a; return a; }
int128_t __asm_movss(int128_t a, ...) { (void)a; return a; }
int128_t __asm_movups(int128_t a, ...) { (void)a; return a; }
int128_t __asm_orps(int128_t a, ...) { (void)a; return a; }
int128_t __asm_paddd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_pand(int128_t a, ...) { (void)a; return a; }
int128_t __asm_pcmpeqb(int128_t a, ...) { (void)a; return a; }
int128_t __asm_pcmpeqd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_pcmpgtb(int128_t a, ...) { (void)a; return a; }
int128_t __asm_pcmpgtd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_pmovmskb(int128_t a, ...) { (void)a; return a; }
int128_t __asm_psadbw(int128_t a, ...) { (void)a; return a; }
int128_t __asm_pshufd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_pslld(int128_t a, ...) { (void)a; return a; }
int128_t __asm_psrldq(int128_t a, ...) { (void)a; return a; }
int128_t __asm_psrlq(int128_t a, ...) { (void)a; return a; }
int128_t __asm_psubb(int128_t a, ...) { (void)a; return a; }
int128_t __asm_psubd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_punpcklbw(int128_t a, ...) { (void)a; return a; }
int128_t __asm_punpckldq(int128_t a, ...) { (void)a; return a; }
int128_t __asm_punpcklqdq(int128_t a, ...) { (void)a; return a; }
int128_t __asm_punpcklwd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_pxor(int128_t a, ...) { (void)a; return a; }
int128_t __asm_shufps(int128_t a, ...) { (void)a; return a; }
int128_t __asm_subsd(int128_t a, ...) { (void)a; return a; }
int128_t __asm_xorps(int128_t a, ...) { (void)a; return a; }

int32_t unknown_154346ee(void) {  return 0; }
int32_t unknown_1543470c(void) {  return 0; }
int32_t unknown_1543471d(void) {  return 0; }
int32_t unknown_1543475d(void) {  return 0; }
int32_t unknown_1543476e(void) {  return 0; }
int32_t unknown_1543477f(void) {  return 0; }
int32_t unknown_15434793(void) {  return 0; }
int32_t unknown_154347ac(void) {  return 0; }
int32_t unknown_154347c7(void) {  return 0; }
int32_t unknown_154347da(void) {  return 0; }
int32_t unknown_154347e7(void) {  return 0; }
int32_t unknown_154347fa(void) {  return 0; }
int32_t unknown_15434822(void) {  return 0; }
int32_t unknown_15434834(void) {  return 0; }
int32_t unknown_15434852(void) {  return 0; }
int32_t unknown_15434866(void) {  return 0; }
int32_t unknown_1543487e(void) {  return 0; }
int32_t unknown_15434891(void) {  return 0; }
int32_t unknown_154348a4(void) {  return 0; }
int32_t unknown_154348b7(void) {  return 0; }
int32_t unknown_154348cc(void) {  return 0; }
int32_t unknown_15435e17(void) {  return 0; }
int32_t unknown_154361a5(void) {  return 0; }
int32_t unknown_154361ad(void) {  return 0; }
int32_t unknown_154361b9(void) {  return 0; }
int32_t unknown_154364bd(void) {  return 0; }
int32_t unknown_15437cd8(void) {  return 0; }
int32_t unknown_15437db2(void) {  return 0; }
int32_t unknown_15437fa7(void) {  return 0; }
int32_t unknown_1543819f(void) {  return 0; }
int32_t unknown_1543f9de(int32_t a1) { (void)a1; return 0; }
int32_t unknown_1543fc3f(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_1543fc95(int32_t a1) { (void)a1; return 0; }
int32_t unknown_1c116c13(void) {  return 0; }
int32_t unknown_1c2e6168(void) {  return 0; }
int32_t unknown_1c2e657f(void) {  return 0; }
int32_t unknown_1c2e7594(void) {  return 0; }
int32_t unknown_1c4c4937(void) {  return 0; }
int32_t unknown_1c898030(void) {  return 0; }
int32_t unknown_1cc559b3(void) {  return 0; }
int32_t unknown_1ccd5555(void) {  return 0; }
int32_t unknown_1ccd5985(void) {  return 0; }
int32_t unknown_1ccd5c6a(void) {  return 0; }
int32_t unknown_1ccd5fc1(void) {  return 0; }
int32_t unknown_1ccd6db0(void) {  return 0; }
int32_t unknown_1ccd7cc1(void) {  return 0; }
int32_t unknown_1cfb64d6(void) {  return 0; }
int32_t unknown_2c2272c4(void) {  return 0; }
int32_t unknown_2c24723e(void) {  return 0; }
int32_t unknown_2c24804e(void) {  return 0; }
int32_t unknown_2c6f4907(void) {  return 0; }
int32_t unknown_2c70537f(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_2c70742c(void) {  return 0; }
int32_t unknown_2c707454(int32_t a1) { (void)a1; return 0; }
int32_t unknown_2c707464(void) {  return 0; }
int32_t unknown_2c70746b(void) {  return 0; }
int32_t unknown_2c707472(void) {  return 0; }
int32_t unknown_2c7b83d8(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_2c7b862e(void) {  return 0; }
int32_t unknown_2c8a7740(void) {  return 0; }
int32_t unknown_2cc87f8c(void) {  return 0; }
int32_t unknown_2ccd5662(void) {  return 0; }
int32_t unknown_2ccd5750(void) {  return 0; }
int32_t unknown_2ccd7a34(void) {  return 0; }
int32_t unknown_2ccd7be7(void) {  return 0; }
int32_t unknown_2ccd7fe1(void) {  return 0; }
int32_t unknown_2ccd80ae(void) {  return 0; }
int32_t unknown_2cfd48f5(void) {  return 0; }
int32_t unknown_2cfd54b5(void) {  return 0; }
int32_t unknown_324342ba(void) {  return 0; }
int32_t unknown_324343bd(void) {  return 0; }
int32_t unknown_324344a9(void) {  return 0; }
int32_t unknown_324345c7(void) {  return 0; }
int32_t unknown_32434c31(void) {  return 0; }
int32_t unknown_32434d0c(void) {  return 0; }
int32_t unknown_32434e27(void) {  return 0; }
int32_t unknown_32434f4c(void) {  return 0; }
int32_t unknown_324383f2(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_32439cff(void) {  return 0; }
int32_t unknown_32439ded(void) {  return 0; }
int32_t unknown_32439ebb(void) {  return 0; }
int32_t unknown_32439f96(void) {  return 0; }
int32_t unknown_3243a08b(void) {  return 0; }
int32_t unknown_3243a182(void) {  return 0; }
int32_t unknown_3243a281(void) {  return 0; }
int32_t unknown_3243a359(void) {  return 0; }
int32_t unknown_3243ab11(void) {  return 0; }
int32_t unknown_3243abf9(void) {  return 0; }
int32_t unknown_3243bb09(void) {  return 0; }
int32_t unknown_3243bbe2(void) {  return 0; }
int32_t unknown_3243bcdf(void) {  return 0; }
int32_t unknown_3243bdc8(void) {  return 0; }
int32_t unknown_3243bef1(void) {  return 0; }
int32_t unknown_3243bffa(void) {  return 0; }
int32_t unknown_3243c509(void) {  return 0; }
int32_t unknown_3243c5e2(void) {  return 0; }
int32_t unknown_3243c6df(void) {  return 0; }
int32_t unknown_3243c7c8(void) {  return 0; }
int32_t unknown_3243c8f1(void) {  return 0; }
int32_t unknown_3243c9fa(void) {  return 0; }
int32_t unknown_3243cf3f(void) {  return 0; }
int32_t unknown_3243d02d(void) {  return 0; }
int32_t unknown_3243d0f6(void) {  return 0; }
int32_t unknown_3243d1d0(void) {  return 0; }
int32_t unknown_3243d2cc(void) {  return 0; }
int32_t unknown_3243d3c3(void) {  return 0; }
int32_t unknown_3243d4f4(void) {  return 0; }
int32_t unknown_3243d5f5(void) {  return 0; }
int32_t unknown_3243dba9(void) {  return 0; }
int32_t unknown_3243dc82(void) {  return 0; }
int32_t unknown_3243dd7f(void) {  return 0; }
int32_t unknown_3243de68(void) {  return 0; }
int32_t unknown_3243df91(void) {  return 0; }
int32_t unknown_3243e09a(void) {  return 0; }
int32_t unknown_3243e672(void) {  return 0; }
int32_t unknown_3243e743(void) {  return 0; }
int32_t unknown_3243e83b(void) {  return 0; }
int32_t unknown_3243e91a(void) {  return 0; }
int32_t unknown_3243e9d7(void) {  return 0; }
int32_t unknown_3243eab2(void) {  return 0; }
int32_t unknown_3243ebab(void) {  return 0; }
int32_t unknown_3243ec93(void) {  return 0; }
int32_t unknown_3243ed9f(void) {  return 0; }
int32_t unknown_3243ee8b(void) {  return 0; }
int32_t unknown_3243ef96(void) {  return 0; }
int32_t unknown_3243f073(void) {  return 0; }
int32_t unknown_3243f124(void) {  return 0; }
int32_t unknown_3243f1fc(void) {  return 0; }
int32_t unknown_3243f2ee(void) {  return 0; }
int32_t unknown_3243f3d8(void) {  return 0; }
int32_t unknown_3243f4e4(void) {  return 0; }
int32_t unknown_3243f5e6(void) {  return 0; }
int32_t unknown_3244013f(void) {  return 0; }
int32_t unknown_3244022d(void) {  return 0; }
int32_t unknown_324402fb(void) {  return 0; }
int32_t unknown_324403d6(void) {  return 0; }
int32_t unknown_324404d0(void) {  return 0; }
int32_t unknown_324405c7(void) {  return 0; }
int32_t unknown_324406ec(void) {  return 0; }
int32_t unknown_324407ed(void) {  return 0; }
int32_t unknown_32440bd9(void) {  return 0; }
int32_t unknown_32440cbf(void) {  return 0; }
int32_t unknown_32441048(void) {  return 0; }
int32_t unknown_3244112e(void) {  return 0; }
int32_t unknown_324416ff(void) {  return 0; }
int32_t unknown_324417ed(void) {  return 0; }
int32_t unknown_324418bb(void) {  return 0; }
int32_t unknown_32441996(void) {  return 0; }
int32_t unknown_32441a90(void) {  return 0; }
int32_t unknown_32441b87(void) {  return 0; }
int32_t unknown_32441c86(void) {  return 0; }
int32_t unknown_32441d5e(void) {  return 0; }
int32_t unknown_324420c7(void) {  return 0; }
int32_t unknown_324421ad(void) {  return 0; }
int32_t unknown_32442a7f(void) {  return 0; }
int32_t unknown_32442b6d(void) {  return 0; }
int32_t unknown_32442c3b(void) {  return 0; }
int32_t unknown_32442d16(void) {  return 0; }
int32_t unknown_32442e10(void) {  return 0; }
int32_t unknown_32442f07(void) {  return 0; }
int32_t unknown_3244302c(void) {  return 0; }
int32_t unknown_3244312d(void) {  return 0; }
int32_t unknown_3244390f(void) {  return 0; }
int32_t unknown_324439fd(void) {  return 0; }
int32_t unknown_32443acb(void) {  return 0; }
int32_t unknown_32443ba6(void) {  return 0; }
int32_t unknown_32443ca0(void) {  return 0; }
int32_t unknown_32443d97(void) {  return 0; }
int32_t unknown_32443eb6(void) {  return 0; }
int32_t unknown_32443fb6(void) {  return 0; }
int32_t unknown_324444cf(void) {  return 0; }
int32_t unknown_324445bd(void) {  return 0; }
int32_t unknown_3244468b(void) {  return 0; }
int32_t unknown_32444766(void) {  return 0; }
int32_t unknown_3244485b(void) {  return 0; }
int32_t unknown_32444952(void) {  return 0; }
int32_t unknown_32444a71(void) {  return 0; }
int32_t unknown_32444b71(void) {  return 0; }
int32_t unknown_32444ebb(void) {  return 0; }
int32_t unknown_32444f8c(void) {  return 0; }
int32_t unknown_324456ff(void) {  return 0; }
int32_t unknown_324457ed(void) {  return 0; }
int32_t unknown_324458bb(void) {  return 0; }
int32_t unknown_32445996(void) {  return 0; }
int32_t unknown_32445a90(void) {  return 0; }
int32_t unknown_32445b87(void) {  return 0; }
int32_t unknown_32445cac(void) {  return 0; }
int32_t unknown_32445dad(void) {  return 0; }
int32_t unknown_32445fc7(int32_t a1, int32_t a2, int32_t a3, int32_t a4, int32_t a5, int32_t a6) { (void)a1; (void)a2; (void)a3; (void)a4; (void)a5; (void)a6; return 0; }
int32_t unknown_3244602f(int32_t a1, int32_t a2, int32_t a3, int32_t a4, int32_t a5) { (void)a1; (void)a2; (void)a3; (void)a4; (void)a5; return 0; }
int32_t unknown_32446259(void) {  return 0; }
int32_t unknown_32446332(void) {  return 0; }
int32_t unknown_3244642f(void) {  return 0; }
int32_t unknown_32446518(void) {  return 0; }
int32_t unknown_3244663b(void) {  return 0; }
int32_t unknown_32446743(void) {  return 0; }
int32_t unknown_3c051527(void) {  return 0; }
int32_t unknown_3c257317(void) {  return 0; }
int32_t unknown_3c258002(void) {  return 0; }
int32_t unknown_3c25803f(void) {  return 0; }
int32_t unknown_3c258064(void) {  return 0; }
int32_t unknown_3c45551b(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_3c8a9a27(void) {  return 0; }
int32_t unknown_3c8b5d78(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_3ccd81b4(void) {  return 0; }
int32_t unknown_3ced61e1(void) {  return 0; }
int32_t unknown_4c267ef2(void) {  return 0; }
int32_t unknown_4c267f22(void) {  return 0; }
int32_t unknown_4c267f5f(void) {  return 0; }
int32_t unknown_4c26976f(void) {  return 0; }
int32_t unknown_4c2b2771(void) {  return 0; }
int32_t unknown_4c2b778e(void) {  return 0; }
int32_t unknown_4c4e4f87(void) {  return 0; }
int32_t unknown_4ca8b831(void) {  return 0; }
int32_t unknown_4cb2c22b(void) {  return 0; }
int32_t unknown_4ccd51bf(void) {  return 0; }
int32_t unknown_4ce47dc8(void) {  return 0; }
int32_t unknown_4ce56273(void) {  return 0; }
int32_t unknown_5c001052(int32_t a1, int32_t a2, int32_t a3) {
    int32_t obj[3];
    obj[0] = a1;
    obj[1] = a2;
    obj[2] = a3;
    g_cb = a1;
    g_inst = a2;
    g_self = (int32_t)(intptr_t)obj;
    function_1060();
    return obj[2];
}
int32_t unknown_5c207506(void) {  return 0; }
int32_t unknown_5c42496c(void) {  return 0; }
int32_t unknown_5c424992(void) {  return 0; }
int32_t unknown_5c4249b0(void) {  return 0; }
int32_t unknown_5c4249ce(void) {  return 0; }
int32_t unknown_5c8854f8(void) {  return 0; }
int32_t unknown_5cc457bf(void) {  return 0; }
int32_t unknown_5ccd48b6(void) {  return 0; }
int32_t unknown_5cce74b0(void) {  return 0; }
int32_t unknown_5cdae17a(void) {  return 0; }
int32_t unknown_5cf67b14(void) {  return 0; }
int32_t unknown_6c2154ea(void) {  return 0; }
int32_t unknown_6c2173c1(void) {  return 0; }
int32_t unknown_6cae580b(void) {  return 0; }
int32_t unknown_6cb857e4(void) {  return 0; }
int32_t unknown_6cec621f(void) {  return 0; }
int32_t unknown_7c006aa8(void) {  return 0; }
int32_t unknown_7c4d502a(void) {  return 0; }
int32_t unknown_7c4e4f68(void) {  return 0; }
int32_t unknown_7c7f614e(void) {  return 0; }
int32_t unknown_7c7f8fbf(void) {  return 0; }
int32_t unknown_7c7f9302(int32_t a1) { (void)a1; return 0; }
int32_t unknown_7c7f9702(int32_t a1) { (void)a1; return 0; }
int32_t unknown_7c7fa20d(void) {  return 0; }
int32_t unknown_7c7fb4ed(void) {  return 0; }
int32_t unknown_7c7fbecd(void) {  return 0; }
int32_t unknown_7c7fc8cd(void) {  return 0; }
int32_t unknown_7c7fd57d(void) {  return 0; }
int32_t unknown_7c7fdf59(int32_t a1) { (void)a1; return 0; }
int32_t unknown_7c8006f6(int32_t a1) { (void)a1; return 0; }
int32_t unknown_7c800b76(int32_t a1) { (void)a1; return 0; }
int32_t unknown_7c8010e2(int32_t a1) { (void)a1; return 0; }
int32_t unknown_7c801c06(int32_t a1) { (void)a1; return 0; }
int32_t unknown_7c802412(int32_t a1) { (void)a1; return 0; }
int32_t unknown_7c803e7d(void) {  return 0; }
int32_t unknown_7c804a36(int32_t a1) { (void)a1; return 0; }
int32_t unknown_7cbccc50(void) {  return 0; }
int32_t unknown_7cdaf56b(void) {  return 0; }
int32_t unknown_7cee7aae(void) {  return 0; }
int32_t unknown_7ceea58f(void) {  return 0; }
int32_t unknown_8c172682(int32_t * a1) { (void)a1; return 0; }
int32_t unknown_8c262886(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_8c26291d(int32_t a1) { (void)a1; return 0; }
int32_t unknown_8c262cbb(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_8c262d71(int32_t a1) { (void)a1; return 0; }
int32_t unknown_8c262ec0(int32_t a1) { (void)a1; return 0; }
int32_t unknown_8c262f48(int32_t a1) { (void)a1; return 0; }
int32_t unknown_8c2630c2(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_8c263171(int32_t a1) { (void)a1; return 0; }
int32_t unknown_8c263489(int32_t a1) { (void)a1; return 0; }
int32_t unknown_8c263558(int32_t a1) { (void)a1; return 0; }
int32_t unknown_8c263c08(void) {  return 0; }
int32_t unknown_8c263cbf(void) {  return 0; }
int32_t unknown_8c263d22(void) {  return 0; }
int32_t unknown_8c263d94(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_8c263e9d(void) {  return 0; }
int32_t unknown_8c263edb(void) {  return 0; }
int32_t unknown_8c263f32(void) {  return 0; }
int32_t unknown_8c26502e(void) {  return 0; }
int32_t unknown_8c265350(void) {  return 0; }
int32_t unknown_8c265548(void) {  return 0; }
int32_t unknown_8c265636(void) {  return 0; }
int32_t unknown_8c26571f(void) {  return 0; }
int32_t unknown_8c2679ab(void) {  return 0; }
int32_t unknown_8c269026(void) {  return 0; }
int32_t unknown_8c26912c(void) {  return 0; }
int32_t unknown_8c269226(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_8c269275(void) {  return 0; }
int32_t unknown_8c2694d2(void) {  return 0; }
int32_t unknown_8c269529(void) {  return 0; }
int32_t unknown_8c2695a4(void) {  return 0; }
int32_t unknown_8c26988a(void) {  return 0; }
int32_t unknown_8c2698e1(void) {  return 0; }
int32_t unknown_8c26995c(void) {  return 0; }
int32_t unknown_8c26a3ff(void) {  return 0; }
int32_t unknown_8c26a456(void) {  return 0; }
int32_t unknown_8c26a4ce(void) {  return 0; }
int32_t unknown_8c26b69f(void) {  return 0; }
int32_t unknown_8c26b6f6(void) {  return 0; }
int32_t unknown_8c26b76e(void) {  return 0; }
int32_t unknown_8c26c099(void) {  return 0; }
int32_t unknown_8c26c0f0(void) {  return 0; }
int32_t unknown_8c26c168(void) {  return 0; }
int32_t unknown_8c26cac0(void) {  return 0; }
int32_t unknown_8c26cb17(void) {  return 0; }
int32_t unknown_8c26cb8f(void) {  return 0; }
int32_t unknown_8c26d4ab(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_8c26d4fa(void) {  return 0; }
int32_t unknown_8c26d73c(void) {  return 0; }
int32_t unknown_8c26d793(void) {  return 0; }
int32_t unknown_8c26d80b(void) {  return 0; }
int32_t unknown_8c26dfc3(void) {  return 0; }
int32_t unknown_8c26e00b(void) {  return 0; }
int32_t unknown_8c26fcbf(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_8c26fd13(void) {  return 0; }
int32_t unknown_8c2707cd(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_8c270824(void) {  return 0; }
int32_t unknown_8c270890(void) {  return 0; }
int32_t unknown_8c270c3c(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_8c270c93(void) {  return 0; }
int32_t unknown_8c270cff(void) {  return 0; }
int32_t unknown_8c27128a(void) {  return 0; }
int32_t unknown_8c2712e1(void) {  return 0; }
int32_t unknown_8c27135c(void) {  return 0; }
int32_t unknown_8c271cbd(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_8c271d14(void) {  return 0; }
int32_t unknown_8c271d80(void) {  return 0; }
int32_t unknown_8c27206c(void) {  return 0; }
int32_t unknown_8c272168(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_8c2721b7(void) {  return 0; }
int32_t unknown_8c272228(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_8c272277(void) {  return 0; }
int32_t unknown_8c2722fc(void) {  return 0; }
int32_t unknown_8c2725e7(void) {  return 0; }
int32_t unknown_8c27263e(void) {  return 0; }
int32_t unknown_8c2726b9(void) {  return 0; }
int32_t unknown_8c272fe8(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_8c273037(void) {  return 0; }
int32_t unknown_8c2730bc(void) {  return 0; }
int32_t unknown_8c2731b6(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_8c273204(void) {  return 0; }
int32_t unknown_8c273278(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_8c2732c7(void) {  return 0; }
int32_t unknown_8c27348c(void) {  return 0; }
int32_t unknown_8c27359c(void) {  return 0; }
int32_t unknown_8c27404b(void) {  return 0; }
int32_t unknown_8c2740a2(void) {  return 0; }
int32_t unknown_8c27411a(void) {  return 0; }
int32_t unknown_8c274ad6(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_8c274b2d(void) {  return 0; }
int32_t unknown_8c274b99(void) {  return 0; }
int32_t unknown_8c274e2c(void) {  return 0; }
int32_t unknown_8c274f37(void) {  return 0; }
int32_t unknown_8c275028(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_8c275077(void) {  return 0; }
int32_t unknown_8c436966(void) {  return 0; }
int32_t unknown_8c43af8b(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_8c97505b(void) {  return 0; }
int32_t unknown_8c9a5fca(void) {  return 0; }
int32_t unknown_8ca458de(void) {  return 0; }
int32_t unknown_8cff6804(void) {  return 0; }
int32_t unknown_9c126bb0(void) {  return 0; }
int32_t unknown_9c1f7534(void) {  return 0; }
int32_t unknown_9c4049ec(void) {  return 0; }
int32_t unknown_9c404a4e(void) {  return 0; }
int32_t unknown_9c4f4eb0(void) {  return 0; }
int32_t unknown_9c8262fd(void) {  return 0; }
int32_t unknown_9c826893(void) {  return 0; }
int32_t unknown_9c82703f(void) {  return 0; }
int32_t unknown_9c82756b(void) {  return 0; }
int32_t unknown_9c827590(void) {  return 0; }
int32_t unknown_9c8275b9(void) {  return 0; }
int32_t unknown_9c827636(void) {  return 0; }
int32_t unknown_9c82772c(void) {  return 0; }
int32_t unknown_9c86733a(void) {  return 0; }
int32_t unknown_9c86738f(void) {  return 0; }
int32_t unknown_9cca78fa(void) {  return 0; }
int32_t unknown_9ccd38c1(void) {  return 0; }
int32_t unknown_9ccd4f1d(void) {  return 0; }
int32_t unknown_9ccd5097(void) {  return 0; }
int32_t unknown_9ccd51a4(void) {  return 0; }
int32_t unknown_9ccd51ff(void) {  return 0; }
int32_t unknown_9cce67bd(void) {  return 0; }
int32_t unknown_9cd51325(int32_t a1) { (void)a1; return 0; }
int32_t unknown_9cf66619(void) {  return 0; }
int32_t unknown_a431403(int32_t a1) { (void)a1; return 0; }
int32_t unknown_a43140f(int32_t a1) { (void)a1; return 0; }
int32_t unknown_a43142f(int32_t a1) { (void)a1; return 0; }
int32_t unknown_a431440(int32_t a1) { (void)a1; return 0; }
int32_t unknown_a43566a(void) {  return 0; }
int32_t unknown_a435a9a(void) {  return 0; }
int32_t unknown_a435c1b(void) {  return 0; }
int32_t unknown_a435d7f(void) {  return 0; }
int32_t unknown_a435d9f(void) {  return 0; }
int32_t unknown_a435dc3(void) {  return 0; }
int32_t unknown_a435df2(void) {  return 0; }
int32_t unknown_a4360d6(void) {  return 0; }
int32_t unknown_a436121(void) {  return 0; }
int32_t unknown_a436150(void) {  return 0; }
int32_t unknown_a436299(void) {  return 0; }
int32_t unknown_a436496(void) {  return 0; }
int32_t unknown_a43659d(void) {  return 0; }
int32_t unknown_a436ab8(void) {  return 0; }
int32_t unknown_a436d94(void) {  return 0; }
int32_t unknown_a436ec5(void) {  return 0; }
int32_t unknown_a436f48(void) {  return 0; }
int32_t unknown_a437078(void) {  return 0; }
int32_t unknown_a437264(void) {  return 0; }
int32_t unknown_a4372f5(void) {  return 0; }
int32_t unknown_a43797a(void) {  return 0; }
int32_t unknown_a4379c0(void) {  return 0; }
int32_t unknown_a4379d8(void) {  return 0; }
int32_t unknown_a437d53(void) {  return 0; }
int32_t unknown_a437d85(void) {  return 0; }
int32_t unknown_a437dd6(void) {  return 0; }
int32_t unknown_a43f791(int32_t a1) { (void)a1; return 0; }
int32_t unknown_a43fc4c(int32_t a1) { (void)a1; return 0; }
int32_t unknown_ac0b60e4(void) {  return 0; }
int32_t unknown_ac0b65f5(void) {  return 0; }
int32_t unknown_ac0b66f8(void) {  return 0; }
int32_t unknown_ac0b6a14(void) {  return 0; }
int32_t unknown_ac13552a(void) {  return 0; }
int32_t unknown_ac135bf3(void) {  return 0; }
int32_t unknown_ac13647c(void) {  return 0; }
int32_t unknown_ac13653b(void) {  return 0; }
int32_t unknown_ac136625(void) {  return 0; }
int32_t unknown_ac136690(void) {  return 0; }
int32_t unknown_ac136ad5(void) {  return 0; }
int32_t unknown_ac136b21(void) {  return 0; }
int32_t unknown_ac136d22(void) {  return 0; }
int32_t unknown_ac555114(void) {  return 0; }
int32_t unknown_ac555402(void) {  return 0; }
int32_t unknown_ac713850(void) {  return 0; }
int32_t unknown_ac71386f(void) {  return 0; }
int32_t unknown_ac71388e(void) {  return 0; }
int32_t unknown_ac7138b0(int32_t a1) { (void)a1; return 0; }
int32_t unknown_ac713fd2(void) {  return 0; }
int32_t unknown_ac714013(void) {  return 0; }
int32_t unknown_ac71442b(void) {  return 0; }
int32_t unknown_ac71445e(void) {  return 0; }
int32_t unknown_ac71518e(void) {  return 0; }
int32_t unknown_ac7151ad(void) {  return 0; }
int32_t unknown_ac7151cc(void) {  return 0; }
int32_t unknown_ac7151ee(void) {  return 0; }
int32_t unknown_ac715279(void) {  return 0; }
int32_t unknown_ac715298(void) {  return 0; }
int32_t unknown_ac7152b7(void) {  return 0; }
int32_t unknown_ac7152d9(void) {  return 0; }
int32_t unknown_ac717839(void) {  return 0; }
int32_t unknown_ac71787e(void) {  return 0; }
int32_t unknown_ac7178c3(void) {  return 0; }
int32_t unknown_ac717908(void) {  return 0; }
int32_t unknown_ac717a7d(void) {  return 0; }
int32_t unknown_ac71803d(void) {  return 0; }
int32_t unknown_ac71805c(void) {  return 0; }
int32_t unknown_ac71807b(void) {  return 0; }
int32_t unknown_ac71809d(void) {  return 0; }
int32_t unknown_ac718143(void) {  return 0; }
int32_t unknown_ac718162(void) {  return 0; }
int32_t unknown_ac718181(void) {  return 0; }
int32_t unknown_ac7181a3(int32_t a1) { (void)a1; return 0; }
int32_t unknown_ac7190bb(void) {  return 0; }
int32_t unknown_ac7191d3(void) {  return 0; }
int32_t unknown_ac719655(void) {  return 0; }
int32_t unknown_ac719a0d(void) {  return 0; }
int32_t unknown_ac71a573(void) {  return 0; }
int32_t unknown_ac71aa7e(int32_t a1) { (void)a1; return 0; }
int32_t unknown_ac71b815(void) {  return 0; }
int32_t unknown_ac71c20f(void) {  return 0; }
int32_t unknown_ac71cc34(void) {  return 0; }
int32_t unknown_ac71d8b2(void) {  return 0; }
int32_t unknown_ac72092d(void) {  return 0; }
int32_t unknown_ac720d9c(void) {  return 0; }
int32_t unknown_ac72140d(void) {  return 0; }
int32_t unknown_ac721e1d(void) {  return 0; }
int32_t unknown_ac722111(void) {  return 0; }
int32_t unknown_ac7223a1(void) {  return 0; }
int32_t unknown_ac72276a(void) {  return 0; }
int32_t unknown_ac723161(void) {  return 0; }
int32_t unknown_ac723531(void) {  return 0; }
int32_t unknown_ac723641(void) {  return 0; }
int32_t unknown_ac7241bf(void) {  return 0; }
int32_t unknown_ac724c38(void) {  return 0; }
int32_t unknown_ac724ed1(void) {  return 0; }
int32_t unknown_ac724fd8(void) {  return 0; }
int32_t unknown_ac726752(void) {  return 0; }
int32_t unknown_ac726771(void) {  return 0; }
int32_t unknown_ac726790(void) {  return 0; }
int32_t unknown_ac7267b2(void) {  return 0; }
int32_t unknown_ac854887(void) {  return 0; }
int32_t unknown_ac8651bf(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_ac865d3f(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_ac867404(int32_t a1) { (void)a1; return 0; }
int32_t unknown_ac86741c(void) {  return 0; }
int32_t unknown_ac867480(void) {  return 0; }
int32_t unknown_ac9166a9(void) {  return 0; }
int32_t unknown_ac916952(void) {  return 0; }
int32_t unknown_ac91697b(void) {  return 0; }
int32_t unknown_ac9b5f8a(void) {  return 0; }
int32_t unknown_accd38d8(void) {  return 0; }
int32_t unknown_acce67d0(void) {  return 0; }
int32_t unknown_acd0df8b(int32_t a1) { (void)a1; return 0; }
int32_t unknown_acfc558c(void) {  return 0; }
int32_t unknown_acfc5ad9(void) {  return 0; }
int32_t unknown_acfc6df1(void) {  return 0; }
int32_t unknown_acfcabb0(int32_t a1, int32_t a2, int32_t a3, int32_t a4, int32_t a5, int32_t a6, int32_t a7) { (void)a1; (void)a2; (void)a3; (void)a4; (void)a5; (void)a6; (void)a7; return 0; }
int32_t unknown_acfcf783(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_bc182884(void) {  return 0; }
int32_t unknown_bc185342(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_bc282462(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc282497(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2824ac(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2824bf(void) {  return 0; }
int32_t unknown_bc2824f2(void) {  return 0; }
int32_t unknown_bc282535(void) {  return 0; }
int32_t unknown_bc28256a(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28257f(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc282592(void) {  return 0; }
int32_t unknown_bc2825d5(void) {  return 0; }
int32_t unknown_bc282643(void) {  return 0; }
int32_t unknown_bc282837(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc282a31(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc282a99(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_bc282ab0(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc282ac5(void) {  return 0; }
int32_t unknown_bc282b29(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc283ddf(void) {  return 0; }
int32_t unknown_bc283e0a(void) {  return 0; }
int32_t unknown_bc283e1c(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc283e2c(void) {  return 0; }
int32_t unknown_bc283e3a(void) {  return 0; }
int32_t unknown_bc283e48(void) {  return 0; }
int32_t unknown_bc283e56(void) {  return 0; }
int32_t unknown_bc283e64(void) {  return 0; }
int32_t unknown_bc283e72(void) {  return 0; }
int32_t unknown_bc284e3d(void) {  return 0; }
int32_t unknown_bc284ff6(void) {  return 0; }
int32_t unknown_bc289357(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc289379(void) {  return 0; }
int32_t unknown_bc28939c(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2893b1(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2893c4(void) {  return 0; }
int32_t unknown_bc2893fb(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_bc28944b(void) {  return 0; }
int32_t unknown_bc289757(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc289779(void) {  return 0; }
int32_t unknown_bc289795(void) {  return 0; }
int32_t unknown_bc2897a7(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2897f4(void) {  return 0; }
int32_t unknown_bc28a267(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc28a291(void) {  return 0; }
int32_t unknown_bc28a2b4(void) {  return 0; }
int32_t unknown_bc28a2c6(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28a2d6(void) {  return 0; }
int32_t unknown_bc28a2e4(void) {  return 0; }
int32_t unknown_bc28a320(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28a36c(void) {  return 0; }
int32_t unknown_bc28a7ec(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc28a7fe(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28a80e(void) {  return 0; }
int32_t unknown_bc28a81c(void) {  return 0; }
int32_t unknown_bc28a82a(void) {  return 0; }
int32_t unknown_bc28a838(void) {  return 0; }
int32_t unknown_bc28a846(void) {  return 0; }
int32_t unknown_bc28a854(void) {  return 0; }
int32_t unknown_bc28b547(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc28b578(void) {  return 0; }
int32_t unknown_bc28b586(void) {  return 0; }
int32_t unknown_bc28b594(void) {  return 0; }
int32_t unknown_bc28b5cf(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28b60e(void) {  return 0; }
int32_t unknown_bc28bf27(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc28bf58(void) {  return 0; }
int32_t unknown_bc28bf66(void) {  return 0; }
int32_t unknown_bc28bf74(void) {  return 0; }
int32_t unknown_bc28bf81(void) {  return 0; }
int32_t unknown_bc28bf8e(void) {  return 0; }
int32_t unknown_bc28bfc9(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28c008(void) {  return 0; }
int32_t unknown_bc28c927(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc28c951(void) {  return 0; }
int32_t unknown_bc28c975(void) {  return 0; }
int32_t unknown_bc28c987(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28c997(void) {  return 0; }
int32_t unknown_bc28c9a5(void) {  return 0; }
int32_t unknown_bc28c9e1(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28ca2d(void) {  return 0; }
int32_t unknown_bc28d5d7(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc28d608(void) {  return 0; }
int32_t unknown_bc28d616(void) {  return 0; }
int32_t unknown_bc28d624(void) {  return 0; }
int32_t unknown_bc28d631(void) {  return 0; }
int32_t unknown_bc28d66c(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28d6ab(void) {  return 0; }
int32_t unknown_bc28e0da(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28e0ff(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28e121(void) {  return 0; }
int32_t unknown_bc28e146(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28e15d(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28e172(void) {  return 0; }
int32_t unknown_bc28e185(void) {  return 0; }
int32_t unknown_bc28e1ac(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28e200(void) {  return 0; }
int32_t unknown_bc28e23c(void) {  return 0; }
int32_t unknown_bc28e267(void) {  return 0; }
int32_t unknown_bc28e295(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28e2ad(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28e2c3(void) {  return 0; }
int32_t unknown_bc28e2da(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28e2ef(void) {  return 0; }
int32_t unknown_bc28e316(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28e366(void) {  return 0; }
int32_t unknown_bc28fd5f(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc28fd7f(void) {  return 0; }
int32_t unknown_bc28fda2(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28fdb6(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc28fdc9(void) {  return 0; }
int32_t unknown_bc28fdda(void) {  return 0; }
int32_t unknown_bc28fdff(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_bc28fe47(void) {  return 0; }
int32_t unknown_bc29074e(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc290760(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc290770(void) {  return 0; }
int32_t unknown_bc29077e(void) {  return 0; }
int32_t unknown_bc290bcd(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc290bde(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc290bed(void) {  return 0; }
int32_t unknown_bc291137(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc291159(void) {  return 0; }
int32_t unknown_bc29117c(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc29118d(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2911a0(void) {  return 0; }
int32_t unknown_bc2911b1(void) {  return 0; }
int32_t unknown_bc2911fc(void) {  return 0; }
int32_t unknown_bc291c5e(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc291c6d(void) {  return 0; }
int32_t unknown_bc292467(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc292489(void) {  return 0; }
int32_t unknown_bc2924ac(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2924c1(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2924d4(void) {  return 0; }
int32_t unknown_bc292510(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_bc292560(void) {  return 0; }
int32_t unknown_bc29330f(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc293331(void) {  return 0; }
int32_t unknown_bc293354(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc293365(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc293378(void) {  return 0; }
int32_t unknown_bc293399(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2933ec(void) {  return 0; }
int32_t unknown_bc293ed7(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc293f01(void) {  return 0; }
int32_t unknown_bc293f21(void) {  return 0; }
int32_t unknown_bc293f33(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc293f6c(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc293fb8(void) {  return 0; }
int32_t unknown_bc294a8b(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc2950e3(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc29510e(void) {  return 0; }
int32_t unknown_bc295131(void) {  return 0; }
int32_t unknown_bc295142(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc295151(void) {  return 0; }
int32_t unknown_bc29518d(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2951d5(void) {  return 0; }
int32_t unknown_bc2952a2(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc2952cc(void) {  return 0; }
int32_t unknown_bc2952ef(void) {  return 0; }
int32_t unknown_bc295300(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc29530f(void) {  return 0; }
int32_t unknown_bc29531c(void) {  return 0; }
int32_t unknown_bc295329(void) {  return 0; }
int32_t unknown_bc295365(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2953ad(void) {  return 0; }
int32_t unknown_bc295c63(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc295c8d(void) {  return 0; }
int32_t unknown_bc295cad(void) {  return 0; }
int32_t unknown_bc295cbc(void) {  return 0; }
int32_t unknown_bc295d06(void) {  return 0; }
int32_t unknown_bc295e9f(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc295ec7(void) {  return 0; }
int32_t unknown_bc295ed8(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc295ee7(void) {  return 0; }
int32_t unknown_bc295ef4(void) {  return 0; }
int32_t unknown_bc295f01(void) {  return 0; }
int32_t unknown_bc295f26(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc295f74(void) {  return 0; }
int32_t unknown_bc296807(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296838(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc29684f(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296864(void) {  return 0; }
int32_t unknown_bc29688a(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_bc2968d6(void) {  return 0; }
int32_t unknown_bc29691e(void) {  return 0; }
int32_t unknown_bc29694d(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc29697f(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296995(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2969b0(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2969ca(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc2969e4(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296a0f(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_bc296a63(void) {  return 0; }
int32_t unknown_bc296ab7(void) {  return 0; }
int32_t unknown_bc296ae4(void) {  return 0; }
int32_t unknown_bc296b11(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296b27(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296b44(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296b5e(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296b78(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296bb9(void) {  return 0; }
int32_t unknown_bc296c16(void) {  return 0; }
int32_t unknown_bc296c45(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296c77(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296c8d(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296ca8(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296cc2(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296cdc(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296d07(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_bc296d5b(void) {  return 0; }
int32_t unknown_bc296da9(void) {  return 0; }
int32_t unknown_bc296dd8(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296e0a(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296e20(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296e3b(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296e55(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296e6f(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296e9a(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_bc296eee(void) {  return 0; }
int32_t unknown_bc296f5e(void) {  return 0; }
int32_t unknown_bc296f95(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296fac(int32_t a1) { (void)a1; return 0; }
int32_t unknown_bc296fc1(void) {  return 0; }
int32_t unknown_bc296ff7(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_bc297045(void) {  return 0; }
int32_t unknown_bc573824(void) {  return 0; }
int32_t unknown_bc574822(void) {  return 0; }
int32_t unknown_bc574ee3(void) {  return 0; }
int32_t unknown_bc57505d(void) {  return 0; }
int32_t unknown_bc5750dd(void) {  return 0; }
int32_t unknown_bc5866fd(void) {  return 0; }
int32_t unknown_bc733833(void) {  return 0; }
int32_t unknown_bc733861(void) {  return 0; }
int32_t unknown_bc733880(void) {  return 0; }
int32_t unknown_bc7338a2(void) {  return 0; }
int32_t unknown_bc733fb0(void) {  return 0; }
int32_t unknown_bc733fee(void) {  return 0; }
int32_t unknown_bc73440c(void) {  return 0; }
int32_t unknown_bc73443f(void) {  return 0; }
int32_t unknown_bc735171(void) {  return 0; }
int32_t unknown_bc73519f(void) {  return 0; }
int32_t unknown_bc7351be(void) {  return 0; }
int32_t unknown_bc7351e0(void) {  return 0; }
int32_t unknown_bc73525c(void) {  return 0; }
int32_t unknown_bc73528a(void) {  return 0; }
int32_t unknown_bc7352a9(void) {  return 0; }
int32_t unknown_bc7352cb(void) {  return 0; }
int32_t unknown_bc73781a(void) {  return 0; }
int32_t unknown_bc737859(void) {  return 0; }
int32_t unknown_bc73789e(void) {  return 0; }
int32_t unknown_bc7378e3(void) {  return 0; }
int32_t unknown_bc737a5b(void) {  return 0; }
int32_t unknown_bc738020(void) {  return 0; }
int32_t unknown_bc73804e(void) {  return 0; }
int32_t unknown_bc73806d(void) {  return 0; }
int32_t unknown_bc73808f(void) {  return 0; }
int32_t unknown_bc738126(void) {  return 0; }
int32_t unknown_bc738154(void) {  return 0; }
int32_t unknown_bc738173(void) {  return 0; }
int32_t unknown_bc738195(void) {  return 0; }
int32_t unknown_bc73909e(void) {  return 0; }
int32_t unknown_bc7391b7(void) {  return 0; }
int32_t unknown_bc739633(void) {  return 0; }
int32_t unknown_bc7399eb(void) {  return 0; }
int32_t unknown_bc73a551(void) {  return 0; }
int32_t unknown_bc73aa5f(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_bc73b7f2(void) {  return 0; }
int32_t unknown_bc73c1ec(void) {  return 0; }
int32_t unknown_bc73cc12(void) {  return 0; }
int32_t unknown_bc73d88f(void) {  return 0; }
int32_t unknown_bc74090b(void) {  return 0; }
int32_t unknown_bc740d7a(void) {  return 0; }
int32_t unknown_bc7413eb(void) {  return 0; }
int32_t unknown_bc741dfb(void) {  return 0; }
int32_t unknown_bc7420f5(void) {  return 0; }
int32_t unknown_bc742385(void) {  return 0; }
int32_t unknown_bc742748(void) {  return 0; }
int32_t unknown_bc743145(void) {  return 0; }
int32_t unknown_bc743515(void) {  return 0; }
int32_t unknown_bc743625(void) {  return 0; }
int32_t unknown_bc74419d(void) {  return 0; }
int32_t unknown_bc744c15(void) {  return 0; }
int32_t unknown_bc744eb5(void) {  return 0; }
int32_t unknown_bc744fbc(void) {  return 0; }
int32_t unknown_bc746730(void) {  return 0; }
int32_t unknown_bc746763(void) {  return 0; }
int32_t unknown_bc746782(void) {  return 0; }
int32_t unknown_bc7467a4(void) {  return 0; }
int32_t unknown_bc954ed4(void) {  return 0; }
int32_t unknown_c106b8a(void) {  return 0; }
int32_t unknown_c3342db(void) {  return 0; }
int32_t unknown_ca05ef9(void) {  return 0; }
int32_t unknown_cc3d6424(void) {  return 0; }
int32_t unknown_cc3d71bb(void) {  return 0; }
int32_t unknown_cc3d7797(void) {  return 0; }
int32_t unknown_cc3d77b6(void) {  return 0; }
int32_t unknown_cc3d8010(void) {  return 0; }
int32_t unknown_cc3d8020(void) {  return 0; }
int32_t unknown_cc806139(void) {  return 0; }
int32_t unknown_cc808fb1(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_cc8092ef(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc8096ef(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc80a1fb(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc80b4db(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc80bebb(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc80c8bb(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc80d56b(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc80df49(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc8106e5(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc810b65(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc8110cf(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc811bf5(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc8123ff(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc813e6b(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc814a25(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
int32_t unknown_cc87541b(void) {  return 0; }
int32_t unknown_ccd807d(void) {  return 0; }
int32_t unknown_ccf6010(void) {  return 0; }
int32_t unknown_ccf6028(void) {  return 0; }
int32_t unknown_cdef5f6(void) {  return 0; }
int32_t unknown_dc116c50(void) {  return 0; }
int32_t unknown_dc395928(void) {  return 0; }
int32_t unknown_dc395959(void) {  return 0; }
int32_t unknown_dc395bad(void) {  return 0; }
int32_t unknown_dc396186(void) {  return 0; }
int32_t unknown_dc3964eb(void) {  return 0; }
int32_t unknown_dc39777e(void) {  return 0; }
int32_t unknown_dc3e6b98(void) {  return 0; }
int32_t unknown_dc3e721a(void) {  return 0; }
int32_t unknown_dc3e72f0(void) {  return 0; }
int32_t unknown_dc3e7868(void) {  return 0; }
int32_t unknown_dc3f6c33(void) {  return 0; }
int32_t unknown_dc3f72a3(void) {  return 0; }
int32_t unknown_dc3f73a4(void) {  return 0; }
int32_t unknown_dc3f7fce(void) {  return 0; }
int32_t unknown_dc815406(void) {  return 0; }
int32_t unknown_dc81634d(void) {  return 0; }
int32_t unknown_dc8170d7(void) {  return 0; }
int32_t unknown_dc8176fc(void) {  return 0; }
int32_t unknown_dc817707(void) {  return 0; }
int32_t unknown_dcd3e09b(int32_t a1, int32_t a2) { (void)a1; (void)a2; return 0; }
int32_t unknown_dcdbf586(void) {  return 0; }
int32_t unknown_ec226bfe(void) {  return 0; }
int32_t unknown_ec227288(void) {  return 0; }
int32_t unknown_ec227375(void) {  return 0; }
int32_t unknown_ec4d4fae(void) {  return 0; }
int32_t unknown_ec8f285d(void) {  return 0; }
int32_t unknown_ecc8d8ce(void) {  return 0; }
int32_t unknown_fc8537fc(int32_t a1) { (void)a1; return 0; }
int32_t unknown_fc8547f8(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_fc854ec0(void) {  return 0; }
int32_t unknown_fc854f4c(void) {  return 0; }
int32_t unknown_fc85503a(void) {  return 0; }
int32_t unknown_fc8550b6(void) {  return 0; }
int32_t unknown_fc8666d6(void) {  return 0; }
int32_t unknown_fcdce59a(void) {  return 0; }
int32_t unknown_fcfc279c(int32_t * a1, int32_t * a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_fcfc3b75(void) {  return 0; }
int32_t unknown_fcfc3c0a(void) {  return 0; }
int32_t unknown_fcfc3c34(void) {  return 0; }
int32_t unknown_fcfc5d0b(void) {  return 0; }
int32_t unknown_fcfc63b1(void) {  return 0; }
int32_t unknown_fcfc8402(void) {  return 0; }
int32_t unknown_fcfc8495(void) {  return 0; }
int32_t unknown_fcfc84b5(void) {  return 0; }
int32_t unknown_fcfcfb8c(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_fcfcfba6(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_fcfd5ee1(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_fcfd5f4e(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return 0; }
int32_t unknown_fcfd74d0(int32_t a1, int32_t a2, int32_t a3, int32_t a4) { (void)a1; (void)a2; (void)a3; (void)a4; return 0; }
