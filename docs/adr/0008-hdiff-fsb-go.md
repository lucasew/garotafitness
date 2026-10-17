# ADR-0008: uncompressed HDIFF and FSB5 remux in Go

Status: accepted

`reconstruct/x5` applies uncompressed HDIFF13 in Go. `reconstruct/x5n` applies
the uncompressed HDIFFSF20 stream inside an HDIFF19 directory patch in Go.
Behavior follows `third_party/hdiffpatch` v2.5.3 and `third_party/hdiffpatch4`
v4.8.0 with a null decompressor, including the bounds checks those `patch.c`
files compile in.

`reconstruct/fsb` remuxes Ogg Vorbis to FSB5 in Go. Packet framing follows
`third_party/oggvorbis2fsb5` and libogg. The guest was invoked as
`("fsb", "-", "-")`, so the loop-point chunk is not written.

Rejected: keeping the wazero guests on those paths. Rejected: a Go port of
the compressed HDiff plugins. The guests already reject a non-empty HDIFF13
compress type and a non-zero HDIFFSF20 `compressedSize`, and no committed
fixture exercises a compressed diff.
