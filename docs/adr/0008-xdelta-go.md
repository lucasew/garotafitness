# ADR-0008: xdelta decode in Go

Status: accepted

`reconstruct/xdelta` applies uncompressed VCDIFF in Go. Behavior follows
`third_party/xdelta3` v3.1.0 with the old guest build: 32-bit sizes, 64-bit
copy offsets, the default code table, Adler32 on, secondary compressors off.
`Apply` and `ApplyStream` both use that decoder.

The New Holland fixture pins the xdelta.wasm output: 133888 bytes, sha256
`bb21747f2de2a59b6b8c7e0b530411dee93e757cf1061b571fd1a39e8a678eb9`.

Rejected: keeping the wazero guest on the apply path. Rejected: a Go port of
secondary LZMA or a custom code table. No committed fixture exercises them,
and the header parser still rejects those files.
