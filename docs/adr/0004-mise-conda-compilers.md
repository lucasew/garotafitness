# ADR-0004: mise conda backend for compilers

Status: accepted

`mise.toml` is the toolchain. Go uses the mise registry. make, g++, clang,
and emscripten use the mise `conda:` backend (`conda:make`, `conda:gxx`,
`conda:clang`, `conda:emscripten`). WASM guests compile with `emcc` from
`conda:emscripten` 4.0.9 (same pin as gameweb).

Rejected a host `g++`. Rejected `nix shell` / `nix develop` for this repo.
Rejected mise registry clang when conda ships the compiler. Rejected zig
and conda wasi-sdk (conda has no wasi-sdk).

Agents compile Guests with `mise exec -- make` (or `emcc` on PATH inside
`mise exec`). Do not invent a second toolchain.
