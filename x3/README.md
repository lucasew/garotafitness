This package reconstructs the uncompressed, single-source MODIFY records used
by RimWorld's RTPatch update. The patch header identifies format 9.00; its
reader is PATCHW32 11.00, stored at offsets 0x56b1d3–0x5a3b33 of the installer's
decompressed payload. The Go implementation comes from static analysis of
that reader, including opcode dispatch at VA 0x1001874e.

Records carry source and target sizes and two rolling checksums. Both versions
are checked. File names use CP866; the final installed MD5 manifest uses
Windows-1251. These encodings must be decoded separately.

The supported instructions copy source ranges, repeat byte patterns, record
literal gaps, reuse copy templates, and add deltas at relative positions.
Unsupported record flags, compressed records, multiple source versions, invalid
paths, and out-of-bounds instructions return errors. The original DLL is never
loaded. The final installed-file MD5 check in the extractor provides an
independent check after all reconstruction steps.
