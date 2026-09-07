# ADR-0006: SREP hybrid (C++ kernel, Go plumbing)

Status: accepted

C++ keeps Future-LZ (`decompress_FUTURE_LZ` and the match heap).
Go owns `NewReader`, the block loop, wazero, and the output buffer.
The Guest exports `srep_open` / `srep_block` / `srep_close`. Pthreads
are not linked. A full Go port of the kernel is later-work.

WASM C++ compile uses `conda:emscripten` (`emcc`, same 4.0.9 pin as
gameweb). Native C++ stays `conda:gxx`. `-O0` because conda binaryen
117 rejects emcc 4.0.9 wasm-opt flags.

Rejected: `os/exec` of native srep. Rejected cgo. Rejected zig.
Rejected compiling the full srep CLI to WASM.
