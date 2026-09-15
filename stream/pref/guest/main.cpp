// Official Precomp restore. Go writes in.pcf and reads out.bin.
#include "precomp_dll.h"

#include <cstring>

extern "C" int pref_restore(void) {
	Switches sw;
	char msg[256];
	std::memset(msg, 0, sizeof msg);
	char in_name[] = "in.pcf";
	char out_name[] = "out.bin";
	return recompress_file(in_name, out_name, msg, sw) ? 0 : 1;
}
