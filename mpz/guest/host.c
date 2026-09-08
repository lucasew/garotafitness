#include "host.h"

uint8_t pe_image[PE_IMAGE_SIZE];
int32_t g_self;
int32_t g172, g173, g174, g175, g176, g177;

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
}

uint32_t host_written(void) { return g_dpos; }

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

#define H_IN  ((int32_t *)1)
#define H_OUT ((int32_t *)2)

int32_t * CreateFileA(char *n, int32_t a, int32_t b, struct _SECURITY_ATTRIBUTES *s, int32_t c, int32_t d, void *t) {
    (void)a; (void)b; (void)s; (void)c; (void)d; (void)t;
    if (n && n[0] == 'i') return H_IN;
    return H_OUT;
}
int32_t ReadFile(int32_t *h, int32_t *buf, int32_t n, int32_t *got, void *ov) {
    (void)h; (void)ov;
    uint32_t want = n < 0 ? 0 : (uint32_t)n;
    if (want > g_slen - g_spos) want = g_slen - g_spos;
    if (buf && want) memcpy(buf, g_src + g_spos, want);
    g_spos += want;
    if (got) *got = (int32_t)want;
    return 1;
}
int32_t WriteFile(int32_t *h, const int32_t *buf, int32_t n, int32_t *got, void *ov) {
    (void)h; (void)ov;
    uint32_t want = n < 0 ? 0 : (uint32_t)n;
    if (want > g_dcap - g_dpos) want = g_dcap - g_dpos;
    if (buf && want) memcpy(g_dst + g_dpos, buf, want);
    g_dpos += want;
    if (got) *got = (int32_t)want;
    return 1;
}
int32_t CloseHandle(int32_t *h) { (void)h; return 1; }
int32_t SetFilePointer(int32_t *h, int32_t off, int32_t *hi, int32_t whence) {
    (void)hi;
    uint32_t base = 0;
    if (h == H_IN) {
        if (whence == 1) base = g_spos;
        else if (whence == 2) base = g_slen;
        int64_t p = (int64_t)base + off;
        if (p < 0) p = 0;
        if (p > (int64_t)g_slen) p = g_slen;
        g_spos = (uint32_t)p;
        return (int32_t)g_spos;
    }
    if (whence == 1) base = g_dpos;
    else if (whence == 2) base = g_dpos;
    int64_t p = (int64_t)base + off;
    if (p < 0) p = 0;
    if (p > (int64_t)g_dcap) p = g_dcap;
    g_dpos = (uint32_t)p;
    return (int32_t)g_dpos;
}
int32_t GetFileType(int32_t *h) { (void)h; return 1; }
int32_t * GetStdHandle(int32_t n) { return n == -10 ? H_IN : H_OUT; }
int32_t * VirtualAlloc(void *a, int32_t n, int32_t t, int32_t p) { (void)a; (void)t; (void)p; return (int32_t *)(n > 0 ? calloc(1, (size_t)n) : NULL); }
int32_t VirtualFree(void *a, int32_t n, int32_t t) { (void)n; (void)t; free(a); return 1; }
int32_t * HeapAlloc(int32_t *h, int32_t f, int32_t n) { (void)h; (void)f; return (int32_t *)(n > 0 ? calloc(1, (size_t)n) : NULL); }
int32_t HeapFree(int32_t *h, int32_t f, int32_t *p) { (void)h; (void)f; free(p); return 1; }
int32_t * HeapReAlloc(int32_t *h, int32_t f, int32_t *p, int32_t n) { (void)h; (void)f; return (int32_t *)realloc(p, n > 0 ? (size_t)n : 0); }
int32_t * HeapCreate(int32_t a, int32_t b, int32_t c) { (void)a; (void)b; (void)c; return (int32_t *)1; }
int32_t HeapDestroy(int32_t *h) { (void)h; return 1; }
int32_t GetLastError(void) { return g_last_err; }
void SetLastError(int32_t e) { g_last_err = e; }
int32_t GetVersion(void) { return 0x00000004; }
int32_t GetVersionExA(struct _OSVERSIONINFOA *v) { if (v) { memset(v, 0, sizeof(*v)); v->e1 = 4; } return 1; }
int32_t GetCommandLineA(void) { return 0; }
int32_t GetModuleFileNameA(void *m, char *b, int32_t n) { (void)m; if (b && n > 0) { b[0] = 0; } return 0; }
int32_t * GetModuleHandleA(char *n) { (void)n; return NULL; }
int32_t * LoadLibraryA(char *n) { (void)n; return NULL; }
int32_t (*GetProcAddress(int32_t *m, char *n))(void) { (void)m; (void)n; return NULL; }
void ExitProcess(int32_t c) { (void)c; abort(); }
void RtlUnwind(int32_t *a, int32_t *b, struct _EXCEPTION_RECORD *c, int32_t *d) { (void)a; (void)b; (void)c; (void)d; }
int32_t TlsAlloc(void) { return 0; }
int32_t TlsFree(int32_t i) { (void)i; return 1; }
int32_t TlsGetValue(int32_t i) { (void)i; return 0; }
int32_t TlsSetValue(int32_t i, int32_t v) { (void)i; (void)v; return 1; }
int32_t InterlockedIncrement(int32_t *p) { if (!p) return 0; return ++*p; }
int32_t InterlockedDecrement(int32_t *p) { if (!p) return 0; return --*p; }
void InitializeCriticalSection(struct _RTL_CRITICAL_SECTION *c) { (void)c; }
void DeleteCriticalSection(struct _RTL_CRITICAL_SECTION *c) { (void)c; }
void EnterCriticalSection(struct _RTL_CRITICAL_SECTION *c) { (void)c; }
void LeaveCriticalSection(struct _RTL_CRITICAL_SECTION *c) { (void)c; }
int32_t FlushFileBuffers(int32_t *h) { (void)h; return 1; }
int32_t SetEndOfFile(int32_t *h) { (void)h; return 1; }
int32_t DeleteFileA(char *n) { (void)n; return 1; }
int32_t SetUnhandledExceptionFilter(int32_t a) { (void)a; return 0; }
int32_t GetCurrentThreadId(void) { return 1; }
int32_t GetStartupInfoA(struct _STARTUPINFOA *s) { if (s) memset(s, 0, sizeof(*s)); return 1; }
int32_t GetCPInfo(int32_t a, struct _cpinfo *c) { (void)a; if (c) memset(c, 0, sizeof(*c)); return 1; }
int32_t GetEnvironmentStrings(void) { return 0; }
int32_t GetEnvironmentStringsW(void) { return 0; }
int32_t FreeEnvironmentStringsA(int32_t a) { (void)a; return 1; }
int32_t FreeEnvironmentStringsW(int32_t a) { (void)a; return 1; }
int32_t GetEnvironmentVariableA(int32_t a, int32_t b, int32_t c) { (void)a; (void)b; (void)c; return 0; }
int32_t SetHandleCount(int32_t n) { return n; }
int32_t SetStdHandle(int32_t n, int32_t h) { (void)n; (void)h; return 1; }
int32_t GetStringTypeA(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e) { (void)a;(void)b;(void)c;(void)d;(void)e; return 1; }
int32_t GetStringTypeW(int32_t a, int32_t b, int32_t c, int32_t d) { (void)a;(void)b;(void)c;(void)d; return 1; }
int32_t LCMapStringA(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f) { (void)a;(void)b;(void)c;(void)d;(void)e;(void)f; return 0; }
int32_t LCMapStringW(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f) { (void)a;(void)b;(void)c;(void)d;(void)e;(void)f; return 0; }
int32_t MultiByteToWideChar(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f) { (void)a;(void)b;(void)c;(void)d;(void)e;(void)f; return 0; }
int32_t WideCharToMultiByte(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f, int32_t g, int32_t h) { (void)a;(void)b;(void)c;(void)d;(void)e;(void)f;(void)g;(void)h; return 0; }
int32_t IsBadCodePtr(int32_t a) { (void)a; return 0; }
int32_t IsBadReadPtr(int32_t a, int32_t b) { (void)a; (void)b; return 0; }
int32_t IsBadWritePtr(int32_t a, int32_t b) { (void)a; (void)b; return 0; }

/* CRT helpers used by MP3::Input / ReadFrame (RetDec lost the bodies). */
int32_t function_1001a410(int32_t a1) { (void)a1; return (int32_t)g_spos; }
int32_t function_1001a42c(int32_t a1, int32_t a2, int32_t a3) { (void)a1; (void)a2; (void)a3; return (int32_t)g_slen; }
int32_t function_1001a5a4(void) { g_spos = 0; return 0; }
int32_t function_1001a5d0(int32_t a1, int32_t a2, int32_t a3) {
    (void)a1;
    if (a3 == 0) g_spos = (uint32_t)a2;
    else if (a3 == 1) g_spos += (uint32_t)a2;
    else g_spos = g_slen + (uint32_t)a2;
    if (g_spos > g_slen) g_spos = g_slen;
    return 0;
}
int32_t function_1001d7ce(int32_t a1, int32_t a2) {
    (void)a1; (void)a2;
    if (g_spos >= g_slen) return -1;
    return g_src[g_spos++];
}
