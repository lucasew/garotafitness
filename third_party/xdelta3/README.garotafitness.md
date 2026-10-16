[xdelta3](https://github.com/jmacd/xdelta), tag `v3.1.0`, commit
`4b4aed71a959fe11852e45242bb6524be85d3709` (Apache 2.0, `COPYING`).

The source and headers are unmodified. `reconstruct/xdelta` applies uncompressed
VCDIFF in Go, matching this tree with `XD3_USE_LARGESIZET=0`, checksums on, and
secondary compressors off. RimWorld bundle patches have no secondary compression
or custom code table.
