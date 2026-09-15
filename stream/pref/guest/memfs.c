// In-memory FILE* for standalone wasm: emcc fopen/open do not
// import WASI path_open, so named files must stay in the guest.
#define PREF_MEMFS
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

enum { maxSlots = 128, maxName = 256 };

typedef struct {
	char name[maxName];
	uint8_t *data;
	size_t len;
	size_t cap;
	int used;
} slot;

typedef struct {
	slot *s;
	size_t pos;
} cookie;

static slot files[maxSlots];
static uint8_t *outCopy;
static uint32_t outLen;

static slot *findSlot(const char *name) {
	for (int i = 0; i < maxSlots; i++) {
		if (files[i].used && strcmp(files[i].name, name) == 0) {
			return &files[i];
		}
	}
	return NULL;
}

static slot *makeSlot(const char *name) {
	slot *s = findSlot(name);
	if (s != NULL) {
		return s;
	}
	for (int i = 0; i < maxSlots; i++) {
		if (!files[i].used) {
			s = &files[i];
			memset(s, 0, sizeof *s);
			strncpy(s->name, name, maxName - 1);
			s->used = 1;
			return s;
		}
	}
	return NULL;
}

static int grow(slot *s, size_t n) {
	if (n <= s->cap) {
		return 0;
	}
	size_t cap = s->cap ? s->cap : 4096;
	while (cap < n) {
		if (cap > (SIZE_MAX / 2)) {
			return -1;
		}
		cap *= 2;
	}
	uint8_t *p = realloc(s->data, cap);
	if (p == NULL) {
		return -1;
	}
	if (cap > s->cap) {
		memset(p + s->cap, 0, cap - s->cap);
	}
	s->data = p;
	s->cap = cap;
	return 0;
}

static ssize_t cread(void *c, char *buf, size_t n) {
	cookie *k = c;
	if (k->pos >= k->s->len) {
		return 0;
	}
	if (n > k->s->len - k->pos) {
		n = k->s->len - k->pos;
	}
	memcpy(buf, k->s->data + k->pos, n);
	k->pos += n;
	return (ssize_t)n;
}

static ssize_t cwrite(void *c, const char *buf, size_t n) {
	cookie *k = c;
	if (grow(k->s, k->pos + n) != 0) {
		return -1;
	}
	memcpy(k->s->data + k->pos, buf, n);
	k->pos += n;
	if (k->pos > k->s->len) {
		k->s->len = k->pos;
	}
	return (ssize_t)n;
}

static int cseek(void *c, off_t *off, int whence) {
	cookie *k = c;
	int64_t pos = (int64_t)k->pos;
	if (whence == SEEK_CUR) {
		pos += *off;
	} else if (whence == SEEK_END) {
		pos = (int64_t)k->s->len + *off;
	} else {
		pos = *off;
	}
	if (pos < 0) {
		return -1;
	}
	k->pos = (size_t)pos;
	*off = (off_t)k->pos;
	return 0;
}

static int cclose(void *c) {
	free(c);
	return 0;
}

static const char *norm(const char *path) {
	if (path == NULL) {
		return "";
	}
	if (path[0] == '.' && path[1] == '/') {
		return path + 2;
	}
	return path;
}

FILE *pref_fopen(const char *path, const char *mode) {
	const char *name = norm(path);
	int write = mode != NULL && strpbrk(mode, "wa+") != NULL;
	int trunc = mode != NULL && strchr(mode, 'w') != NULL;
	int append = mode != NULL && strchr(mode, 'a') != NULL;
	slot *s = findSlot(name);
	if (s == NULL) {
		if (!write) {
			return NULL;
		}
		s = makeSlot(name);
		if (s == NULL) {
			return NULL;
		}
	}
	if (trunc) {
		s->len = 0;
	}
	cookie *k = malloc(sizeof *k);
	if (k == NULL) {
		return NULL;
	}
	k->s = s;
	k->pos = append ? s->len : 0;
	cookie_io_functions_t fns = {cread, cwrite, cseek, cclose};
	FILE *f = fopencookie(k, mode != NULL ? mode : "rb", fns);
	if (f == NULL) {
		free(k);
	}
	return f;
}

int pref_remove(const char *path) {
	slot *s = findSlot(norm(path));
	if (s == NULL) {
		return 0;
	}
	free(s->data);
	memset(s, 0, sizeof *s);
	return 0;
}

void pref_put(const char *name, const uint8_t *p, uint32_t n) {
	slot *s = makeSlot(norm(name));
	if (s == NULL) {
		return;
	}
	free(s->data);
	s->data = NULL;
	s->len = s->cap = 0;
	if (n == 0) {
		return;
	}
	if (grow(s, n) != 0) {
		return;
	}
	memcpy(s->data, p, n);
	s->len = n;
}

uint32_t pref_take(const char *name, uint8_t **ptr) {
	slot *s = findSlot(norm(name));
	if (s == NULL || s->len == 0) {
		*ptr = NULL;
		return 0;
	}
	free(outCopy);
	outCopy = malloc(s->len);
	if (outCopy == NULL) {
		*ptr = NULL;
		return 0;
	}
	memcpy(outCopy, s->data, s->len);
	outLen = (uint32_t)s->len;
	*ptr = outCopy;
	return outLen;
}

uint8_t *pref_out_ptr(void) { return outCopy; }
uint32_t pref_out_len(void) { return outLen; }
