/* Mechanical host for RetDec FULL.c (RAZOR 1.00). INV-03: no PE execution. */
#ifndef RZW_HOST_H
#define RZW_HOST_H

#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <setjmp.h>
#include <signal.h>
#include <wchar.h>
#include <math.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef int8_t int3_t;
typedef __int128 int128_t;
typedef int64_t int864_t;
typedef float float32_t;
typedef double float64_t;
typedef double float80_t;

#ifndef NULL
#define NULL ((void *)0)
#endif

#define PE_IMAGE_VA0 0x429000u
#define PE_IMAGE_SIZE 0x17000u
extern uint8_t pe_image[PE_IMAGE_SIZE];
extern int64_t g_self;

void pe_image_init(void);
void host_set_files(const uint8_t *src, uint32_t slen, uint8_t *dst, uint32_t dcap);
uint32_t host_written(void);
void *host_xmalloc(size_t n);

/* ---- LLVM bit ops ---- */
static inline int32_t llvm_ctlz_i32(uint32_t v, bool z) {
    (void)z;
    return v ? __builtin_clz(v) : 32;
}
static inline int64_t llvm_ctlz_i64(uint64_t v, bool z) {
    (void)z;
    return v ? __builtin_clzll(v) : 64;
}
static inline int64_t llvm_cttz_i64(uint64_t v, bool z) {
    (void)z;
    return v ? __builtin_ctzll(v) : 64;
}

/* ---- SSE2 helpers used by rANS nibble CDF ---- */
static inline void r128_u16(int128_t v, uint16_t o[8]) { memcpy(o, &v, 16); }
static inline int128_t r128_from_u16(const uint16_t o[8]) {
    int128_t v;
    memcpy(&v, o, 16);
    return v;
}
static inline void r128_u8(int128_t v, uint8_t o[16]) { memcpy(o, &v, 16); }
static inline int128_t r128_from_u8(const uint8_t o[16]) {
    int128_t v;
    memcpy(&v, o, 16);
    return v;
}

static inline void __asm_movaps(int128_t v) { (void)v; }
static inline int128_t __asm_movaps_1(int128_t v) { return v; }
static inline int128_t __asm_movdqa(int128_t v) { return v; }
static inline int128_t __asm_movapd(int128_t v) { return v; }
static inline int128_t __asm_movd(int32_t x) { return (int128_t)(uint32_t)x; }
static inline int32_t __asm_movd_2(int128_t v) { return (int32_t)(uint32_t)v; }
static inline int32_t __asm_movd_3(int128_t v) { return (int32_t)(uint32_t)v; }
static inline int32_t __asm_movd_5(int128_t v) { return (int32_t)(uint32_t)v; }
static inline int128_t __asm_movq(int64_t x) { return (int128_t)(uint64_t)x; }
static inline int64_t __asm_movq_13(int128_t v) { return (int64_t)(uint64_t)v; }

static inline int128_t __asm_pxor(int128_t a, int128_t b) { return a ^ b; }
static inline int128_t __asm_pand(int128_t a, int128_t b) { return a & b; }
static inline int128_t __asm_por(int128_t a, int128_t b) { return a | b; }
static inline void __asm_por_4(int128_t a, int128_t b) { (void)a; (void)b; }

static inline int128_t __asm_psubw(int128_t a, int128_t b) {
    uint16_t x[8], y[8];
    r128_u16(a, x);
    r128_u16(b, y);
    for (int i = 0; i < 8; i++) x[i] = (uint16_t)(x[i] - y[i]);
    return r128_from_u16(x);
}
static inline int128_t __asm_psubw_8(int128_t a, int128_t b) { return __asm_psubw(a, b); }
static inline int128_t __asm_paddw(int128_t a, int128_t b) {
    uint16_t x[8], y[8];
    r128_u16(a, x);
    r128_u16(b, y);
    for (int i = 0; i < 8; i++) x[i] = (uint16_t)(x[i] + y[i]);
    return r128_from_u16(x);
}
static inline int128_t __asm_paddd(int128_t a, int128_t b) {
    uint32_t x[4], y[4];
    memcpy(x, &a, 16);
    memcpy(y, &b, 16);
    for (int i = 0; i < 4; i++) x[i] += y[i];
    int128_t r;
    memcpy(&r, x, 16);
    return r;
}
static inline int128_t __asm_paddd_11(int128_t a, int128_t b) { return __asm_paddd(a, b); }
static inline int128_t __asm_psraw(int128_t a, int32_t n) {
    int16_t x[8];
    memcpy(x, &a, 16);
    int s = n & 15;
    for (int i = 0; i < 8; i++) x[i] = (int16_t)(x[i] >> s);
    int128_t r;
    memcpy(&r, x, 16);
    return r;
}
static inline int128_t __asm_punpcklwd(int128_t a, int128_t b) {
    uint16_t x[8], y[8], o[8];
    r128_u16(a, x);
    r128_u16(b, y);
    o[0] = x[0];
    o[1] = y[0];
    o[2] = x[1];
    o[3] = y[1];
    o[4] = x[2];
    o[5] = y[2];
    o[6] = x[3];
    o[7] = y[3];
    return r128_from_u16(o);
}
static inline int128_t __asm_punpckhdq(int128_t a, int128_t b) {
    uint32_t x[4], y[4], o[4];
    memcpy(x, &a, 16);
    memcpy(y, &b, 16);
    o[0] = x[2];
    o[1] = y[2];
    o[2] = x[3];
    o[3] = y[3];
    int128_t r;
    memcpy(&r, o, 16);
    return r;
}
static inline int128_t __asm_pshufd(int128_t a, int32_t imm) {
    uint32_t x[4], o[4];
    memcpy(x, &a, 16);
    unsigned u = (unsigned)imm;
    o[0] = x[u & 3];
    o[1] = x[(u >> 2) & 3];
    o[2] = x[(u >> 4) & 3];
    o[3] = x[(u >> 6) & 3];
    int128_t r;
    memcpy(&r, o, 16);
    return r;
}
static inline int128_t __asm_packsswb(int128_t a, int128_t b) {
    int16_t x[8], y[8];
    uint8_t o[16];
    memcpy(x, &a, 16);
    memcpy(y, &b, 16);
    for (int i = 0; i < 8; i++) {
        int v = x[i];
        if (v > 127) v = 127;
        if (v < -128) v = -128;
        o[i] = (uint8_t)v;
    }
    for (int i = 0; i < 8; i++) {
        int v = y[i];
        if (v > 127) v = 127;
        if (v < -128) v = -128;
        o[8 + i] = (uint8_t)v;
    }
    return r128_from_u8(o);
}
static inline int32_t __asm_pmovmskb(int128_t a) {
    uint8_t x[16];
    r128_u8(a, x);
    int32_t m = 0;
    for (int i = 0; i < 16; i++)
        if (x[i] & 0x80) m |= 1 << i;
    return m;
}
static inline int128_t __asm_psrldq(int128_t a, int32_t n) {
    uint8_t x[16], o[16];
    r128_u8(a, x);
    int s = n < 0 ? 0 : n;
    if (s > 16) s = 16;
    memset(o, 0, 16);
    if (s < 16) memcpy(o, x + s, (size_t)(16 - s));
    return r128_from_u8(o);
}
static inline int128_t __asm_pcmpgtd(int128_t a, int128_t b) {
    int32_t x[4], y[4], o[4];
    memcpy(x, &a, 16);
    memcpy(y, &b, 16);
    for (int i = 0; i < 4; i++) o[i] = x[i] > y[i] ? -1 : 0;
    int128_t r;
    memcpy(&r, o, 16);
    return r;
}
static inline int128_t __asm_pmaddwd(int128_t a, int128_t b) {
    int16_t x[8], y[8];
    int32_t o[4];
    memcpy(x, &a, 16);
    memcpy(y, &b, 16);
    o[0] = (int32_t)x[0] * y[0] + (int32_t)x[1] * y[1];
    o[1] = (int32_t)x[2] * y[2] + (int32_t)x[3] * y[3];
    o[2] = (int32_t)x[4] * y[4] + (int32_t)x[5] * y[5];
    o[3] = (int32_t)x[6] * y[6] + (int32_t)x[7] * y[7];
    int128_t r;
    memcpy(&r, o, 16);
    return r;
}

static inline double __asm_addsd(double a, double b) { return a + b; }
static inline double __asm_divsd(double a, double b) { return b != 0 ? a / b : 0; }
static inline double __asm_mulsd(double a, double b) { return a * b; }
static inline double __asm_movsd(double a) { return a; }
static inline double __asm_movsd_14(double a) { return a; }
static inline double __asm_cvtsi2sd(int32_t a) { return (double)a; }

static inline void __asm_rep_movsd_memcpy(char *dst, char *src, int32_t ndwords) {
    if (dst && src && ndwords > 0) memcpy(dst, src, (size_t)ndwords * 4);
}
static inline void __asm_rep_stosb_memset(char *dst, int32_t v, int32_t n) {
    if (dst && n > 0) memset(dst, (int)v, (size_t)n);
}
static inline void __asm_rep_stosd_memset(char *dst, int32_t v, int32_t n) {
    if (!dst || n <= 0) return;
    for (int32_t i = 0; i < n; i++) ((int32_t *)dst)[i] = v;
}
static inline void __asm_rep_stosq_memset(char *dst, int64_t v, int32_t n) {
    if (!dst || n <= 0) return;
    for (int32_t i = 0; i < n; i++) ((int64_t *)dst)[i] = v;
}

#define __asm_sti(...) (0)
#define __asm_hlt(...) (0)
#define __asm_int(...) (0)
#define __asm_int1(...) (0)
#define __asm_int3(...) (0)
#define __asm_mfence(...) (0)
#define __asm_pause(...) (0)
#define __asm_wait(...) (0)
#define __asm_fnsave(...) (0)
#define __asm_sldt(...) (0)
static inline int32_t __asm_in(int32_t a, ...) { (void)a; return 0; }
static inline int32_t __asm_in_7(int32_t a, ...) { (void)a; return 0; }
static inline int32_t __asm_in_9(int32_t a, ...) { (void)a; return 0; }
static inline int32_t __asm_insb(int16_t a) { (void)a; return 0; }
static inline int32_t __asm_out(int32_t a, int32_t b) { (void)a; (void)b; return 0; }
static inline int32_t __asm_out_6(int32_t a, int32_t b) { (void)a; (void)b; return 0; }
static inline int32_t __asm_out_10(int32_t a, int32_t b) { (void)a; (void)b; return 0; }
static inline int32_t __asm_out_12(int32_t a, int32_t b) { (void)a; (void)b; return 0; }
static inline int32_t __asm_outsb(int16_t a, char b) { (void)a; (void)b; return 0; }
static inline int32_t __asm_outsd(int16_t a, int32_t b) { (void)a; (void)b; return 0; }

int64_t __readgsqword(int32_t off);
int32_t __readfsdword(int32_t off);
void __writefsdword(int32_t off, int32_t val);

/* Win32 / CRT stand-ins. */
struct _COORD { int16_t e0; int16_t e1; };
struct _EXCEPTION_RECORD {
    int32_t e0; int32_t e1; struct _EXCEPTION_RECORD *e2; int64_t *e3; int32_t e4; int32_t e5[1];
};
struct _M128A { int64_t e0; int64_t e1; };
struct _CONTEXT {
    int64_t e0, e1, e2, e3, e4, e5;
    int32_t e6, e7;
    int16_t e8, e9, e10, e11, e12, e13;
    int32_t e14;
    int64_t e15, e16, e17, e18, e19, e20, e21, e22, e23, e24, e25, e26, e27, e28, e29, e30, e31, e32, e33, e34, e35, e36, e37, e38;
    struct _M128A e39[26];
    int64_t e40, e41, e42, e43, e44, e45;
};
struct _EXCEPTION_POINTERS { struct _EXCEPTION_RECORD *e0; struct _CONTEXT *e1; };
struct _FILETIME { int32_t e0; int32_t e1; };
struct _IMAGE_RUNTIME_FUNCTION_ENTRY { int32_t e0; int32_t e1; int64_t e2; };
struct _IO_FILE { int32_t e0; };
struct _KNONVOLATILE_CONTEXT_POINTERS { int64_t e0; int64_t e1; };
struct _LARGE_INTEGER { int64_t e0; };
struct _LIST_ENTRY { struct _LIST_ENTRY *e0; struct _LIST_ENTRY *e1; };
struct _MEMORYSTATUSEX {
    int32_t e0, e1; int64_t e2, e3, e4, e5, e6, e7, e8;
};
struct _OVERLAPPED { int32_t e0, e1; int64_t e2; int64_t *e3; };
struct _RTL_CRITICAL_SECTION_DEBUG;
struct _RTL_CRITICAL_SECTION {
    struct _RTL_CRITICAL_SECTION_DEBUG *e0; int32_t e1, e2; int64_t *e3, *e4; int32_t e5;
};
struct _RTL_CRITICAL_SECTION_DEBUG {
    int16_t e0, e1; struct _RTL_CRITICAL_SECTION *e2; struct _LIST_ENTRY e3;
    int32_t e4, e5, e6; int16_t e7, e8;
};
struct _SECURITY_ATTRIBUTES { int32_t e0; int64_t *e1; bool e2; };
struct _SMALL_RECT { int16_t e0, e1, e2, e3; };
struct _CONSOLE_SCREEN_BUFFER_INFO {
    struct _COORD e0, e1; int16_t e2; struct _SMALL_RECT e3; struct _COORD e4;
};
struct _STARTUPINFOA {
    int32_t e0; char *e1, *e2, *e3;
    int32_t e4, e5, e6, e7, e8, e9, e10, e11;
    int16_t e12, e13; char *e14; int64_t *e15, *e16, *e17;
};
struct _SYSTEMTIME { int16_t e0, e1, e2, e3, e4, e5, e6, e7; };
struct _TYPEDEF___sigset_t { int32_t e0[1]; };
struct _UNWIND_HISTORY_TABLE_ENTRY { int64_t e0; struct _IMAGE_RUNTIME_FUNCTION_ENTRY *e1; };
struct _UNWIND_HISTORY_TABLE {
    int32_t e0; char e1, e2, e3, e4; int64_t e5, e6;
    struct _UNWIND_HISTORY_TABLE_ENTRY e7[1];
};
struct _WIN32_FIND_DATAW {
    int32_t e0; struct _FILETIME e1, e2, e3;
    int32_t e4, e5, e6, e7;
    int16_t e8[1]; int16_t e9[14];
    int32_t e10, e11; int16_t e12;
};
/* setjmp.h already defines struct __jmp_buf_tag */
struct _OSVERSIONINFOA { int32_t e0, e1, e2, e3, e4; char e5[128]; };
struct _cpinfo { int32_t e0; char e1[1]; char e2[1]; };

int64_t __C_specific_handler(void);
int32_t (*__dllonexit(int32_t (*a1)(), void (***a2)(), void (***a3)()))();
int32_t __getmainargs(int32_t *a1, char ***a2, char ***a3, int32_t a4, int64_t *a5);
int64_t __iob_func(void);
int64_t __lconv_init(void);
void __set_app_type(int32_t a1);
void __setusermatherr(int64_t a1);
void _amsg_exit(int32_t a1);
void _cexit(void);
void _endthreadex(int32_t a1);
void _initterm(void (**a1)(), void (**a2)());
void _lock(int32_t a1);
void _unlock(int32_t a1);
int32_t _vsnwprintf(int16_t *a1, int32_t a2, int16_t *a3, int64_t a4);
int32_t _vswprintf(int16_t *a1, int16_t *a2, int64_t a3);
int16_t *_wsetlocale(int32_t a1, int16_t *a2);

int64_t *AddVectoredExceptionHandler(int32_t a1, int32_t (*a2)(struct _EXCEPTION_POINTERS *));
int32_t RemoveVectoredExceptionHandler(int64_t *a1);
bool GlobalMemoryStatusEx(struct _MEMORYSTATUSEX *a1);
bool PathMatchSpecW(int16_t *a1, int16_t *a2);
char RtlAddFunctionTable(struct _IMAGE_RUNTIME_FUNCTION_ENTRY *a1, int32_t a2, int64_t a3);
void RtlCaptureContext(struct _CONTEXT *a1);
struct _IMAGE_RUNTIME_FUNCTION_ENTRY *RtlLookupFunctionEntry(int64_t a1, int64_t *a2, struct _UNWIND_HISTORY_TABLE *a3);
int64_t (*RtlVirtualUnwind(int32_t a1, int64_t a2, int64_t a3, struct _IMAGE_RUNTIME_FUNCTION_ENTRY *a4, struct _CONTEXT *a5, int64_t **a6, int64_t *a7, struct _KNONVOLATILE_CONTEXT_POINTERS *a8))(struct _EXCEPTION_RECORD *, int64_t *, struct _CONTEXT *, int64_t *);
int64_t function_43c718(int64_t a1, int64_t *a2, int64_t a3);
int64_t function_43c798(int32_t a1, int64_t a2, int64_t a3, int64_t a4);
int32_t _wcsicmp(int16_t *a, int16_t *b);
int16_t *_wcslwr(int16_t *s);
char *_ultoa(uint32_t v, char *buf, int32_t base);
void __writeNullptrQword(int64_t a);
int32_t *_errno(void);
int32_t __readfsbyte(int32_t off);
void __frontend_reg_store_fpr(int64_t a, ...);
void *_onexit(void *f);
char *_strdup(const char *s);

int64_t *CreateFileA(char *n, int32_t a, int32_t b, struct _SECURITY_ATTRIBUTES *s, int32_t c, int32_t d, void *t);
int64_t *CreateFileW(int16_t *n, int32_t a, int32_t b, struct _SECURITY_ATTRIBUTES *s, int32_t c, int32_t d, void *t);
int32_t ReadFile(int64_t *h, void *buf, int32_t n, int32_t *got, void *ov);
int32_t WriteFile(int64_t *h, const void *buf, int32_t n, int32_t *got, void *ov);
int32_t CloseHandle(int64_t *h);
int32_t SetFilePointer(int64_t *h, int32_t off, int32_t *hi, int32_t whence);
int32_t GetFileType(int64_t *h);
int64_t *GetStdHandle(int32_t n);
int32_t GetLastError(void);
void SetLastError(int32_t e);
int32_t GetVersion(void);
int32_t GetVersionExA(struct _OSVERSIONINFOA *v);
int32_t GetCommandLineA(void);
int16_t *GetCommandLineW(void);
int32_t GetModuleFileNameA(void *m, char *b, int32_t n);
int64_t *GetModuleHandleA(char *n);
int64_t *LoadLibraryA(char *n);
int32_t (*GetProcAddress(int64_t *m, char *n))(void);
void ExitProcess(int32_t c);
void RtlUnwind(int64_t *a, int64_t *b, struct _EXCEPTION_RECORD *c, int64_t *d);
int32_t TlsAlloc(void);
int32_t TlsFree(int32_t i);
int64_t TlsGetValue(int32_t i);
int32_t TlsSetValue(int32_t i, int64_t v);
int32_t InterlockedIncrement(int32_t *p);
int32_t InterlockedDecrement(int32_t *p);
void InitializeCriticalSection(struct _RTL_CRITICAL_SECTION *c);
void DeleteCriticalSection(struct _RTL_CRITICAL_SECTION *c);
void EnterCriticalSection(struct _RTL_CRITICAL_SECTION *c);
void LeaveCriticalSection(struct _RTL_CRITICAL_SECTION *c);
int32_t TryEnterCriticalSection(struct _RTL_CRITICAL_SECTION *c);
int32_t FlushFileBuffers(int64_t *h);
int32_t SetEndOfFile(int64_t *h);
int32_t DeleteFileA(char *n);
int32_t SetUnhandledExceptionFilter(int32_t a);
int32_t GetCurrentThreadId(void);
int32_t GetStartupInfoA(struct _STARTUPINFOA *s);
int32_t GetCPInfo(int32_t a, struct _cpinfo *c);
int32_t GetEnvironmentStrings(void);
int32_t GetEnvironmentStringsW(void);
int32_t FreeEnvironmentStringsA(int32_t a);
int32_t FreeEnvironmentStringsW(int32_t a);
int32_t GetEnvironmentVariableA(int32_t a, int32_t b, int32_t c);
int32_t SetHandleCount(int32_t n);
int32_t SetStdHandle(int32_t n, int32_t h);
int32_t GetStringTypeA(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e);
int32_t GetStringTypeW(int32_t a, int32_t b, int32_t c, int32_t d);
int32_t LCMapStringA(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f);
int32_t LCMapStringW(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f);
int32_t MultiByteToWideChar(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f);
int32_t WideCharToMultiByte(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f, int32_t g, int32_t h);
int32_t IsBadCodePtr(int32_t a);
int32_t IsBadReadPtr(int32_t a, int32_t b);
int32_t IsBadWritePtr(int32_t a, int32_t b);
void Sleep(int32_t ms);
int32_t GetFileAttributesW(int16_t *n);
int32_t SetFileTime(int64_t *h, void *a, void *b, struct _FILETIME *c);
int64_t *CreateEventA(struct _SECURITY_ATTRIBUTES *s, bool m, bool i, char *n);
int64_t *CreateSemaphoreA(struct _SECURITY_ATTRIBUTES *s, int32_t a, int32_t b, char *n);
int32_t ReleaseSemaphore(int64_t *h, int32_t a, int32_t *b);
int32_t SetEvent(int64_t *h);
int32_t ResetEvent(int64_t *h);
int32_t WaitForSingleObject(int64_t *h, int32_t ms);
int32_t WaitForMultipleObjects(int32_t n, int64_t **h, bool all, int32_t ms);
int32_t GetTickCount(void);
int32_t timeGetTime(void);
int32_t QueryPerformanceCounter(int64_t *p);
int32_t GetCurrentProcessId(void);
int64_t *GetCurrentProcess(void);
int64_t *GetCurrentThread(void);
int32_t GetProcessAffinityMask(int64_t *h, int64_t *a, int64_t *s);
int32_t SetProcessAffinityMask(int64_t *h, int64_t m);
int32_t GetProcessTimes(int64_t *h, void *a, void *b, void *c, void *d);
int32_t GetThreadPriority(int64_t *h);
int32_t SetThreadPriority(int64_t *h, int32_t p);
int32_t GetThreadContext(int64_t *h, struct _CONTEXT *c);
int32_t SetThreadContext(int64_t *h, struct _CONTEXT *c);
int32_t SuspendThread(int64_t *h);
int32_t ResumeThread(int64_t *h);
int32_t TerminateProcess(int64_t *h, int32_t c);
int32_t DuplicateHandle(int64_t *a, int64_t *b, int64_t *c, int64_t **d, int32_t e, bool f, int32_t g);
int32_t GetHandleInformation(int64_t *h, int32_t *f);
int32_t CreateDirectoryW(int16_t *n, struct _SECURITY_ATTRIBUTES *s);
int32_t SetCurrentDirectoryA(char *n);
int32_t SetCurrentDirectoryW(int16_t *n);
int32_t GetFullPathNameW(int16_t *n, int32_t c, int16_t *b, int16_t **f);
int32_t GetTempPathW(int32_t n, int16_t *b);
int32_t GetTempFileNameW(int16_t *p, int16_t *x, int32_t u, int16_t *b);
int32_t SetFileAttributesW(int16_t *n, int32_t a);
int64_t *FindFirstFileW(int16_t *n, struct _WIN32_FIND_DATAW *d);
int32_t FindNextFileW(int64_t *h, struct _WIN32_FIND_DATAW *d);
int32_t FindClose(int64_t *h);
int32_t FileTimeToLocalFileTime(struct _FILETIME *a, struct _FILETIME *b);
int32_t FileTimeToSystemTime(struct _FILETIME *a, struct _SYSTEMTIME *b);
void GetSystemTimeAsFileTime(struct _FILETIME *t);
int32_t GetConsoleScreenBufferInfo(int64_t *h, struct _CONSOLE_SCREEN_BUFFER_INFO *i);
int32_t SetConsoleTitleA(char *t);
int32_t IsDebuggerPresent(void);
void OutputDebugStringA(char *s);
void RaiseException(int32_t a, int32_t b, int32_t c, int64_t *d);
int32_t UnhandledExceptionFilter(struct _EXCEPTION_POINTERS *p);
int32_t VirtualProtect(void *a, int64_t n, int32_t p, int32_t *o);
int32_t VirtualQuery(void *a, void *b, int64_t n);
int16_t **CommandLineToArgvW(int16_t *c, int32_t *n);
int64_t *VirtualAlloc(void *a, int64_t n, int32_t t, int32_t p);
int32_t VirtualFree(void *a, int64_t n, int32_t t);
int64_t *HeapAlloc(int64_t *h, int32_t f, int64_t n);
int32_t HeapFree(int64_t *h, int32_t f, void *p);
int64_t *HeapReAlloc(int64_t *h, int32_t f, void *p, int64_t n);
int64_t *HeapCreate(int32_t a, int64_t b, int64_t c);
int32_t HeapDestroy(int64_t *h);

#ifdef __cplusplus
}
#endif
#endif
