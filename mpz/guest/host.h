/* Mechanical host for RetDec FULL.c (MpzSlimmer). INV-03: no PE execution. */
#ifndef MPZ_HOST_H
#define MPZ_HOST_H

#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef int64_t int80_t;
typedef int64_t int128_t;
typedef float float32_t;
typedef double float64_t;
typedef double float80_t;

#ifndef NULL
#define NULL ((void *)0)
#endif

/* Win32 / CRT stand-ins (decode path only). */
struct _EXCEPTION_RECORD { int32_t e0; int32_t e1; struct _EXCEPTION_RECORD * e2; int32_t * e3; int32_t e4; int32_t e5[1]; };
struct _M128A { int32_t e0; int64_t e1; };
struct _CONTEXT {
    int32_t e0,e1,e2,e3,e4,e5,e6,e7;
    int16_t e8,e9,e10,e11,e12,e13;
    int32_t e14,e15,e16,e17,e18,e19,e20,e21,e22,e23,e24,e25,e26,e27,e28,e29,e30,e31,e32,e33,e34,e35,e36,e37,e38;
    struct _M128A e39[26];
    int32_t e40,e41,e42,e43,e44,e45;
};
struct _EXCEPTION_POINTERS { struct _EXCEPTION_RECORD * e0; struct _CONTEXT * e1; };
struct _LIST_ENTRY { struct _LIST_ENTRY * e0; struct _LIST_ENTRY * e1; };
struct _RTL_CRITICAL_SECTION_DEBUG;
struct _RTL_CRITICAL_SECTION { struct _RTL_CRITICAL_SECTION_DEBUG * e0; int32_t e1; int32_t e2; int32_t * e3; int32_t * e4; int32_t e5; };
struct _RTL_CRITICAL_SECTION_DEBUG { int16_t e0; int16_t e1; struct _RTL_CRITICAL_SECTION * e2; struct _LIST_ENTRY e3; int32_t e4; int32_t e5; int32_t e6; int16_t e7; int16_t e8; };
struct _OSVERSIONINFOA { int32_t e0,e1,e2,e3,e4; char e5[128]; };
struct _OVERLAPPED { int32_t e0,e1,e2; int32_t * e3; };
struct _SECURITY_ATTRIBUTES { int32_t e0; int32_t * e1; bool e2; };
struct _STARTUPINFOA {
    int32_t e0; char * e1,*e2,*e3; int32_t e4,e5,e6,e7,e8,e9,e10,e11; int16_t e12,e13; char * e14; int32_t * e15,*e16,*e17;
};
struct _cpinfo { int32_t e0; char e1[1]; char e2[1]; };
struct struct1 { int32_t e0,e1,e2,e3; };
struct struct2 { int32_t e0,e1,e2,e3; };
struct struct3 { int32_t e0,e1,e2,e3; };
struct struct4 { int32_t e0,e1,e2,e3; };
struct struct5 { int32_t e0,e1,e2,e3; };
struct struct6 { int32_t e0,e1,e2,e3; };
struct struct7 { int32_t e0,e1,e2,e3; };
struct struct8 { int32_t e0,e1,e2,e3; };

#define PE_IMAGE_VA0 0x10024000u
#define PE_IMAGE_SIZE 0x15000u
extern uint8_t pe_image[PE_IMAGE_SIZE];
extern int32_t g_self;
extern int32_t g172, g173, g174, g175, g176, g177;

void pe_image_init(void);
void host_set_files(const uint8_t *src, uint32_t slen, uint8_t *dst, uint32_t dcap);
uint32_t host_written(void);

int32_t __readfsdword(int32_t off);
void __writefsdword(int32_t off, int32_t val);
void __asm_rep_movsd_memcpy(char *dst, char *src, int32_t ndwords);
int32_t __asm_sti(void);
int32_t __asm_fnclex(void);
int32_t __asm_in(int32_t a, ...);
int32_t __asm_insb(int32_t a);
int32_t __asm_int3(void);
int32_t __asm_iretd(void);
int32_t __asm_maskmovq(int32_t a, int32_t b);
int32_t __asm_out(int32_t a, int32_t b);
int32_t __asm_out_6(int32_t a, int32_t b);
int32_t __asm_str(void);

int32_t * CreateFileA(char *n, int32_t a, int32_t b, struct _SECURITY_ATTRIBUTES *s, int32_t c, int32_t d, void *t);
int32_t ReadFile(int32_t *h, int32_t *buf, int32_t n, int32_t *got, void *ov);
int32_t WriteFile(int32_t *h, const int32_t *buf, int32_t n, int32_t *got, void *ov);
int32_t CloseHandle(int32_t *h);
int32_t SetFilePointer(int32_t *h, int32_t off, int32_t *hi, int32_t whence);
int32_t GetFileType(int32_t *h);
int32_t * GetStdHandle(int32_t n);
int32_t * VirtualAlloc(void *a, int32_t n, int32_t t, int32_t p);
int32_t VirtualFree(void *a, int32_t n, int32_t t);
int32_t * HeapAlloc(int32_t *h, int32_t f, int32_t n);
int32_t HeapFree(int32_t *h, int32_t f, int32_t *p);
int32_t * HeapReAlloc(int32_t *h, int32_t f, int32_t *p, int32_t n);
int32_t * HeapCreate(int32_t a, int32_t b, int32_t c);
int32_t HeapDestroy(int32_t *h);
int32_t GetLastError(void);
void SetLastError(int32_t e);
int32_t GetVersion(void);
int32_t GetVersionExA(struct _OSVERSIONINFOA *v);
int32_t GetCommandLineA(void);
int32_t GetModuleFileNameA(void *m, char *b, int32_t n);
int32_t * GetModuleHandleA(char *n);
int32_t * LoadLibraryA(char *n);
int32_t (*GetProcAddress(int32_t *m, char *n))(void);
void ExitProcess(int32_t c);
void RtlUnwind(int32_t *a, int32_t *b, struct _EXCEPTION_RECORD *c, int32_t *d);
int32_t TlsAlloc(void);
int32_t TlsFree(int32_t i);
int32_t TlsGetValue(int32_t i);
int32_t TlsSetValue(int32_t i, int32_t v);
int32_t InterlockedIncrement(int32_t *p);
int32_t InterlockedDecrement(int32_t *p);
void InitializeCriticalSection(struct _RTL_CRITICAL_SECTION *c);
void DeleteCriticalSection(struct _RTL_CRITICAL_SECTION *c);
void EnterCriticalSection(struct _RTL_CRITICAL_SECTION *c);
void LeaveCriticalSection(struct _RTL_CRITICAL_SECTION *c);
int32_t FlushFileBuffers(int32_t *h);
int32_t SetEndOfFile(int32_t *h);
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

#ifdef __cplusplus
}
#endif
#endif
