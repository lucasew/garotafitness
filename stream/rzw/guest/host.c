#include "host.h"

uint8_t pe_image[PE_IMAGE_SIZE];
int64_t g_self;

#include "image.inc"

static const uint8_t *g_src;
static uint32_t g_slen, g_spos;
static uint8_t *g_dst;
static uint32_t g_dcap, g_dpos;
static int32_t g_fs0;
static int32_t g_last_err;
static int64_t g_gs48;

void pe_image_init(void) {
    memcpy(pe_image, kPeImageInit, PE_IMAGE_SIZE);
    for (size_t i = 0; i + 8 <= PE_IMAGE_SIZE; i += 8) {
        uint64_t v;
        memcpy(&v, pe_image + i, 8);
        if (v >= PE_IMAGE_VA0 && v < PE_IMAGE_VA0 + PE_IMAGE_SIZE) {
            v = (uint64_t)(uintptr_t)(pe_image + (v - PE_IMAGE_VA0));
            memcpy(pe_image + i, &v, 8);
        }
    }
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

void *host_xmalloc(size_t n) {
    if (n == 0) n = 1;
    return calloc(1, n);
}

int64_t __readgsqword(int32_t off) { (void)off; return g_gs48; }
int32_t __readfsdword(int32_t off) { (void)off; return g_fs0; }
void __writefsdword(int32_t off, int32_t val) { (void)off; g_fs0 = val; }

#define H_IN  ((int64_t *)1)
#define H_OUT ((int64_t *)2)

int64_t *CreateFileA(char *n, int32_t a, int32_t b, struct _SECURITY_ATTRIBUTES *s, int32_t c, int32_t d, void *t) {
    (void)a; (void)b; (void)s; (void)c; (void)d; (void)t;
    if (n && (n[0] == 'i' || n[0] == 'I')) return H_IN;
    return H_OUT;
}
int64_t *CreateFileW(int16_t *n, int32_t a, int32_t b, struct _SECURITY_ATTRIBUTES *s, int32_t c, int32_t d, void *t) {
    (void)n; (void)a; (void)b; (void)s; (void)c; (void)d; (void)t;
    return H_IN;
}
int32_t ReadFile(int64_t *h, void *buf, int32_t n, int32_t *got, void *ov) {
    (void)h; (void)ov;
    uint32_t want = n < 0 ? 0 : (uint32_t)n;
    if (want > g_slen - g_spos) want = g_slen - g_spos;
    if (buf && want) memcpy(buf, g_src + g_spos, want);
    g_spos += want;
    if (got) *got = (int32_t)want;
    return 1;
}
int32_t WriteFile(int64_t *h, const void *buf, int32_t n, int32_t *got, void *ov) {
    (void)h; (void)ov;
    uint32_t want = n < 0 ? 0 : (uint32_t)n;
    if (want > g_dcap - g_dpos) want = g_dcap - g_dpos;
    if (buf && want) memcpy(g_dst + g_dpos, buf, want);
    g_dpos += want;
    if (got) *got = (int32_t)want;
    return 1;
}
int32_t CloseHandle(int64_t *h) { (void)h; return 1; }
int32_t SetFilePointer(int64_t *h, int32_t off, int32_t *hi, int32_t whence) {
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
int32_t GetFileType(int64_t *h) { (void)h; return 1; }
int64_t *GetStdHandle(int32_t n) { return n == -10 ? H_IN : H_OUT; }
int32_t GetLastError(void) { return g_last_err; }
void SetLastError(int32_t e) { g_last_err = e; }
int32_t GetVersion(void) { return 0x00000004; }
int32_t GetVersionExA(struct _OSVERSIONINFOA *v) {
    if (v) { memset(v, 0, sizeof(*v)); v->e1 = 4; }
    return 1;
}
int32_t GetCommandLineA(void) { return 0; }
int16_t *GetCommandLineW(void) { return NULL; }
int32_t GetModuleFileNameA(void *m, char *b, int32_t n) {
    (void)m;
    if (b && n > 0) b[0] = 0;
    return 0;
}
int64_t *GetModuleHandleA(char *n) { (void)n; return NULL; }
int64_t *LoadLibraryA(char *n) { (void)n; return NULL; }
int32_t (*GetProcAddress(int64_t *m, char *n))(void) { (void)m; (void)n; return NULL; }
void ExitProcess(int32_t c) { (void)c; abort(); }
void RtlUnwind(int64_t *a, int64_t *b, struct _EXCEPTION_RECORD *c, int64_t *d) {
    (void)a; (void)b; (void)c; (void)d;
}
int32_t TlsAlloc(void) { return 0; }
int32_t TlsFree(int32_t i) { (void)i; return 1; }
int64_t TlsGetValue(int32_t i) { (void)i; return 0; }
int32_t TlsSetValue(int32_t i, int64_t v) { (void)i; (void)v; return 1; }
int32_t InterlockedIncrement(int32_t *p) { if (!p) return 0; return ++*p; }
int32_t InterlockedDecrement(int32_t *p) { if (!p) return 0; return --*p; }
void InitializeCriticalSection(struct _RTL_CRITICAL_SECTION *c) { (void)c; }
void DeleteCriticalSection(struct _RTL_CRITICAL_SECTION *c) { (void)c; }
void EnterCriticalSection(struct _RTL_CRITICAL_SECTION *c) { (void)c; }
void LeaveCriticalSection(struct _RTL_CRITICAL_SECTION *c) { (void)c; }
int32_t TryEnterCriticalSection(struct _RTL_CRITICAL_SECTION *c) { (void)c; return 1; }
int32_t FlushFileBuffers(int64_t *h) { (void)h; return 1; }
int32_t SetEndOfFile(int64_t *h) { (void)h; return 1; }
int32_t DeleteFileA(char *n) { (void)n; return 1; }
int32_t SetUnhandledExceptionFilter(int32_t a) { (void)a; return 0; }
int32_t GetCurrentThreadId(void) { return 1; }
int32_t GetStartupInfoA(struct _STARTUPINFOA *s) {
    if (s) memset(s, 0, sizeof(*s));
    return 1;
}
int32_t GetCPInfo(int32_t a, struct _cpinfo *c) {
    (void)a;
    if (c) memset(c, 0, sizeof(*c));
    return 1;
}
int32_t GetEnvironmentStrings(void) { return 0; }
int32_t GetEnvironmentStringsW(void) { return 0; }
int32_t FreeEnvironmentStringsA(int32_t a) { (void)a; return 1; }
int32_t FreeEnvironmentStringsW(int32_t a) { (void)a; return 1; }
int32_t GetEnvironmentVariableA(int32_t a, int32_t b, int32_t c) {
    (void)a; (void)b; (void)c;
    return 0;
}
int32_t SetHandleCount(int32_t n) { return n; }
int32_t SetStdHandle(int32_t n, int32_t h) { (void)n; (void)h; return 1; }
int32_t GetStringTypeA(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e) {
    (void)a; (void)b; (void)c; (void)d; (void)e;
    return 1;
}
int32_t GetStringTypeW(int32_t a, int32_t b, int32_t c, int32_t d) {
    (void)a; (void)b; (void)c; (void)d;
    return 1;
}
int32_t LCMapStringA(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f) {
    (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
    return 0;
}
int32_t LCMapStringW(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f) {
    (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
    return 0;
}
int32_t MultiByteToWideChar(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f) {
    (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
    return 0;
}
int32_t WideCharToMultiByte(int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f, int32_t g, int32_t h) {
    (void)a; (void)b; (void)c; (void)d; (void)e; (void)f; (void)g; (void)h;
    return 0;
}
int32_t IsBadCodePtr(int32_t a) { (void)a; return 0; }
int32_t IsBadReadPtr(int32_t a, int32_t b) { (void)a; (void)b; return 0; }
int32_t IsBadWritePtr(int32_t a, int32_t b) { (void)a; (void)b; return 0; }
void Sleep(int32_t ms) { (void)ms; }
int32_t GetFileAttributesW(int16_t *n) { (void)n; return 0x80; }
int32_t SetFileTime(int64_t *h, void *a, void *b, struct _FILETIME *c) {
    (void)h; (void)a; (void)b; (void)c;
    return 1;
}
int64_t *CreateEventA(struct _SECURITY_ATTRIBUTES *s, bool m, bool i, char *n) {
    (void)s; (void)m; (void)i; (void)n;
    return (int64_t *)3;
}
int64_t *CreateSemaphoreA(struct _SECURITY_ATTRIBUTES *s, int32_t a, int32_t b, char *n) {
    (void)s; (void)a; (void)b; (void)n;
    return (int64_t *)4;
}
int32_t ReleaseSemaphore(int64_t *h, int32_t a, int32_t *b) {
    (void)h; (void)a; (void)b;
    return 1;
}
int32_t SetEvent(int64_t *h) { (void)h; return 1; }
int32_t ResetEvent(int64_t *h) { (void)h; return 1; }
int32_t WaitForSingleObject(int64_t *h, int32_t ms) { (void)h; (void)ms; return 0; }
int32_t WaitForMultipleObjects(int32_t n, int64_t **h, bool all, int32_t ms) {
    (void)n; (void)h; (void)all; (void)ms;
    return 0;
}
int32_t GetTickCount(void) { return 0; }
int32_t timeGetTime(void) { return 0; }
int32_t QueryPerformanceCounter(int64_t *p) {
    if (p) *p = 0;
    return 1;
}
int32_t GetCurrentProcessId(void) { return 1; }
int64_t *GetCurrentProcess(void) { return (int64_t *)5; }
int64_t *GetCurrentThread(void) { return (int64_t *)6; }
int32_t GetProcessAffinityMask(int64_t *h, int64_t *a, int64_t *s) {
    (void)h;
    if (a) *a = 1;
    if (s) *s = 1;
    return 1;
}
int32_t SetProcessAffinityMask(int64_t *h, int64_t m) { (void)h; (void)m; return 1; }
int32_t GetProcessTimes(int64_t *h, void *a, void *b, void *c, void *d) {
    (void)h; (void)a; (void)b; (void)c; (void)d;
    return 1;
}
int32_t GetThreadPriority(int64_t *h) { (void)h; return 0; }
int32_t SetThreadPriority(int64_t *h, int32_t p) { (void)h; (void)p; return 1; }
int32_t GetThreadContext(int64_t *h, struct _CONTEXT *c) { (void)h; (void)c; return 1; }
int32_t SetThreadContext(int64_t *h, struct _CONTEXT *c) { (void)h; (void)c; return 1; }
int32_t SuspendThread(int64_t *h) { (void)h; return 0; }
int32_t ResumeThread(int64_t *h) { (void)h; return 1; }
int32_t TerminateProcess(int64_t *h, int32_t c) { (void)h; (void)c; return 1; }
int32_t DuplicateHandle(int64_t *a, int64_t *b, int64_t *c, int64_t **d, int32_t e, bool f, int32_t g) {
    (void)a; (void)c; (void)e; (void)f; (void)g;
    if (d) *d = b;
    return 1;
}
int32_t GetHandleInformation(int64_t *h, int32_t *f) {
    (void)h;
    if (f) *f = 0;
    return 1;
}
int32_t CreateDirectoryW(int16_t *n, struct _SECURITY_ATTRIBUTES *s) {
    (void)n; (void)s;
    return 1;
}
int32_t SetCurrentDirectoryA(char *n) { (void)n; return 1; }
int32_t SetCurrentDirectoryW(int16_t *n) { (void)n; return 1; }
int32_t GetFullPathNameW(int16_t *n, int32_t c, int16_t *b, int16_t **f) {
    (void)n; (void)c; (void)b; (void)f;
    return 0;
}
int32_t GetTempPathW(int32_t n, int16_t *b) {
    (void)n;
    if (b) b[0] = 0;
    return 0;
}
int32_t GetTempFileNameW(int16_t *p, int16_t *x, int32_t u, int16_t *b) {
    (void)p; (void)x; (void)u;
    if (b) b[0] = 0;
    return 0;
}
int32_t SetFileAttributesW(int16_t *n, int32_t a) { (void)n; (void)a; return 1; }
int64_t *FindFirstFileW(int16_t *n, struct _WIN32_FIND_DATAW *d) {
    (void)n; (void)d;
    return (int64_t *)-1;
}
int32_t FindNextFileW(int64_t *h, struct _WIN32_FIND_DATAW *d) { (void)h; (void)d; return 0; }
int32_t FindClose(int64_t *h) { (void)h; return 1; }
int32_t FileTimeToLocalFileTime(struct _FILETIME *a, struct _FILETIME *b) {
    if (a && b) *b = *a;
    return 1;
}
int32_t FileTimeToSystemTime(struct _FILETIME *a, struct _SYSTEMTIME *b) {
    (void)a;
    if (b) memset(b, 0, sizeof(*b));
    return 1;
}
void GetSystemTimeAsFileTime(struct _FILETIME *t) {
    if (t) memset(t, 0, sizeof(*t));
}
int32_t GetConsoleScreenBufferInfo(int64_t *h, struct _CONSOLE_SCREEN_BUFFER_INFO *i) {
    (void)h;
    if (i) memset(i, 0, sizeof(*i));
    return 1;
}
int32_t SetConsoleTitleA(char *t) { (void)t; return 1; }
int32_t IsDebuggerPresent(void) { return 0; }
void OutputDebugStringA(char *s) { (void)s; }
void RaiseException(int32_t a, int32_t b, int32_t c, int64_t *d) {
    (void)a; (void)b; (void)c; (void)d;
}
int32_t UnhandledExceptionFilter(struct _EXCEPTION_POINTERS *p) { (void)p; return 0; }
int32_t VirtualProtect(void *a, int64_t n, int32_t p, int32_t *o) {
    (void)a; (void)n; (void)p;
    if (o) *o = 0;
    return 1;
}
int32_t VirtualQuery(void *a, void *b, int64_t n) { (void)a; (void)b; (void)n; return 0; }
int16_t **CommandLineToArgvW(int16_t *c, int32_t *n) {
    (void)c;
    if (n) *n = 0;
    return NULL;
}
int64_t *VirtualAlloc(void *a, int64_t n, int32_t t, int32_t p) {
    (void)a; (void)t; (void)p;
    return (int64_t *)(n > 0 ? calloc(1, (size_t)n) : NULL);
}
int32_t VirtualFree(void *a, int64_t n, int32_t t) {
    (void)n; (void)t;
    free(a);
    return 1;
}
int64_t *HeapAlloc(int64_t *h, int32_t f, int64_t n) {
    (void)h; (void)f;
    return (int64_t *)(n > 0 ? calloc(1, (size_t)n) : NULL);
}
int32_t HeapFree(int64_t *h, int32_t f, void *p) {
    (void)h; (void)f;
    free(p);
    return 1;
}
int64_t *HeapReAlloc(int64_t *h, int32_t f, void *p, int64_t n) {
    (void)h; (void)f;
    return (int64_t *)realloc(p, n > 0 ? (size_t)n : 0);
}
int64_t *HeapCreate(int32_t a, int64_t b, int64_t c) {
    (void)a; (void)b; (void)c;
    return (int64_t *)1;
}
int32_t HeapDestroy(int64_t *h) { (void)h; return 1; }

int64_t __C_specific_handler(void) { return 0; }
int32_t (*__dllonexit(int32_t (*a1)(), void (***a2)(), void (***a3)()))() {
    (void)a1; (void)a2; (void)a3;
    return NULL;
}
int32_t __getmainargs(int32_t *a1, char ***a2, char ***a3, int32_t a4, int64_t *a5) {
    (void)a1; (void)a2; (void)a3; (void)a4; (void)a5;
    return 0;
}
int64_t __iob_func(void) { return 0; }
int64_t __lconv_init(void) { return 0; }
void __set_app_type(int32_t a1) { (void)a1; }
void __setusermatherr(int64_t a1) { (void)a1; }
void _amsg_exit(int32_t a1) { (void)a1; }
void _cexit(void) {}
void _endthreadex(int32_t a1) { (void)a1; }
void _initterm(void (**a1)(), void (**a2)()) { (void)a1; (void)a2; }
void _lock(int32_t a1) { (void)a1; }
void _unlock(int32_t a1) { (void)a1; }
int32_t _vsnwprintf(int16_t *a1, int32_t a2, int16_t *a3, int64_t a4) {
    (void)a1; (void)a2; (void)a3; (void)a4;
    return 0;
}
int32_t _vswprintf(int16_t *a1, int16_t *a2, int64_t a3) {
    (void)a1; (void)a2; (void)a3;
    return 0;
}
int16_t *_wsetlocale(int32_t a1, int16_t *a2) { (void)a1; (void)a2; return NULL; }

int64_t *AddVectoredExceptionHandler(int32_t a1, int32_t (*a2)(struct _EXCEPTION_POINTERS *)) {
    (void)a1; (void)a2;
    return (int64_t *)1;
}
int32_t RemoveVectoredExceptionHandler(int64_t *a1) { (void)a1; return 1; }
bool GlobalMemoryStatusEx(struct _MEMORYSTATUSEX *a1) {
    if (a1) {
        memset(a1, 0, sizeof(*a1));
        a1->e0 = (int32_t)sizeof(*a1);
        a1->e2 = 1ll << 32;
        a1->e3 = 1ll << 32;
    }
    return true;
}
bool PathMatchSpecW(int16_t *a1, int16_t *a2) { (void)a1; (void)a2; return true; }
char RtlAddFunctionTable(struct _IMAGE_RUNTIME_FUNCTION_ENTRY *a1, int32_t a2, int64_t a3) {
    (void)a1; (void)a2; (void)a3;
    return 1;
}
void RtlCaptureContext(struct _CONTEXT *a1) { (void)a1; }
struct _IMAGE_RUNTIME_FUNCTION_ENTRY *RtlLookupFunctionEntry(int64_t a1, int64_t *a2, struct _UNWIND_HISTORY_TABLE *a3) {
    (void)a1; (void)a2; (void)a3;
    return NULL;
}
int64_t (*RtlVirtualUnwind(int32_t a1, int64_t a2, int64_t a3, struct _IMAGE_RUNTIME_FUNCTION_ENTRY *a4, struct _CONTEXT *a5, int64_t **a6, int64_t *a7, struct _KNONVOLATILE_CONTEXT_POINTERS *a8))(struct _EXCEPTION_RECORD *, int64_t *, struct _CONTEXT *, int64_t *) {
    (void)a1; (void)a2; (void)a3; (void)a4; (void)a5; (void)a6; (void)a7; (void)a8;
    return NULL;
}
int64_t function_43c718(int64_t a1, int64_t *a2, int64_t a3) {
    (void)a1; (void)a2; (void)a3;
    return 0;
}
int64_t function_43c798(int32_t a1, int64_t a2, int64_t a3, int64_t a4) {
    (void)a1; (void)a2; (void)a3; (void)a4;
    return 0;
}
int32_t _wcsicmp(int16_t *a, int16_t *b) {
    if (!a || !b) return 0;
    while (*a && *b && *a == *b) { a++; b++; }
    return (int32_t)*a - (int32_t)*b;
}
int16_t *_wcslwr(int16_t *s) {
    if (!s) return s;
    for (int16_t *p = s; *p; p++) {
        if (*p >= 'A' && *p <= 'Z') *p = (int16_t)(*p + 32);
    }
    return s;
}
char *_ultoa(uint32_t v, char *buf, int32_t base) {
    if (!buf) return buf;
    if (base < 2) base = 10;
    char tmp[40];
    int n = 0;
    uint32_t x = v;
    do {
        int d = (int)(x % (uint32_t)base);
        tmp[n++] = (char)(d < 10 ? '0' + d : 'a' + d - 10);
        x /= (uint32_t)base;
    } while (x && n < 39);
    int i = 0;
    while (n) buf[i++] = tmp[--n];
    buf[i] = 0;
    return buf;
}
void __writeNullptrQword(int64_t a) { (void)a; }
static int32_t g_errno_val;
int32_t *_errno(void) { return &g_errno_val; }
int32_t __readfsbyte(int32_t off) { (void)off; return 0; }
void __frontend_reg_store_fpr(int64_t a, ...) { (void)a; }
void *_onexit(void *f) { return f; }
char *_strdup(const char *s) {
    if (!s) return NULL;
    size_t n = strlen(s) + 1;
    char *p = (char *)malloc(n);
    if (p) memcpy(p, s, n);
    return p;
}
