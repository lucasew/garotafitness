#ifdef __EMSCRIPTEN__
#define __declspec(x)
#endif
#ifndef O_BINARY
#define O_BINARY 0
#endif
#include <assert.h>
#include <unistd.h>
static inline int setmode(int fd, int mode) {
	(void)fd;
	(void)mode;
	return 0;
}
