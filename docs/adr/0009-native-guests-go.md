# ADR-0009: remaining codec guests in Go

Status: accepted

These paths no longer load a wazero guest:

- `stream/srep` runs the Future-LZ kernel in Go, including the match heap and the in-memory spill. This supersedes ADR-0006. That record stays as the history of the wasm split.
- `stream/mpz` runs a mechanical translation of `guest/decoder.c` (the lifted MPZ 5.4.5.1 decoder). `guest/c2go.py` writes `lift.go` and `tables.go`. The 5.4.5.0 complemented-literal path stays in the Go reader.
- `stream/mpzz` decodes with the existing Go OGGRE decoder.
- `stream/tor` decodes FreeArc Tornado byte, bit, Huffman, and arithmetic streams.
- `stream/xt2png` reencodes preflate diffs to deflate in Go.
- `stream/xt3u` recompresses with LZ4 HC and LZ4 fast, and restores zstd frames that use a raw content prefix.
- `reconstruct/sevenz` encodes with a port of 7-Zip 26.02 `LzmaEncode`. `reconstruct/fgpack` encodes with a separate port of LZMA SDK 19.00. The match finders are not shared. `ulikunitz/xz` is not used for these writes. Allocator blocks come from mmap: a C one-past-end pointer, or freeing a block while a live `*T` still names it, makes the Go garbage collector abort.

`stream/pref` stays on `prefdec.wasm`. A faithful `recompress_file` port is not in this change. `reconstruct/xdelta` stays on its wasm guest here; the Go VCDIFF applier is PR 20.

Committed `magic2dec.wasm` and `rzwdec.wasm` are still unused by their readers.

Numbering: ADR-0008 on this branch (PR 21) is uncompressed HDIFF and FSB5 remux. PR 20 also adds an ADR-0008, for xdelta. Whichever of those two merges second has to take the next free number. This record is 0009 because it sits on the HDIFF branch.

Rejected: a from-scratch codec, a third-party LZ4 or zstd library, cgo, and a subprocess.
