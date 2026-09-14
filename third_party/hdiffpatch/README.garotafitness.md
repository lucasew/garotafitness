[HDiffPatch](https://github.com/sisong/HDiffPatch), tag `v2.5.3`, commit
`fddc5a50ebd396d43475ac51d130888595d5f902`.

These four unmodified files come from `libHDiffPatch/HPatch`. Their MIT license
is included in each file. The wrapper in `reconstruct/x5/guest/main.c` accepts uncompressed
HDIFF13 patches and checks the declared source and target sizes. Build with
`make -C reconstruct/x5` using the project's emscripten toolchain.
