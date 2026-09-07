# ADR-0004: mise conda backend for compilers

Status: accepted

`mise.toml` is the toolchain. Go uses the mise registry. make, g++, clang,
and wasi-sdk use the mise `conda:` backend (`conda:make`, `conda:gxx`, later
`conda:wasi-sdk` or the conda package that provides it).

Rejected a host `g++`. Rejected `nix shell` / `nix develop` for this repo.
Rejected mise registry clang when conda ships the compiler.

Agents compile Guests with `mise exec -- make` (or the conda `g++` on PATH
inside `mise exec`). Do not invent a second toolchain.
