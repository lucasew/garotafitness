[HDiffPatch](https://github.com/sisong/HDiffPatch), tag `v2.5.3`, commit
`fddc5a50ebd396d43475ac51d130888595d5f902`.

These four unmodified files come from `libHDiffPatch/HPatch`. Their MIT license
is included in each file. `reconstruct/x5` applies uncompressed HDIFF13 in Go
(`patch_decompress` with a null decompressor) and keeps the bounds checks in
`patch.c`. A non-empty compress type is rejected.
