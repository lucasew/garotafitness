[xdelta3](https://github.com/jmacd/xdelta), tag `v3.1.0`, commit
`4b4aed71a959fe11852e45242bb6524be85d3709` (Apache 2.0, `COPYING`).

The source and headers are unmodified. `xdelta/guest/main.c` wraps the memory
decoder, with checksums enabled. The Guest omits secondary compressors; the
RimWorld bundle patches have no secondary compression or custom code table.
