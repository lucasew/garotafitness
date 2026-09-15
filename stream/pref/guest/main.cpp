// Official Precomp restore. Host copies the PCF into guest memory.
#include "precomp_dll.h"

#include <cstdint>
#include <cstdio>
#include <cstring>

extern "C" {
void pref_put(const char *name, const uint8_t *p, uint32_t n);
uint32_t pref_take(const char *name, uint8_t **ptr);
}

extern "C" int pref_restore(const uint8_t *in, uint32_t n) {
	Switches sw;
	char msg[256];
	std::memset(msg, 0, sizeof msg);
	pref_put("/in.pcf", in, n);
	char in_name[] = "/in.pcf";
	char out_name[] = "/out.bin";
	int ok = recompress_file(in_name, out_name, msg, sw) ? 0 : 1;
	if (ok != 0 && msg[0]) {
		std::fprintf(stderr, "%s\n", msg);
	}
	uint8_t *out = nullptr;
	if (ok == 0 && pref_take("/out.bin", &out) == 0) {
		ok = 1;
	}
	return ok;
}
