# garotafitness

Extract a local FitGirl repack into its installed game tree:

```sh
go build -o bin/garotafitness ./cmd/garotafitness
bin/garotafitness extract '/path/to/repack' /path/to/output
```

`-v` raises slog detail (volumes, reconstruction records, recipe commands;
`-vv` includes each written member). `--help` and `--version` come from
lewkit `x/cmd`.

Keep `setup.exe`, the required `fg-*.bin` volumes, and `MD5/fitgirl-bins.md5`
together in the source directory. The optional soundtrack volume is extracted
when present. `setup.exe` supplies metadata and the installed-file checksum
manifest; it is read as data. Extraction runs entirely inside the Go process.

Reconstruction follows the extraction records recovered from `setup.exe` and
the recipes stored in its volumes. Those records supply archive filters,
working directories, compression parameters, patch inputs, output moves, and
cleanup paths. The extractor checks archive CRCs, patch checksums, and the
installed-file manifest before writing the final tree. Temporary files stay in
memory. MPZ soundtrack decoding can take several minutes.

Supported recipe operations include FSB remuxing, LZMA packing, x2, RTPatch,
HDiffPatch, VCDIFF, file moves, copies, and deletions. The extractor parses these
operations and calls its own implementations; it never runs the batch files or
their executables. Unknown operations and unresolved reconstruction arguments
return errors.

Compiled WASM modules are included. To rebuild the reconstruction modules, use
the compilers in `mise.toml` and run `mise run build:reconstruction`. The MPZ
translation is documented in [mpz/guest/README.md](mpz/guest/README.md), and the
RTPatch format in [x3/README.md](x3/README.md).

RimWorld is the integration-test corpus. Independent fixtures also exercise
different filenames, archive destinations, file counts, and compression options.
Run regression tests with `go test -p 1 ./...`. With the corpus mounted at the
path used in `arc_test.go`, the full extraction test can also be enabled:

```sh
GAROTAFITNESS_FULL_EXTRACT=all go test . -run '^TestExtractInstalledRimWorld$' -timeout=1h
GAROTAFITNESS_FULL_EXTRACT=required go test . -run '^TestExtractInstalledRimWorld$' -timeout=1h
```
