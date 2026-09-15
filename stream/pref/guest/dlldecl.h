#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif
#ifdef __EMSCRIPTEN__
#define __declspec(x)
#endif
#ifndef O_BINARY
#define O_BINARY 0
#endif
#include <assert.h>
#include <stdio.h>
#include <unistd.h>
#ifdef __cplusplus
extern "C" {
#endif
FILE *pref_fopen(const char *path, const char *mode);
int pref_remove(const char *path);
#ifdef __cplusplus
}
#endif
#ifndef PREF_MEMFS
#define fopen pref_fopen
#define remove pref_remove
#endif
static inline int setmode(int fd, int mode) {
	(void)fd;
	(void)mode;
	return 0;
}
