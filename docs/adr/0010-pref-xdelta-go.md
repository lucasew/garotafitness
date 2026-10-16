# ADR-0010: pref and xdelta in Go

Status: accepted

`reconstruct/xdelta` applies uncompressed VCDIFF in Go. Behavior follows
`third_party/xdelta3` v3.1.0 with the old guest build: 32-bit sizes, 64-bit
copy offsets, the default code table, Adler32 on, secondary compressors off.
The New Holland fixture is 133888 bytes, sha256
`bb21747f2de2a59b6b8c7e0b530411dee93e757cf1061b571fd1a39e8a678eb9`.
This is the same decoder as PR 20. Numbered 0010 because ADR-0008 on this
branch is uncompressed HDIFF and FSB5, and ADR-0009 is the earlier native
guest set.

`stream/pref` restores schnaader/precomp-cpp 0.4.8 PCF in Go. The payload may
be wrapped in bzip2 or xz. Uncompressed copies pass through. PDF, ZIP, gzip,
PNG, multi-PNG, SWF, brute, and raw zlib streams are rebuilt with the same
single-thread preflate reencoder as `stream/xt2png`. Base64 is reencoded in
Go. GIF is recompressed from the PGF pixel stream. bzip2 chunks are encoded
with libbzip2 1.0.6. JPG chunks restore packJPG, or Brunsli when the brotli
metadata flag is clear, and can strip the motion-JPEG DHT. MP3 chunks restore
packMP3. A Brunsli metadata section stored with brotli returns an error.

`stream/magic2` and `stream/rzw` were already Go. Their unused wasm modules
are removed, along with `prefdec.wasm` and `internal/wasmrun`. No package
embeds a wasm guest after this change.

Rejected: a from-scratch deflate, cgo, and a subprocess.
