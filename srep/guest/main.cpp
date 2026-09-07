// Hybrid SREP v3 guest: official Future-LZ kernel only.
// Go owns the header, block loop, and output buffer.

#define _FILE_OFFSET_BITS 64

#include "kernel.h"

#include "../../third_party/srep/Compression/SREP/decompress.cpp"

namespace {

const unsigned kVMBlock = 8 * mb;
const unsigned kVMMem = 512 * mb;

MEMORY_MANAGER *mm;
VIRTUAL_MEMORY_MANAGER *vm;
LZ_MATCH_HEAP *lz_matches;
unsigned g_base_len;

} // namespace

extern "C" void srep_close();

extern "C" int srep_open(unsigned base_len) {
  srep_close();
  g_base_len = base_len;
  mm = new MEMORY_MANAGER(kVMMem);
  vm = new VIRTUAL_MEMORY_MANAGER(const_cast<char *>("/vm"), kVMBlock);
  lz_matches = new LZ_MATCH_HEAP();
  FUTURE_LZ_MATCH barrier;
  barrier.dest = Offset(-1);
  lz_matches->insert(barrier);
  return 0;
}

extern "C" int srep_block(void *stat, unsigned stat_bytes, void *lits, unsigned lit_bytes,
                          void *out, unsigned out_bytes, unsigned long long block_start) {
  if (!mm || !vm || !lz_matches || !out)
    return 1;
  if ((stat_bytes > 0 && !stat) || (lit_bytes > 0 && !lits))
    return 1;
  char *statp = static_cast<char *>(stat);
  char *litp = static_cast<char *>(lits);
  char *outp = static_cast<char *>(out);
  const bool ok = decompress_FUTURE_LZ(
      false, g_base_len, nullptr, Offset(block_start), reinterpret_cast<STAT *>(statp),
      reinterpret_cast<STAT *>(statp + stat_bytes), litp, litp + lit_bytes, outp, outp + out_bytes,
      *mm, *vm, *lz_matches, unsigned(-1));
  return ok ? 0 : 1;
}

extern "C" void srep_close() {
  delete lz_matches;
  lz_matches = nullptr;
  delete vm;
  vm = nullptr;
  delete mm;
  mm = nullptr;
}
